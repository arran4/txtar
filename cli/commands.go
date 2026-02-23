package cli

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"txtar"
)

// Create is a subcommand `txtar create` -- Create a new archive
//
// Flags:
//
//	recursive:	-r --recursive	(default: false)	Recursive
//	trim:		-t --trim		(default: false)	Trim directory prefix
//	name:		--name			(default: "")		Name filter (glob pattern)
//	depth:		--depth			(default: -1)		Max depth
//	files:		...				Files/dirs to add
func Create(recursive bool, trim bool, name string, depth int, files ...string) {
	a := new(txtar.Archive)
	for _, file := range files {
		err := filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			rel, err := filepath.Rel(file, path)
			if err != nil {
				return err
			}

			// Calculate depth
			d := 0
			if rel != "." {
				d = strings.Count(rel, string(os.PathSeparator)) + 1
			}

			if info.IsDir() {
				if !recursive && rel != "." {
					return filepath.SkipDir
				}
				if depth >= 0 && d > depth {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip symlinks to avoid including files outside the intended scope
			if info.Mode()&os.ModeSymlink != 0 {
				return nil
			}

			if depth >= 0 && d > depth {
				return nil
			}

			// Apply filters
			if name != "" {
				matched, _ := filepath.Match(name, filepath.Base(path))
				if !matched {
					return nil
				}
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			storeName := path
			if trim {
				if rel == "." {
					storeName = filepath.Base(path)
				} else {
					storeName = rel
				}
			}

			a.Set(storeName, data)
			return nil
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error walking path %s: %v\n", file, err)
		}
	}
	fmt.Print(string(txtar.Format(a)))
}

// List is a subcommand `txtar list` -- List files in archive with index, offset, size, name
//
// Flags:
//
//	archive:	@1	Archive file
func List(archive string) {
	a, err := txtar.ParseFile(archive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing archive: %v\n", err)
		os.Exit(1)
	}

	// Calculate offsets
	// Start with comment
	offset := int64(len(txtar.FixNL(a.Comment)))

	for i, f := range a.Files {
		marker := fmt.Sprintf("-- %s --\n", f.Name)
		// Note: Format uses fixNL on data, so ensure we account for that.
		// But marker line is exactly formatted.
		// Wait, does Format add extra newlines or spacing?
		// fmt.Fprintf(&buf, "-- %s --\n", f.Name)
		// buf.Write(fixNL(f.Data))

		// So offset points to start of marker? Or start of file content?
		// Usually "file offset" means where the file starts.
		// If it means content offset, add marker length.
		// If it means entry offset, current offset is correct.
		// I'll print entry offset (start of marker).

		// Wait, user asked for "file offsets".
		// In tar/zip, offset usually means offset of the header.
		// I'll print the current offset (header start).

		size := int64(len(f.Data))

		fmt.Printf("%d %d %d %s\n", i, offset, size, f.Name)

		offset += int64(len(marker))
		offset += int64(len(txtar.FixNL(f.Data)))
	}
}

// Add is a subcommand `txtar add` -- Add files to archive
//
// Flags:
//
//	recursive:	-r --recursive	(default: false)	Recursive
//	archive:	@1	Archive file
//	files:		...	Files to add
func Add(recursive bool, archive string, files ...string) {
	a, err := txtar.ParseFile(archive)
	if err != nil {
		if os.IsNotExist(err) {
			a = new(txtar.Archive)
		} else {
			fmt.Fprintf(os.Stderr, "Error parsing archive: %v\n", err)
			os.Exit(1)
		}
	}

	for _, file := range files {
		if recursive {
			filepath.Walk(file, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				// Skip symlinks
				if info.Mode()&os.ModeSymlink != 0 {
					return nil
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				a.Set(path, data)
				return nil
			})
		} else {
			info, err := os.Stat(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error stating file %s: %v\n", file, err)
				continue
			}
			if info.IsDir() {
				fmt.Fprintf(os.Stderr, "Skipping directory %s (use -r)\n", file)
				continue
			}
			data, err := os.ReadFile(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file, err)
				continue
			}
			a.Set(file, data)
		}
	}

	if err := os.WriteFile(archive, txtar.Format(a), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing archive: %v\n", err)
		os.Exit(1)
	}
}

// Append is a subcommand `txtar append` -- Alias for add
//
// Flags:
//
//	recursive:	-r --recursive	(default: false)	Recursive
//	archive:	@1	Archive file
//	files:		...	Files to append
func Append(recursive bool, archive string, files ...string) {
	Add(recursive, archive, files...)
}

// Delete is a subcommand `txtar delete` -- Delete files from archive
//
// Flags:
//
//	archive:	@1	Archive file
//	files:		...	Files to delete (names or glob patterns)
func Delete(archive string, files ...string) {
	a, err := txtar.ParseFile(archive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing archive: %v\n", err)
		os.Exit(1)
	}

	// Collect files to delete
	var toDelete []string

	for _, pattern := range files {
		// Check if it's a pattern
		if strings.ContainsAny(pattern, "*?[]") {
			for _, f := range a.Files {
				matched, _ := filepath.Match(pattern, f.Name)
				if matched {
					toDelete = append(toDelete, f.Name)
				}
			}
		} else {
			// Exact match
			toDelete = append(toDelete, pattern)
		}
	}

	for _, name := range toDelete {
		a.Delete(name)
	}

	if err := os.WriteFile(archive, txtar.Format(a), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing archive: %v\n", err)
		os.Exit(1)
	}
}

// Cat is a subcommand `txtar cat` -- Extract or display archive content
//
// Flags:
//
//	archive:	@1				Archive file
//	txt:		-t --txt		(default: false)	Extract/cat content of files inside archive
//	files:		...				Files to extract (names in archive)
func Cat(archive string, txt bool, files ...string) {
	if !txt {
		data, err := os.ReadFile(archive)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading archive: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(string(data))
		return
	}

	a, err := txtar.ParseFile(archive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing archive: %v\n", err)
		os.Exit(1)
	}

	fsys, _ := txtar.FS(a)

	if len(files) == 0 {
		for _, f := range a.Files {
			fmt.Print(string(f.Data))
		}
		return
	}

	for _, file := range files {
		content, err := fs.ReadFile(fsys, file)
		if err != nil {
			found := false
			for _, f := range a.Files {
				matched, _ := filepath.Match(file, f.Name)
				if matched {
					fmt.Print(string(f.Data))
					found = true
				}
			}
			if !found {
				fmt.Fprintf(os.Stderr, "File %s not found in archive\n", file)
			}
		} else {
			fmt.Print(string(content))
		}
	}
}

// Comment is a subcommand `txtar comment` -- Show or set archive comment
//
// Flags:
//
//	comment:	-c --comment	(default: "")	Set comment to text
//	file:		-f --file		(default: "")	Set comment from file (use - for stdin)
//	archive:	@1				Archive file
func Comment(comment string, file string, archive string) {
	a, err := txtar.ParseFile(archive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing archive: %v\n", err)
		os.Exit(1)
	}

	if comment == "" && file == "" {
		fmt.Print(string(a.Comment))
		return
	}

	if comment != "" && file != "" {
		fmt.Fprintf(os.Stderr, "Error: cannot specify both --comment and --file\n")
		os.Exit(1)
	}

	var text string
	if comment != "" {
		text = comment
	} else if file == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			os.Exit(1)
		}
		text = string(data)
	} else {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading file %s: %v\n", file, err)
			os.Exit(1)
		}
		text = string(data)
	}

	a.SetComment(text)
	if err := os.WriteFile(archive, txtar.Format(a), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing archive: %v\n", err)
		os.Exit(1)
	}
}
