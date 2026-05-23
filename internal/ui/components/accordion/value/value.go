package value

import (
	"image/color"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	scrollStepInterval = 300
	scrollIdleInterval = 2500
)

type scrollTick struct {
	gen int
}

type scrollState struct {
	offset int
	dir    int
	gen    int
}

// Model renders a single accordion value with horizontal scroll when content
// exceeds width.
type Model struct {
	rawValue string
	colour   color.Color
	bg       color.Color
	width    int
	scroll   scrollState
}

// New returns a value model for the given content and styling.
func New(raw string, colour, bg color.Color, width int) Model {
	return Model{
		rawValue: raw,
		colour:   colour,
		bg:       bg,
		width:    width,
		scroll: scrollState{
			offset: 0,
			dir:    1,
			gen:    0,
		},
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Start schedules the first scroll tick if the content exceeds the available
// width.
func (m Model) Start(gen int) (Model, tea.Cmd) {
	m.scroll.gen = gen
	if !m.needsScroll() {
		return m, nil
	}

	return m, m.tick(scrollIdleInterval * time.Millisecond)
}

// Update handles scroll animation ticks.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if tick, ok := msg.(scrollTick); ok {
		return m.handleScrollTick(tick)
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
		shown = ansi.Cut(m.rawValue, m.scroll.offset, m.scroll.offset+m.width)
	}

	return tea.NewView(lipgloss.NewStyle().
		Width(m.width).
		Foreground(m.colour).
		Background(m.bg).
		Render(shown))
}

func (m Model) handleScrollTick(msg scrollTick) (Model, tea.Cmd) {
	if msg.gen != m.scroll.gen || m.width <= 0 {
		return m, nil
	}

	maxOff := ansi.StringWidth(m.rawValue) - m.width
	if maxOff <= 0 {
		return m, nil
	}

	m.scroll.offset += m.scroll.dir

	switch {
	case m.scroll.offset >= maxOff:
		m.scroll.offset = maxOff
		m.scroll.dir = -1

		return m, m.tick(scrollIdleInterval * time.Millisecond)
	case m.scroll.offset <= 0:
		m.scroll.offset = 0
		m.scroll.dir = 1

		return m, m.tick(scrollIdleInterval * time.Millisecond)
	default:
		return m, m.tick(scrollStepInterval * time.Millisecond)
	}
}

func (m Model) needsScroll() bool {
	return ansi.StringWidth(m.rawValue) > m.width
}

func (m Model) tick(d time.Duration) tea.Cmd {
	gen := m.scroll.gen

	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return scrollTick{gen: gen}
	})
}
