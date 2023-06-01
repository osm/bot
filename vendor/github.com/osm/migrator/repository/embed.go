package repository

import (
	"embed"
	"io/fs"
)

// embedFiles holds the implementation of the Source interface.
type embedFiles struct {
	dir     string
	embedFS embed.FS
}

// FromEmbedded creates a new embedded file based repository
//
// It expects an embed.FS object and a valid directory path to be passed on
// invocation. Migration files should be named:
// "0001_something_something.sql".
// The version number should be incremented for each migration.
func FromEmbedded(embedFS embed.FS, dir string) Source {
	return &embedFiles{embedFS: embedFS, dir: dir}
}

// Load loads all migrations from the dir given when the repository
// was created.
func (r *embedFiles) Load() (map[int]string, error) {
	// List all files within the directory
	files, err := fs.ReadDir(r.embedFS, r.dir)
	if err != nil {
		return nil, err
	}

	return loadFiles(files, r.dir, &r.embedFS)
}
