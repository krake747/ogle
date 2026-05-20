// Package card implements a single service card as a tea.Model.
package card

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// FocusMsg tells a card it is now focused.
type FocusMsg struct{}

// BlurMsg tells a card it is no longer focused.
type BlurMsg struct{}

// HoverMsg tells a card the mouse is hovering over it.
type HoverMsg struct{}

// UnhoverMsg tells a card the mouse is no longer hovering over it.
type UnhoverMsg struct{}

// ScrollTick advances the scrolling text window for a truncated service name.
type ScrollTick struct {
	gen int
}

const (
	cols               = 3
	listRatio          = 30
	listMinTermWidth   = 80
	pctDivisor         = 100
	maxCardH           = 12
	terminalCellAspect = 2
	borderW            = 2
	scrollStepInterval = 300
	scrollIdleInterval = 2500
)

// Model is a tea.Model representing a single service card.
type Model struct {
	def            domain.ServiceDef
	w, h           int
	focused        bool
	hovered        bool
	th             *theme.Theme
	scrollOffset   int
	scrollDir      int
	nextScrollTime time.Time
	focusGen       int
}

// New returns a Model for the given service definition and terminal dimensions.
func New(def domain.ServiceDef, w, h int, th *theme.Theme) Model {
	return Model{
		def:            def,
		w:              w,
		h:              h,
		focused:        false,
		hovered:        false,
		th:             th,
		scrollDir:      1,
		scrollOffset:   0,
		nextScrollTime: time.Time{},
		focusGen:       0,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height
		m.scrollOffset = 0
		m.scrollDir = 1

		if m.focused && m.needsScroll() {
			m.focusGen++
			m.nextScrollTime = time.Now().Add(scrollIdleInterval * time.Millisecond)

			return m, tickScroll(m.nextScrollTime, m.focusGen)
		}

	case FocusMsg:
		m.focused = true
		m.scrollOffset = 0
		m.scrollDir = 1

		if m.needsScroll() {
			m.focusGen++
			m.nextScrollTime = time.Now().Add(scrollIdleInterval * time.Millisecond)

			return m, tickScroll(m.nextScrollTime, m.focusGen)
		}

	case BlurMsg:
		m.focused = false
		m.scrollOffset = 0
		m.scrollDir = 1

	case HoverMsg:
		m.hovered = true

	case UnhoverMsg:
		m.hovered = false

	case ScrollTick:
		if !m.focused || !m.needsScroll() || msg.gen != m.focusGen {
			return m, nil
		}

		if time.Now().Before(m.nextScrollTime) {
			return m, tickScroll(m.nextScrollTime, m.focusGen)
		}

		maxOff := m.maxScrollOffset()

		m.scrollOffset += m.scrollDir

		switch {
		case m.scrollOffset >= maxOff:
			m.scrollOffset = maxOff
			m.scrollDir = -1
			m.nextScrollTime = time.Now().Add(scrollIdleInterval * time.Millisecond)

		case m.scrollOffset <= 0:
			m.scrollOffset = 0
			m.scrollDir = 1
			m.nextScrollTime = time.Now().Add(scrollIdleInterval * time.Millisecond)

		default:
			m.nextScrollTime = time.Now().Add(scrollStepInterval * time.Millisecond)
		}

		return m, tickScroll(m.nextScrollTime, m.focusGen)

	case msgs.ThemeChanged:
		m.th = msg.Theme
	}

	return m, nil
}

func tickScroll(t time.Time, gen int) tea.Cmd {
	return tea.Tick(max(0, time.Until(t)), func(_ time.Time) tea.Msg {
		return ScrollTick{gen: gen}
	})
}

// View satisfies tea.Model.
func (m Model) View() tea.View {
	cardW, cardH := m.cardWidth(), m.cardHeight()

	if cardW <= 0 || cardH <= 0 {
		return tea.NewView("")
	}

	innerW := cardW - borderW
	innerH := cardH - borderW
	name := m.def.Name

	var shown string

	switch {
	case len(name) <= innerW:
		shown = name

	case m.focused:
		shown = name[m.scrollOffset : m.scrollOffset+innerW]

	default:
		shown = name[:innerW-1] + "…"
	}

	content := lipgloss.NewStyle().Width(innerW).Align(lipgloss.Center).Render(shown)
	padded := lipgloss.PlaceVertical(innerH, lipgloss.Center, content)

	borderFg := m.th.CarouselBlurred
	if m.focused {
		borderFg = m.th.CarouselFocused
	} else if m.hovered {
		borderFg = m.th.CarouselHover
	}

	return tea.NewView(lipgloss.NewStyle().
		Width(cardW).
		Height(cardH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderFg).
		BorderBackground(m.th.CarouselBackground).
		Background(m.th.CarouselBackground).
		Render(padded))
}

func (m Model) cardWidth() int {
	carouselW := max(m.w, listMinTermWidth) * listRatio / pctDivisor

	return carouselW / cols
}

func (m Model) cardHeight() int {
	h := min(m.cardWidth()/terminalCellAspect, maxCardH)
	if h%2 == 0 {
		h--
	}

	return max(h, 1)
}

func (m Model) needsScroll() bool {
	return len(m.def.Name) > m.cardWidth()-borderW
}

func (m Model) maxScrollOffset() int {
	return len(m.def.Name) - (m.cardWidth() - borderW)
}
