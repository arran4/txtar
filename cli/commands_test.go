package cli

import (
	"os"
	"path/filepath"
	"testing"
	"txtar"
)

func TestDelete(t *testing.T) {
	// Helper to verify archive content
	verifyArchive := func(t *testing.T, archivePath string, expectedFiles map[string]string) {
		t.Helper()
		a, err := txtar.ParseFile(archivePath)
		if err != nil {
			t.Fatalf("Failed to parse archive: %v", err)
		}

		if len(a.Files) != len(expectedFiles) {
			t.Errorf("Expected %d files, got %d", len(expectedFiles), len(a.Files))
		}

		found := make(map[string]bool)
		for _, f := range a.Files {
			content, ok := expectedFiles[f.Name]
			if !ok {
				t.Errorf("Unexpected file in archive: %s", f.Name)
				continue
			}
			found[f.Name] = true
			if string(f.Data) != content {
				t.Errorf("File %s content mismatch. Got %q, want %q", f.Name, f.Data, content)
			}
		}

		for name := range expectedFiles {
			if !found[name] {
				t.Errorf("Expected file %s not found in archive", name)
			}
		}
	}

	tests := []struct {
		name     string
		initial  map[string]string
		args     []string
		expected map[string]string
	}{
		{
			name: "delete single file",
			initial: map[string]string{
				"file1": "content1\n",
				"file2": "content2\n",
			},
			args: []string{"file1"},
			expected: map[string]string{
				"file2": "content2\n",
			},
		},
		{
			name: "delete multiple files",
			initial: map[string]string{
				"file1": "content1\n",
				"file2": "content2\n",
				"file3": "content3\n",
			},
			args: []string{"file1", "file3"},
			expected: map[string]string{
				"file2": "content2\n",
			},
		},
		{
			name: "delete with wildcard",
			initial: map[string]string{
				"foo.txt": "content1\n",
				"bar.txt": "content2\n",
				"baz.go":  "content3\n",
			},
			args: []string{"*.txt"},
			expected: map[string]string{
				"baz.go": "content3\n",
			},
		},
		{
			name: "delete with wildcard question mark",
			initial: map[string]string{
				"file1": "content1\n",
				"file2": "content2\n",
				"fileA": "content3\n",
			},
			args: []string{"file?"},
			expected: map[string]string{},
		},
		{
			name: "delete non-existent",
			initial: map[string]string{
				"file1": "content1\n",
			},
			args: []string{"non-existent"},
			expected: map[string]string{
				"file1": "content1\n",
			},
		},
		{
			name: "delete mixed patterns",
			initial: map[string]string{
				"a.txt": "A\n",
				"b.txt": "B\n",
				"c.go":  "C\n",
			},
			args: []string{"*.go", "a.txt"},
			expected: map[string]string{
				"b.txt": "B\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup temporary directory for each test case
			tmpDir := t.TempDir()
			archivePath := filepath.Join(tmpDir, "test.txtar")

			// Create initial archive
			a := new(txtar.Archive)
			for name, content := range tt.initial {
				// Note: txtar.Format ensures newline at end of file content if missing.
				// Our test data includes it explicitly to avoid confusion.
				a.Files = append(a.Files, txtar.File{Name: name, Data: []byte(content)})
			}
			data := txtar.Format(a)
			if err := os.WriteFile(archivePath, data, 0644); err != nil {
				t.Fatal(err)
			}

			// Run Delete
			Delete(archivePath, tt.args...)

			// Verify result
			verifyArchive(t, archivePath, tt.expected)
		})
	}
}
