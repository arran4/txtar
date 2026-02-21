package txtar

import (
	"bytes"
	"strings"
	"testing"
)

func TestSetComment(t *testing.T) {
	a := &Archive{
		Comment: []byte("old comment\n"),
		Files: []File{
			{Name: "file1", Data: []byte("data1")},
		},
	}

	a.SetComment("new comment")

	if string(a.Comment) != "new comment" {
		t.Errorf("got %q, want %q", a.Comment, "new comment")
	}

	formatted := Format(a)
	// FixNL is called on comment during Format, so it adds \n if missing
	if !strings.HasPrefix(string(formatted), "new comment\n-- file1 --") {
		t.Errorf("Format output incorrect: %q", formatted)
	}
}

func TestReadComment(t *testing.T) {
	text := `comment line 1
comment line 2
-- file1 --
data1
`
	r := NewReader(bytes.NewBufferString(text))

	comment, err := r.ReadComment()
	if err != nil {
		t.Fatalf("ReadComment failed: %v", err)
	}

	expected := "comment line 1\ncomment line 2\n"
	if string(comment) != expected {
		t.Errorf("got %q, want %q", comment, expected)
	}

	// Verify Next works
	f, err := r.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}
	if f.Name != "file1" {
		t.Errorf("got file %q, want file1", f.Name)
	}
}

func TestReadComment_AlreadySkipped(t *testing.T) {
	text := `-- file1 --
data1
`
	r := NewReader(bytes.NewBufferString(text))

	_, err := r.Next()
	if err != nil {
		t.Fatalf("Next failed: %v", err)
	}

	_, err = r.ReadComment()
	if err == nil {
		t.Error("ReadComment should fail after Next")
	}
}
