// Package states implements the State pattern for the startup flow.
// Each concrete type represents one startup state and owns its own transitions.
package states

import tea "charm.land/bubbletea/v2"

// State is implemented by every startup flow state.
type State interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (State, tea.Cmd)
	View() string
}
