// Package parser provides parsing and validation of Docker Compose files.
package parser

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"go.yaml.in/yaml/v3"

	"github.com/ma-tf/ogle/internal/domain"
)

var (
	ErrReadComposeFile  = errors.New("failed to read compose file")
	ErrParseComposeFile = errors.New("failed to parse compose file")
)

// Port parsing constants.
const (
	portPartsBindAddrMin   = 3 // minimum parts to have a bind address (addr:host:container)
	portPartsHostContainer = 2 // parts count for host:container format
)

//nolint:exhaustruct // compile-time assertion that Service satisfies Parser.
var _ Parser = Service{}

// Parser validates and parses Compose Files into Projects.
//
//mockery:generate: true
type Parser interface {
	Parse(path string) (*domain.Project, error)
}

// composeFile is the minimal YAML structure required for parsing.
type composeFile struct {
	Name     string `yaml:"name"`
	Services map[string]struct {
		Image         string            `yaml:"image"`
		ContainerName string            `yaml:"container_name"` //nolint:tagliatelle // Docker Compose uses snake_case
		Build         any               `yaml:"build"`
		Labels        map[string]string `yaml:"labels"`
		Ports         []any             `yaml:"ports"`
	} `yaml:"services"`
}

// Service exposes compose file validation and parsing.
type Service struct {
	ctx    context.Context
	logger *slog.Logger
}

// New constructs a Service with the given logger.
func New(ctx context.Context, logger *slog.Logger) Service {
	return Service{
		ctx:    ctx,
		logger: logger,
	}
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
			Labels:        svc.Labels,
			Ports:         normalizePorts(svc.Ports),
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

// normalizePorts converts Docker Compose port declarations into normalised
// display format "host→container/protocol". Returns an empty slice if ports
// is nil or empty.
func normalizePorts(ports []any) []string {
	if len(ports) == 0 {
		return nil
	}

	result := make([]string, 0, len(ports))
	for _, p := range ports {
		switch v := p.(type) {
		case string:
			// Short form: "8080:80", "8080:80/tcp", "127.0.0.1:8080:80/tcp", "80", etc.
			result = append(result, normalizeShortPort(v))
		case map[string]any:
			// Long form: {target: 80, published: 8080, protocol: tcp}
			result = append(result, normalizeLongPort(v))
		}
	}

	return result
}

// normalizeShortPort converts a short-form port string to "host→container/proto" format.
// Handles: "8080:80", "8080:80/tcp", "127.0.0.1:8080:80/tcp", "80", etc.
func normalizeShortPort(s string) string {
	// Strip bind address (e.g., "127.0.0.1:8080:80/tcp" → "8080:80/tcp")
	parts := splitByColon(s)
	protocol := "tcp" // default

	// Check if the last part contains a protocol
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if idx := findSlash(lastPart); idx >= 0 {
			protocol = lastPart[idx+1:]
			parts[len(parts)-1] = lastPart[:idx]
		}
	}

	// Now parts should be like ["8080", "80"] or ["80"] or ["127.0.0.1", "8080", "80"]
	// If we have 3+ parts, the first is the bind address, skip it
	if len(parts) >= portPartsBindAddrMin {
		parts = parts[1:]
	}

	// Normalize based on what we have
	if len(parts) == portPartsHostContainer {
		// "8080:80" → "8080→80/tcp"
		return parts[0] + "→" + parts[1] + "/" + protocol
	} else if len(parts) == 1 {
		// "80" → "→80/tcp" (no host port)
		return "→" + parts[0] + "/" + protocol
	}

	// Fallback (shouldn't happen with valid input)
	return s
}

// normalizeLongPort converts a long-form port object to "host→container/proto" format.
func normalizeLongPort(m map[string]any) string {
	protocol := "tcp"
	if p, ok := m["protocol"].(string); ok && p != "" {
		protocol = p
	}

	container := ""

	switch t := m["target"].(type) {
	case float64:
		container = strconv.Itoa(int(t))
	case string:
		container = t
	}

	host := ""

	switch pub := m["published"].(type) {
	case float64:
		host = strconv.Itoa(int(pub))
	case string:
		host = pub
	}

	if host != "" {
		return host + "→" + container + "/" + protocol
	}

	return "→" + container + "/" + protocol
}

// splitByColon splits a string by colons. Simple helper to avoid importing strings unnecessarily.
func splitByColon(s string) []string {
	var (
		result  []string
		current string
	)

	for _, ch := range s {
		if ch == ':' {
			result = append(result, current)
			current = ""
		} else {
			current += string(ch)
		}
	}

	result = append(result, current)

	return result
}

// findSlash returns the index of the first '/' in s, or -1 if not found.
func findSlash(s string) int {
	for i, ch := range s {
		if ch == '/' {
			return i
		}
	}

	return -1
}
