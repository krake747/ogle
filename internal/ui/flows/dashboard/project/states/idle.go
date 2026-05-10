package states

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
)

type idleKeyMap struct {
	Quit key.Binding
}

func (k idleKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k idleKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Quit}}
}

//nolint:gochecknoglobals // list of key bindings should be global and immutable
var defaultIdleKeys = idleKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
}

// Idle is the initial project state. It renders a minimal project summary
// while the full dashboard implementation is deferred.
type Idle struct {
	project *domain.Project
	keys    idleKeyMap
	help    help.Model
	w, h    int
}

// NewIdle returns an Idle state initialised with the given project.
func NewIdle(project *domain.Project) State {
	//nolint:exhaustruct // keys and help have defaults
	return &Idle{
		project: project,
		keys:    defaultIdleKeys,
		help:    help.New(),
	}
}

// Init implements State.
func (s *Idle) Init() tea.Cmd { return nil }

// SetSize implements State.
func (s *Idle) SetSize(w, h int) {
	s.w = w
	s.h = h
	s.help.SetWidth(w)
}

// Update handles the quit key.
func (s *Idle) Update(msg tea.Msg) (State, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if key.Matches(keyMsg, s.keys.Quit) {
			return s, tea.Quit
		}
	}

	return s, nil
}

// View renders a minimal project summary with a help bar on the last row.
func (s *Idle) View() string {
	helpBar := s.help.View(s.keys)

	var body string
	if s.project == nil {
		body = "ogle\n\n[no project loaded]\n"
	} else {
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

		sb.WriteString("\n\n[dashboard not yet implemented]\n")
		body = sb.String()
	}

	if s.h <= 0 {
		return body + "\n" + helpBar
	}

	bodyLines := strings.Count(body, "\n")
	if strings.HasSuffix(body, "\n") {
		bodyLines--
	}

	padding := max(s.h-bodyLines-1, 1)

	return body + strings.Repeat("\n", padding) + helpBar
}
