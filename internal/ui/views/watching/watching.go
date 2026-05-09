// Package watching provides the watching view: displayed during startup when
// no compose files are present (cold start) and when a file disappears at
// runtime (disconnected mode).
package watching

import (
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

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
}

// New returns a cold-mode Model watching dir.
func New(dir string) Model {
	return Model{
		mode:       ModeCold,
		dir:        dir,
		targetFile: "",
		state:      stateIdle,
		notice:     "",
		watcherErr: nil,
	}
}

// NewDisconnected returns a disconnected-mode Model waiting for targetFile to
// reappear. targetFile must be a basename (e.g. "docker-compose.yaml").
func NewDisconnected(targetFile string) Model {
	return Model{
		mode:       ModeDisconnected,
		dir:        "",
		targetFile: filepath.Base(targetFile),
		state:      stateIdle,
		notice:     "",
		watcherErr: nil,
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

// Init satisfies tea.Model. The watching view has no startup commands.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles key input. In stateError, 'r' emits msgs.RetryWatcher which
// app.go intercepts to retry watcher initialisation.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if m.state == stateError && keyMsg.String() == "r" {
			return m, func() tea.Msg { return msgs.RetryWatcher{} }
		}
	}

	return m, nil
}

// View renders the watching screen. All output fits within 80 columns.
func (m Model) View() string {
	var body string

	switch m.mode {
	case ModeCold:
		body = fmt.Sprintf("Watching %s for a compose file...", m.dir)
	case ModeDisconnected:
		body = fmt.Sprintf("Disconnected — waiting for %s...", m.targetFile)
	}

	out := "ogle\n\n" + body + "\n"

	switch m.state {
	case stateIdle:
		// No additional output in idle state.
	case stateNotice:
		out += "\nnotice: " + m.notice + "\n"
	case stateError:
		out += fmt.Sprintf("\nError: %v\n", m.watcherErr)
	}

	out += "\n"

	switch m.state {
	case stateIdle, stateNotice:
		out += "ctrl+c quit\n"
	case stateError:
		out += "r retry   ctrl+c quit\n"
	}

	return out
}
