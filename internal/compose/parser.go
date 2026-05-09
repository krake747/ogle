package compose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

var (
	ErrReadComposeFile  = errors.New("failed to read compose file")
	ErrParseComposeFile = errors.New("failed to parse compose file")
)

// composeFile is the minimal YAML structure required for parsing.
type composeFile struct {
	Name     string `yaml:"name"`
	Services map[string]struct {
		Image         string `yaml:"image"`
		ContainerName string `yaml:"container_name"`
		Build         any    `yaml:"build"`
	} `yaml:"services"`
}

// Project represents a parsed Docker Compose project.
type Project struct {
	Name     string
	File     string
	Services []Service
}

// Service represents a single service within a Docker Compose project.
type Service struct {
	Name          string
	Image         string
	ContainerName string
}

// Validate returns nil if path exists on disk and can be parsed as a valid
// compose YAML file. It returns a wrapped ErrReadComposeFile or
// ErrParseComposeFile on failure.
func Validate(path string) error {
	_, err := readAndUnmarshal(path)

	return err
}

// Parse reads and parses the compose file at path into a Project. path must
// be an absolute path to an existing, valid compose file; callers should use
// ScanAll and Validate before calling Parse.
func Parse(path string) (*Project, error) {
	cf, err := readAndUnmarshal(path)
	if err != nil {
		return nil, err
	}

	name := cf.Name
	if name == "" {
		name = filepath.Base(filepath.Dir(path))
	}

	services := make([]Service, 0, len(cf.Services))
	for serviceName, svc := range cf.Services {
		services = append(services, Service{
			Name:          serviceName,
			Image:         svc.Image,
			ContainerName: svc.ContainerName,
		})
	}

	return &Project{
		Name:     name,
		File:     path,
		Services: services,
	}, nil
}

// readAndUnmarshal reads path from disk and unmarshals it into a composeFile.
// It is the shared implementation used by both Validate and Parse.
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
