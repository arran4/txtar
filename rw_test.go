package txtar_test

import (
	"io"
	"testing"
	"txtar"
)

func TestArchiveReadWrite(t *testing.T) {
	a := new(txtar.Archive)
	a.Set("foo.txt", []byte("foo content"))
	a.Set("bar.txt", []byte("bar content"))

	if len(a.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(a.Files))
	}

	a.Set("foo.txt", []byte("foo content updated"))
	if len(a.Files) != 2 {
		t.Errorf("expected 2 files after update, got %d", len(a.Files))
	}

	for _, f := range a.Files {
		if f.Name == "foo.txt" && string(f.Data) != "foo content updated" {
			t.Errorf("expected foo content updated, got %q", f.Data)
		}
	}

	a.Delete("bar.txt")
	if len(a.Files) != 1 {
		t.Errorf("expected 1 file after delete, got %d", len(a.Files))
	}
	if a.Files[0].Name != "foo.txt" {
		t.Errorf("expected foo.txt remaining, got %s", a.Files[0].Name)
	}
}

func TestFileSystemReadWrite(t *testing.T) {
	a := new(txtar.Archive)
	fsys, err := txtar.FS(a)
	if err != nil {
		t.Fatal(err)
	}

	// Create a file
	w, err := fsys.Create("new.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("hello world")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	// Verify it exists in Archive
	if len(a.Files) != 1 {
		t.Errorf("expected 1 file in archive, got %d", len(a.Files))
	}
	if a.Files[0].Name != "new.txt" {
		t.Errorf("expected new.txt, got %s", a.Files[0].Name)
	}
	if string(a.Files[0].Data) != "hello world" {
		t.Errorf("expected 'hello world', got %q", a.Files[0].Data)
	}

	// Verify it exists in FileSystem
	f, err := fsys.Open("new.txt")
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if string(content) != "hello world" {
		t.Errorf("FileSystem read 'hello world', got %q", content)
	}

	// Remove the file
	if err := fsys.Remove("new.txt"); err != nil {
		t.Fatal(err)
	}

	// Verify it's gone from Archive
	if len(a.Files) != 0 {
		t.Errorf("expected 0 files in archive, got %d", len(a.Files))
	}

	// Verify it's gone from FileSystem
	_, err = fsys.Open("new.txt")
	if err == nil {
		t.Error("Open(new.txt) succeeded after remove, expected error")
	}
}

func TestRename(t *testing.T) {
	a := new(txtar.Archive)
	fsys, err := txtar.FS(a)
	if err != nil {
		t.Fatal(err)
	}

	w, err := fsys.Create("old.txt")
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("content"))
	w.Close()

	if err := fsys.Rename("old.txt", "new.txt"); err != nil {
		t.Fatal(err)
	}

	if _, err := fsys.Open("old.txt"); err == nil {
		t.Error("Open(old.txt) succeeded after rename")
	}

	f, err := fsys.Open("new.txt")
	if err != nil {
		t.Fatal(err)
	}
	content, _ := io.ReadAll(f)
	if string(content) != "content" {
		t.Errorf("got %q, want content", content)
	}
}
