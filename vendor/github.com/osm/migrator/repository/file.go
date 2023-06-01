package repository

import (
	"os"
)

// files holds the implementation of the Source interface.
type files struct {
	dir string
}

// FromFiles creates a new file based repository
//
// It expects a valid directory path to be passed on invocation.
// Migration files should be named: "0001_something_something.sql".
// The version number should be incremented for each migration.
func FromFiles(dir string) Source {
	return &files{dir: dir}
}

// Load loads all migrations from the dir given when the repository
// was created.
func (r *files) Load() (map[int]string, error) {
	// List all files within the directory
	files, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}

	return loadFiles(files, r.dir, nil)
}
