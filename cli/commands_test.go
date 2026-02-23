package cli

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"txtar"
)

func TestList(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "test.txtar")

	// Create an archive
	a := &txtar.Archive{
		Comment: []byte("my comment\n"),
		Files: []txtar.File{
			{Name: "file1.txt", Data: []byte("content1\n")},
			{Name: "dir/file2.txt", Data: []byte("content2")}, // no newline
		},
	}
	data := txtar.Format(a)
	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run List
	// We need to handle os.Exit if List fails, but List calls os.Exit(1) on error.
	// Since our input is valid, it shouldn't exit.
	// If it does, the test binary will exit, which is a failure.
	List(archivePath)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Expected output
	// Comment: "my comment\n" (len 11)
	// Offset 0: 11.
	// Marker 1: "-- file1.txt --\n" (len 16).
	// Content 1: "content1\n" (len 9).
	//
	// Offset 1: 11 + 16 + 9 = 36.
	// Marker 2: "-- dir/file2.txt --\n" (len 22).
	// Content 2: "content2" (len 8).

	expectedLines := []string{
		"0 11 9 file1.txt",
		"1 36 9 dir/file2.txt",
	}

	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("Output missing line: %q. Got:\n%s", line, output)
		}
	}
}

func BenchmarkList(b *testing.B) {
	tmpDir := b.TempDir()
	archivePath := filepath.Join(tmpDir, "bench.txtar")

	// Create a large archive
	a := &txtar.Archive{
		Comment: []byte("benchmark archive\n"),
	}
	// Add 1000 files of 1KB each
	content := bytes.Repeat([]byte("a"), 1024)
	for i := 0; i < 1000; i++ {
		a.Files = append(a.Files, txtar.File{
			Name: fmt.Sprintf("file%d", i),
			Data: content,
		})
	}

	if err := os.WriteFile(archivePath, txtar.Format(a), 0644); err != nil {
		b.Fatal(err)
	}

	// Silence output
	oldStdout := os.Stdout
	devNull, _ := os.Open(os.DevNull)
	os.Stdout = devNull // Note: os.Stdout is *os.File
	// Wait, os.DevNull might not work for assignment to os.Stdout directly if I need to restore it?
	// Yes it works.

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		List(archivePath)
	}

	os.Stdout = oldStdout
	devNull.Close()
}
