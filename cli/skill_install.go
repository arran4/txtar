package cli

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// downloadAndExtractTarball downloads a tar.gz from url and extracts it to targetDir.
// Returns the commit/revision hash from the top-level directory name if it matches typical GitHub format.
func downloadAndExtractTarball(url, targetDir string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	gzr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	revision := ""

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read tar: %w", err)
		}

		// Skip pax headers and empty names
		if header.Typeflag == tar.TypeXGlobalHeader || header.Name == "" {
			continue
		}

		// GitHub tarballs have a top level directory like repo-name-commithash/
		parts := strings.Split(header.Name, "/")
		if len(parts) > 0 && revision == "" {
			topLevel := parts[0]
			// Try to extract commit hash if present
			if idx := strings.LastIndex(topLevel, "-"); idx != -1 && len(topLevel)-idx-1 >= 7 {
				revision = topLevel[idx+1:]
			} else {
				revision = topLevel
			}
		}

		// Strip the top-level directory
		if len(parts) <= 1 {
			continue
		}
		relPath := strings.Join(parts[1:], "/")

		targetPath := filepath.Join(targetDir, relPath)

		// Guard against path traversal
		if !strings.HasPrefix(targetPath, filepath.Clean(targetDir)+string(os.PathSeparator)) {
			return "", fmt.Errorf("invalid file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				return "", err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return "", err
			}
			f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return "", err
			}
			f.Close()
		}
	}

	return revision, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		targetPath := filepath.Join(dst, relPath)

		// Guard against path traversal (should be caught by Rel, but double check)
		if !strings.HasPrefix(targetPath, filepath.Clean(dst)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", targetPath)
		}

		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		if info.Mode()&os.ModeSymlink != 0 {
			// Skip symlinks for security
			return nil
		}

		// Regular file
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		// Do not copy executable permissions from remote content to prevent skills from becoming plugins silently
		mode := info.Mode() & 0666
		return os.WriteFile(targetPath, data, mode)
	})
}

func hashDir(dir string) (string, error) {
	h := sha256.New()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || info.Name() == metadataFileName {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// resolveSource downloads/copies the source to a temporary directory and returns the path to the temp dir containing the skill(s).
func resolveSource(source string) (string, string, error) {
	tempDir, err := os.MkdirTemp("", "txtar-skill-*")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	var revision string
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// Treat as URL to tarball
		return "", "", fmt.Errorf("direct HTTP source not fully implemented yet, use owner/repo")
	} else if strings.Contains(source, "/") && !strings.HasPrefix(source, ".") && !strings.HasPrefix(source, "/") {
		// Treat as GitHub owner/repo
		parts := strings.SplitN(source, "/", 3) // owner, repo, [path...]
		if len(parts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub source format: %s", source)
		}
		owner, repo := parts[0], parts[1]

		url := fmt.Sprintf("https://github.com/%s/%s/archive/HEAD.tar.gz", owner, repo)
		rev, err := downloadAndExtractTarball(url, tempDir)
		if err != nil {
			os.RemoveAll(tempDir)
			return "", "", fmt.Errorf("failed to fetch from GitHub: %w", err)
		}
		revision = rev

	} else {
		// Treat as local path
		absPath, err := filepath.Abs(source)
		if err != nil {
			os.RemoveAll(tempDir)
			return "", "", err
		}
		info, err := os.Stat(absPath)
		if err != nil {
			os.RemoveAll(tempDir)
			return "", "", err
		}
		if !info.IsDir() {
			os.RemoveAll(tempDir)
			return "", "", fmt.Errorf("source must be a directory")
		}
		if err := copyDir(absPath, tempDir); err != nil {
			os.RemoveAll(tempDir)
			return "", "", err
		}
		rev, _ := hashDir(tempDir)
		revision = rev
	}

	return tempDir, revision, nil
}
