// Package watching provides the watching view: displayed during startup when
// no compose files are present (cold start) and when a file disappears at
// runtime (disconnected mode).
package watching

import (
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

// Mode controls the heading and file-matching behaviour of the watching view.
type Mode int

const (
	// ModeCold is the cold-start state: no compose file has ever been loaded.
	ModeCold Mode = iota

	// ModeDisconnected is the runtime-disconnect state: a file was loaded but
	// has since disappeared. FileAvailabilityChanged events are only acted on
	// if the specific target filename reappears.
	ModeDisconnected
)

type state int

const (
	stateIdle   state = iota // monitoring; no notice
	stateNotice              // file exists but YAML is invalid (transient)
	stateError               // watcher.New failed; recoverable via 'r'
)

// Model is the watching view. It is a value type; all mutating methods return
// a new Model.
type Model struct {
	mode       Mode
	dir        string // watched directory (displayed in cold mode)
	targetFile string // basename watched in disconnected mode
	state      state
	notice     string // set in stateNotice
	watcherErr error  // set in stateError
	parsing    bool
	width      int
	height     int
}

// New returns a cold-mode Model watching dir.
func New(dir string, width, height int) Model {
	return Model{
		mode:       ModeCold,
		dir:        dir,
		targetFile: "",
		state:      stateIdle,
		notice:     "",
		watcherErr: nil,
		parsing:    false,
		width:      width,
		height:     height,
	}
}

// NewDisconnected returns a disconnected-mode Model waiting for targetFile to
// reappear. targetFile must be a basename (e.g. "docker-compose.yaml").
func NewDisconnected(targetFile string, width, height int) Model {
	return Model{
		mode:       ModeDisconnected,
		dir:        "",
		targetFile: filepath.Base(targetFile),
		state:      stateIdle,
		notice:     "",
		watcherErr: nil,
		parsing:    false,
		width:      width,
		height:     height,
	}
}

// SetNotice enters stateNotice with the provided message. Used when a file
// exists on disk but fails compose.Validate.
func (m Model) SetNotice(msg string) Model {
	m.state = stateNotice
	m.notice = msg

	return m
}

// ClearNotice returns the model to stateIdle.
func (m Model) ClearNotice() Model {
	m.state = stateIdle
	m.notice = ""

	return m
}

// SetError enters stateError. Used when watcher.New fails.
func (m Model) SetError(err error) Model {
	m.state = stateError
	m.watcherErr = err

	return m
}

// ClearError returns the model to stateIdle.
func (m Model) ClearError() Model {
	m.state = stateIdle
	m.watcherErr = nil

	return m
}

// SetParsing sets the parsing indicator. When true, a "Parsing..." notice is
// shown inline. Consistent with SetNotice / SetError.
func (m Model) SetParsing(v bool) Model {
	m.parsing = v

	return m
}

// Init satisfies tea.Model. The watching view has no startup commands.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles key input. In stateError, 'r' emits msgs.RetryWatcher which
// app.go intercepts to retry watcher initialisation.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = sz.Width
		m.height = sz.Height

		return m, nil
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if m.state == stateError && keyMsg.String() == "r" {
			return m, func() tea.Msg { return msgs.RetryWatcher{} }
		}
	}

	return m, nil
}

// View renders the watching screen with the title pinned top-left and the
// body block anchored to the bottom-left.
func (m Model) View() tea.View {
	w := m.width
	h := m.height

	var bodyText string

	switch m.mode {
	case ModeCold:
		bodyText = fmt.Sprintf("Watching %s for a compose file...", m.dir)
	case ModeDisconnected:
		bodyText = fmt.Sprintf("Disconnected — waiting for %s...", m.targetFile)
	}

	body := lipgloss.NewStyle().Width(w).Render(bodyText)

	switch m.state {
	case stateIdle:
		// nothing extra in idle
	case stateNotice:
		body += "\n\n" + lipgloss.NewStyle().Width(w).Render("notice: "+m.notice)
	case stateError:
		body += "\n\n" + lipgloss.NewStyle().Width(w).Render(fmt.Sprintf("Error: %v", m.watcherErr))
	}

	// Rendered after the state block so it appears in all states, consistent
	// with fileselect. parsing.go clears the flag before entering notice/error
	// states, so this is a no-op in practice when state != stateIdle.
	if m.parsing {
		body += "\n\nParsing..."
	}

	var footer string

	switch m.state {
	case stateIdle, stateNotice:
		footer = "ctrl+c quit"
	case stateError:
		footer = "r retry   ctrl+c quit"
	}

	content := body + "\n\n" + footer

	// Lipgloss owns everything below the title: Height(h-1) with bottom
	// alignment pins the body+footer block to the last rows of the screen.
	return tea.NewView("ogle\n" + lipgloss.NewStyle().
		Width(w).
		Height(h-1).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(content))
}
