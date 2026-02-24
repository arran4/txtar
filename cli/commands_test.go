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
		t.Run(tt.name, func(t *testing.T) {			// Setup temporary directory for each test case
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
		t.Run(tt.name, func(t *testing.T) {			// Capture stdout
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

func TestCat(t *testing.T) {
	// Setup temporary directory
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.txtar")

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
		want     string
		contains bool // if false, exact match required
	}{
		{
			name: "no args (all files)",
			args: []string{},
			want: "content1\ncontent2\ncontent3\n",
		},
		{
			name: "single file",
			args: []string{"file2.go"},
			want: "content2\n",
		},
		{
			name: "multiple files ordered",
			args: []string{"file1.txt", "file2.go"},
			want: "content1\ncontent2\n",
		},
		{
			name: "multiple files reordered",
			args: []string{"file2.go", "file1.txt"},
			want: "content2\ncontent1\n",
		},
		{
			name: "pattern",
			args: []string{"*.txt"},
			want: "content1\n",
		},
		{
			name: "pattern with subdir",
			args: []string{"*/*.txt"},
			want: "content3\n",
		},
		{
			name: "multiple patterns",
			args: []string{"*.txt", "*.go"},
			want: "content1\ncontent2\n",
		},
		{
			name: "overlapping patterns",
			args: []string{"*.txt", "*.txt"},
			want: "content1\ncontent1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			Cat(archivePath, true, tt.args...)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			got := buf.String()

			if got != tt.want {
				t.Errorf("Cat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestComment(t *testing.T) {
	// Helper to create a fresh archive for each test
	setupArchive := func(t *testing.T) string {
		t.Helper()
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.txtar")
		initialComment := "initial comment"
		a := new(txtar.Archive)
		a.Comment = []byte(initialComment)
		data := txtar.Format(a)
		if err := os.WriteFile(archivePath, data, 0644); err != nil {
			t.Fatal(err)
		}
		return archivePath
	}

	// Case 1: Read existing comment
	t.Run("ReadComment", func(t *testing.T) {
		archivePath := setupArchive(t)

		oldStdout := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout = w
		defer func() { os.Stdout = oldStdout }()

		Comment("", "", archivePath)

		w.Close()
		out, _ := io.ReadAll(r)
		got := string(out)

		// txtar format adds a newline to the comment if missing during Format.
		expected := "initial comment\n"
		if got != expected {
			t.Errorf("Expected output %q, got %q", expected, got)
		}
	})

	// Case 2: Set comment from string
	t.Run("SetCommentString", func(t *testing.T) {
		archivePath := setupArchive(t)
		newComment := "new comment from string"
		Comment(newComment, "", archivePath)

		// Verify archive content
		readA, err := txtar.ParseFile(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		expected := newComment + "\n"
		if string(readA.Comment) != expected {
			t.Errorf("Expected comment %q, got %q", expected, string(readA.Comment))
		}
	})

	// Case 3: Set comment from file
	t.Run("SetCommentFile", func(t *testing.T) {
		archivePath := setupArchive(t)
		commentFile := filepath.Join(filepath.Dir(archivePath), "comment.txt")
		fileComment := "comment from file"
		if err := os.WriteFile(commentFile, []byte(fileComment), 0644); err != nil {
			t.Fatal(err)
		}

		Comment("", commentFile, archivePath)

		// Verify archive content
		readA, err := txtar.ParseFile(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		expected := fileComment + "\n"
		if string(readA.Comment) != expected {
			t.Errorf("Expected comment %q, got %q", expected, string(readA.Comment))
		}
	})

	// Case 4: Set comment from stdin
	t.Run("SetCommentStdin", func(t *testing.T) {
		archivePath := setupArchive(t)
		stdinComment := "comment from stdin"
		oldStdin := os.Stdin
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdin = r
		defer func() { os.Stdin = oldStdin }()

		go func() {
			w.Write([]byte(stdinComment))
			w.Close()
		}()

		Comment("", "-", archivePath)

		// Verify archive content
		readA, err := txtar.ParseFile(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		expected := stdinComment + "\n"
		if string(readA.Comment) != expected {
			t.Errorf("Expected comment %q, got %q", expected, string(readA.Comment))
		}
	})
}
