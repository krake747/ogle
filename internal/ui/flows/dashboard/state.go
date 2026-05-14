package dashboard

import tea "charm.land/bubbletea/v2"

// State is implemented by every dashboard flow state.
type State interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (State, tea.Cmd)
	View() string
	SetSize(w, h int)
}
