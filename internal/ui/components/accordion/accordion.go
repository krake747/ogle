package accordion

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

// Model is the accordion component state.
type Model struct {
	services     []domain.ServiceDef
	runtime      *domain.ServiceRuntimeData
	selectedName string
	w, h         int
	th           *theme.Theme
}

// New returns a Model with the given project, dimensions, and theme.
func New(project *domain.Project, w, h int, th *theme.Theme) Model {
	selectedName := ""
	if len(project.Services) > 0 {
		selectedName = project.Services[0].Name
	}

	return Model{
		services:     project.Services,
		selectedName: selectedName,
		runtime:      nil,
		w:            w,
		h:            h,
		th:           th,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles selection, runtime data, resize, and theme changes.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height

	case msgs.ThemeChanged:
		m.th = msg.Theme

	case msgs.ServiceSelected:
		m.selectedName = msg.ServiceName

	case msgs.ServicesPolled:
		if msg.Err == nil && m.selectedName != "" {
			m.runtime = msg.Runtimes[m.selectedName]
		}
	}

	return m, nil
}

// View renders the inspector detail for the selected service.
func (m Model) View() tea.View {
	if m.selectedName == "" || m.w == 0 {
		return tea.NewView("")
	}

	def, ok := m.lookupDef(m.selectedName)
	if !ok {
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

	stateColour := m.th.StateMuted
	if m.runtime != nil {
		stateColour = colourForState(m.runtime.State, m.th)
	}

	bg := m.th.AccordionBackground
	lbl := lipgloss.NewStyle().Foreground(m.th.AccordionLabel).Background(bg).Render
	val := lipgloss.NewStyle().Foreground(m.th.AccordionValue).Background(bg).Render

	var stateVal string
	if stateStr == dash {
		stateVal = val(dash)
	} else {
		stateVal = lipgloss.NewStyle().Foreground(stateColour).Background(bg).Render(stateStr)
	}

	line := lipgloss.NewStyle().Width(m.w).Background(m.th.AccordionBackground).Render

	lines := []string{
		line(lbl("Image:        ") + val(def.Image)),
		line(lbl("Container ID: ") + val(containerID)),
		line(lbl("Created:      ") + val(createdAt)),
		line(lbl("State:        ") + stateVal),
		line(lbl("Ports:        ") + val(strings.Join(def.Ports, ", "))),
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m Model) lookupDef(name string) (domain.ServiceDef, bool) {
	for _, svc := range m.services {
		if svc.Name == name {
			return svc, true
		}
	}

	return domain.ServiceDef{}, false
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
