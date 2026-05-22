package accordion

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is the accordion component state.
type Model struct {
	w, h int
	th   *theme.Theme
}

// New returns a Model with the given dimensions and theme.
func New(w, h int, th *theme.Theme) Model {
	return Model{w: w, h: h, th: th}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles resize and theme changes.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
	case msgs.ThemeChanged:
		m.th = msg.Theme
	}

	return m, nil
}

// View renders the accordion placeholder.
func (m Model) View() tea.View {
	return tea.NewView(
		lipgloss.Place(m.w, m.h,
			lipgloss.Left, lipgloss.Center,
			"accordion",
			lipgloss.WithWhitespaceStyle(lipgloss.NewStyle().Background(m.th.CarouselBackground)),
		),
	)
}
