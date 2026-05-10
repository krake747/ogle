// Package parser provides parsing and validation of Docker Compose files.
package parser

import (
	"errors"
	"fmt"
	"log/slog"
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
		ContainerName string `yaml:"containerName"`
		Build         any    `yaml:"build"`
	} `yaml:"services"`
}

// Project represents a parsed Docker Compose project.
type Project struct {
	Name     string
	File     string
	Services []ServiceDef
}

// ServiceDef represents a single service declared within a Docker Compose project.
type ServiceDef struct {
	Name          string
	Image         string
	ContainerName string
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
func (s Service) Parse(path string) (*Project, error) {
	cf, err := readAndUnmarshal(path)
	if err != nil {
		return nil, err
	}

	name := cf.Name
	if name == "" {
		name = filepath.Base(filepath.Dir(path))
	}

	services := make([]ServiceDef, 0, len(cf.Services))
	for serviceName, svc := range cf.Services {
		services = append(services, ServiceDef{
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
