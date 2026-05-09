package states

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/compose"
)

// scanDoneMsg is the result of the initial ScanAll+Validate sweep.
type scanDoneMsg struct{ valid []string }

// parseDoneMsg is the result of a compose.Parse call.
type parseDoneMsg struct {
	project *compose.Project
	err     error
}

// ScanCmd runs ScanAll then Validate on each candidate, returning only paths
// that pass Validate.
func ScanCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		candidates := compose.ScanAll(dir)
		valid := validateFiles(candidates)

		return scanDoneMsg{valid: valid}
	}
}

func ParseCmd(path string) tea.Cmd {
	return func() tea.Msg {
		project, err := compose.Parse(path)

		return parseDoneMsg{project: project, err: err}
	}
}

func validateFiles(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if compose.Validate(p) == nil {
			out = append(out, p)
		}
	}

	return out
}
