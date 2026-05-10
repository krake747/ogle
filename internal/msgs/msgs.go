package msgs

import "github.com/ma-tf/ogle/internal/services/parser"

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
	Project *parser.Project
}

// WatcherError is delivered when watcher.New fails — either on initial startup
// or after a retry attempt. The startup flow forwards this to the watching view
// to enter watchingError state.
type WatcherError struct{ Err error }

// RetryWatcher is emitted by the watching view when the user presses 'r' in
// the watchingError state. app.go handles it by retrying watcher.New.
type RetryWatcher struct{}
