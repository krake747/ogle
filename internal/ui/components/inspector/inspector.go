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
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	dash          = "—"
	shortIDLen    = 12
	secsPerMinute = 60
	secsPerHour   = 3600
)

// Model renders the detail header for a single service.
type Model struct {
	def     domain.ServiceDef
	runtime *domain.ServiceRuntimeData
	theme   *theme.Theme
	w       int
}

// New returns a Model for the given service.
func New(th *theme.Theme, def domain.ServiceDef, w, _ int) Model {
	return Model{
		def:     def,
		runtime: nil,
		theme:   th,
		w:       w,
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
		m.w = msg.Width

	case msgs.ServicesPolled:
		if msg.Err == nil && m.def.Name != "" {
			m.runtime = msg.Runtimes[m.def.Name]
		}
	}

	return m, nil
}

// View renders the detail header for this service.
func (m Model) View() tea.View {
	if m.def.Name == "" || m.w == 0 {
		return tea.NewView("")
	}

	stateStr := dash
	containerID := dash
	createdAt := dash

	if m.runtime != nil {
		if m.runtime.Status != "" {
			stateStr = m.runtime.Status
		}

		if m.runtime.ContainerID != "" {
			containerID = m.runtime.ContainerID
		}

		if len(containerID) > shortIDLen {
			containerID = containerID[:shortIDLen]
		}

		if !m.runtime.CreatedAt.IsZero() {
			createdAt = formatAge(time.Since(m.runtime.CreatedAt))
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
		"Created:       " + createdAt,
		"State:         " + stateLabel,
		"Ports:         " + strings.Join(m.def.Ports, ", "),
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func formatAge(d time.Duration) string {
	secs := max(int(d.Seconds()), 0)

	switch {
	case secs < secsPerMinute:
		return fmt.Sprintf("%ds ago", secs)
	case secs < secsPerHour:
		return fmt.Sprintf("%dm ago", secs/secsPerMinute)
	default:
		return fmt.Sprintf("%dh ago", secs/secsPerHour)
	}
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
