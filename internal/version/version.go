// Package version holds build metadata set at compile time via ldflags.
package version

//nolint:gochecknoglobals // build metadata variables set via ldflags
var (
	// Version is the semantic version of the build.
	Version = "dev"
	// Commit is the git commit hash of the build.
	Commit = "none"
	// Date is the build date.
	Date = "unknown"
)
