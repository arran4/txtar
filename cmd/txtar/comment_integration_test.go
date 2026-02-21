package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommentIntegration(t *testing.T) {
	// 1. Build the binary
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "txtar")

	// We assume we are running in cmd/txtar directory or can find it.
	// But go test ./... runs in each package dir.
	// So we are in cmd/txtar.
	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	// 2. Prepare test data
	content := "hello world\n"
	fileName := "test.txt"
	filePath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	archivePath := filepath.Join(tmpDir, "archive.txtar")

	// 3. Create archive
	// Usage: txtar create [-r] [-t] [--name] [--depth] file...
	// We want to store relative path "test.txt".
	// So we pass filePath but use -t (trim) maybe?
	// Or just run from tmpDir?
	// Let's run create command with Cwd = tmpDir
	createCmd := exec.Command(binPath, "create", fileName)
	createCmd.Dir = tmpDir
	out, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(archivePath, out, 0644); err != nil {
		t.Fatal(err)
	}

	// 4. Set comment
	comment := "This is a comment"
	setCmd := exec.Command(binPath, "comment", archivePath, comment)
	if out, err := setCmd.CombinedOutput(); err != nil {
		t.Fatalf("set comment failed: %v\n%s", err, out)
	}

	// 5. Read comment
	readCmd := exec.Command(binPath, "comment", archivePath)
	out, err = readCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("read comment failed: %v\n%s", err, out)
	}
	if strings.TrimSpace(string(out)) != comment {
		t.Errorf("read comment: got %q, want %q", string(out), comment)
	}

	// 6. Verify content
	// Usage: txtar cat archive [files...]
	// Default prints content? No, checks -t flag for extraction content.
	// Wait, Cat implementation:
	// if !txt { print archive content }
	// if txt { if files given, extract/print content of files; else print all files content }

	// Check full archive content (should contain comment)
	catCmd := exec.Command(binPath, "cat", archivePath)
	out, err = catCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cat failed: %v\n%s", err, out)
	}
	archiveContent := string(out)
	if !strings.HasPrefix(archiveContent, comment+"\n-- "+fileName+" --") {
		t.Errorf("archive content unexpected: %q", archiveContent)
	}

	// Check file content extraction
	catFileCmd := exec.Command(binPath, "cat", "-t", archivePath, fileName)
	out, err = catFileCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cat file failed: %v\n%s", err, out)
	}
	if string(out) != content {
		t.Errorf("file content: got %q, want %q", string(out), content)
	}

    // 7. Update comment
    newComment := "Updated comment"
    updateCmd := exec.Command(binPath, "comment", archivePath, newComment)
    if out, err := updateCmd.CombinedOutput(); err != nil {
        t.Fatalf("update comment failed: %v\n%s", err, out)
    }

    // Verify update
    readUpdateCmd := exec.Command(binPath, "comment", archivePath)
    out, err = readUpdateCmd.CombinedOutput()
    if err != nil {
        t.Fatalf("read updated comment failed: %v\n%s", err, out)
    }
    if strings.TrimSpace(string(out)) != newComment {
        t.Errorf("read updated comment: got %q, want %q", string(out), newComment)
    }
}
