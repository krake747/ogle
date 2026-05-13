package states

import (
	"context"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
	logs "github.com/ma-tf/ogle/internal/services/docker/logs"
	"github.com/ma-tf/ogle/internal/ui/components/inspector"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	gracePeriodDuration = 5 * time.Second
	logStreamRetryDelay = 5 * time.Second
	halfPaneDivisor     = 2
)

const (
	focusLeft  = 0
	focusRight = 1 // reserved for tab/focus switching (out of scope this iteration)
)

// Dashboard is the main project state. It renders a two-pane horizontal split:
// service list on the left, Service Inspector on the right.
type Dashboard struct {
	ctx             context.Context
	project         *domain.Project
	keys            dashboardKeyMap
	help            help.Model
	serviceList     servicelist.Model
	inspector       inspector.Model
	layout          PaneLayout
	focus           int
	selectedService domain.ServiceDef
	showLabels      bool
	theme           *theme.Theme
	themeName       string
	pollInterval    time.Duration
	logBufferCap    int
	connection      ConnectionMachine
	logView         LogPane
	drag            DragCoordinator
}

// NewDashboard returns a Dashboard state initialised with the given project.
func NewDashboard(
	ctx context.Context,
	project *domain.Project,
	th *theme.Theme,
	themeName string,
	poll time.Duration,
	logBufCap int,
	streamer logs.Streamer,
) State {
	var first domain.ServiceDef
	if len(project.Services) > 0 {
		first = project.Services[0]
	}

	return &Dashboard{
		ctx:             ctx,
		project:         project,
		keys:            defaultDashboardKeys,
		help:            help.New(),
		serviceList:     servicelist.New(project, th, 0, 0),
		inspector:       inspector.New(first, th),
		layout:          NewPaneLayout(th),
		focus:           focusLeft,
		selectedService: first,
		connection: ConnectionMachine{
			state:       inspector.ConnectStateConnecting,
			unavailable: inspector.UnavailableState{SecondsUntilRetry: 0},
		},
		logView: LogPane{
			streamer:   streamer,
			buffer:     newLogBuffer(logBufCap),
			scrollRows: 0,
			paused:     false,
			state:      inspector.LogAreaConnecting,
		},
		drag:         newDragCoordinator(),
		showLabels:   false,
		theme:        th,
		themeName:    themeName,
		pollInterval: poll,
		logBufferCap: logBufCap,
	}
}

// Init implements State. Fires Docker Connect and the grace-period timer in
// parallel. Watcher subscription is owned by the root orchestrator and must
// not be touched here.
func (d *Dashboard) Init() tea.Cmd {
	graceTick := tea.Tick(gracePeriodDuration, func(_ time.Time) tea.Msg {
		return gracePeriodExpiredMsg{}
	})

	return tea.Batch(
		svcdocker.Connect(d.ctx),
		graceTick,
	)
}

// SetSize implements State.
func (d *Dashboard) SetSize(w, h int) {
	d.help.SetWidth(w)
	d.layout = d.layout.SetSize(w, h)

	b := d.layout.ServiceListBounds()
	d.serviceList = d.serviceList.SetBounds(b.X, b.Y, b.W, b.H)

	lb := d.layout.LogViewBounds()
	d.inspector = d.inspector.SetBounds(lb.W, lb.H, lb.Y)
}

// Update handles all Dashboard messages.
func (d *Dashboard) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if s, cmd, handled := d.handleKeyPressMsg(msg); handled {
			return s, cmd
		}
	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			d.drag.HandleClick(msg)
		}
	case tea.MouseMotionMsg:
		if d.drag.HandleMotion(msg, d.layout) {
			return d, nil
		}
	case tea.MouseReleaseMsg:
		text, handled := d.drag.HandleRelease(msg, d.layout, d.serviceList.View(), d.inspector.View(), d.footerView())
		if handled {
			if text != "" {
				return d, tea.SetClipboard(text)
			}

			return d, nil
		}
	case tea.MouseWheelMsg:
		if d.handleMouseWheel(msg) {
			return d, nil
		}
	case msgs.ServiceSelected:
		return d, d.handleServiceSelected(msg)
	case msgs.ProjectLoaded:
		return d, d.handleProjectLoaded(msg)
	case msgs.DaemonConnected:
		return d, d.handleDaemonConnectedMsg()
	case msgs.DaemonUnavailable:
		d.logView.Close()

		return d, d.handleDaemonUnavailable()
	case msgs.ServiceActionCompleted:
		d.handleServiceActionCompleted(msg)

		return d, nil
	case gracePeriodExpiredMsg:
		return d, d.handleGracePeriodExpired()
	case retryTickMsg:
		return d, d.handleRetryTick()
	case msgs.LogLine:
		return d, d.handleLogLine(msg)
	case msgs.LogStreamError:
		d.handleLogStreamError()

		return d, nil
	case msgs.LogStreamContainerNotFound:
		return d, d.handleLogStreamContainerNotFound()
	case logStreamRetryMsg:
		return d, d.handleLogStreamRetry()
	}

	var inspectorCmd tea.Cmd

	d.inspector, inspectorCmd = d.inspector.Update(msg)

	var listCmd tea.Cmd

	d.serviceList, listCmd = d.serviceList.Update(msg)

	return d, tea.Batch(inspectorCmd, listCmd)
}

func (d *Dashboard) handleKeyPressMsg(msg tea.KeyPressMsg) (State, tea.Cmd, bool) {
	if !d.serviceList.IsFiltering() {
		if key.Matches(msg, d.keys.Quit) {
			return d, tea.Quit, true
		}

		if key.Matches(msg, d.keys.Settings) {
			return NewSettings(d.ctx, d.project, d.themeName, d.pollInterval, d.logBufferCap, d.theme), nil, true
		}
	}

	if cmd := d.handleKeyPress(msg); cmd != nil {
		return d, cmd, true
	}

	return nil, nil, false
}

func (d *Dashboard) handleMouseWheel(msg tea.MouseWheelMsg) bool {
	lb := d.layout.LogViewBounds()
	if msg.Y < lb.Y || msg.Y >= lb.Y+lb.H {
		return false
	}

	switch msg.Button {
	case tea.MouseWheelUp:
		d.logView.scrollRows++
		d.logView.paused = true
	case tea.MouseWheelDown:
		if d.logView.scrollRows > 0 {
			d.logView.scrollRows--
		}

		if d.logView.scrollRows == 0 {
			d.logView.paused = false
		}
	}

	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logView.Paused(), d.logView.State())

	return true
}

func (d *Dashboard) handleServiceSelected(msg msgs.ServiceSelected) tea.Cmd {
	d.selectedService = msg.Service
	d.inspector = d.inspector.SetService(msg.Service)
	d.logView.Close()
	d.logView.Clear()

	var cmd tea.Cmd

	if d.connection.ConnectState() == inspector.ConnectStateConnected {
		cmd = d.startLogStream(msg.Service)
	} else {
		d.logView.state = inspector.LogAreaConnecting
	}

	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logView.Paused(), d.logView.State())

	return cmd
}

func (d *Dashboard) handleProjectLoaded(msg msgs.ProjectLoaded) tea.Cmd {
	d.project = msg.Project
	d.serviceList = d.serviceList.SetProject(msg.Project)
	d.logView.Close()
	d.logView.Clear()

	if len(msg.Project.Services) > 0 {
		d.selectedService = msg.Project.Services[0]
		d.inspector = d.inspector.SetService(d.selectedService)
	}

	var cmd tea.Cmd

	if d.connection.ConnectState() == inspector.ConnectStateConnected && len(msg.Project.Services) > 0 {
		cmd = d.startLogStream(d.selectedService)
	} else {
		d.logView.state = inspector.LogAreaConnecting
	}

	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logView.Paused(), d.logView.State())

	return cmd
}

func (d *Dashboard) handleDaemonConnectedMsg() tea.Cmd {
	d.connection.HandleConnected()
	d.inspector = d.inspector.SetConnectState(d.connection.ConnectState())

	cmd := d.startLogStream(d.selectedService)

	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logView.Paused(), d.logView.State())

	return cmd
}

func (d *Dashboard) handleDaemonUnavailable() tea.Cmd {
	cmd := d.connection.HandleUnavailable()
	if cmd == nil {
		return nil
	}

	d.inspector = d.inspector.SetUnavailable(d.connection.Unavailable())

	if d.logView.State() == inspector.LogAreaNotFound {
		d.logView.state = inspector.LogAreaUnavailable
		d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logView.Paused(), d.logView.State())
	}

	return cmd
}

func (d *Dashboard) handleGracePeriodExpired() tea.Cmd {
	cmd := d.connection.HandleGracePeriodExpired()
	if cmd == nil {
		return nil
	}

	d.inspector = d.inspector.SetUnavailable(d.connection.Unavailable())

	return cmd
}

func (d *Dashboard) handleRetryTick() tea.Cmd {
	cmd := d.connection.HandleRetryTick(d.ctx)
	if cmd == nil {
		return nil
	}

	if d.connection.ConnectState() == inspector.ConnectStateConnecting {
		d.inspector = d.inspector.SetConnectState(inspector.ConnectStateConnecting)
	} else {
		d.inspector = d.inspector.SetUnavailable(d.connection.Unavailable())
	}

	return cmd
}

func (d *Dashboard) handleServiceActionCompleted(msg msgs.ServiceActionCompleted) {
	optimistic := domain.ServiceStateRunning
	if msg.Action == domain.ServiceActionStop {
		optimistic = domain.ServiceStateExited
	}

	if msg.Err != nil {
		d.serviceList = d.serviceList.SetActionError(msg.ServiceName, string(msg.Action)+" failed")
	} else {
		d.serviceList = d.serviceList.SetActionSuccess(msg.ServiceName, optimistic)
	}
}

func (d *Dashboard) handleLogLine(msg msgs.LogLine) tea.Cmd {
	cmd := d.logView.HandleLogLine(msg)
	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logView.Paused(), d.logView.State())

	return cmd
}

func (d *Dashboard) handleLogStreamError() {
	d.logView.HandleStreamError()
	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logView.Paused(), d.logView.State())
}

func (d *Dashboard) handleLogStreamContainerNotFound() tea.Cmd {
	cmd := d.logView.HandleContainerNotFound()
	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logView.Paused(), d.logView.State())

	return cmd
}

func (d *Dashboard) handleLogStreamRetry() tea.Cmd {
	name := logs.ContainerName(d.project.Name, d.selectedService.Name, d.selectedService.ContainerName)
	cmd := d.logView.HandleRetry(d.ctx, name)
	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logView.Paused(), d.logView.State())

	return cmd
}

// View renders the two-pane dashboard layout with a help bar on the last row.
func (d *Dashboard) View() string {
	if d.layout.w == 0 || d.layout.h == 0 {
		return ""
	}

	full := d.renderFull()
	if d.drag.Active() {
		full = d.drag.ApplyHighlight(full, d.layout)
	}

	return full
}

func (d *Dashboard) renderFull() string {
	return d.layout.View(d.serviceList.View(), d.inspector.View(), d.focus == focusLeft) +
		"\n" + d.footerView()
}

func (d *Dashboard) footerView() string {
	km := combinedKeyMap{
		dashboard:      d.keys,
		list:           d.serviceList.KeyMap(),
		actionBindings: d.actionBindings(),
	}

	return d.help.View(km)
}

// actionBindings returns the context-sensitive action key bindings for the
// help bar. Returns nil when Docker is unavailable or an action is in-flight.
func (d *Dashboard) actionBindings() []key.Binding {
	if d.connection.ConnectState() != inspector.ConnectStateConnected {
		return nil
	}

	state, hasState, inFlight := d.serviceList.SelectedEffectiveState()
	if inFlight {
		return nil
	}

	var bindings []key.Binding

	switch {
	case hasState && state == domain.ServiceStateRunning:
		bindings = append(bindings, d.keys.ActionStop, d.keys.ActionRestart)
	default:
		bindings = append(bindings, d.keys.ActionStart)
	}

	return append(bindings, d.keys.ActionRebuild)
}

// startLogStream closes any existing stream, starts a new one for svc, and
// returns a Next() cmd. Sets logView.state to LogAreaStreaming.
// If svc has no name (empty project) it is a no-op.
func (d *Dashboard) startLogStream(svc domain.ServiceDef) tea.Cmd {
	if svc.Name == "" {
		return nil
	}

	name := logs.ContainerName(d.project.Name, svc.Name, svc.ContainerName)

	return d.logView.StartStream(d.ctx, name)
}

// computeDisplayLines delegates to logView.ComputeDisplayLines with the
// current layout bounds and theme.
func (d *Dashboard) computeDisplayLines() []string {
	lb := d.layout.LogViewBounds()
	stderrStyle := lipgloss.NewStyle().Foreground(d.theme.LogStderr)

	return d.logView.ComputeDisplayLines(lb.W, lb.H, stderrStyle)
}

func (d *Dashboard) handleZoom() {
	d.layout = d.layout.ToggleMode()
	if d.layout.IsLogFullscreen() {
		d.keys.Zoom = key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "split"))
		d.focus = focusRight
	} else {
		d.keys.Zoom = key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "fullscreen"))
		d.focus = focusLeft
	}

	b := d.layout.ServiceListBounds()
	d.serviceList = d.serviceList.SetBounds(b.X, b.Y, b.W, b.H)
	lb := d.layout.LogViewBounds()
	d.inspector = d.inspector.SetBounds(lb.W, lb.H, lb.Y)
}

func (d *Dashboard) handleToggleLabels() {
	d.showLabels = !d.showLabels
	d.inspector = d.inspector.SetShowLabels(d.showLabels)
}

func (d *Dashboard) handleScrollUp() {
	lb := d.layout.LogViewBounds()
	d.logView.ScrollUp(lb.H)
	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logView.Paused(), d.logView.State())
}

func (d *Dashboard) handleScrollDown() {
	lb := d.layout.LogViewBounds()
	d.logView.ScrollDown(lb.H)
	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logView.Paused(), d.logView.State())
}

func (d *Dashboard) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	if d.serviceList.IsFiltering() {
		return nil
	}

	for _, kb := range []keyBinding{
		{d.keys.Zoom, d.handleZoom},
		{d.keys.ToggleLabels, d.handleToggleLabels},
		{d.keys.ScrollUp, d.handleScrollUp},
		{d.keys.ScrollDown, d.handleScrollDown},
	} {
		if key.Matches(msg, kb.binding) {
			kb.handle()

			return nil
		}
	}

	if d.connection.ConnectState() != inspector.ConnectStateConnected {
		return nil
	}

	state, hasState, inFlight := d.serviceList.SelectedEffectiveState()
	if inFlight {
		return nil
	}

	name := d.selectedService.Name
	if name == "" {
		return nil
	}

	file := d.project.File
	proj := d.project.Name

	switch {
	case key.Matches(msg, d.keys.ActionStop) && hasState && state == domain.ServiceStateRunning:
		d.serviceList = d.serviceList.SetActionInFlight(name, "stopping…")

		return svcdocker.Stop(d.ctx, file, proj, name)

	case key.Matches(msg, d.keys.ActionStart) &&
		(!hasState || state == domain.ServiceStateExited || state == domain.ServiceStateNotCreated):
		d.serviceList = d.serviceList.SetActionInFlight(name, "starting…")

		return svcdocker.Start(d.ctx, file, proj, name)

	case key.Matches(msg, d.keys.ActionRestart) && hasState && state == domain.ServiceStateRunning:
		d.serviceList = d.serviceList.SetActionInFlight(name, "restarting…")

		return svcdocker.Restart(d.ctx, file, proj, name)

	case key.Matches(msg, d.keys.ActionRebuild):
		d.serviceList = d.serviceList.SetActionInFlight(name, "rebuilding…")

		return svcdocker.Rebuild(d.ctx, file, proj, name)
	}

	return nil
}
