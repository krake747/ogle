package main

import "github.com/ma-tf/ogle/cmd"

//nolint:gochecknoglobals // build metadata variables
var (
	// version, commit, and date are set at build time using ldflags.
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.Execute(version, commit, date)
}
