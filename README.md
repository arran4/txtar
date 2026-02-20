# txtar

`txtar` is a Go package and CLI tool for the txtar archive format, originally from `golang.org/x/tools/txtar`.

This version adds read/write capabilities to the archive and a filesystem implementation, along with a full-featured CLI.

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

```go
a := new(txtar.Archive)
a.Set("file.txt", []byte("content"))
a.Delete("file.txt")
```

### FileSystem

```go
fsys, err := txtar.FS(a)
w, err := fsys.Create("file.txt")
w.Write([]byte("content"))
w.Close()
```

## License

BSD-style (see LICENSE).
