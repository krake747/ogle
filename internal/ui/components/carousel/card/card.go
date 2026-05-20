// Package card implements a single service card as a tea.Model.
package card

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
)

const (
	cols               = 2
	chevronW           = 2
	chevronCount       = 2
	listRatio          = 30
	listMinTermWidth   = 80
	pctDivisor         = 100
	maxCardH           = 8
	terminalCellAspect = 2
	borderW            = 2
)

// Model is a tea.Model representing a single service card.
type Model struct {
	def  domain.ServiceDef
	w, h int
}

// New returns a Model for the given service definition and terminal dimensions.
func New(def domain.ServiceDef, w, h int) Model {
	return Model{
		def: def,
		w:   w,
		h:   h,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.w = msg.Width
		m.h = msg.Height
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

	return tea.NewView(lipgloss.NewStyle().
		Width(cardW).
		Height(cardH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Render(padded))
}

func (m Model) cardWidth() int {
	carouselW := max(m.w, listMinTermWidth) * listRatio / pctDivisor

	return (carouselW - chevronW*chevronCount) / cols
}

func (m Model) cardHeight() int {
	return min(m.cardWidth()/terminalCellAspect, maxCardH)
}
