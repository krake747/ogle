package states

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/compose"
)

// Idle is the initial project state. It renders a minimal project summary —
// the same view previously provided by internal/ui/views/dashboard — while
// the full dashboard implementation is deferred.
//
// Idle supersedes internal/ui/views/dashboard (deleted in migration step 4).
type Idle struct {
	project *compose.Project
}

// NewIdle returns an Idle state loaded with the given project.
func NewIdle(project *compose.Project) State {
	return Idle{project: project}
}

// Init has no startup commands.
func (s Idle) Init() tea.Cmd { return nil }

// Update handles the quit key. Full input handling is deferred.
func (s Idle) Update(msg tea.Msg) (State, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if keyMsg.String() == "q" {
			return s, tea.Quit
		}
	}

	return s, nil
}

// View renders a minimal project summary. Fits within 80 columns.
func (s Idle) View() string {
	if s.project == nil {
		return "ogle\n\n[no project loaded]\n\nq quit\n"
	}

	var sb strings.Builder

	sb.WriteString("ogle\n\n")
	fmt.Fprintf(&sb, "Project: %s   (%s)\n", s.project.Name, s.project.File)

	count := len(s.project.Services)
	fmt.Fprintf(&sb, "Services: %d", count)

	if count > 0 {
		names := make([]string, 0, count)
		for _, svc := range s.project.Services {
			names = append(names, svc.Name)
		}

		sb.WriteString("  —  " + strings.Join(names, ", "))
	}

	sb.WriteString("\n\n[dashboard not yet implemented]\n\nq quit\n")

	return sb.String()
}
