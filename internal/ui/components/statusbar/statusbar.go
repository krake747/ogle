// Package statusbar provides a transient one-line status message bar.
// It renders an info or error message for a fixed duration, then clears itself.
package statusbar

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const msgDuration = 3 * time.Second

// Model holds status bar state.
type Model struct {
	msg     string
	msgEnd  time.Time
	isError bool
	th      *theme.Theme
	width   int
}

// New returns a Model with no active message.
func New(th *theme.Theme) Model {
	return Model{
		msg:     "",
		msgEnd:  time.Time{},
		isError: false,
		th:      th,
		width:   0,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles status message lifecycle and theme/size changes.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case msgs.ThemeChanged:
		m.th = msg.Theme

	case msgs.DisplayError:
		m.msg = msg.Err
		m.msgEnd = time.Now().Add(msgDuration)
		m.isError = true

		return m, tea.Tick(msgDuration, func(time.Time) tea.Msg {
			return msgs.ClearStatusMsg{}
		})

	case msgs.DisplayStatus:
		m.msg = msg.Msg
		m.msgEnd = time.Now().Add(msgDuration)
		m.isError = false

		return m, tea.Tick(msgDuration, func(time.Time) tea.Msg {
			return msgs.ClearStatusMsg{}
		})

	case msgs.ClearStatusMsg:
		if time.Now().After(m.msgEnd) {
			m.msg = ""
		}
	}

	return m, nil
}

// View renders the status bar. Returns an empty view when no message is active.
func (m Model) View() tea.View {
	if m.msg == "" {
		return tea.NewView("")
	}

	fg := m.th.StatusInfo
	if m.isError {
		fg = m.th.ActionError
	}

	style := lipgloss.NewStyle().
		Foreground(fg).
		Background(m.th.StatusBarBackground).
		Width(m.width)

	return tea.NewView(style.Render(m.msg))
}
