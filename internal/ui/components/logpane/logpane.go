package logpane

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

const (
	defaultCap    = 1000
	scrollHalfDiv = 2
)

// Model stores raw log text lines with scroll and pause support.
type Model struct {
	lines      []string
	cap        int
	scrollRows int
	paused     bool
	h          int
}

// New returns a Model.
func New() Model {
	return Model{
		lines:      nil,
		cap:        defaultCap,
		scrollRows: 0,
		paused:     false,
		h:          0,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles log lines, resize, and scroll keys.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.LogLine:
		m.lines = append(m.lines, msg.Text)
		if len(m.lines) > m.cap {
			m.lines = m.lines[len(m.lines)-m.cap:]
		}

		if !m.paused {
			m.scrollRows = 0
		}

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgup"))):
			half := max(m.h/scrollHalfDiv, 1)
			m.scrollRows += half
			m.paused = true
		case key.Matches(msg, key.NewBinding(key.WithKeys("pgdn"))):
			half := max(m.h/scrollHalfDiv, 1)

			m.scrollRows -= half
			if m.scrollRows < 0 {
				m.scrollRows = 0
			}

			if m.scrollRows == 0 {
				m.paused = false
			}
		}

	case tea.WindowSizeMsg:
		m.h = msg.Height
	}

	return m, nil
}

// View returns visible log lines sliced by scroll offset.
func (m Model) View() string {
	if m.h <= 0 || len(m.lines) == 0 {
		return ""
	}

	totalRows := len(m.lines)
	maxScroll := max(totalRows-m.h, 0)
	m.scrollRows = clamp(m.scrollRows, 0, maxScroll)

	end := max(totalRows-m.scrollRows, 0)
	start := max(end-m.h, 0)

	out := m.lines[start:end]

	if m.paused {
		out = append(out, "── paused ──")
	}

	return strings.Join(out, "\n")
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}
