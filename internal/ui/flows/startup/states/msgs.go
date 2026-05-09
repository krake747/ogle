package states

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/compose"
)

// ---- internal messages -----------------------------------------------------

// scanDoneMsg is the result of the initial ScanAll+Validate sweep.
type scanDoneMsg struct{ valid []string }

// parseDoneMsg is the result of a compose.Parse call.
type parseDoneMsg struct {
	project *compose.Project
	err     error
}

// ---- cmds ------------------------------------------------------------------

// ScanCmd runs ScanAll then Validate on each candidate. Only paths that pass
// Validate are included in the result. Injected into Scanning as its Scan
// field at construction time by startup.New.
func ScanCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		candidates := compose.ScanAll(dir)
		valid := validateFiles(candidates)

		return scanDoneMsg{valid: valid}
	}
}

// ParseCmd runs compose.Parse on path and returns a parseDoneMsg. Applied at
// transition time (when a file is chosen) by startup.makeHandleFiles and by
// Selecting.Update.
func ParseCmd(path string) tea.Cmd {
	return func() tea.Msg {
		project, err := compose.Parse(path)

		return parseDoneMsg{project: project, err: err}
	}
}

// ---- internal helpers ------------------------------------------------------

// validateFiles filters paths to those that pass compose.Validate.
func validateFiles(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if compose.Validate(p) == nil {
			out = append(out, p)
		}
	}

	return out
}
