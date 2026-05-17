package logpane

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

const (
	servicePanelHeight = 5
	helpbarHeight      = 2
	defaultCap         = 1000
)

// Model stores raw log text lines backed by a viewport for windowed rendering.
type Model struct {
	lines    []string
	cap      int
	viewport viewport.Model
	lineCh   <-chan string
	h        int
}

// New returns a Model reading from the given line channel.
func New(w, h int, lineCh <-chan string) Model {
	vp := viewport.New(viewport.WithWidth(w), viewport.WithHeight(0))
	vp.MouseWheelEnabled = true
	vp.KeyMap = viewport.KeyMap{}

	return Model{
		lines:    nil,
		cap:      defaultCap,
		viewport: vp,
		lineCh:   lineCh,
		h:        h,
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
		return m.drainLines()

	case tea.WindowSizeMsg:
		wasAtBottom := m.viewport.AtBottom()
		m.h = msg.Height
		m.viewport.SetWidth(msg.Width)
		h := min(len(m.lines), max(m.h-servicePanelHeight-helpbarHeight, 0))
		m.viewport.SetHeight(h)

		if wasAtBottom || m.viewport.PastBottom() {
			m.viewport.GotoBottom()
		}
	}

	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)

	return m, cmd
}

func (m Model) drainLines() (Model, tea.Cmd) {
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
			h := min(len(m.lines), max(m.h-servicePanelHeight-helpbarHeight, 0))
			m.viewport.SetHeight(h)

			if wasAtBottom {
				m.viewport.GotoBottom()
			}

			return m, nil
		}
	}
}

// View returns the viewport-rendered window of log lines.
func (m Model) View() tea.View {
	return tea.NewView(m.viewport.View())
}
