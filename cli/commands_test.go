package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"txtar"
)

func TestCat(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.txtar")

	// Create a sample archive
	a := &txtar.Archive{
		Comment: []byte("comment\n"),
		Files: []txtar.File{
			{Name: "file1.txt", Data: []byte("content1\n")},
			{Name: "dir/file2.txt", Data: []byte("content2\n")},
			{Name: "file3.txt", Data: []byte("content3\n")},
		},
	}
	data := txtar.Format(a)
	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Helper to capture output
	captureOutput := func(f func()) (string, string) {
		oldStdout := os.Stdout
		oldStderr := os.Stderr
		rOut, wOut, _ := os.Pipe()
		rErr, wErr, _ := os.Pipe()
		os.Stdout = wOut
		os.Stderr = wErr

		outC := make(chan string)
		errC := make(chan string)

		go func() {
			var buf bytes.Buffer
			io.Copy(&buf, rOut)
			outC <- buf.String()
		}()
		go func() {
			var buf bytes.Buffer
			io.Copy(&buf, rErr)
			errC <- buf.String()
		}()

		f()

		wOut.Close()
		wErr.Close()
		os.Stdout = oldStdout
		os.Stderr = oldStderr

		out := <-outC
		err := <-errC
		return out, err
	}

	tests := []struct {
		name       string
		txt        bool
		files      []string
		wantStdout string
		wantStderr string
	}{
		{
			name:       "cat archive raw",
			txt:        false,
			files:      nil,
			wantStdout: string(data),
			wantStderr: "",
		},
		{
			name:       "cat all files",
			txt:        true,
			files:      nil,
			wantStdout: "content1\ncontent2\ncontent3\n",
			wantStderr: "",
		},
		{
			name:       "cat specific file",
			txt:        true,
			files:      []string{"file1.txt"},
			wantStdout: "content1\n",
			wantStderr: "",
		},
		{
			name:       "cat file in subdir",
			txt:        true,
			files:      []string{"dir/file2.txt"},
			wantStdout: "content2\n",
			wantStderr: "",
		},
		{
			name:       "cat glob pattern",
			txt:        true,
			files:      []string{"file*.txt"},
			wantStdout: "content1\ncontent3\n",
			wantStderr: "",
		},
		{
			name:       "cat non-existent file",
			txt:        true,
			files:      []string{"nonexistent.txt"},
			wantStdout: "",
			wantStderr: "File nonexistent.txt not found in archive\n",
		},
		{
			name:       "cat mixed existing and non-existent",
			txt:        true,
			files:      []string{"file1.txt", "missing.txt"},
			wantStdout: "content1\n",
			wantStderr: "File missing.txt not found in archive\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr := captureOutput(func() {
				Cat(archivePath, tt.txt, tt.files...)
			})

			if stdout != tt.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, tt.wantStdout)
			}
			if stderr != tt.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr, tt.wantStderr)
			}
		})
	}
}
