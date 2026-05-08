package msgs

import "github.com/ma-tf/ogle/internal/compose"

// FileAvailabilityChanged is emitted by the watcher whenever compose file
// presence in the watched directory changes. Files contains the absolute paths
// of all compose filenames that currently exist on disk. Consumers are
// responsible for calling compose.Validate on each path before use; the watcher
// only performs existence checks.
type FileAvailabilityChanged struct {
	Files []string
}

// FileSelected is emitted by the fileselect view when the user confirms a file
// choice from the picker.
type FileSelected struct {
	Path string
}

// ProjectLoaded is emitted by the startup flow after a successful
// compose.Parse call and signals the app root to transition to the dashboard.
type ProjectLoaded struct {
	Project *compose.Project
}
