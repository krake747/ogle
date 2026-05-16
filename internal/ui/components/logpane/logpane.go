package logpane

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

const defaultCap = 1000

// Model stores raw log text lines backed by a viewport for windowed rendering.
// Lines arrive asynchronously via lineCh and are flushed when a
// msgs.LogLinesAvailable message is received.
type Model struct {
	lines    []string
	cap      int
	viewport viewport.Model
	lineCh   <-chan string
}

// New returns a Model reading from the given line channel.
func New(w, h int, lineCh <-chan string) Model {
	vp := viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))
	vp.KeyMap = viewport.KeyMap{}

	return Model{
		lines:    nil,
		cap:      defaultCap,
		viewport: vp,
		lineCh:   lineCh,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update drains the line channel on availability signals and delegates to the viewport.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.LogLinesAvailable:
		if m.lineCh == nil {
			return m, nil
		}

		for {
			select {
			case line, ok := <-m.lineCh:
				if !ok {
					m.lineCh = nil

					return m, nil
				}

				m.lines = append(m.lines, line)
				if len(m.lines) > m.cap {
					m.lines = m.lines[len(m.lines)-m.cap:]
				}
			default:
				wasAtBottom := m.viewport.AtBottom()
				m.viewport.SetContentLines(m.lines)

				if wasAtBottom {
					m.viewport.GotoBottom()
				}

				return m, nil
			}
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
