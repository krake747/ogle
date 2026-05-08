package compose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

var (
	ErrNoComposeFileFound    = errors.New("no compose file found")
	ErrCouldNotGetWorkingDir = errors.New("failed to get working directory")
	ErrReadComposeFile       = errors.New("failed to read compose file")
	ErrParseComposeFile      = errors.New("failed to parse compose file")
)

// composeFile is the minimal YAML structure we need.
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

type Service struct {
	Name          string
	Image         string
	ContainerName string
}

func resolveFile(projectPath string) (string, error) {
	if projectPath != "" {
		if _, err := os.Stat(projectPath); err != nil {
			return "", fmt.Errorf("%w: %w", ErrNoComposeFileFound, err)
		}

		return projectPath, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrCouldNotGetWorkingDir, err)
	}

	composeFileNames := []string{
		"compose.yml",
		"compose.yaml",
		"docker-compose.yml",
		"docker-compose.yaml",
	}

	for _, fallbackPath := range composeFileNames {
		path := filepath.Join(cwd, fallbackPath)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("%w in %s (searched for %s)", ErrNoComposeFileFound, cwd, strings.Join(composeFileNames, ", "))
}

func Parse(projectPath string) (*Project, error) {
	path, err := resolveFile(projectPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadComposeFile, err)
	}

	var cf composeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParseComposeFile, err)
	}

	var name string = cf.Name
	if name == "" {
		name = filepath.Base(filepath.Dir(path))
	}

	services := make([]Service, 0, len(cf.Services))
	for serviceName, service := range cf.Services {
		services = append(services, Service{
			Name:          serviceName,
			Image:         service.Image,
			ContainerName: service.ContainerName,
		})
	}

	return &Project{
		Name:     name,
		File:     path,
		Services: services,
	}, nil
}
