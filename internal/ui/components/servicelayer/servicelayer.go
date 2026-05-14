// Package servicelayer implements a Bubble Tea sub-model for a single Service
// Layer in the compositor. Each Service Layer owns a Service Inspector,
// Log Stream, Log Buffer, and scroll state. All Service Layers are peers —
// none is foreground or background.
package servicelayer

import (
	"context"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	logs "github.com/ma-tf/ogle/internal/services/docker/logs"
	"github.com/ma-tf/ogle/internal/ui/components/inspector"
	"github.com/ma-tf/ogle/internal/ui/components/logpane"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

//nolint:gochecknoglobals // package-level key bindings are shared across all Model instances
var (
	keyScrollUp      = key.NewBinding(key.WithKeys("pgup"))
	keyScrollDown    = key.NewBinding(key.WithKeys("pgdn"))
	wheelScrollLines = 2
)

// Model is a Bubble Tea sub-model for a single Service Layer.
type Model struct {
	ctx         context.Context
	projectName string
	service     domain.ServiceDef
	inspector   inspector.Model
	logPane     *logpane.LogPane
	theme       *theme.Theme
	focused     bool
	z           int
	w, h        int
	connected   bool
}

// New creates a Model for the given service. Call Init() to start the log
// stream once the Docker daemon is confirmed connected.
func New(
	ctx context.Context,
	projectName string,
	service domain.ServiceDef,
	th *theme.Theme,
	streamer logs.Streamer,
	logBufCap int,
) *Model {
	lp := logpane.NewLogPane(streamer, logBufCap)

	return &Model{
		ctx:         ctx,
		projectName: projectName,
		service:     service,
		inspector:   inspector.New(service, th),
		logPane:     lp,
		theme:       th,
		focused:     false,
		z:           0,
		w:           0,
		h:           0,
		connected:   false,
	}
}

// Init implements tea.Model. Sets log area to LogAreaConnecting; defers
// stream start until DaemonConnected is received.
func (s *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for this Service Layer.
func (s *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.LogLine:
		if msg.ServiceName != s.service.Name {
			return s, nil
		}

		cmd := s.logPane.HandleLogLine(msg)
		s.syncInspectorLogView()

		return s, s.tag(cmd)

	case msgs.LogStreamError:
		if msg.ServiceName != s.service.Name {
			return s, nil
		}

		cmd := s.logPane.HandleStreamError()
		s.syncInspectorLogView()

		return s, cmd

	case msgs.LogStreamContainerNotFound:
		if msg.ServiceName != s.service.Name {
			return s, nil
		}

		cmd := s.logPane.HandleContainerNotFound()
		s.syncInspectorLogView()

		// logStreamRetryMsg is unexported in logpane; route through logPane.Update
		// so the retry is handled internally when it fires.
		return s, cmd

	case msgs.DaemonConnected:
		return s, s.handleDaemonConnected()

	case msgs.DaemonUnavailable:
		s.handleDaemonUnavailable()

		return s, nil

	case tea.KeyPressMsg:
		if !s.focused {
			return s, nil
		}

		switch {
		case key.Matches(msg, keyScrollUp):
			s.logPane.ScrollUp(s.h)
			s.syncInspectorLogView()
		case key.Matches(msg, keyScrollDown):
			s.logPane.ScrollDown(s.h)
			s.syncInspectorLogView()
		}

		return s, nil

	case tea.WindowSizeMsg:
		s.SetSize(msg.Width, msg.Height)

		return s, nil
	}

	// Pass unrecognised messages through logPane.Update so that
	// logStreamRetryMsg (unexported in logpane) is handled internally.
	var logCmd tea.Cmd

	s.logPane, logCmd = s.logPane.Update(msg)

	if logCmd != nil {
		s.syncInspectorLogView()

		return s, s.tag(logCmd)
	}

	var inspCmd tea.Cmd

	s.inspector, inspCmd = s.inspector.Update(msg)

	return s, inspCmd
}

// SetSize stores new dimensions and propagates them to the inspector.
func (s *Model) SetSize(w, h int) {
	s.w = w
	s.h = h
	s.inspector = s.inspector.SetBounds(w, h, 0)
	s.syncInspectorLogView()
}

// SetFocused controls whether scroll key events are consumed by this layer.
func (s *Model) SetFocused(focused bool) {
	s.focused = focused
}

// SetZ sets the Z-index used when building the compositor.
func (s *Model) SetZ(z int) {
	s.z = z
}

// Z returns the current Z-index.
func (s *Model) Z() int {
	return s.z
}

// ServiceName returns the name of the Service this layer represents.
func (s *Model) ServiceName() string {
	return s.service.Name
}

// SetShowLabels controls label section visibility in the Service Inspector.
func (s *Model) SetShowLabels(show bool) {
	s.inspector = s.inspector.SetShowLabels(show)
}

// SetUnavailable pushes the Docker unavailable countdown into the inspector.
func (s *Model) SetUnavailable(u inspector.UnavailableState) {
	s.inspector = s.inspector.SetUnavailable(u)
}

// SetConnectState pushes a connectivity state change into the inspector.
func (s *Model) SetConnectState(cs inspector.ConnectState) {
	s.inspector = s.inspector.SetConnectState(cs)
}

// ScrollUp scrolls the log view up by paneHeight/2 rows.
func (s *Model) ScrollUp(paneHeight int) {
	s.logPane.ScrollUp(paneHeight)
	s.syncInspectorLogView()
}

// ScrollDown scrolls the log view down by paneHeight/2 rows.
func (s *Model) ScrollDown(paneHeight int) {
	s.logPane.ScrollDown(paneHeight)
	s.syncInspectorLogView()
}

// WheelUp scrolls the log view up by one row (mouse wheel).
func (s *Model) WheelUp() {
	s.logPane.ScrollUp(wheelScrollLines)
	s.syncInspectorLogView()
}

// WheelDown scrolls the log view down by one row (mouse wheel).
func (s *Model) WheelDown() {
	s.logPane.ScrollDown(wheelScrollLines)
	s.syncInspectorLogView()
}

// Close stops the log stream for this layer. Call before discarding a layer.
func (s *Model) Close() {
	s.logPane.Close()
}

// View renders the Service Inspector for this layer.
func (s *Model) View() string {
	return s.inspector.View()
}

// startLogStream begins the log stream and returns a tagged cmd.
func (s *Model) startLogStream() tea.Cmd {
	if s.service.Name == "" {
		return nil
	}

	name := logs.ContainerName(s.projectName, s.service.Name, s.service.ContainerName)

	return s.tag(s.logPane.StartStream(s.ctx, name))
}

// syncInspectorLogView pushes the current log state into the inspector model.
func (s *Model) syncInspectorLogView() {
	stderrStyle := lipgloss.NewStyle().Foreground(s.theme.LogStderr)
	lines := s.logPane.ComputeDisplayLines(s.w, s.h, stderrStyle)
	s.inspector = s.inspector.SetLogView(lines, s.logPane.Paused(), s.logPane.State())
}

// tag wraps a cmd so that any log messages it emits carry this layer's
// ServiceName. Handles msgs.LogLine, msgs.LogStreamError, and
// msgs.LogStreamContainerNotFound.
func (s *Model) tag(cmd tea.Cmd) tea.Cmd {
	if cmd == nil {
		return nil
	}

	name := s.service.Name

	return func() tea.Msg {
		msg := cmd()

		switch m := msg.(type) {
		case msgs.LogLine:
			m.ServiceName = name

			return m
		case msgs.LogStreamError:
			m.ServiceName = name

			return m
		case msgs.LogStreamContainerNotFound:
			m.ServiceName = name

			return m
		}

		return msg
	}
}

// handleDaemonConnected handles the DaemonConnected message.
func (s *Model) handleDaemonConnected() tea.Cmd {
	s.connected = true
	s.inspector = s.inspector.SetConnectState(inspector.ConnectStateConnected)
	cmd := s.startLogStream()
	s.syncInspectorLogView()

	return cmd
}

// handleDaemonUnavailable handles the DaemonUnavailable message.
func (s *Model) handleDaemonUnavailable() {
	s.connected = false
	s.logPane.Close()

	if s.logPane.State() == inspector.LogAreaNotFound {
		s.logPane.MarkUnavailable()
	}

	s.syncInspectorLogView()
}
