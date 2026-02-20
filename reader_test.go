package txtar

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestReader(t *testing.T) {
	text := `comment1
comment2
-- file1 --
File 1 text.
-- foo ---
More file 1 text.
-- file 2 --
File 2 text.
-- empty --
-- noNL --
hello world
-- empty filename line --
some content
-- --`

	r := NewReader(strings.NewReader(text))

	// Read comment
	var comment bytes.Buffer
	if _, err := io.Copy(&comment, r); err != nil {
		t.Fatalf("reading comment: %v", err)
	}
	wantComment := "comment1\ncomment2\n"
	if got := comment.String(); got != wantComment {
		t.Errorf("comment: got %q, want %q", got, wantComment)
	}

	// File 1
	f, err := r.Next()
	if err != nil {
		t.Fatalf("Next file1: %v", err)
	}
	if f.Name != "file1" {
		t.Errorf("file1 name: got %q, want %q", f.Name, "file1")
	}
	var b1 bytes.Buffer
	if _, err := io.Copy(&b1, r); err != nil {
		t.Fatalf("reading file1: %v", err)
	}
	want1 := "File 1 text.\n-- foo ---\nMore file 1 text.\n"
	if got := b1.String(); got != want1 {
		t.Errorf("file1 content: got %q, want %q", got, want1)
	}

	// File 2
	f, err = r.Next()
	if err != nil {
		t.Fatalf("Next file2: %v", err)
	}
	if f.Name != "file 2" {
		t.Errorf("file2 name: got %q, want %q", f.Name, "file 2")
	}
	var b2 bytes.Buffer
	if _, err := io.Copy(&b2, r); err != nil {
		t.Fatalf("reading file2: %v", err)
	}
	want2 := "File 2 text.\n"
	if got := b2.String(); got != want2 {
		t.Errorf("file2 content: got %q, want %q", got, want2)
	}

	// Empty file
	f, err = r.Next()
	if err != nil {
		t.Fatalf("Next empty: %v", err)
	}
	if f.Name != "empty" {
		t.Errorf("empty name: got %q, want %q", f.Name, "empty")
	}
	var b3 bytes.Buffer
	if _, err := io.Copy(&b3, r); err != nil {
		t.Fatalf("reading empty: %v", err)
	}
	if b3.Len() != 0 {
		t.Errorf("empty content: got %q, want empty", b3.String())
	}

	// No NL
	f, err = r.Next()
	if err != nil {
		t.Fatalf("Next noNL: %v", err)
	}
	if f.Name != "noNL" {
		t.Errorf("noNL name: got %q, want %q", f.Name, "noNL")
	}
	var b4 bytes.Buffer
	if _, err := io.Copy(&b4, r); err != nil {
		t.Fatalf("reading noNL: %v", err)
	}
	// Note: Parse adds newline, Reader returns raw bytes.
	// The input text has `hello world\n`.
	want4 := "hello world\n"
	if got := b4.String(); got != want4 {
		t.Errorf("noNL content: got %q, want %q", got, want4)
	}

	// Last file "empty filename line"
	f, err = r.Next()
	if err != nil {
		t.Fatalf("Next last: %v", err)
	}
	if f.Name != "empty filename line" {
		t.Errorf("last name: got %q, want %q", f.Name, "empty filename line")
	}
	var b5 bytes.Buffer
	if _, err := io.Copy(&b5, r); err != nil {
		t.Fatalf("reading last: %v", err)
	}
	// The content includes `-- --` because it's not a valid marker (empty name).
	// Reader returns exactly what is in stream.
	want5 := "some content\n-- --"
	if got := b5.String(); got != want5 {
		t.Errorf("last content: got %q, want %q", got, want5)
	}

	// End
	_, err = r.Next()
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
}

func TestReader_NoComment(t *testing.T) {
	text := `-- file1 --
content`
	r := NewReader(strings.NewReader(text))

	// Read comment (should be empty)
	var comment bytes.Buffer
	if _, err := io.Copy(&comment, r); err != nil {
		t.Fatalf("reading comment: %v", err)
	}
	if comment.Len() != 0 {
		t.Errorf("expected empty comment, got %q", comment.String())
	}

	f, err := r.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if f.Name != "file1" {
		t.Errorf("got %q, want file1", f.Name)
	}

	var b bytes.Buffer
	io.Copy(&b, r)
	if got := b.String(); got != "content" {
		t.Errorf("got %q, want content", got)
	}
}

func TestReader_SkipComment(t *testing.T) {
	text := `comment
-- file1 --
content`
	r := NewReader(strings.NewReader(text))

	// Don't read comment, call Next directly
	f, err := r.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if f.Name != "file1" {
		t.Errorf("got %q, want file1", f.Name)
	}

	var b bytes.Buffer
	io.Copy(&b, r)
	if got := b.String(); got != "content" {
		t.Errorf("got %q, want content", got)
	}
}

func TestReader_SmallBuffer(t *testing.T) {
	text := `comment
-- file1 --
content of file1`
	r := NewReader(strings.NewReader(text))

	// Skip comment
	f, err := r.Next()
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if f.Name != "file1" {
		t.Fatalf("got %q, want file1", f.Name)
	}

	var b bytes.Buffer
	buf := make([]byte, 3) // Read 3 bytes at a time
	for {
		n, err := r.Read(buf)
		b.Write(buf[:n])
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
	}

	if got := b.String(); got != "content of file1" {
		t.Errorf("got %q, want content of file1", got)
	}
}

func TestReader_All(t *testing.T) {
	text := `-- file1 --
content1
-- file2 --
content2`
	r := NewReader(strings.NewReader(text))

	var names []string
	var contents []string

	for f, err := range r.All() {
		if err != nil {
			t.Fatalf("All: %v", err)
		}
		names = append(names, f.Name)
		b, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		contents = append(contents, string(b))
	}

	wantNames := []string{"file1", "file2"}
	if len(names) != len(wantNames) {
		t.Fatalf("got %d files, want %d", len(names), len(wantNames))
	}
	for i, name := range names {
		if name != wantNames[i] {
			t.Errorf("file %d: got %q, want %q", i, name, wantNames[i])
		}
	}

	wantContents := []string{"content1\n", "content2"} // file2 has no trailing newline in input
	for i, content := range contents {
		if content != wantContents[i] {
			t.Errorf("content %d: got %q, want %q", i, content, wantContents[i])
		}
	}
}

func TestReader_AllWithData(t *testing.T) {
	text := `-- file1 --
content1
-- file2 --
content2`
	r := NewReader(strings.NewReader(text))

	var names []string
	var contents []string

	for f, err := range r.AllWithData() {
		if err != nil {
			t.Fatalf("AllWithData: %v", err)
		}
		names = append(names, f.Name)
		contents = append(contents, string(f.Data))
	}

	wantNames := []string{"file1", "file2"}
	if len(names) != len(wantNames) {
		t.Fatalf("got %d files, want %d", len(names), len(wantNames))
	}
	for i, name := range names {
		if name != wantNames[i] {
			t.Errorf("file %d: got %q, want %q", i, name, wantNames[i])
		}
	}

	wantContents := []string{"content1\n", "content2"}
	for i, content := range contents {
		if content != wantContents[i] {
			t.Errorf("content %d: got %q, want %q", i, content, wantContents[i])
		}
	}
}
