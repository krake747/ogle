package servicelist2

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// serviceItem is a per-service list item that renders a coloured state icon,
// service name, and optional action label.
type serviceItem struct {
	def         domain.ServiceDef
	runtime     *domain.ServiceRuntimeData
	inFlight    bool
	actionLabel string
	actionError string
	theme       *theme.Theme
}

func newServiceItem(def domain.ServiceDef, th *theme.Theme) serviceItem {
	return serviceItem{
		def:         def,
		theme:       th,
		runtime:     nil,
		inFlight:    false,
		actionLabel: "",
		actionError: "",
	}
}

func (m serviceItem) Init() tea.Cmd { return nil }

func (m serviceItem) Update(_ tea.Msg) (serviceItem, tea.Cmd) { return m, nil }

func (m serviceItem) View() string {
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

func (m serviceItem) Title() string { return m.View() }

func (m serviceItem) Description() string {
	if m.runtime == nil {
		return "—"
	}

	return string(m.runtime.State)
}

func (m serviceItem) FilterValue() string { return m.def.Name }

func (m serviceItem) ServiceDef() domain.ServiceDef { return m.def }

func (m serviceItem) State() (domain.ServiceState, bool, bool) {
	if m.inFlight {
		return "", false, true
	}

	if m.runtime == nil {
		return "", false, false
	}

	return m.runtime.State, true, false
}

func (m serviceItem) SetRuntime(rt *domain.ServiceRuntimeData) serviceItem {
	m.runtime = rt

	return m
}

func (m serviceItem) SetActionInFlight(label string) serviceItem {
	m.inFlight = true
	m.actionLabel = label
	m.actionError = ""

	return m
}

func (m serviceItem) SetActionSuccess(optimisticState domain.ServiceState) serviceItem {
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

func (m serviceItem) SetActionError(err string) serviceItem {
	m.inFlight = false
	m.actionLabel = ""
	m.actionError = err

	return m
}
