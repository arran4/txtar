package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestList(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name      string
		content   string
		expected  []string
	}{
		{
			name: "standard archive",
			content: `comment
-- file1 --
content1
-- file2 --
content2
`,
			expected: []string{
				"0 8 9 file1",
				"1 29 9 file2",
			},
		},
		{
			name: "missing final newline",
			content: "-- file1 --\nabc",
			expected: []string{
				"0 0 4 file1",
			},
		},
		{
			name: "empty file",
			content: "-- file1 --\n-- file2 --\n",
			expected: []string{
				"0 0 0 file1",
				"1 12 0 file2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archivePath := filepath.Join(dir, strings.ReplaceAll(tt.name, " ", "_")+".txtar")
			if err := os.WriteFile(archivePath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			List(archivePath)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			got := buf.String()

			for _, want := range tt.expected {
				if !strings.Contains(got, want) {
					t.Errorf("Output missing %q\nGot:\n%s", want, got)
				}
			}
		})
	}
}
