// Package servicetitle provides a per-service title component that renders the
// coloured icon, name, and action label for a service list entry.
package servicetitle

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is a value-type component that tracks service runtime state and renders
// a single service title line. It implements list.Item directly.
type Model struct {
	def         domain.ServiceDef
	runtime     *domain.ServiceRuntimeData
	inFlight    bool
	actionLabel string
	actionError string
	theme       *theme.Theme
}

// New returns a Model with the given service definition and theme.
func New(def domain.ServiceDef, th *theme.Theme) Model {
	//nolint:exhaustruct // zero-value fields are intentional defaults
	return Model{def: def, theme: th}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update satisfies tea.Model. No per-item messages are handled yet.
func (m Model) Update(_ tea.Msg) (Model, tea.Cmd) { return m, nil }

// View renders the service title: a coloured state icon, the service name, and
// an optional action suffix.
func (m Model) View() string {
	icon := "●"
	colour := m.theme.StateMuted

	switch {
	case m.inFlight:
		icon = "◌"
		colour = m.theme.StateTransient
	case m.runtime == nil:

	default:
		switch m.runtime.State {
		case domain.ServiceStateRunning:
			icon = "●"
			colour = m.theme.StateRunning
		case domain.ServiceStateExited, domain.ServiceStateDead:
			icon = "●"
			colour = m.theme.StateExited
		case domain.ServiceStateNotCreated:
			icon = "○"
		case domain.ServiceStatePaused:
			icon = "●"
			colour = m.theme.StatePaused
		case domain.ServiceStateRestarting:
			icon = "●"
			colour = m.theme.StateTransient
		case domain.ServiceStateUnknown:
			icon = "●"
		}
	}

	rendered := lipgloss.NewStyle().Foreground(colour).Render(icon) + " " + m.def.Name

	if m.inFlight && m.actionLabel != "" {
		rendered += "  " + m.actionLabel
	}

	if !m.inFlight && m.actionError != "" {
		rendered += "  " + lipgloss.NewStyle().Foreground(m.theme.ActionError).Render(m.actionError)
	}

	return rendered
}

// Title returns the rendered title string. Implements list.Item.
func (m Model) Title() string { return m.View() }

// Description returns the runtime state or "—" when unknown. Implements list.Item.
func (m Model) Description() string {
	if m.runtime == nil {
		return "—"
	}

	return string(m.runtime.State)
}

// FilterValue returns the service name. Implements list.Item.
func (m Model) FilterValue() string { return m.def.Name }

// ServiceDef returns the service definition.
func (m Model) ServiceDef() domain.ServiceDef { return m.def }

// State returns the effective service state. hasState is false when runtime is
// nil and no optimistic state has been applied. inFlight is true when an action
// is in progress.
func (m Model) State() (domain.ServiceState, bool, bool) {
	if m.inFlight {
		return "", false, true
	}

	if m.runtime == nil {
		return "", false, false
	}

	return m.runtime.State, true, false
}

// SetRuntime returns a copy of the model with the runtime data replaced.
func (m Model) SetRuntime(rt *domain.ServiceRuntimeData) Model {
	m.runtime = rt

	return m
}

// SetActionInFlight returns a copy of the model with in-flight state set.
func (m Model) SetActionInFlight(label string) Model {
	m.inFlight = true
	m.actionLabel = label
	m.actionError = ""

	return m
}

// SetActionSuccess returns a copy with in-flight cleared and an optimistic
// ServiceState applied.
func (m Model) SetActionSuccess(optimisticState domain.ServiceState) Model {
	m.inFlight = false
	m.actionLabel = ""
	m.actionError = ""

	if m.runtime == nil {
		m.runtime = &domain.ServiceRuntimeData{
			ContainerID: "",
			State:       optimisticState,
			Health:      "",
			StateAge:    0,
		}
	} else {
		rt := *m.runtime
		rt.State = optimisticState
		m.runtime = &rt
	}

	return m
}

// SetActionError returns a copy with in-flight cleared and the error message set.
func (m Model) SetActionError(err string) Model {
	m.inFlight = false
	m.actionLabel = ""
	m.actionError = err

	return m
}
