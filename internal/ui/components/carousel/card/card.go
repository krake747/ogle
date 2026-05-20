// Package card implements a single service card as a tea.Model.
package card

import (
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

const (
	cols               = 2
	listRatio          = 30
	listMinTermWidth   = 80
	pctDivisor         = 100
	maxCardH           = 8
	terminalCellAspect = 2
	borderW            = 2
)

// Model is a tea.Model representing a single service card.
type Model struct {
	def     domain.ServiceDef
	w, h    int
	focused bool
	th      *theme.Theme
}

// New returns a Model for the given service definition and terminal dimensions.
func New(def domain.ServiceDef, w, h int, th *theme.Theme) Model {
	return Model{
		def:     def,
		w:       w,
		h:       h,
		focused: false,
		th:      th,
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

	case FocusMsg:
		m.focused = true

	case BlurMsg:
		m.focused = false

	case msgs.ThemeChanged:
		m.th = msg.Theme
	}

	return m, nil
}

// View satisfies tea.Model.
func (m Model) View() tea.View {
	cardW, cardH := m.cardWidth(), m.cardHeight()

	if cardW <= 0 || cardH <= 0 {
		return tea.NewView("")
	}

	innerW := cardW - borderW
	name := m.def.Name

	var shown string

	if len(name) <= innerW {
		shown = name
	} else {
		shown = name[:innerW-1] + "…"
	}

	content := lipgloss.NewStyle().Width(innerW).Render(shown)
	padded := lipgloss.PlaceVertical(cardH, lipgloss.Top, content)

	borderFg := m.th.CarouselBlurred
	if m.focused {
		borderFg = m.th.CarouselFocused
	}

	return tea.NewView(lipgloss.NewStyle().
		Width(cardW).
		Height(cardH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderFg).
		Render(padded))
}

func (m Model) cardWidth() int {
	carouselW := max(m.w, listMinTermWidth) * listRatio / pctDivisor

	return carouselW / cols
}

func (m Model) cardHeight() int {
	return min(m.cardWidth()/terminalCellAspect, maxCardH)
}
