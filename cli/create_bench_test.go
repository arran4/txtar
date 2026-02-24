package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkCreate(b *testing.B) {
	// Setup temporary directory structure once
	tmpDir, err := os.MkdirTemp("", "bench-create")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate 1000 files in nested directories
	for i := 0; i < 1000; i++ {
		dir := filepath.Join(tmpDir, fmt.Sprintf("dir%d", i%10))
		if err := os.MkdirAll(dir, 0755); err != nil {
			b.Fatal(err)
		}
		file := filepath.Join(dir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(file, []byte("content"), 0644); err != nil {
			b.Fatal(err)
		}
	}

	// Disable stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	f, err := os.Open(os.DevNull)
	if err != nil {
		b.Fatal(err)
	}
	os.Stdout = f
	defer f.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Run Create on the temp directory
		// We use recursive=true, trim=false, glob="", depth=-1
		Create(true, false, "", -1, tmpDir)
	}
}
