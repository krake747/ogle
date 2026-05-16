package inspector

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	dash          = "—"
	shortIDLen    = 12
	secsPerMinute = 60
	secsPerHour   = 3600
)

// Model renders the detail header for a single service. It is a value type;
// all mutating methods return a new Model.
type Model struct {
	def     domain.ServiceDef
	runtime *domain.ServiceRuntimeData
	theme   *theme.Theme
	w, h    int
}

// New returns a Model for the given service.
func New(th *theme.Theme, def domain.ServiceDef, w, h int) Model {
	return Model{
		def:     def,
		runtime: nil,
		theme:   th,
		w:       w,
		h:       h,
	}
}

// ServiceName returns the service this model represents.
// func (m Model) ServiceName() string { return m.def.Name }

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles dimension changes and runtime data.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width - servicelist.ListWidth(msg.Width)
		m.h = msg.Height

	case msgs.ServicesPolled:
		if msg.Err == nil && m.def.Name != "" {
			m.runtime = msg.Runtimes[m.def.Name]
		}
	}

	return m, nil
}

// View renders the detail header for this service.
func (m Model) View() string {
	if m.def.Name == "" || m.w == 0 {
		return ""
	}

	stateStr := dash
	healthStr := dash
	containerID := dash

	if m.runtime != nil {
		stateStr = string(m.runtime.State)

		ageStr := formatAge(m.runtime.StateAge)
		stateStr += " (" + ageStr + " ago)"

		if m.runtime.ContainerID != "" {
			containerID = m.runtime.ContainerID
			if len(containerID) > shortIDLen {
				containerID = containerID[:shortIDLen]
			}
		}

		healthStr = string(m.runtime.Health)
		if m.runtime.Health != "" {
			healthColour := colourForHealth(m.runtime.Health, m.theme)
			healthStr = lipgloss.NewStyle().Foreground(healthColour).Render(healthStr)
		}
	}

	stateColour := m.theme.StateRunning
	if m.runtime != nil {
		stateColour = colourForState(m.runtime.State, m.theme)
	}

	stateLabel := lipgloss.NewStyle().Foreground(stateColour).Render(stateStr)

	lines := []string{
		"Image:         " + m.def.Image,
		"Container ID:  " + containerID,
		"State:         " + stateLabel,
		"Health:        " + healthStr,
		"Ports:         " + strings.Join(m.def.Ports, ", "),
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func formatAge(d time.Duration) string {
	secs := max(int(d.Seconds()), 0)

	switch {
	case secs < secsPerMinute:
		return fmt.Sprintf("%ds", secs)
	case secs < secsPerHour:
		return fmt.Sprintf("%dm", secs/secsPerMinute)
	default:
		return fmt.Sprintf("%dh", secs/secsPerHour)
	}
}

func colourForHealth(h domain.ServiceHealth, th *theme.Theme) color.Color {
	switch h {
	case domain.ServiceHealthHealthy:
		return th.StateRunning
	case domain.ServiceHealthUnhealthy:
		return th.StateExited
	case domain.ServiceHealthStarting:
		return th.StateTransient
	case domain.ServiceHealthNoHealthcheck, domain.ServiceHealthUnknown:
		return th.StateMuted
	}

	return th.StateMuted
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
