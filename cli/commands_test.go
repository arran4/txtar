package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreate_SymlinkVulnerability(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a sensitive file
	sensitiveFile := filepath.Join(tmpDir, "sensitive.txt")
	sensitiveContent := "This is sensitive data"
	if err := os.WriteFile(sensitiveFile, []byte(sensitiveContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a directory to archive
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.Mkdir(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink pointing to the sensitive file
	linkPath := filepath.Join(dataDir, "link_to_sensitive")
	if err := os.Symlink(sensitiveFile, linkPath); err != nil {
		t.Fatal(err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run Create
	// func Create(recursive bool, trim bool, name string, depth int, files ...string)
	Create(true, false, "", -1, dataDir)

	w.Close()
	os.Stdout = oldStdout

	out, _ := io.ReadAll(r)
	output := string(out)

	// Check if sensitive content is present
	if strings.Contains(output, sensitiveContent) {
		t.Errorf("Security vulnerability: Symlink was followed and sensitive file content was included in archive")
	}
}

func TestAdd_SymlinkVulnerability(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a sensitive file
	sensitiveFile := filepath.Join(tmpDir, "sensitive.txt")
	sensitiveContent := "This is sensitive data"
	if err := os.WriteFile(sensitiveFile, []byte(sensitiveContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a directory to archive
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.Mkdir(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink pointing to the sensitive file
	linkPath := filepath.Join(dataDir, "link_to_sensitive")
	if err := os.Symlink(sensitiveFile, linkPath); err != nil {
		t.Fatal(err)
	}

	// Create an initial archive
	archivePath := filepath.Join(tmpDir, "archive.txt")
	if err := os.WriteFile(archivePath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	// Run Add
	// func Add(recursive bool, archive string, files ...string)
	Add(true, archivePath, dataDir)

	// Read the archive
	archiveContent, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	output := string(archiveContent)

	// Check if sensitive content is present
	if strings.Contains(output, sensitiveContent) {
		t.Errorf("Security vulnerability: Add command followed symlink and sensitive file content was included in archive")
	}
}
