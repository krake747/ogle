// Package flows defines the cross-flow contract used by every node in the
// UI flow tree (orchestrators and the root shim).
package flows

import tea "charm.land/bubbletea/v2"

// State is the cross-flow interface implemented by every node in the flow tree:
// leaf orchestrators (startup.Model, project.Model) and the root orchestrator
// (dashboard.Model). Update returns (State, tea.Cmd) rather than
// (tea.Model, tea.Cmd) so that callers within the tree never need type
// assertions — the return type is already constrained to the tree.
//
// A type cannot satisfy both flows.State and tea.Model simultaneously in Go
// (same method name, conflicting Update signatures), which is why app.go
// exists as a thin adapter shim.
type State interface {
	Init() tea.Cmd
	Update(tea.Msg) (State, tea.Cmd)
	View() string
}
