// Package domain defines the core types and service interfaces for ogle.
package domain

import "time"

// ServiceState is the current Docker container state for a Service.
type ServiceState string

// ServiceState constants enumerate the possible Docker container states.
const (
	ServiceStateRunning    ServiceState = "running"
	ServiceStateExited     ServiceState = "exited"
	ServiceStatePaused     ServiceState = "paused"
	ServiceStateRestarting ServiceState = "restarting"
	ServiceStateDead       ServiceState = "dead"
	ServiceStateNotCreated ServiceState = "not created"
	ServiceStateUnknown    ServiceState = "unknown"
)

// ServiceHealth is the Docker health check result for a Service.
type ServiceHealth string

// ServiceHealth constants enumerate the possible Docker health check results.
const (
	ServiceHealthHealthy       ServiceHealth = "healthy"
	ServiceHealthUnhealthy     ServiceHealth = "unhealthy"
	ServiceHealthStarting      ServiceHealth = "starting"
	ServiceHealthNoHealthcheck ServiceHealth = "no healthcheck"
	ServiceHealthUnknown       ServiceHealth = "unknown"
)

// ServiceRuntimeData holds the live Docker state for a Service. It is nil
// until the first successful DaemonConnected message is received.
type ServiceRuntimeData struct {
	ContainerID string
	State       ServiceState
	Health      ServiceHealth
	StateAge    time.Duration
}

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
	Labels        map[string]string
}
