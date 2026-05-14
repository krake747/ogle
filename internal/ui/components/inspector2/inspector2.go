package inspector2

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	dash       = "—"
	shortIDLen = 12
)

// Model renders the detail header for the selected service. It is a value type;
// all mutating methods return a new Model.
type Model struct {
	selected string
	runtime  *domain.ServiceRuntimeData
	theme    *theme.Theme
	w, h     int
}

// New returns a Model with no selected service.
func New(th *theme.Theme) Model {
	return Model{
		selected: "",
		runtime:  nil,
		theme:    th,
		w:        0,
		h:        0,
	}
}

// SetBounds returns a copy with new dimensions.
func (m Model) SetBounds(w, h int) Model {
	m.w = w
	m.h = h

	return m
}

// Selected returns the currently selected service name.
func (m Model) Selected() string { return m.selected }

// View renders the detail header for the selected service.
func (m Model) View() string {
	if m.selected == "" || m.w == 0 {
		return ""
	}

	stateStr := dash
	healthStr := dash
	containerID := dash

	if m.runtime != nil {
		stateStr = string(m.runtime.State)
		if m.runtime.ContainerID != "" {
			containerID = m.runtime.ContainerID
			if len(containerID) > shortIDLen {
				containerID = containerID[:shortIDLen]
			}
		}

		healthStr = string(m.runtime.Health)
	}

	title := m.theme.ServiceListTitle.Render(m.selected)

	stateColour := m.theme.StateRunning
	if m.runtime != nil {
		stateColour = colourForState(m.runtime.State, m.theme)
	}

	stateLabel := lipgloss.NewStyle().Foreground(stateColour).Render(stateStr)

	lines := []string{
		title,
		"",
		"Container ID:  " + containerID,
		"State:         " + stateLabel,
		"Health:        " + healthStr,
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func colourForState(s domain.ServiceState, th *theme.Theme) color.Color {
	switch s {
	case domain.ServiceStateRunning:
		return th.StateRunning
	case domain.ServiceStateExited, domain.ServiceStateDead:
		return th.StateExited
	case domain.ServiceStatePaused:
		return th.StatePaused
	case domain.ServiceStateRestarting:
		return th.StateTransient
	case domain.ServiceStateNotCreated, domain.ServiceStateUnknown:
		return th.StateMuted
	default:
		return th.StateMuted
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles domain messages that affect inspector state.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.ServiceSelected:
		m.selected = msg.Service.Name
		m.runtime = nil
	case msgs.ServicesPolled:
		if msg.Err == nil && m.selected != "" {
			m.runtime = msg.Runtimes[m.selected]
		}
	}

	return m, nil
}
