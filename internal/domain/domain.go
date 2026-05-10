// Package domain defines the core types and service interfaces for ogle.
package domain

// Project is the parsed, named in-memory representation of a Compose File.
type Project struct {
	Name     string
	File     string
	Services []ServiceDef
}

// ServiceDef is a single Service declared within a Compose File.
type ServiceDef struct {
	Name          string
	Image         string
	ContainerName string
}
