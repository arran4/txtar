package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkListMemory(b *testing.B) {
	dir := b.TempDir()
	archivePath := filepath.Join(dir, "large.txtar")

	f, err := os.Create(archivePath)
	if err != nil {
		b.Fatal(err)
	}

	_, _ = fmt.Fprintln(f, "This is a large archive for testing memory usage.")

	chunk := make([]byte, 10240) // 10KB
	for i := range chunk {
		chunk[i] = 'a'
	}

	// Write 1000 files -> ~10MB
	for i := 0; i < 1000; i++ {
		_, _ = fmt.Fprintf(f, "-- file%d.txt --\n", i)
		_, _ = f.Write(chunk)
		_, _ = f.Write([]byte("\n"))
	}
	_ = f.Close()

	// Redirect stdout
	oldStdout := os.Stdout
	devNull, _ := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
	os.Stdout = devNull
	defer func() {
		os.Stdout = oldStdout
		_ = devNull.Close()
	}()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		List(archivePath)
	}
}
