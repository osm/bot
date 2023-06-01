package repository

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// versionRegexp contains the regular expression that validates
// the version number that is extracted from each filename.
var versionRegexp *regexp.Regexp

// init compiles the regular expression
func init() {
	versionRegexp = regexp.MustCompile("^[0-9]+$")
}

func loadFiles(files []fs.DirEntry, dir string, efs *embed.FS) (map[int]string, error) {
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
		var d []byte
		var err error
		if efs != nil {
			d, err = efs.ReadFile(path.Join(dir, n))
		} else {
			d, err = os.ReadFile(path.Join(dir, n))
		}
		if err != nil {
			return nil, err
		}

		// Store contents of the sql file in the map
		ret[vi] = string(d)
	}

	// Return an error if no migrations were found
	if len(ret) == 0 {
		return nil, fmt.Errorf("no migrations found in %s", dir)
	}

	// Return the repo
	return ret, nil
}
