// Package inspector implements the Service Inspector component — the right pane
// of the Dashboard. It renders a stacked layout: a compact detail header above
// the Log Stream area.
package inspector

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// ConnectState represents the Docker daemon connectivity state as seen by the
// Service Inspector.
type ConnectState int

const (
	// ConnectStateConnecting is the initial state: the ping is in-flight or
	// the grace period has not yet expired.
	ConnectStateConnecting ConnectState = iota

	// ConnectStateConnected means the daemon ping succeeded.
	ConnectStateConnected

	// ConnectStateUnavailable means the daemon cannot be reached. The Log Stream
	// area shows a retry countdown.
	ConnectStateUnavailable
)

// UnavailableState carries the live countdown until the next retry attempt.
type UnavailableState struct {
	SecondsUntilRetry int
}

// LogAreaState describes what the log area should render.
type LogAreaState int

const (
	// LogAreaConnecting is shown while the daemon ping is in-flight or the
	// grace period is active.
	LogAreaConnecting LogAreaState = iota

	// LogAreaStreaming means a stream is attached and lines are available.
	LogAreaStreaming

	// LogAreaUnavailable is shown when the daemon is unreachable; the buffer
	// is frozen.
	LogAreaUnavailable

	// LogAreaNotFound is shown when the container does not exist yet.
	LogAreaNotFound
)

// Model is the Service Inspector component. It is a value type; all mutating
// methods return a new Model.
type Model struct {
	service      domain.ServiceDef
	runtime      *domain.ServiceRuntimeData
	connectState ConnectState
	unavailable  UnavailableState
	theme        *theme.Theme
	showLabels   bool
	labels       labelsModel
	logLines     []string
	logPaused    bool
	logState     LogAreaState
	width        int
	height       int
	y            int
}

// New returns a Model for the given service with ConnectStateConnecting.
func New(service domain.ServiceDef, th *theme.Theme) Model {
	return Model{
		service:      service,
		runtime:      nil,
		connectState: ConnectStateConnecting,
		labels:       newLabelsModel(service),
		theme:        th,
		unavailable:  UnavailableState{SecondsUntilRetry: 0},
		showLabels:   false,
		logLines:     nil,
		logPaused:    false,
		logState:     LogAreaConnecting,
		width:        0,
		height:       0,
		y:            0,
	}
}

// SetService replaces the currently displayed service. Called on ServiceSelected
// and after Live Reload resets the selected service.
func (m Model) SetService(def domain.ServiceDef) Model {
	m.service = def
	m.runtime = nil
	m.labels = newLabelsModel(def)

	return m
}

// SetRuntime updates the runtime data for the current service.
func (m Model) SetRuntime(rd *domain.ServiceRuntimeData) Model {
	m.runtime = rd

	return m
}

// SetConnectState updates the connectivity state.
func (m Model) SetConnectState(cs ConnectState) Model {
	m.connectState = cs

	return m
}

// SetUnavailable transitions to ConnectStateUnavailable with the given countdown.
func (m Model) SetUnavailable(u UnavailableState) Model {
	m.connectState = ConnectStateUnavailable
	m.unavailable = u

	return m
}

// SetShowLabels controls label section visibility.
func (m Model) SetShowLabels(show bool) Model {
	m.showLabels = show

	return m
}

// SetBounds propagates terminal dimensions and vertical origin to the inspector.
func (m Model) SetBounds(w, h, y int) Model {
	m.width = w
	m.height = h
	m.y = y

	return m
}

// SetLogView updates the log area content. lines contains the pre-computed,
// pre-styled display rows. paused indicates whether auto-tail is suspended.
// state controls which placeholder (if any) is shown instead of lines.
func (m Model) SetLogView(lines []string, paused bool, state LogAreaState) Model {
	m.logLines = lines
	m.logPaused = paused
	m.logState = state

	return m
}

// Init satisfies tea.Model (not used directly; the Dashboard owns Init).
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages directed at the Service Inspector.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.showLabels {
		var cmd tea.Cmd

		m.labels, cmd = m.labels.update(m.adjustMouseY(msg))

		return m, cmd
	}

	return m, nil
}

// adjustMouseY rewrites the Y field of mouse messages to be relative to the
// label section top (inspector origin + header rows), so that mouseRow receives
// a section-local coordinate.
func (m Model) adjustMouseY(msg tea.Msg) tea.Msg {
	offset := m.y + HeaderLines

	switch ev := msg.(type) {
	case tea.MouseMotionMsg:
		ev.Y -= offset

		return ev
	case tea.MouseClickMsg:
		ev.Y -= offset

		return ev
	case tea.MouseReleaseMsg:
		ev.Y -= offset

		return ev
	}

	return msg
}

// View renders the Service Inspector: detail header above the Log Stream area.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	header := renderHeader(m.service, m.runtime, m.width)

	var body string
	if m.showLabels {
		body = m.labels.view(m.width, m.height-HeaderLines, m.theme)
	} else {
		body = m.renderLogArea()
	}

	return header + "\n" + body
}

// renderLogArea returns the Log Stream content appropriate for the current state.
func (m Model) renderLogArea() string {
	switch m.logState {
	case LogAreaConnecting:
		return "Connecting to Docker…"
	case LogAreaUnavailable:
		return fmt.Sprintf("Docker unavailable — retrying in %ds…", m.unavailable.SecondsUntilRetry)
	case LogAreaNotFound:
		return "No container — service not started"
	case LogAreaStreaming:
		content := strings.Join(m.logLines, "\n")
		if m.logPaused {
			pausedStyle := lipgloss.NewStyle().Faint(true)
			indicator := pausedStyle.Render("── paused · PgDn to resume ──")

			if content != "" {
				content += "\n"
			}

			content += indicator
		}

		return content
	default:
		return ""
	}
}
