package states

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/services/scanner"
)

// scanDoneMsg is the result of the initial ScanAll+Validate sweep.
type scanDoneMsg struct{ valid []string }

// parseDoneMsg is the result of a parser.Service.Parse call.
type parseDoneMsg struct {
	project *parser.Project
	err     error
}

// ScanCmd runs ScanAll then Validate on each candidate, returning only paths
// that pass Validate.
func ScanCmd(dir string, scannerSvc scanner.Service, parserSvc parser.Service) tea.Cmd {
	return func() tea.Msg {
		candidates := scannerSvc.ScanAll(dir)
		valid := validateFiles(candidates, parserSvc)

		return scanDoneMsg{valid: valid}
	}
}

// ParseCmd runs parser.Service.Parse asynchronously, returning a parseDoneMsg.
func ParseCmd(path string, parserSvc parser.Service) tea.Cmd {
	return func() tea.Msg {
		project, err := parserSvc.Parse(path)

		return parseDoneMsg{project: project, err: err}
	}
}

func validateFiles(paths []string, parserSvc parser.Service) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if parserSvc.Validate(p) == nil {
			out = append(out, p)
		}
	}

	return out
}
