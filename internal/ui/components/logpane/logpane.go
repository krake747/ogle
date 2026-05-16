package logpane

import (
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

const (
	defaultCap = 1000
)

// Model stores raw log text lines backed by a viewport for windowed rendering.
type Model struct {
	lines    []string
	cap      int
	viewport viewport.Model
}

// New returns a Model with the given width. Height is teminal height.
func New(w, h int) Model {
	vp := viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))
	vp.KeyMap = viewport.KeyMap{}

	return Model{
		lines:    nil,
		cap:      defaultCap,
		viewport: vp,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles log lines, resize, and delegates to the viewport.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.LogLine:
		m.lines = append(m.lines, strings.TrimRight(msg.Text, "\n"))
		if len(m.lines) > m.cap {
			m.lines = m.lines[len(m.lines)-m.cap:]
		}

		wasAtBottom := m.viewport.AtBottom()
		m.viewport.SetContentLines(m.lines)

		if wasAtBottom {
			m.viewport.GotoBottom()
		}

	case tea.WindowSizeMsg:
		m.viewport.SetWidth(msg.Width)
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

// View returns the viewport-rendered window of log lines.
func (m Model) View() string {
	return m.viewport.View()
}
