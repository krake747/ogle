package compose

import (
	"os"
	"path/filepath"
)

// KnownFilenames returns the ordered list of compose filenames recognised by
// ogle, from highest to lowest priority. Files are returned as a new slice on
// each call so callers cannot mutate the canonical list.
func KnownFilenames() []string {
	return []string{
		"compose.yml",
		"compose.yaml",
		"docker-compose.yml",
		"docker-compose.yaml",
	}
}

// ScanAll returns the absolute paths of all KnownFilenames that exist in dir,
// in priority order. Files that do not exist are silently omitted. No YAML
// validation is performed; call Validate on each path before use.
func ScanAll(dir string) []string {
	var found []string
	for _, name := range KnownFilenames() {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			found = append(found, path)
		}
	}

	return found
}
