package clifs

import (
	"io"
	"os"
)

type FS interface {
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(name string, data []byte, perm os.FileMode) error
	Stdout() io.Writer
	Stderr() io.Writer
}

type DefaultFS struct{}

func (DefaultFS) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (DefaultFS) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (DefaultFS) Stdout() io.Writer {
	return os.Stdout
}

func (DefaultFS) Stderr() io.Writer {
	return os.Stderr
}
