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
	createCmd := exec.Command(binPath, "create", fileName)
	createCmd.Dir = tmpDir
	out, err := createCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(archivePath, out, 0644); err != nil {
		t.Fatal(err)
	}

	// 4. Set comment with -c
	comment := "This is a comment"
	setCmd := exec.Command(binPath, "comment", "-c", comment, archivePath)
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

    // 7. Update comment with -c
    newComment := "Updated comment"
    updateCmd := exec.Command(binPath, "comment", "-c", newComment, archivePath)
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

    // 8. Set comment from file with -f
    commentFile := filepath.Join(tmpDir, "comment.txt")
    fileComment := "Comment from file"
    if err := os.WriteFile(commentFile, []byte(fileComment), 0644); err != nil {
        t.Fatal(err)
    }
    fileCmd := exec.Command(binPath, "comment", "-f", commentFile, archivePath)
    if out, err := fileCmd.CombinedOutput(); err != nil {
        t.Fatalf("set comment from file failed: %v\n%s", err, out)
    }

    // Verify file update
    readFileCmd := exec.Command(binPath, "comment", archivePath)
    out, err = readFileCmd.CombinedOutput()
    if err != nil {
        t.Fatalf("read file comment failed: %v\n%s", err, out)
    }
    if strings.TrimSpace(string(out)) != fileComment {
         t.Errorf("read file comment: got %q, want %q", string(out), fileComment)
    }
}
