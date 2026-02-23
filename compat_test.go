package txtar

import (
	"bytes"
	"os"
	"reflect"
	"testing"
)

func TestParseFileCompatibility(t *testing.T) {
	// Create a sample txtar content where last file has no newline
	content := `Comment line 1
-- file1.txt --
Content of file 1
-- file2.txt --
Content of file 2 without newline`

	// Parse using Parse (reference implementation)
	archive1 := Parse([]byte(content))

	// Parse using ParseFile (new implementation)
	tmpfile, err := os.CreateTemp("", "test*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	archive2, err := ParseFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	// Compare archives
	if !reflect.DeepEqual(archive1, archive2) {
		t.Errorf("Archives do not match.")
		t.Errorf("Parse Comment: %q", archive1.Comment)
		t.Errorf("ParseFile Comment: %q", archive2.Comment)

		if len(archive1.Files) != len(archive2.Files) {
			t.Errorf("File counts differ: %d vs %d", len(archive1.Files), len(archive2.Files))
		}

		for i := 0; i < len(archive1.Files) && i < len(archive2.Files); i++ {
			f1 := archive1.Files[i]
			f2 := archive2.Files[i]
			if f1.Name != f2.Name {
				t.Errorf("File %d name differs: %q vs %q", i, f1.Name, f2.Name)
			}
			if !bytes.Equal(f1.Data, f2.Data) {
				t.Errorf("File %d data differs:\nParse: %q\nParseFile: %q", i, f1.Data, f2.Data)
			}
		}
	}
}
