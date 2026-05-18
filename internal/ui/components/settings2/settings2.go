package settings2

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	boxWidth  = 30
	boxHeight = 5
)

// Model is a stub overlay for the settings page.
type Model struct {
	th   *theme.Theme
	w, h int
}

// New returns a Model.
func New(th *theme.Theme, w, h int) Model {
	return Model{th: th, w: w, h: h}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "esc" {
			return m, func() tea.Msg {
				return msgs.SettingsVisibilityChanged{Visible: false}
			}
		}
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height
	}

	return m, nil
}

// View renders the overlay box.
func (m Model) View() tea.View {
	return tea.NewView(
		lipgloss.NewStyle().
			Width(boxWidth).
			Height(boxHeight).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.th.StateMuted).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center).
			Render("Settings2\n\nesc to close"),
	)
}
