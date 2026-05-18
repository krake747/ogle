package servicelist

import (
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

//nolint:exhaustruct // satisfies bubbles list.DefaultItem
var _ list.DefaultItem = serviceItem{}

// serviceItem is a per-service list item that renders a coloured state icon,
// service name, and optional action label.
type serviceItem struct {
	def         domain.ServiceDef
	runtime     *domain.ServiceRuntimeData
	inFlight    bool
	actionLabel string
	actionError string
	th          *theme.Theme
}

func newServiceItem(def domain.ServiceDef, th *theme.Theme) serviceItem {
	return serviceItem{
		def:         def,
		th:          th,
		runtime:     nil,
		inFlight:    false,
		actionLabel: "",
		actionError: "",
	}
}

func (m serviceItem) Init() tea.Cmd {
	return nil
}

//nolint:gocognit // type switch routing to many message types is inherently complex
func (m serviceItem) Update(msg tea.Msg) (serviceItem, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.ServicesPolled:
		if msg.Err == nil && m.def.Name != "" {
			m.runtime = msg.Runtimes[m.def.Name]
		}

	case msgs.ServiceStop:
		if m.def.Name != msg.ServiceName {
			break
		}

		m.inFlight = true
		m.actionLabel = "stopping…"
		m.actionError = ""

	case msgs.ServiceStart:
		if m.def.Name != msg.ServiceName {
			break
		}

		m.inFlight = true
		m.actionLabel = "starting…"
		m.actionError = ""

	case msgs.ServiceRestart:
		if m.def.Name != msg.ServiceName {
			break
		}

		m.inFlight = true
		m.actionLabel = "restarting…"
		m.actionError = ""

	case msgs.ServiceRebuild:
		if m.def.Name != msg.ServiceName {
			break
		}

		m.inFlight = true
		m.actionLabel = "rebuilding…"
		m.actionError = ""

	case msgs.ServiceActionCompleted:
		if m.def.Name != msg.ServiceName {
			break
		}

		m.inFlight = false
		m.actionLabel = ""

		m.actionError = ""
		if msg.Err != nil {
			m.actionError = msg.Err.Error()

			break
		}

		opt := domain.ServiceStateRunning
		if msg.Action == domain.ServiceActionStop {
			opt = domain.ServiceStateExited
		}

		if m.runtime == nil {
			m.runtime = &domain.ServiceRuntimeData{
				State:       opt,
				ContainerID: "",
				Status:      "",
				CreatedAt:   time.Time{},
			}
		} else {
			rt := *m.runtime
			rt.State = opt
			m.runtime = &rt
		}
	}

	return m, nil
}

func (m serviceItem) View() tea.View {
	icon := "●"
	colour := m.th.StateMuted

	switch {
	case m.inFlight:
		icon = "◌"
		colour = m.th.StateTransient
	case m.runtime == nil:

	default:
		switch m.runtime.State {
		case domain.ServiceStateRunning:
			icon = "●"
			colour = m.th.StateRunning
		case domain.ServiceStateExited, domain.ServiceStateDead:
			icon = "●"
			colour = m.th.StateExited
		case domain.ServiceStateNotCreated:
			icon = "○"
		case domain.ServiceStatePaused:
			icon = "●"
			colour = m.th.StatePaused
		case domain.ServiceStateRestarting:
			icon = "●"
			colour = m.th.StateTransient
		case domain.ServiceStateUnknown:
			icon = "●"
		}
	}

	renderedText := lipgloss.NewStyle().Foreground(m.th.StateMuted).Render(m.def.Name)
	rendered := lipgloss.NewStyle().Foreground(colour).Render(icon, renderedText)

	if m.inFlight && m.actionLabel != "" {
		rendered += "  " + m.actionLabel
	}

	if !m.inFlight && m.actionError != "" {
		rendered += "  " + lipgloss.NewStyle().Foreground(m.th.ActionError).Render(m.actionError)
	}

	return tea.NewView(rendered)
}

func (m serviceItem) Title() string {
	return m.View().Content
}

func (m serviceItem) Description() string {
	if m.runtime == nil {
		return "—"
	}

	return string(m.runtime.State)
}

func (m serviceItem) FilterValue() string {
	return m.def.Name
}

func (m serviceItem) ServiceName() string {
	return m.def.Name
}
