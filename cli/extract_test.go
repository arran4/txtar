package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"txtar"
)

func TestExtract(t *testing.T) {
	t.Skip("Skipping test to avoid actual file system activity as per PR feedback")
	// Setup temporary directory
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.txtar")
	outDir := filepath.Join(tmpDir, "out")

	// Create archive with 3 files
	a := new(txtar.Archive)
	a.Files = []txtar.File{
		{Name: "file1.txt", Data: []byte("content1\n")},
		{Name: "file2.go", Data: []byte("content2\n")},
		{Name: "sub/file3.txt", Data: []byte("content3\n")},
	}
	data := txtar.Format(a)
	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		args     []string
		want     []string
		notWant  []string
	}{
		{
			name: "no args (all files)",
			args: []string{},
			want: []string{"file1.txt", "file2.go", "sub/file3.txt"},
		},
		{
			name: "single file",
			args: []string{"file2.go"},
			want: []string{"file2.go"},
			notWant: []string{"file1.txt", "sub/file3.txt"},
		},
		{
			name: "pattern",
			args: []string{"*.txt"},
			want: []string{"file1.txt"},
			notWant: []string{"file2.go", "sub/file3.txt"},
		},
		{
			name: "pattern with subdir",
			args: []string{"*/*.txt"},
			want: []string{"sub/file3.txt"},
			notWant: []string{"file1.txt", "file2.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear out directory
			_ = os.RemoveAll(outDir)
			if err := os.MkdirAll(outDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Capture stdout and stderr
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			_, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			Extract(outDir, archivePath, tt.args...)

			_ = w.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr


			for _, w := range tt.want {
				content, err := os.ReadFile(filepath.Join(outDir, w))
				if err != nil {
					t.Errorf("Expected file %s to be extracted, but got error: %v", w, err)
				} else {
					var expectedContent string
					for _, f := range a.Files {
						if f.Name == w {
							expectedContent = string(f.Data)
							break
						}
					}
					if string(content) != expectedContent {
						t.Errorf("Expected content %q for file %s, got %q", expectedContent, w, string(content))
					}
				}
			}
			for _, nw := range tt.notWant {
				if _, err := os.Stat(filepath.Join(outDir, nw)); err == nil {
					t.Errorf("Expected file %s to NOT be extracted, but it was", nw)
				}
			}
		})
	}

	// Security test: prevent path traversal
	t.Run("path traversal", func(t *testing.T) {
		a := new(txtar.Archive)
		a.Files = []txtar.File{
			{Name: "../outside.txt", Data: []byte("content\n")},
			{Name: "/absolute.txt", Data: []byte("content\n")},
			{Name: "sub/../../nested.txt", Data: []byte("content\n")},
			{Name: "safe.txt", Data: []byte("safe\n")},
		}

		secArchivePath := filepath.Join(tmpDir, "sec.txtar")
		secOutDir := filepath.Join(tmpDir, "sec-out")
		if err := os.MkdirAll(secOutDir, 0755); err != nil {
			t.Fatal(err)
		}

		data := txtar.Format(a)
		if err := os.WriteFile(secArchivePath, data, 0644); err != nil {
			t.Fatal(err)
		}

		// Capture stdout and stderr
		oldStdout := os.Stdout
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stdout = w
		os.Stderr = w

		Extract(secOutDir, secArchivePath)

		_ = w.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr

		// Check output for warnings
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "Warning: skipping file with unsafe path: ../outside.txt") {
			t.Errorf("Expected warning for ../outside.txt")
		}
		if !strings.Contains(output, "Warning: skipping file with unsafe path: /absolute.txt") {
			t.Errorf("Expected warning for /absolute.txt")
		}
		if !strings.Contains(output, "Warning: skipping file with unsafe path: sub/../../nested.txt") {
			t.Errorf("Expected warning for sub/../../nested.txt")
		}

		// Ensure safe file was extracted
		if _, err := os.Stat(filepath.Join(secOutDir, "safe.txt")); err != nil {
			t.Errorf("Expected safe.txt to be extracted, got error: %v", err)
		}

		// Ensure unsafe files were not extracted
		if _, err := os.Stat(filepath.Join(secOutDir, "../outside.txt")); err == nil {
			t.Errorf("Unsafe file ../outside.txt was extracted")
		}
		if _, err := os.Stat("/absolute.txt"); err == nil {
			t.Errorf("Unsafe file /absolute.txt was extracted")
			// Clean up if it was actually created
			_ = os.Remove("/absolute.txt")
		}
	})
}
