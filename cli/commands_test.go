package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func BenchmarkCreate(b *testing.B) {
	dir := b.TempDir()
	for i := 0; i < 1000; i++ {
		err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file%d.txt", i)), []byte("some content"), 0644)
		if err != nil {
			b.Fatal(err)
		}
	}

	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	r, w, _ := os.Pipe()
	os.Stdout = w

	go func() {
		io.Copy(io.Discard, r)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Create(true, false, "", -1, dir)
	}

	w.Close()
}

func TestCreate(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"file1.txt": "content1\n",
		"subdir/file2.txt": "content2\n",
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Capture stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()
	r, w, _ := os.Pipe()
	os.Stdout = w

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		io.Copy(&buf, r)
		wg.Done()
	}()

	// Call Create
	Create(true, true, "", -1, dir)

	w.Close()
	wg.Wait()

	output := buf.String()

	// Verify output contains the files
	expected1 := "-- file1.txt --\ncontent1\n"
	if !strings.Contains(output, expected1) {
		t.Errorf("Output missing file1.txt content. Got:\n%s", output)
	}

	expected2 := "-- subdir/file2.txt --\ncontent2\n"
	if !strings.Contains(output, expected2) {
		t.Errorf("Output missing subdir/file2.txt content. Got:\n%s", output)
	}
}
