package cli

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestCreate(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "txtar-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create some files
	// dir/
	//   file1.txt
	//   subdir/
	//     file2.txt
	err = os.MkdirAll(tmpDir+"/subdir", 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(tmpDir+"/file1.txt", []byte("content1"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(tmpDir+"/subdir/file2.txt", []byte("content2"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		recursive bool
		trim      bool
		glob      string
		depth     int
		files     []string
		want      []string
		notWant   []string
	}{
		{
			name:      "simple non-recursive",
			recursive: false,
			depth:     -1,
			files:     []string{tmpDir + "/file1.txt"},
			want:      []string{tmpDir + "/file1.txt"},
		},
		{
			name:      "recursive",
			recursive: true,
			depth:     -1,
			files:     []string{tmpDir},
			want:      []string{tmpDir + "/file1.txt", tmpDir + "/subdir/file2.txt"},
		},
		{
			name:      "recursive with trim",
			recursive: true,
			trim:      true,
			depth:     -1,
			files:     []string{tmpDir},
			want:      []string{"file1.txt", "subdir/file2.txt"},
		},
		{
			name:      "depth 0",
			recursive: true,
			depth:     0,
			files:     []string{tmpDir},
			want:      []string{}, // tmpDir itself is a dir, files inside are depth 1
		},
		{
			name:      "depth 1",
			recursive: true,
			depth:     1,
			files:     []string{tmpDir},
			want:      []string{tmpDir + "/file1.txt"},
			notWant:   []string{tmpDir + "/subdir/file2.txt"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			Create(tt.recursive, tt.trim, tt.glob, tt.depth, tt.files...)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			got := buf.String()

			for _, w := range tt.want {
				if !strings.Contains(got, "-- "+w+" --") {
					t.Errorf("expected output to contain %q, but it didn't\nGot:\n%s", w, got)
				}
			}
			for _, nw := range tt.notWant {
				if strings.Contains(got, "-- "+nw+" --") {
					t.Errorf("expected output to NOT contain %q, but it did\nGot:\n%s", nw, got)
				}
			}
		})
	}
}
