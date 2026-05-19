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

// serviceItem is a per-service list item that renders a coloured state icon
// and service name.
type serviceItem struct {
	def      domain.ServiceDef
	runtime  *domain.ServiceRuntimeData
	inFlight bool
	th       *theme.Theme
}

func newServiceItem(def domain.ServiceDef, th *theme.Theme) serviceItem {
	return serviceItem{
		def:      def,
		th:       th,
		runtime:  nil,
		inFlight: false,
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

		return m, func() tea.Msg { return msgs.DisplayStatus{Msg: m.def.Name + ": stopping…"} }

	case msgs.ServiceStart:
		if m.def.Name != msg.ServiceName {
			break
		}

		m.inFlight = true

		return m, func() tea.Msg { return msgs.DisplayStatus{Msg: m.def.Name + ": starting…"} }

	case msgs.ServiceRestart:
		if m.def.Name != msg.ServiceName {
			break
		}

		m.inFlight = true

		return m, func() tea.Msg { return msgs.DisplayStatus{Msg: m.def.Name + ": restarting…"} }

	case msgs.ServiceRebuild:
		if m.def.Name != msg.ServiceName {
			break
		}

		m.inFlight = true

		return m, func() tea.Msg { return msgs.DisplayStatus{Msg: m.def.Name + ": rebuilding…"} }

	case msgs.ServiceActionCompleted:
		if m.def.Name != msg.ServiceName {
			break
		}

		m.inFlight = false

		if msg.Err != nil {
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
	textColour := m.th.StateMuted

	switch {
	case m.inFlight:
		icon = "◌"
		colour = m.th.StateTransient
		textColour = m.th.Text
	case m.runtime == nil:

	default:
		switch m.runtime.State {
		case domain.ServiceStateRunning:
			icon = "●"
			colour = m.th.StateRunning
			textColour = m.th.Text
		case domain.ServiceStateExited, domain.ServiceStateDead:
			icon = "●"
			colour = m.th.StateExited
			textColour = m.th.Text
		case domain.ServiceStateNotCreated:
			icon = "○"
		case domain.ServiceStatePaused:
			icon = "●"
			colour = m.th.StatePaused
			textColour = m.th.Text
		case domain.ServiceStateRestarting:
			icon = "●"
			colour = m.th.StateTransient
			textColour = m.th.Text
		case domain.ServiceStateUnknown:
			icon = "●"
		}
	}

	renderedText := lipgloss.NewStyle().Foreground(textColour).Render(m.def.Name)
	rendered := lipgloss.NewStyle().Foreground(colour).Render(icon, renderedText)

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
