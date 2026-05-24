package value

import (
	"image/color"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// StartMsg triggers the value's horizontal scroll animation.
type StartMsg struct {
	Gen int
}

// Model renders a single accordion value with horizontal scroll when content
// exceeds width.
type Model struct {
	rawValue string
	colour   color.Color
	bg       color.Color
	width    int
	scroll   ScrollState
}

// New returns a value model for the given content and styling.
func New(raw string, colour, bg color.Color, width int) Model {
	return Model{
		rawValue: raw,
		colour:   colour,
		bg:       bg,
		width:    width,
		scroll: ScrollState{
			Offset:     0,
			Dir:        1,
			Gen:        0,
			InstanceID: time.Now().UnixNano(),
		},
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles scroll start requests and animation ticks.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StartMsg:
		m.scroll.Gen = msg.Gen

		if !m.needsScroll() {
			return m, nil
		}

		return m, m.tick(scrollIdleInterval * time.Millisecond)

	case ScrollState:
		return m.handleScrollTick(msg)
	}

	return m, nil
}

// View renders the value with optional scroll windowing.
func (m Model) View() tea.View {
	if m.width <= 0 {
		return tea.NewView("")
	}

	shown := m.rawValue
	if m.needsScroll() {
		shown = ansi.Cut(m.rawValue, m.scroll.Offset, m.scroll.Offset+m.width)
	}

	return tea.NewView(lipgloss.NewStyle().
		Width(m.width).
		Foreground(m.colour).
		Background(m.bg).
		Render(shown))
}

func (m Model) handleScrollTick(msg ScrollState) (Model, tea.Cmd) {
	s, interval, ok := m.scroll.HandleTick(msg, m.rawValue, m.width)
	m.scroll = s

	if !ok {
		return m, nil
	}

	return m, m.tick(interval)
}

func (m Model) needsScroll() bool {
	return ansi.StringWidth(m.rawValue) > m.width
}

func (m Model) tick(d time.Duration) tea.Cmd {
	gen := m.scroll.Gen
	id := m.scroll.InstanceID

	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return ScrollState{Offset: 0, Dir: 0, Gen: gen, InstanceID: id}
	})
}
