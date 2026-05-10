package states

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/services/scanner"
)

// scanDoneMsg is the result of the initial ScanAll+Validate sweep.
type scanDoneMsg struct{ valid []string }

// parseDoneMsg is the result of a domain.Parser.Parse call.
type parseDoneMsg struct {
	project *domain.Project
	err     error
}

// ScanCmd runs ScanAll then Validate on each candidate, returning only paths
// that pass Validate.
func ScanCmd(
	dir string,
	sc scanner.Scanner,
	p parser.Parser,
) tea.Cmd {
	return func() tea.Msg {
		candidates := sc.ScanAll(dir)
		valid := validateFiles(candidates, p)

		return scanDoneMsg{valid: valid}
	}
}

// ParseCmd runs parser.Parser.Parse asynchronously, returning a parseDoneMsg.
func ParseCmd(
	path string,
	p parser.Parser,
) tea.Cmd {
	return func() tea.Msg {
		project, err := p.Parse(path)

		return parseDoneMsg{project: project, err: err}
	}
}

func validateFiles(paths []string, p parser.Parser) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if p.Validate(path) == nil {
			out = append(out, path)
		}
	}

	return out
}
