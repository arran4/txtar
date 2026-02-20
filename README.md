# txtar

`txtar` is a Go package and CLI tool for the txtar archive format, forked from [golang.org/x/tools/txtar](https://cs.opensource.google/go/x/tools/+/master:txtar/).

This project extends the original library with significant enhancements, including:

- **Read/Write Library**: Adds support for modifying archives programmatically (`Set`, `Delete`).
- **Extended Filesystem**: Implements a full `fs.FS` interface with write capabilities (`Create`, `Remove`, `Rename`), which the original library lacks.
- **Streaming Version**: Creating a streaming version for efficient processing of large archives.
- **CLI Version**: A comprehensive command-line tool for creating, extracting, and managing archives.

## Installation

```bash
go install github.com/username/txtar/cmd/txtar@latest
```

## CLI Usage

The `txtar` CLI provides commands to manage txtar archives.

### Create

Create a new archive from files or directories.

```bash
txtar create -r directory > archive.txtar
```

Flags:
- `-r, --recursive`: Recursive (default: false)
- `-t, --trim`: Trim directory prefix (default: false)
- `--name`: Name filter (glob pattern)
- `--depth`: Max depth

### List

List files in an archive.

```bash
txtar list archive.txtar
```

### Add / Append

Add files to an existing archive.

```bash
txtar add archive.txtar file1 file2
```

### Delete

Delete files from an archive.

```bash
txtar delete archive.txtar file1
```

### Cat

Extract content or display the archive.

```bash
# Display archive content
txtar cat archive.txtar

# Extract file content
txtar cat -t archive.txtar file1
```

## Library Usage

### Archive Read/Write

The `Archive` struct now supports direct modification:

```go
a := new(txtar.Archive)
a.Set("file.txt", []byte("content"))
a.Delete("file.txt")
```

### FileSystem

The library provides a filesystem implementation that supports standard `fs.FS` operations as well as write operations:

```go
fsys, err := txtar.FS(a)

// Create a new file within the archive
w, err := fsys.Create("file.txt")
if err != nil {
    log.Fatal(err)
}
w.Write([]byte("content"))
w.Close()

// Read the file using standard fs.FS
data, err := fs.ReadFile(fsys, "file.txt")
```

## License

BSD-style (see LICENSE).
