package repository

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// files holds the implementation of the Source interface.
type files struct{ dir string }

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
	files, err := ioutil.ReadDir(r.dir)
	if err != nil {
		return nil, err
	}

	// Store all migrations within this map
	ret := make(map[int]string)

	// Iterate over the files
	for _, f := range files {
		// Get the name of the file
		n := f.Name()

		// Check that the entry is a file and that it ends with .sql
		// Continue with the next file if it doesn't pass
		if f.IsDir() || !strings.HasSuffix(n, ".sql") {
			continue
		}

		// Split the filename on underscore (_)
		// The first part should contain the version number
		v := strings.FieldsFunc(n, func(c rune) bool { return c == '_' })[0]

		// Make sure that the extract part is actually an integer
		if !versionRegexp.MatchString(v) {
			return nil, fmt.Errorf("%s does not contain a valid version number", n)
		}

		// Convert the string to int
		// The error can be ignored here since we know that v is an integer
		// since it passed the regexp matching from the lines above
		vi, _ := strconv.Atoi(v)

		// Make sure we don't add multiple files with the same version
		if _, ok := ret[vi]; ok {
			return nil, fmt.Errorf("unable to add %s, another file with the same version has already been added", n)
		}

		// Read the contents of the file
		d, err := ioutil.ReadFile(path.Join(r.dir, n))
		if err != nil {
			return nil, err
		}

		// Store contents of the sql file in the map
		ret[vi] = string(d)
	}

	// Return an error if no migrations were found
	if len(ret) == 0 {
		return nil, fmt.Errorf("no migrations found in %s", r.dir)
	}

	// Return the repo
	return ret, nil
}

// versionRegexp contains the regular expression that validates
// the version number that is extracted from each filename.
var versionRegexp *regexp.Regexp

// init compiles the regular expression
func init() {
	versionRegexp = regexp.MustCompile("^[0-9]+$")
}
