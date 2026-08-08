package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"txtar"
)

func TestDescription(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "test.txtar")

	// Create initial archive
	a := new(txtar.Archive)
	a.Comment = []byte("line 1\nline 2\nline 3\n")
	a.Set("file1.txt", []byte("content1\n"))

	if err := os.WriteFile(archivePath, txtar.Format(a), 0644); err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	t.Run("Show", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		Description(false, false, "", archivePath)

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)

		expected := "line 1\nline 2\nline 3\n"
		if buf.String() != expected {
			t.Errorf("Expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("Replace", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "replace.txtar")
		_ = os.WriteFile(tempFile, txtar.Format(a), 0644)

		Description(true, false, "", tempFile, "new line 1", "new line 2")

		readA, err := txtar.ParseFile(tempFile)
		if err != nil {
			t.Fatalf("Failed to parse archive: %v", err)
		}
		expected := "new line 1 new line 2\n"
		if string(readA.Comment) != expected {
			t.Errorf("Expected %q, got %q", expected, string(readA.Comment))
		}
	})

	t.Run("Append", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "append.txtar")
		_ = os.WriteFile(tempFile, txtar.Format(a), 0644)

		Description(false, true, "", tempFile, "line 4")

		readA, err := txtar.ParseFile(tempFile)
		if err != nil {
			t.Fatalf("Failed to parse archive: %v", err)
		}
		expected := "line 1\nline 2\nline 3\nline 4\n"
		if string(readA.Comment) != expected {
			t.Errorf("Expected %q, got %q", expected, string(readA.Comment))
		}
	})

	t.Run("Edit", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "edit.txtar")
		_ = os.WriteFile(tempFile, txtar.Format(a), 0644)

		Description(false, false, "2-2", tempFile, "new line 2")

		readA, err := txtar.ParseFile(tempFile)
		if err != nil {
			t.Fatalf("Failed to parse archive: %v", err)
		}
		expected := "line 1\nnew line 2\nline 3\n"
		if string(readA.Comment) != expected {
			t.Errorf("Expected %q, got %q", expected, string(readA.Comment))
		}
	})

	t.Run("EditMultipleLines", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "edit2.txtar")
		_ = os.WriteFile(tempFile, txtar.Format(a), 0644)

		Description(false, false, "1-2", tempFile, "replaced lines 1 and 2")

		readA, err := txtar.ParseFile(tempFile)
		if err != nil {
			t.Fatalf("Failed to parse archive: %v", err)
		}
		expected := "replaced lines 1 and 2\nline 3\n"
		if string(readA.Comment) != expected {
			t.Errorf("Expected %q, got %q", expected, string(readA.Comment))
		}
	})
}
