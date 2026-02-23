package cli

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"txtar"
)

// Helper to create a temp archive
func createArchive(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "archive.txtar")
	err := os.WriteFile(archivePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to create archive: %v", err)
	}
	return archivePath
}

func TestComment_SetCommentString(t *testing.T) {
	archive := createArchive(t, "-- file.txt --\ncontent")
	comment := "This is a comment"

	Comment(comment, "", archive)

	// Verify
	a, err := txtar.ParseFile(archive)
	if err != nil {
		t.Fatalf("failed to parse archive: %v", err)
	}
	// txtar adds a trailing newline
	expected := comment + "\n"
	if string(a.Comment) != expected {
		t.Errorf("expected comment %q, got %q", expected, string(a.Comment))
	}
}

func TestComment_SetCommentFile(t *testing.T) {
	archive := createArchive(t, "-- file.txt --\ncontent")
	commentFile := filepath.Join(filepath.Dir(archive), "comment.txt")
	comment := "Comment from file"
	if err := os.WriteFile(commentFile, []byte(comment), 0644); err != nil {
		t.Fatalf("failed to create comment file: %v", err)
	}

	Comment("", commentFile, archive)

	// Verify
	a, err := txtar.ParseFile(archive)
	if err != nil {
		t.Fatalf("failed to parse archive: %v", err)
	}
	// txtar adds a trailing newline
	expected := comment + "\n"
	if string(a.Comment) != expected {
		t.Errorf("expected comment %q, got %q", expected, string(a.Comment))
	}
}

func TestComment_Stdin(t *testing.T) {
	// Requires redirecting Stdin.
	// We create a pipe and replace os.Stdin.
	// Since os.Stdin is global, this test cannot run in parallel.
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r

	inputComment := "stdin comment"
	go func() {
		defer w.Close()
		io.WriteString(w, inputComment)
	}()

	archive := createArchive(t, "-- file.txt --\ncontent")

	Comment("", "-", archive)

	// Verify
	a, err := txtar.ParseFile(archive)
	if err != nil {
		t.Fatalf("failed to parse archive: %v", err)
	}
	// txtar adds a trailing newline
	expected := inputComment + "\n"
	if string(a.Comment) != expected {
		t.Errorf("expected comment %q, got %q", expected, string(a.Comment))
	}
}

func TestComment_ShowComment(t *testing.T) {
	// Requires capturing stdout.
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	initialComment := "Initial comment\n"
	archive := createArchive(t, initialComment+"-- file.txt --\ncontent")

	// Call Comment (no args)
	Comment("", "", archive)

	w.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read stdout: %v", err)
	}

	if buf.String() != initialComment {
		t.Errorf("expected output %q, got %q", initialComment, buf.String())
	}
}

// Test error cases with subprocess
func TestComment_Errors(t *testing.T) {
	// Check if running as subprocess
	if os.Getenv("TEST_subprocess") == "1" {
		arg := os.Getenv("TEST_ARG")
		archive := os.Getenv("TEST_ARCHIVE")
		switch arg {
		case "both":
			Comment("c", "f", archive)
		case "bad_archive":
			Comment("c", "", "nonexistent.txtar")
		case "bad_file":
			Comment("", "nonexistent.txt", archive)
		}
		// If Comment returns (which it shouldn't on error), we exit 0
		os.Exit(0)
		return
	}

	archive := createArchive(t, "")

	tests := []struct {
		name    string
		arg     string
		archive string
		wantExit bool
	}{
		{"both flags", "both", archive, true},
		{"bad archive", "bad_archive", "nonexistent", true},
		{"bad file", "bad_file", archive, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestComment_Errors")
			cmd.Env = append(os.Environ(), "TEST_subprocess=1", "TEST_ARG="+tc.arg, "TEST_ARCHIVE="+tc.archive)

			// Capture stderr to verify output if needed (optional)
			var stderr bytes.Buffer
			cmd.Stderr = &stderr

			err := cmd.Run()

			if tc.wantExit {
				if e, ok := err.(*exec.ExitError); !ok || e.Success() {
					t.Errorf("expected exit error, got %v. Stderr: %s", err, stderr.String())
				}
			} else {
				if err != nil {
					t.Errorf("expected success, got %v. Stderr: %s", err, stderr.String())
				}
			}
		})
	}
}
