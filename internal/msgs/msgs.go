package msgs

import (
	"time"

	"github.com/ma-tf/ogle/internal/domain"
)

// FileAvailabilityChanged is emitted by the watcher whenever compose file
// presence in the watched directory changes. Files contains the absolute paths
// of all compose filenames that currently exist on disk. Consumers are
// responsible for calling parser.Service.Validate on each path before use;
// the watcher only performs existence checks.
type FileAvailabilityChanged struct {
	Files []string
}

// FileSelected is emitted by the fileselect view when the user confirms a file
// choice from the picker.
type FileSelected struct {
	Path string
}

// ProjectLoaded is emitted by the startup flow after a successful
// parser.Service.Parse call and signals the app root to transition to the dashboard.
type ProjectLoaded struct {
	Project *domain.Project
}

// WatcherError is delivered when watcher.New fails — either on initial startup
// or after a retry attempt. The startup flow forwards this to the watching view
// to enter watchingError state.
type WatcherError struct{ Err error }

// RetryWatcher is emitted by the watching view when the user presses 'r' in
// the watchingError state. app.go handles it by retrying watcher.New.
type RetryWatcher struct{}

// ServiceSelected is emitted by the service list component when the cursor
// moves to a new service.
type ServiceSelected struct {
	Service domain.ServiceDef
}

// DaemonConnected is emitted by the docker service when the Docker daemon ping
// succeeds. It signals the Dashboard to start State Polling and Log Stream.
type DaemonConnected struct{}

// DaemonUnavailable is emitted by the docker service when the Docker daemon
// cannot be reached. The Dashboard shows a retry countdown and freezes Service
// States at their last-known values.
type DaemonUnavailable struct{ Err error }

// ServiceActionCompleted is emitted by a docker action cmd when the
// docker compose subprocess exits, whether successfully or not.
type ServiceActionCompleted struct {
	ServiceName string
	Action      domain.ServiceAction
	Err         error
}

// LogLine carries one demultiplexed log frame from the Docker logs API.
type LogLine struct {
	Text        string
	IsStderr    bool
	ServiceName string
}

// LogStreamError is emitted when the LogStreamer goroutine hits a read error.
type LogStreamError struct {
	Err         error
	ServiceName string
}

// LogStreamContainerNotFound is emitted when the logs endpoint returns 404.
type LogStreamContainerNotFound struct {
	ServiceName string
}

// SettingsApplied is emitted by states.Settings when the user confirms changes.
// dashboard.Model handles it to update the active configuration for the session.
type SettingsApplied struct {
	Theme        string
	PollInterval time.Duration
	LogBufferCap int
}

// OrphanDiscovered is emitted when a running container is found that has no
// corresponding Service in the current Project. Dashboard creates a Service
// Layer for it so logs and state are visible.
type OrphanDiscovered struct {
	Service domain.ServiceDef
}

// OrphanGone is emitted when a previously discovered Orphan container stops
// or disappears. Dashboard closes and removes its Service Layer.
type OrphanGone struct {
	ServiceName string
}
