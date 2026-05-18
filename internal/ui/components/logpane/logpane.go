package logpane

import (
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

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
	wrap     bool
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
		wrap:     false,
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

	case msgs.LogScrollH:
		m.viewport.ScrollRight(msg.Amount)

		return m, nil

	case msgs.ToggleLogWrap:
		m.wrap = !m.wrap
		realIdx := m.realLineIndex(m.viewport.YOffset())
		wasAtBottom := m.viewport.AtBottom()

		if m.wrap {
			m.viewport.SetXOffset(0)
		}

		m.viewport.SoftWrap = m.wrap
		m.viewport.SetContentLines(m.lines)

		if wasAtBottom || m.viewport.PastBottom() {
			m.viewport.GotoBottom()
		} else {
			m.viewport.SetYOffset(realIdx)
		}

		return m, nil

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

// realLineIndex returns the real-line index corresponding to the given virtual
// YOffset, accounting for soft wrapping. Mirrors the viewport's internal
// calculateLine logic for precise scroll restoration on wrap toggle.
func (m Model) realLineIndex(yOffset int) int {
	if len(m.lines) == 0 {
		return 0
	}

	if !m.wrap {
		return min(yOffset, len(m.lines)-1)
	}

	maxW := max(m.viewport.Width(), 1)

	var total int

	for i, line := range m.lines {
		vLines := max(1, (ansi.StringWidth(line)+maxW-1)/maxW)
		if yOffset < total+vLines {
			return i
		}

		total += vLines
	}

	return max(0, len(m.lines)-1)
}

// View returns the viewport-rendered window of log lines.
func (m Model) View() tea.View {
	return tea.NewView(m.viewport.View())
}
