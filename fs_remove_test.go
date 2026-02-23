// Copyright 2024 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package txtar_test

import (
	"io/fs"
	"testing"
	"txtar"
)

func TestRemove(t *testing.T) {
	tests := []struct {
		name      string
		files     string
		remove    string
		expectErr error // specific error or nil
		check     func(*testing.T, fs.FS, *txtar.Archive)
	}{
		{
			name:   "remove existing file",
			files:  "-- test.txt --\ncontent",
			remove: "test.txt",
			check: func(t *testing.T, fsys fs.FS, ar *txtar.Archive) {
				if _, err := fsys.Open("test.txt"); err == nil {
					t.Error("Open(test.txt) succeeded after remove; expected error")
				} else if _, ok := err.(*fs.PathError); !ok {
					t.Errorf("Open(test.txt) returned error %T; expected *fs.PathError", err)
				}
				// Verify underlying archive
				for _, f := range ar.Files {
					if f.Name == "test.txt" {
						t.Error("Archive still contains test.txt after remove")
					}
				}
			},
		},
		{
			name:   "remove non-existent file",
			files:  "",
			remove: "nonexistent.txt",
			check: func(t *testing.T, fsys fs.FS, ar *txtar.Archive) {
				// No check needed, just verifying no error returned by Remove call itself
			},
		},
		{
			name:      "remove invalid path",
			files:     "",
			remove:    "invalid/../path",
			expectErr: fs.ErrInvalid,
			check: func(t *testing.T, fsys fs.FS, ar *txtar.Archive) {
				// No check needed
			},
		},
		{
			name:   "remove file in subdirectory",
			files:  "-- dir/file.txt --\ncontent",
			remove: "dir/file.txt",
			check: func(t *testing.T, fsys fs.FS, ar *txtar.Archive) {
				// Verify file is gone
				if _, err := fsys.Open("dir/file.txt"); err == nil {
					t.Error("Open(dir/file.txt) succeeded after remove")
				}
				// Verify directory is gone (implicit because empty)
				if _, err := fsys.Open("dir"); err == nil {
					t.Error("Open(dir) succeeded after removing last file; expected error (directory should disappear)")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ar := txtar.Parse([]byte(tc.files))
			fsys, err := txtar.FS(ar)
			if err != nil {
				t.Fatalf("txtar.FS failed: %v", err)
			}

			err = fsys.Remove(tc.remove)
			if tc.expectErr != nil {
				if err == nil {
					t.Errorf("Remove(%q) succeeded; expected error %v", tc.remove, tc.expectErr)
				} else {
					// Check error type or value if needed. Here assuming fs.ErrInvalid wrapped in PathError.
					pathErr, ok := err.(*fs.PathError)
					if !ok || pathErr.Err != tc.expectErr {
						t.Errorf("Remove(%q) returned error %v; expected PathError with Err=%v", tc.remove, err, tc.expectErr)
					}
				}
			} else if err != nil {
				t.Errorf("Remove(%q) failed: %v", tc.remove, err)
			}

			if tc.check != nil {
				tc.check(t, fsys, ar)
			}
		})
	}
}
