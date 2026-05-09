package states

import (
	"errors"
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/compose"
	"github.com/ma-tf/ogle/internal/msgs"
)

// Parsing is the invisible state while a compose.Parse call is in flight. It
// holds the last visible state (Watching or Selecting) so View() and input
// forwarding remain unchanged during the parse.
type Parsing struct {
	Path    string
	Parse   tea.Cmd
	Display tea.Model // Watching or Selecting
}

// Init fires the parse command. Only meaningful for the -f startup case;
// mid-session transitions deliver Parse via Update's return value.
func (p Parsing) Init() tea.Cmd {
	return p.Parse
}

// Update handles the parse result. Other messages are forwarded to Display
// to keep the UI responsive during the parse.
func (p Parsing) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if done, ok := msg.(parseDoneMsg); ok {
		return p.handleParseDone(done)
	}

	updated, cmd := p.Display.Update(msg)

	return Parsing{Path: p.Path, Parse: p.Parse, Display: updated}, cmd
}

// View keeps the UI unchanged during the parse.
func (p Parsing) View() tea.View { return p.Display.View() }

func (p Parsing) handleParseDone(done parseDoneMsg) (tea.Model, tea.Cmd) {
	if done.err == nil {
		return p, func() tea.Msg {
			return msgs.ProjectLoaded{Project: done.project}
		}
	}

	// Race: file disappeared between Validate and Parse. The watcher will
	// deliver FileAvailabilityChanged shortly; return to display and wait.
	if errors.Is(done.err, compose.ErrReadComposeFile) {
		return p.Display, nil
	}

	// Parse failed with a real error: surface it on the Display state's sub-model.
	switch d := p.Display.(type) {
	case Selecting:
		return Selecting{Model: d.Model.SetError(p.Path, done.err), HandleFiles: d.HandleFiles}, nil
	case Watching:
		notice := fmt.Sprintf("%s could not be parsed: %v", filepath.Base(p.Path), done.err)

		return Watching{Model: d.Model.SetNotice(notice), HandleFiles: d.HandleFiles}, nil
	default:
		return p.Display, nil
	}
}
