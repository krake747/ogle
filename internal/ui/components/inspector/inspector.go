// Package inspector implements the Service Inspector component — the right pane
// of the Dashboard. It renders a stacked layout: a compact detail header above
// the Log Stream area.
package inspector

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
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

// Model is the Service Inspector component. It is a value type; all mutating
// methods return a new Model.
type Model struct {
	service      domain.ServiceDef
	runtime      *domain.ServiceRuntimeData
	connectState ConnectState
	unavailable  UnavailableState
	showLabels   bool
	labels       labelsModel
	width        int
	height       int
}

// New returns a Model for the given service with ConnectStateConnecting.
func New(service domain.ServiceDef) Model {
	return Model{
		service:      service,
		runtime:      nil,
		connectState: ConnectStateConnecting,
		labels:       newLabelsModel(service),
		unavailable:  UnavailableState{SecondsUntilRetry: 0},
		showLabels:   false,
		width:        0,
		height:       0,
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

// SetBounds propagates terminal dimensions to the inspector.
func (m Model) SetBounds(w, h int) Model {
	m.width = w
	m.height = h

	return m
}

// Init satisfies tea.Model (not used directly; the Dashboard owns Init).
func (m Model) Init() tea.Cmd { return nil }

// Update handles messages directed at the Service Inspector.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.showLabels {
		var cmd tea.Cmd

		m.labels, cmd = m.labels.update(msg)

		return m, cmd
	}

	return m, nil
}

// View renders the Service Inspector: detail header above the Log Stream area.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	header := renderHeader(m.service, m.runtime, m.width)

	var body string
	if m.showLabels {
		body = m.labels.view(m.width, m.height-headerLines)
	} else {
		body = m.renderLogArea()
	}

	return header + "\n" + body
}

// renderLogArea returns the Log Stream placeholder appropriate for the current
// connectivity state.
func (m Model) renderLogArea() string {
	switch m.connectState {
	case ConnectStateConnecting:
		return "Connecting to Docker…"
	case ConnectStateConnected:
		return ""
	case ConnectStateUnavailable:
		return fmt.Sprintf("Docker unavailable — retrying in %ds…", m.unavailable.SecondsUntilRetry)
	default:
		return ""
	}
}
