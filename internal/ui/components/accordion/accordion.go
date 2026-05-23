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
	"github.com/ma-tf/ogle/internal/ui/components/accordion/value"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	dash             = "—"
	shortIDLen       = 12
	labelWidth       = 14
	secsPerMinute    = 60
	secsPerHour      = 3600
	listMinTermWidth = 80
	listRatio        = 30
	pctDivisor       = 100
	numFields        = 5
)

// Model is the accordion component state.
type Model struct {
	services     []domain.ServiceDef
	runtime      *domain.ServiceRuntimeData
	selectedName string
	w, h         int
	th           *theme.Theme
	values       [numFields]value.Model
	scrollGen    int
}

// New returns a Model with the given project, dimensions, and theme.
func New(project *domain.Project, w, h int, th *theme.Theme) Model {
	selectedName := ""
	if len(project.Services) > 0 {
		selectedName = project.Services[0].Name
	}

	m := Model{
		services:     project.Services,
		selectedName: selectedName,
		runtime:      nil,
		w:            w,
		h:            h,
		th:           th,
		values:       [numFields]value.Model{},
		scrollGen:    0,
	}
	for i := range m.values {
		m.values[i] = value.New("", th.AccordionValue, th.AccordionBackground, m.valueWidth())
	}

	return m
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles selection, runtime data, resize, and theme changes.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height

		return m.syncValues()

	case msgs.ThemeChanged:
		m.th = msg.Theme

		return m.syncValues()

	case msgs.ServiceSelected:
		m.selectedName = msg.ServiceName

		return m.syncValues()

	case msgs.ServicesPolled:
		if msg.Err == nil && m.selectedName != "" {
			m.runtime = msg.Runtimes[m.selectedName]
		}

		return m.syncValues()
	}

	var cmds []tea.Cmd

	for i := range m.values {
		var cmd tea.Cmd

		m.values[i], cmd = m.values[i].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the inspector detail for the selected service.
func (m Model) View() tea.View {
	if m.selectedName == "" || m.w == 0 {
		return tea.NewView("")
	}

	vw := m.valueWidth()
	bg := m.th.AccordionBackground

	labels := []string{
		"Image:        ",
		"Container ID: ",
		"Created:      ",
		"State:        ",
		"Ports:        ",
	}
	labelBlock := lipgloss.JoinVertical(lipgloss.Left, labels...)
	labelCol := lipgloss.NewStyle().
		Width(labelWidth).
		Foreground(m.th.AccordionLabel).
		Background(bg).
		Render(labelBlock)

	valStrs := make([]string, numFields)
	for i := range numFields {
		valStrs[i] = m.values[i].View().Content
	}

	valBlock := lipgloss.JoinVertical(lipgloss.Left, valStrs...)
	valCol := lipgloss.NewStyle().
		Width(vw).
		Foreground(m.th.AccordionValue).
		Background(bg).
		Render(valBlock)

	return tea.NewView(lipgloss.JoinHorizontal(lipgloss.Top,
		labelCol,
		valCol,
	))
}

func (m Model) syncValues() (Model, tea.Cmd) {
	vw := m.valueWidth()
	if m.selectedName == "" || vw <= 0 {
		for i := range m.values {
			m.values[i] = value.New("", m.th.AccordionValue, m.th.AccordionBackground, 0)
		}

		return m, nil
	}

	bg := m.th.AccordionBackground
	raws, colours := m.computeFieldContent()

	var cmds []tea.Cmd

	for i := range m.values {
		m.scrollGen++
		m.values[i] = value.New(raws[i], colours[i], bg, vw)

		var cmd tea.Cmd

		m.values[i], cmd = m.values[i].Start(m.scrollGen)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) computeFieldContent() ([numFields]string, [numFields]color.Color) {
	def, ok := m.lookupDef(m.selectedName)

	var (
		raws    [numFields]string
		colours [numFields]color.Color
	)

	if !ok {
		for i := range raws {
			raws[i] = ""
			colours[i] = m.th.AccordionValue
		}

		return raws, colours
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

	portsStr := strings.Join(def.Ports, ", ")
	if portsStr == "" {
		portsStr = dash
	}

	raws = [numFields]string{def.Image, containerID, createdAt, stateStr, portsStr}
	colours = [numFields]color.Color{
		m.th.AccordionValue,
		m.th.AccordionValue,
		m.th.AccordionValue,
		stateColour,
		m.th.AccordionValue,
	}

	return raws, colours
}

func (m Model) valueWidth() int {
	carouselW := max(m.w, listMinTermWidth) * listRatio / pctDivisor

	return max(0, carouselW-labelWidth)
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
