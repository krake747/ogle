// Package scanner provides file discovery for Docker Compose files.
package scanner

import (
	"log/slog"
	"os"
	"path/filepath"
)

//nolint:exhaustruct // compile-time assertion that Service satisfies Scanner.
var _ Scanner = Service{}

// Service performs file discovery for Docker Compose files.
type Service struct {
	logger *slog.Logger
}

// Scanner finds Compose Files on disk.
//
//mockery:generate: true
type Scanner interface {
	KnownFilenames() []string
	ScanAll(dir string) []string
}

// New constructs a Service with the given logger.
func New(logger *slog.Logger) Service {
	return Service{logger: logger}
}

// KnownFilenames returns the ordered list of compose filenames recognised by
// ogle, from highest to lowest priority. Files are returned as a new slice on
// each call so callers cannot mutate the canonical list.
func (s Service) KnownFilenames() []string {
	return knownFilenames()
}

// ScanAll returns the absolute paths of all known compose filenames that exist
// in dir, in priority order. Files that do not exist are silently omitted. No
// YAML validation is performed; call parser.Service.Validate on each path
// before use.
func (s Service) ScanAll(dir string) []string {
	return scanAll(dir)
}

func knownFilenames() []string {
	return []string{
		"compose.yml",
		"compose.yaml",
		"docker-compose.yml",
		"docker-compose.yaml",
	}
}

func scanAll(dir string) []string {
	var found []string

	for _, name := range knownFilenames() {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			found = append(found, path)
		}
	}

	return found
}
