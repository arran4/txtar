package txtar_test

import (
	"errors"
	"io/fs"
	"testing"
	"txtar"
)

func TestFileSystemCreate(t *testing.T) {
	a := new(txtar.Archive)
	fsys, err := txtar.FS(a)
	if err != nil {
		t.Fatal(err)
	}

	// Test happy path: creating a new file at root
	w, err := fsys.Create("hello.txt")
	if err != nil {
		t.Fatalf("Create(hello.txt) failed: %v", err)
	}
	if _, err := w.Write([]byte("hello")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify file exists
	data, err := fs.ReadFile(fsys, "hello.txt")
	if err != nil {
		t.Fatalf("ReadFile(hello.txt) failed: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("got %q, want %q", data, "hello")
	}

	// Test happy path: creating a new file in a new subdirectory
	w, err = fsys.Create("sub/world.txt")
	if err != nil {
		t.Fatalf("Create(sub/world.txt) failed: %v", err)
	}
	if _, err := w.Write([]byte("world")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	data, err = fs.ReadFile(fsys, "sub/world.txt")
	if err != nil {
		t.Fatalf("ReadFile(sub/world.txt) failed: %v", err)
	}
	if string(data) != "world" {
		t.Errorf("got %q, want %q", data, "world")
	}

	// Test happy path: overwriting an existing file
	w, err = fsys.Create("hello.txt")
	if err != nil {
		t.Fatalf("Create(hello.txt) failed for overwrite: %v", err)
	}
	if _, err := w.Write([]byte("updated")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	data, err = fs.ReadFile(fsys, "hello.txt")
	if err != nil {
		t.Fatalf("ReadFile(hello.txt) failed after overwrite: %v", err)
	}
	if string(data) != "updated" {
		t.Errorf("got %q, want %q", data, "updated")
	}

	// Test error case: invalid path (absolute path)
	_, err = fsys.Create("/absolute/path")
	if err == nil {
		t.Error("Create(/absolute/path) succeeded, want error")
	} else {
		var pathErr *fs.PathError
		if !errors.As(err, &pathErr) {
			t.Errorf("got error type %T, want *fs.PathError", err)
		} else if pathErr.Op != "create" {
			t.Errorf("got op %q, want %q", pathErr.Op, "create")
		}
	}

	// Test error case: creating a file that conflicts with an existing directory name
	// "sub" is a directory because of "sub/world.txt"
	w, err = fsys.Create("sub")
	if err != nil {
		t.Fatalf("Create(sub) should succeed (it only creates a fileWriter): %v", err)
	}
	if _, err := w.Write([]byte("conflict")); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	err = w.Close()
	if err == nil {
		t.Error("Close() should fail when creating a file that conflicts with a directory")
	}
}

func TestFileSystemRemove(t *testing.T) {
	a := new(txtar.Archive)
	a.Set("foo.txt", []byte("foo"))
	fsys, err := txtar.FS(a)
	if err != nil {
		t.Fatal(err)
	}

	// Test happy path: removing a file
	if err := fsys.Remove("foo.txt"); err != nil {
		t.Fatalf("Remove(foo.txt) failed: %v", err)
	}

	// Verify file is gone
	_, err = fsys.Open("foo.txt")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("got error %v, want fs.ErrNotExist", err)
	}

	// Test error case: invalid path
	err = fsys.Remove("/invalid")
	if err == nil {
		t.Error("Remove(/invalid) succeeded, want error")
	} else {
		var pathErr *fs.PathError
		if !errors.As(err, &pathErr) {
			t.Errorf("got error type %T, want *fs.PathError", err)
		} else if pathErr.Op != "remove" {
			t.Errorf("got op %q, want %q", pathErr.Op, "remove")
		}
	}
}

func TestFileSystemRename(t *testing.T) {
	a := new(txtar.Archive)
	a.Set("foo.txt", []byte("foo"))
	fsys, err := txtar.FS(a)
	if err != nil {
		t.Fatal(err)
	}

	// Test happy path: renaming a file
	if err := fsys.Rename("foo.txt", "bar.txt"); err != nil {
		t.Fatalf("Rename(foo.txt, bar.txt) failed: %v", err)
	}

	// Verify old file is gone and new file exists
	_, err = fsys.Open("foo.txt")
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("got error %v, want fs.ErrNotExist for old file", err)
	}
	data, err := fs.ReadFile(fsys, "bar.txt")
	if err != nil {
		t.Fatalf("ReadFile(bar.txt) failed: %v", err)
	}
	if string(data) != "foo" {
		t.Errorf("got %q, want %q", data, "foo")
	}

	// Test error case: invalid path
	err = fsys.Rename("/invalid", "new.txt")
	if err == nil {
		t.Error("Rename(/invalid, new.txt) succeeded, want error")
	}
	err = fsys.Rename("bar.txt", "/invalid")
	if err == nil {
		t.Error("Rename(bar.txt, /invalid) succeeded, want error")
	}

	// Test error case: non-existent source
	err = fsys.Rename("nonexistent.txt", "baz.txt")
	if err == nil {
		t.Error("Rename(nonexistent.txt, baz.txt) succeeded, want error")
	}
}
