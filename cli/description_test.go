package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
	"txtar"
)

type MockFS struct {
	Files map[string][]byte
	Out   bytes.Buffer
	Err   bytes.Buffer
}

func (m *MockFS) MkdirAll(path string, perm os.FileMode) error {
	return nil
}

func (m *MockFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	if m.Files == nil {
		m.Files = make(map[string][]byte)
	}
	m.Files[name] = data
	return nil
}

func (m *MockFS) Stdout() io.Writer {
	return &m.Out
}

func (m *MockFS) Stderr() io.Writer {
	return &m.Err
}

func TestDescription(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "test.txtar")

	// Create initial archive
	a := new(txtar.Archive)
	a.Comment = []byte("line 1\nline 2\nline 3\n")
	a.Set("file1.txt", []byte("content1\n"))

	_ = os.WriteFile(archivePath, txtar.Format(a), 0644)

	t.Run("Show", func(t *testing.T) {
		fsys := &MockFS{}
		descriptionWithFS(fsys, false, false, "", archivePath)

		expected := "line 1\nline 2\nline 3\n"
		if fsys.Out.String() != expected {
			t.Errorf("Expected %q, got %q", expected, fsys.Out.String())
		}
	})

	t.Run("Replace", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "replace.txtar")
		_ = os.WriteFile(tempFile, txtar.Format(a), 0644)

		fsys := &MockFS{}
		descriptionWithFS(fsys, true, false, "", tempFile, "new line 1", "new line 2")

		readA := txtar.Parse(fsys.Files[tempFile])
		expected := "new line 1 new line 2\n"
		if string(readA.Comment) != expected {
			t.Errorf("Expected %q, got %q", expected, string(readA.Comment))
		}
	})

	t.Run("Append", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "append.txtar")
		_ = os.WriteFile(tempFile, txtar.Format(a), 0644)

		fsys := &MockFS{}
		descriptionWithFS(fsys, false, true, "", tempFile, "line 4")

		readA := txtar.Parse(fsys.Files[tempFile])
		expected := "line 1\nline 2\nline 3\nline 4\n"
		if string(readA.Comment) != expected {
			t.Errorf("Expected %q, got %q", expected, string(readA.Comment))
		}
	})

	t.Run("Edit", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "edit.txtar")
		_ = os.WriteFile(tempFile, txtar.Format(a), 0644)

		fsys := &MockFS{}
		descriptionWithFS(fsys, false, false, "2-2", tempFile, "new line 2")

		readA := txtar.Parse(fsys.Files[tempFile])
		expected := "line 1\nnew line 2\nline 3\n"
		if string(readA.Comment) != expected {
			t.Errorf("Expected %q, got %q", expected, string(readA.Comment))
		}
	})

	t.Run("EditMultipleLines", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "edit2.txtar")
		_ = os.WriteFile(tempFile, txtar.Format(a), 0644)

		fsys := &MockFS{}
		descriptionWithFS(fsys, false, false, "1-2", tempFile, "replaced lines 1 and 2")

		readA := txtar.Parse(fsys.Files[tempFile])
		expected := "replaced lines 1 and 2\nline 3\n"
		if string(readA.Comment) != expected {
			t.Errorf("Expected %q, got %q", expected, string(readA.Comment))
		}
	})
}
