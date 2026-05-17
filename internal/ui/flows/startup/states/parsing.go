package states

import (
	"errors"
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/parser"
)

// Parsing is the invisible state while a parser.Service.Parse call is in flight. It
// holds the last visible state (Watching or Selecting) so View() and input
// forwarding remain unchanged during the parse.
type Parsing struct {
	path    string
	parse   tea.Cmd
	display tea.Model // Watching or Selecting
}

// NewParsing constructs a Parsing state for the given path, using display as
// the underlying visible state. It activates the "Parsing..." notice on the
// display view.
func NewParsing(
	path string,
	display tea.Model,
	p parser.Parser,
) tea.Model {
	return Parsing{
		path:    path,
		parse:   ParseCmd(path, p),
		display: setParsing(display, true),
	}
}

// Init fires the parse command. Only meaningful for the -f startup case;
// mid-session transitions deliver Parse via Update's return value.
func (p Parsing) Init() tea.Cmd {
	return p.parse
}

// Update handles the parse result. Other messages are forwarded to display
// to keep the UI responsive during the parse.
func (p Parsing) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if done, ok := msg.(parseDoneMsg); ok {
		return p.handleParseDone(done)
	}

	updated, cmd := p.display.Update(msg)

	return Parsing{path: p.path, parse: p.parse, display: updated}, cmd
}

// View keeps the UI unchanged during the parse.
func (p Parsing) View() tea.View {
	return p.display.View()
}

func (p Parsing) handleParseDone(done parseDoneMsg) (tea.Model, tea.Cmd) {
	if done.err == nil {
		return p, func() tea.Msg {
			return msgs.ProjectLoaded{Project: done.project}
		}
	}

	// Race: file disappeared between Validate and Parse. The watcher will
	// deliver FileAvailabilityChanged shortly; return to display and wait.
	if errors.Is(done.err, parser.ErrReadComposeFile) {
		return setParsing(p.display, false), nil
	}

	// Parse failed with a real error: surface it on the display state's sub-model.
	switch d := p.display.(type) {
	case Selecting:
		return d.withParsing(false).withError(p.path, done.err), nil
	case Watching:
		notice := fmt.Sprintf("%s could not be parsed: %v", filepath.Base(p.path), done.err)

		return d.withParsing(false).withNotice(notice), nil
	default:
		return p.display, nil
	}
}

// setParsing sets the parsing indicator on display if it is a known display
// type. Unknown display types are returned unchanged.
func setParsing(display tea.Model, v bool) tea.Model {
	switch d := display.(type) {
	case Watching:
		return d.withParsing(v)
	case Selecting:
		return d.withParsing(v)
	}

	return display
}
