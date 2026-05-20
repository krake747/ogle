package carousel

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
)

type card struct {
	def     domain.ServiceDef
	runtime *domain.ServiceRuntimeData
}

func newCard(def domain.ServiceDef) card {
	return card{def: def, runtime: nil}
}

func (c card) Init() tea.Cmd { return nil }

func (c card) Update(_ tea.Msg) (card, tea.Cmd) { return c, nil }

func (c card) View(cardW, cardH int, focused bool) tea.View {
	focusedFg := lipgloss.Color("#ffffff")
	unfocusedFg := lipgloss.Color("#444444")

	borderColour := unfocusedFg
	if focused {
		borderColour = focusedFg
	}

	return tea.NewView(lipgloss.NewStyle().
		Width(cardW).
		Height(cardH).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColour).
		Render(""))
}
