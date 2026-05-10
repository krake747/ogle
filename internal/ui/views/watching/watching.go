// Package watching provides the watching view: displayed during startup when
// no compose files are present (cold start) and when a file disappears at
// runtime (disconnected mode).
package watching

import (
	"fmt"
	"path/filepath"
	"strings"

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

const (
	minWidth        = 80
	minHeight       = 24
	maxContentWidth = 120
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
		m.width = max(sz.Width, minWidth)
		m.height = max(sz.Height, minHeight)

		return m, nil
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if m.state == stateError && keyMsg.String() == "r" {
			return m, func() tea.Msg { return msgs.RetryWatcher{} }
		}
	}

	return m, nil
}

// View renders the watching screen with a centred layout.
func (m Model) View() string {
	w := m.width
	if w == 0 {
		w = minWidth
	}

	h := m.height
	if h == 0 {
		h = minHeight
	}

	// Effective content width: fluid up to maxContentWidth, centred beyond that.
	contentWidth := min(w, maxContentWidth)

	leftPad := (w - contentWidth) / 2 //nolint:mnd // integer halving for centering

	// Build body text.
	var bodyText string

	switch m.mode {
	case ModeCold:
		bodyText = fmt.Sprintf("Watching %s for a compose file...", m.dir)
	case ModeDisconnected:
		bodyText = fmt.Sprintf("Disconnected — waiting for %s...", m.targetFile)
	}

	// Assemble content lines (each wrapped to contentWidth).
	var lines []string

	lines = append(lines, "ogle")
	lines = append(lines, "")
	lines = append(lines, wrapLine(bodyText, contentWidth)...)

	switch m.state {
	case stateIdle:
		// nothing extra in idle
	case stateNotice:
		lines = append(lines, "")
		lines = append(lines, wrapLine("notice: "+m.notice, contentWidth)...)
	case stateError:
		lines = append(lines, "")
		lines = append(lines, wrapLine(fmt.Sprintf("Error: %v", m.watcherErr), contentWidth)...)
	}

	// Rendered after the state block so it appears in all states, consistent
	// with fileselect. parsing.go clears the flag before entering notice/error
	// states, so this is a no-op in practice when state != stateIdle.
	if m.parsing {
		lines = append(lines, "")
		lines = append(lines, "Parsing...")
	}

	// Footer text.
	var footer string

	switch m.state {
	case stateIdle, stateNotice:
		footer = "ctrl+c quit"
	case stateError:
		footer = "r retry   ctrl+c quit"
	}

	// Vertical centering: content block centred in (h-1) rows, footer on row h.
	availableRows := h - 1 // last row reserved for footer

	topPad := max((availableRows-len(lines))/2, 0) //nolint:mnd // integer halving for centering

	var sb strings.Builder

	pad := strings.Repeat(" ", leftPad)

	// Leading blank lines.
	for range topPad {
		sb.WriteByte('\n')
	}

	// Content.
	for _, l := range lines {
		if l == "" {
			sb.WriteByte('\n')
		} else {
			sb.WriteString(pad + l + "\n")
		}
	}

	// Fill remaining rows before footer.
	rendered := topPad + len(lines)
	for i := rendered; i < availableRows; i++ {
		sb.WriteByte('\n')
	}

	// Footer on last row (no trailing newline — bubbletea handles cursor).
	sb.WriteString(pad + footer)

	return sb.String()
}

// wrapLine wraps s into lines of at most width runes. Operates on runes to
// handle multi-byte characters (e.g. em dash) correctly.
func wrapLine(s string, width int) []string {
	runes := []rune(s)
	if width <= 0 || len(runes) <= width {
		return []string{s}
	}

	var out []string

	for len(runes) > width {
		// Scan backwards from width for a space break point.
		cut := width
		for i := width - 1; i > 0; i-- {
			if runes[i] == ' ' {
				cut = i

				break
			}
		}

		out = append(out, string(runes[:cut]))
		runes = runes[cut:]

		// Trim leading spaces from the remainder.
		for len(runes) > 0 && runes[0] == ' ' {
			runes = runes[1:]
		}
	}

	if len(runes) > 0 {
		out = append(out, string(runes))
	}

	return out
}
