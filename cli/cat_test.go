package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkCatMemory(b *testing.B) {
	// Create a large archive
	dir := b.TempDir()
	archivePath := filepath.Join(dir, "large.txtar")

	f, err := os.Create(archivePath)
	if err != nil {
		b.Fatal(err)
	}

	// Write 10MB archive (1000 files of 10KB)
	// This should be enough to show memory usage difference
	// (reading 10MB into memory vs streaming)
	content := make([]byte, 10000)
    for i := range content {
        content[i] = 'a'
    }
    content[9999] = '\n'

	for i := 0; i < 1000; i++ {
		fmt.Fprintf(f, "-- file%d.txt --\n", i)
		f.Write(content)
	}
	f.Close()

    // Redirect stdout
    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    // Consume pipe in background to avoid blocking
    go func() {
        io.Copy(io.Discard, r)
    }()

    defer func() {
        os.Stdout = oldStdout
        w.Close()
    }()

	b.ResetTimer()
    b.ReportAllocs()
	for i := 0; i < b.N; i++ {
        // Run Cat with -t and no files (stream all)
        Cat(archivePath, true)
	}
}

func TestCatOrder(t *testing.T) {
    // Create archive with file1, file2
    dir := t.TempDir()
    archivePath := filepath.Join(dir, "order.txtar")
    f, err := os.Create(archivePath)
    if err != nil {
        t.Fatal(err)
    }
    fmt.Fprintf(f, "-- file1 --\ncontent1\n-- file2 --\ncontent2\n")
    f.Close()

    // Redirect stdout
    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    // Capture output
    outC := make(chan string)
    go func() {
        var buf []byte
        data, _ := io.ReadAll(r)
        buf = append(buf, data...)
        outC <- string(buf)
    }()

    // Run Cat requesting file2 then file1
    Cat(archivePath, true, "file2", "file1")

    w.Close()
    os.Stdout = oldStdout
    out := <-outC

    expected := "content2\ncontent1\n"
    if out != expected {
        t.Errorf("Expected %q, got %q", expected, out)
    }
}
