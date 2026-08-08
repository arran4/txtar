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

type mockFSType struct {
	files  map[string][]byte
	dirs   map[string]bool
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func (m *mockFSType) MkdirAll(path string, perm os.FileMode) error {
	m.dirs[path] = true
	return nil
}

func (m *mockFSType) WriteFile(name string, data []byte, perm os.FileMode) error {
	m.files[name] = data
	return nil
}

func (m *mockFSType) Stdout() io.Writer {
	return m.stdout
}

func (m *mockFSType) Stderr() io.Writer {
	return m.stderr
}

func TestExtract(t *testing.T) {
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
			// Initialize mock state
			m := &mockFSType{
				files:  make(map[string][]byte),
				dirs:   make(map[string]bool),
				stdout: new(bytes.Buffer),
				stderr: new(bytes.Buffer),
			}

			extractWithFS(m, false, outDir, archivePath, tt.args...)

			for _, w := range tt.want {
				content, ok := m.files[filepath.Join(outDir, w)]
				if !ok {
					t.Errorf("Expected file %s to be extracted, but it was not", w)
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
				if _, ok := m.files[filepath.Join(outDir, nw)]; ok {
					t.Errorf("Expected file %s to NOT be extracted, but it was", nw)
				}
			}
		})
	}

	// Security test: prevent path traversal
	t.Run("path traversal", func(t *testing.T) {
		// Initialize mock state
		m := &mockFSType{
			files:  make(map[string][]byte),
			dirs:   make(map[string]bool),
			stdout: new(bytes.Buffer),
			stderr: new(bytes.Buffer),
		}

		a := new(txtar.Archive)
		a.Files = []txtar.File{
			{Name: "../outside.txt", Data: []byte("content\n")},
			{Name: "/absolute.txt", Data: []byte("content\n")},
			{Name: "sub/../../nested.txt", Data: []byte("content\n")},
			{Name: "safe.txt", Data: []byte("safe\n")},
		}

		secArchivePath := filepath.Join(tmpDir, "sec.txtar")
		secOutDir := filepath.Join(tmpDir, "sec-out")

		data := txtar.Format(a)
		if err := os.WriteFile(secArchivePath, data, 0644); err != nil {
			t.Fatal(err)
		}

		extractWithFS(m, false, secOutDir, secArchivePath)

		output := m.stderr.String()

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
		if _, ok := m.files[filepath.Join(secOutDir, "safe.txt")]; !ok {
			t.Errorf("Expected safe.txt to be extracted")
		}

		// Ensure unsafe files were not extracted
		if _, ok := m.files[filepath.Join(secOutDir, "../outside.txt")]; ok {
			t.Errorf("Unsafe file ../outside.txt was extracted")
		}
		if _, ok := m.files["/absolute.txt"]; ok {
			t.Errorf("Unsafe file /absolute.txt was extracted")
		}
	})
}
