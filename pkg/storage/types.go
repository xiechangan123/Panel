package storage

import (
	"context"
	"io"
	"time"
)

type Storage interface {
	// Delete deletes the given file(s).
	Delete(ctx context.Context, file ...string) error
	// Exists determines if a file exists.
	Exists(ctx context.Context, file string) bool
	// LastModified gets the file's last modified time.
	LastModified(ctx context.Context, file string) (time.Time, error)
	// List lists all files (not directories) in the given path.
	List(ctx context.Context, path string) ([]string, error)
	// Put writes the contents of a file.
	Put(ctx context.Context, file string, content io.Reader) error
	// Size gets the file size of a given file.
	Size(ctx context.Context, file string) (int64, error)
}
