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
	// Helper to capture stdout/stderr
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

		stdout := <-outC
		stderr := <-errC

		rOut.Close()
		rErr.Close()

		return stdout, stderr
	}

	tests := []struct {
		name    string
		archive []txtar.File // using slice for deterministic order
		txt     bool
		files   []string
		wantOut string
		wantErr string
	}{
		{
			name: "cat archive (txt=false)",
			archive: []txtar.File{
				{Name: "file1", Data: []byte("content1\n")},
			},
			txt:     false,
			wantOut: "-- file1 --\ncontent1\n", // Rough expectation, will refine if needed
		},
		{
			name: "cat all files (txt=true, no args)",
			archive: []txtar.File{
				{Name: "file1", Data: []byte("content1\n")},
				{Name: "file2", Data: []byte("content2\n")},
			},
			txt:     true,
			files:   []string{},
			wantOut: "content1\ncontent2\n",
		},
		{
			name: "cat specific file",
			archive: []txtar.File{
				{Name: "file1", Data: []byte("content1\n")},
				{Name: "file2", Data: []byte("content2\n")},
			},
			txt:     true,
			files:   []string{"file1"},
			wantOut: "content1\n",
		},
		{
			name: "cat with pattern",
			archive: []txtar.File{
				{Name: "foo.txt", Data: []byte("foo\n")},
				{Name: "bar.txt", Data: []byte("bar\n")},
				{Name: "baz.go", Data: []byte("baz\n")},
			},
			txt:     true,
			files:   []string{"*.txt"},
			wantOut: "foo\nbar\n",
		},
		{
			name: "file not found",
			archive: []txtar.File{
				{Name: "file1", Data: []byte("content1\n")},
			},
			txt:     true,
			files:   []string{"missing"},
			wantOut: "",
			wantErr: "File missing not found in archive\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			archivePath := filepath.Join(tmpDir, "test.txtar")

			// Create archive
			a := new(txtar.Archive)
			a.Files = tt.archive
			rawArchive := txtar.Format(a)
			if err := os.WriteFile(archivePath, rawArchive, 0644); err != nil {
				t.Fatal(err)
			}

			stdout, stderr := captureOutput(func() {
				Cat(archivePath, tt.txt, tt.files...)
			})

			// For txt=false, compare with raw archive content
			if !tt.txt {
				if stdout != string(rawArchive) {
					t.Errorf("got stdout %q, want %q", stdout, string(rawArchive))
				}
			} else {
				if stdout != tt.wantOut {
					t.Errorf("got stdout %q, want %q", stdout, tt.wantOut)
				}
			}

			if stderr != tt.wantErr {
				t.Errorf("got stderr %q, want %q", stderr, tt.wantErr)
			}
		})
	}
}
