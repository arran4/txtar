package txtar

import (
	"bufio"
	"io"
	"iter"
)

// Reader provides sequential access to the contents of a txtar archive.
// Reader.Read reads from the current file in the archive.
// Reader.Next advances to the next file.
type Reader struct {
	r             *bufio.Reader
	atStartOfLine bool
	nextFile      File
	nextFileValid bool
	pending       []byte
}

// NewReader creates a new Reader reading from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		r:             bufio.NewReader(r),
		atStartOfLine: true,
	}
}

// Next advances to the next entry in the archive.
// It returns the File header for the next file.
// The Data field of the returned File is always nil.
//
// If there are no more files, Next returns io.EOF.
func (r *Reader) Next() (File, error) {
	// If we have a pending next file (found during Read), return it.
	if r.nextFileValid {
		f := r.nextFile
		r.nextFileValid = false
		r.nextFile = File{}
		return f, nil
	}

	// Consume remaining data of current file
	_, err := io.Copy(io.Discard, r)
	if err != nil {
		return File{}, err
	}

	// After consuming, check if we found the next file
	if r.nextFileValid {
		f := r.nextFile
		r.nextFileValid = false
		r.nextFile = File{}
		return f, nil
	}

	return File{}, io.EOF
}

// Read reads from the current file in the archive.
// It returns 0, io.EOF when the end of the file is reached.
func (r *Reader) Read(p []byte) (n int, err error) {
	if r.nextFileValid {
		return 0, io.EOF
	}

	if len(r.pending) > 0 {
		n = copy(p, r.pending)
		r.pending = r.pending[n:]
		return n, nil
	}

	if r.atStartOfLine {
		// Check for marker
		peek, _ := r.r.Peek(3)
		if string(peek) == "-- " {
			// Potential marker.
			// We need to read the whole line to verify.
			line, err := r.r.ReadSlice('\n')

			// Check if it's a marker.
			// err == nil or EOF or ErrBufferFull
			// if ErrBufferFull, it's a partial line. We assume not a marker if too long?
			// But isMarker works on partial data if it doesn't have suffix.
			// isMarker needs suffix " --".
			// If buffer is full and we don't have \n, we might not have the suffix yet.
			// If we assume standard buffer size (4096), lines longer than that starting with "-- " are not markers.

			if err == nil || err == io.EOF {
				name, _ := isMarker(line)
				if name != "" {
					r.nextFile = File{Name: name}
					r.nextFileValid = true
					r.atStartOfLine = true
					return 0, io.EOF
				}
			}

			// Not a marker.
			// Copy to pending.
			r.pending = append([]byte(nil), line...)

			if err == nil {
				r.atStartOfLine = true
			} else if err == bufio.ErrBufferFull {
				r.atStartOfLine = false
			} else if err == io.EOF {
				r.atStartOfLine = false
			}

			// If error was not EOF/BufferFull, return it?
			if err != nil && err != io.EOF && err != bufio.ErrBufferFull {
				return 0, err
			}

			n = copy(p, r.pending)
			r.pending = r.pending[n:]
			return n, nil
		}
	}

	// Normal read until newline
	line, err := r.r.ReadSlice('\n')
	if len(line) > 0 {
		n = copy(p, line)
		if n < len(line) {
			r.pending = append([]byte(nil), line[n:]...)
		}

		if err == nil {
			r.atStartOfLine = true
		} else if err == bufio.ErrBufferFull {
			r.atStartOfLine = false
			err = nil // Hide BufferFull
		} else if err == io.EOF {
			r.atStartOfLine = false
			if len(r.pending) > 0 {
				err = nil
			}
		}
	}
	return n, err
}

// All returns an iterator over the files in the archive.
// The yielded File has a nil Data field.
// The caller must read the file content from r (the Reader itself) before the next iteration.
// This is similar to calling Next() in a loop.
func (r *Reader) All() iter.Seq2[File, error] {
	return func(yield func(File, error) bool) {
		for {
			f, err := r.Next()
			if err == io.EOF {
				return
			}
			if err != nil {
				yield(File{}, err)
				return
			}
			if !yield(f, nil) {
				return
			}
		}
	}
}

// AllWithData returns an iterator over the files in the archive.
// The yielded File has its Data field populated with the file content.
func (r *Reader) AllWithData() iter.Seq2[File, error] {
	return func(yield func(File, error) bool) {
		for {
			f, err := r.Next()
			if err == io.EOF {
				return
			}
			if err != nil {
				yield(File{}, err)
				return
			}
			data, err := io.ReadAll(r)
			if err != nil {
				yield(File{}, err)
				return
			}
			f.Data = data
			if !yield(f, nil) {
				return
			}
		}
	}
}
