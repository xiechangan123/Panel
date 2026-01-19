package storage

import (
	"io"
	"time"
)

type Storage interface {
	// Delete deletes the given file(s).
	Delete(file ...string) error
	// Exists determines if a file exists.
	Exists(file string) bool
	// LastModified gets the file's last modified time.
	LastModified(file string) (time.Time, error)
	// List lists all files (not directories) in the given path.
	List(path string) ([]string, error)
	// Put writes the contents of a file.
	Put(file string, content io.Reader) error
	// Size gets the file size of a given file.
	Size(file string) (int64, error)
}
