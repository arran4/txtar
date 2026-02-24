package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSymlinkVulnerability(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create a secret file outside the target directory
	secretContent := "SUPER_SECRET_CONTENT_DO_NOT_LEAK"
	secretFile := filepath.Join(tmpDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte(secretContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a target directory to archive
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.Mkdir(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink inside the target directory pointing to the secret file
	symlinkPath := filepath.Join(targetDir, "link_to_secret")
	if err := os.Symlink(secretFile, symlinkPath); err != nil {
		t.Fatal(err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run Create on the target directory
	// recursive=true, trim=false, follow=false, name="", depth=-1, files=[targetDir]
	Create(true, false, false, "", -1, targetDir)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check if the secret content is present in the output
	if strings.Contains(output, secretContent) {
		t.Errorf("Vulnerability confirmed: Create command followed symlink and included secret content:\n%s", output)
	} else {
		t.Logf("Secret content not found in Create output. Vulnerability seems fixed.")
	}
}

func TestSymlinkVulnerability_Add(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create a secret file outside the target directory
	secretContent := "SUPER_SECRET_CONTENT_DO_NOT_LEAK_ADD"
	secretFile := filepath.Join(tmpDir, "secret_add.txt")
	if err := os.WriteFile(secretFile, []byte(secretContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a target directory to archive
	targetDir := filepath.Join(tmpDir, "target_add")
	if err := os.Mkdir(targetDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink inside the target directory pointing to the secret file
	symlinkPath := filepath.Join(targetDir, "link_to_secret_add")
	if err := os.Symlink(secretFile, symlinkPath); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(tmpDir, "archive.txt")

	// Run Add on the target directory
	// recursive=true, follow=false, archive=archivePath, files=[targetDir]
	Add(true, false, archivePath, targetDir)

	// Read the archive file
	data, err := os.ReadFile(archivePath)
	if err != nil {
		// If archive wasn't created because it was empty, that's fine too
		if os.IsNotExist(err) {
			return
		}
		t.Fatal(err)
	}
	output := string(data)

	// Check if the secret content is present in the output
	if strings.Contains(output, secretContent) {
		t.Errorf("Vulnerability confirmed: Add command followed symlink and included secret content:\n%s", output)
	} else {
		t.Logf("Secret content not found in Add output. Vulnerability seems fixed.")
	}
}

func TestSymlinkFollowing(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()

	// Create a target file outside the target directory (but we intend to follow it)
	targetContent := "TARGET_CONTENT_TO_FOLLOW"
	targetFile := filepath.Join(tmpDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte(targetContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a directory to archive
	archiveDir := filepath.Join(tmpDir, "archive_dir")
	if err := os.Mkdir(archiveDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink inside the directory pointing to the target file
	symlinkPath := filepath.Join(archiveDir, "link_to_target")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Fatal(err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run Create on the directory with follow=true
	// recursive=true, trim=false, follow=true, name="", depth=-1, files=[archiveDir]
	Create(true, false, true, "", -1, archiveDir)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Check if the content is present in the output
	if !strings.Contains(output, targetContent) {
		t.Errorf("Follow symlink failed: Create command with follow=true did NOT include symlink content:\n%s", output)
	} else {
		t.Logf("Symlink followed as expected.")
	}
}
