package storage

import (
	"time"
)

type Storage interface {
	// MakeDirectory creates a directory.
	MakeDirectory(directory string) error
	// DeleteDirectory deletes the given directory.
	DeleteDirectory(directory string) error
	// Copy the given file to a new location.
	Copy(oldFile, newFile string) error
	// Delete deletes the given file(s).
	Delete(file ...string) error
	// Exists determines if a file exists.
	Exists(file string) bool
	// Files gets all the files from the given directory.
	Files(path string) ([]string, error)
	// Get gets the contents of a file.
	Get(file string) ([]byte, error)
	// LastModified gets the file's last modified time.
	LastModified(file string) (time.Time, error)
	// MimeType gets the file's mime type.
	MimeType(file string) (string, error)
	// Missing determines if a file is missing.
	Missing(file string) bool
	// Move a file to a new location.
	Move(oldFile, newFile string) error
	// Path gets the full path for the file.
	Path(file string) string
	// Put writes the contents of a file.
	Put(file, content string) error
	// Size gets the file size of a given file.
	Size(file string) (int64, error)
}
