// Package parser provides parsing and validation of Docker Compose files.
package parser

import (
	"cmp"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"go.yaml.in/yaml/v3"

	"github.com/ma-tf/ogle/internal/domain"
)

var (
	ErrReadComposeFile  = errors.New("failed to read compose file")
	ErrParseComposeFile = errors.New("failed to parse compose file")
)

//nolint:exhaustruct // compile-time assertion that Service satisfies Parser.
var _ Parser = Service{}

// Parser validates and parses Compose Files into Projects.
type Parser interface {
	Validate(path string) error
	Parse(path string) (*domain.Project, error)
}

// composeFile is the minimal YAML structure required for parsing.
type composeFile struct {
	Name     string `yaml:"name"`
	Services map[string]struct {
		Image         string `yaml:"image"`
		ContainerName string `yaml:"containerName"`
		Build         any    `yaml:"build"`
	} `yaml:"services"`
}

// Service exposes compose file validation and parsing.
type Service struct {
	logger *slog.Logger
}

// New constructs a Service with the given logger.
func New(logger *slog.Logger) Service {
	return Service{logger: logger}
}

// Validate returns nil if path exists on disk and can be parsed as a valid
// compose YAML file. It returns a wrapped ErrReadComposeFile or
// ErrParseComposeFile on failure.
func (s Service) Validate(path string) error {
	_, err := readAndUnmarshal(path)

	return err
}

// Parse reads and parses the compose file at path into a Project. path must
// be an absolute path to an existing, valid compose file; callers should use
// ScanAll and Validate before calling Parse.
func (s Service) Parse(path string) (*domain.Project, error) {
	cf, err := readAndUnmarshal(path)
	if err != nil {
		return nil, err
	}

	name := cf.Name
	if name == "" {
		name = filepath.Base(filepath.Dir(path))
	}

	services := make([]domain.ServiceDef, 0, len(cf.Services))
	for serviceName, svc := range cf.Services {
		services = append(services, domain.ServiceDef{
			Name:          serviceName,
			Image:         svc.Image,
			ContainerName: svc.ContainerName,
		})
	}

	slices.SortFunc(services, func(a, b domain.ServiceDef) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return &domain.Project{
		Name:     name,
		File:     path,
		Services: services,
	}, nil
}

func readAndUnmarshal(path string) (composeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return composeFile{}, fmt.Errorf("%w: %w", ErrReadComposeFile, err)
	}

	var cf composeFile
	if unmarshalErr := yaml.Unmarshal(data, &cf); unmarshalErr != nil {
		return composeFile{}, fmt.Errorf("%w: %w", ErrParseComposeFile, unmarshalErr)
	}

	return cf, nil
}
