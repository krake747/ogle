package dashboard

import (
	"context"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
	logs "github.com/ma-tf/ogle/internal/services/docker/logs"
	"github.com/ma-tf/ogle/internal/ui/components/inspector"
	"github.com/ma-tf/ogle/internal/ui/components/servicelayer"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	gracePeriodDuration      = 5 * time.Second
	settingsLayerZ           = 100
	initCommandsCapacityBase = 2
)

const (
	focusLeft  = 0
	focusRight = 1 // reserved for tab/focus switching (out of scope this iteration)
)

// Screen is the main dashboard state. It renders a two-pane horizontal split:
// service list on the left, the top Service Layer on the right.
type Screen struct {
	ctx          context.Context
	project      *domain.Project
	keys         dashboardKeyMap
	help         help.Model
	serviceList  servicelist.Model
	layout       PaneLayout
	focus        int
	layers       map[string]*servicelayer.Model
	topLayer     string
	nextZ        int
	showLabels   bool
	theme        *theme.Theme
	themeName    string
	settings     *Settings
	zm           *zone.Manager
	logBufferCap int
	connection   ConnectionMachine
	drag         DragCoordinator
}

// NewScreen returns a Screen state initialised with the given project.
// The streamer parameter is retained for API compatibility; each Service Layer
// creates its own streamer via logs.New().
func NewScreen(
	ctx context.Context,
	project *domain.Project,
	th *theme.Theme,
	themeName string,
	logBufCap int,
	_ logs.Streamer,
	zm *zone.Manager,
) State {
	d := &Screen{
		ctx:         ctx,
		project:     project,
		keys:        defaultDashboardKeys,
		help:        help.New(),
		serviceList: servicelist.New(project, th, zm, 0, 0),
		layout:      NewPaneLayout(th, zm),
		focus:       focusLeft,
		connection: ConnectionMachine{
			state:       inspector.ConnectStateConnecting,
			unavailable: inspector.UnavailableState{SecondsUntilRetry: 0},
		},
		drag:         newDragCoordinator(zm),
		showLabels:   false,
		theme:        th,
		themeName:    themeName,
		zm:           zm,
		logBufferCap: logBufCap,
		nextZ:        1,
		layers:       nil,
		topLayer:     "",
		settings:     nil,
	}

	d.layers, d.topLayer = d.initLayers()

	return d
}

// Init implements State. Fires Docker Connect, the grace-period timer, and all
// Service Layer Init commands in parallel.
func (d *Screen) Init() tea.Cmd {
	graceTick := tea.Tick(gracePeriodDuration, func(_ time.Time) tea.Msg {
		return gracePeriodExpiredMsg{}
	})

	cmds := make([]tea.Cmd, 0, initCommandsCapacityBase+len(d.layers))
	cmds = append(cmds, svcdocker.Connect(d.ctx), graceTick)

	for _, layer := range d.layers {
		cmds = append(cmds, layer.Init())
	}

	return tea.Batch(cmds...)
}

// SetSize implements State.
func (d *Screen) SetSize(w, h int) {
	d.help.SetWidth(w)
	d.layout = d.layout.SetSize(w, h)

	b := d.layout.ServiceListBounds()
	d.serviceList = d.serviceList.SetBounds(b.X, b.Y, b.W, b.H)

	lb := d.layout.LogViewBounds()

	for _, layer := range d.layers {
		layer.SetSize(lb.W, lb.H)
	}

	if d.settings != nil {
		d.settings.SetSize(w, h)
	}
}

// Update handles all Screen messages.
func (d *Screen) Update(msg tea.Msg) (State, tea.Cmd) {
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
		// DragCoordinator.HandleMotion disabled — pending compositor adaptation
	case tea.MouseReleaseMsg:
		// DragCoordinator.HandleRelease disabled — pending compositor adaptation
	case tea.MouseWheelMsg:
		if d.handleMouseWheel(msg) {
			return d, nil
		}
	case msgs.ServiceSelected:
		d.handleServiceSelected(msg)

		return d, nil
	case msgs.ProjectLoaded:
		return d, d.handleProjectLoaded(msg)
	case msgs.DaemonConnected:
		return d, d.handleDaemonConnectedMsg()
	case msgs.DaemonUnavailable:
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
		d.handleLogStreamError(msg)

		return d, nil
	case msgs.LogStreamContainerNotFound:
		return d, d.handleLogStreamContainerNotFound(msg)
	case msgs.SettingsApplied:
		return d, d.handleSettingsApplied(msg)
	case msgs.OrphanDiscovered:
		return d, d.handleOrphanDiscovered(msg)
	case msgs.OrphanGone:
		d.handleOrphanGone(msg)

		return d, nil
	}

	// Route unrecognised messages through all layers (handles internal
	// logStreamRetryMsg) and through the service list.
	return d, d.routeUnhandled(msg)
}

// routeUnhandled routes unrecognised messages through all layers and the service list.
func (d *Screen) routeUnhandled(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	for _, layer := range d.layers {
		_, layerCmd := layer.Update(msg)
		if layerCmd != nil {
			cmds = append(cmds, layerCmd)
		}
	}

	var listCmd tea.Cmd

	d.serviceList, listCmd = d.serviceList.Update(msg)

	if listCmd != nil {
		cmds = append(cmds, listCmd)
	}

	return tea.Batch(cmds...)
}

func (d *Screen) handleKeyPressMsg(msg tea.KeyPressMsg) (State, tea.Cmd, bool) {
	if d.settings != nil {
		next, cmd := d.settings.Update(msg)
		d.settings = next

		return d, cmd, true
	}

	if !d.serviceList.IsFiltering() {
		if key.Matches(msg, d.keys.Quit) {
			return d, tea.Quit, true
		}

		if key.Matches(msg, d.keys.Settings) {
			d.settings = NewSettings(d.themeName, d.logBufferCap, d.theme, d.zm)
			d.settings.SetSize(d.layout.w, d.layout.h)

			return d, nil, true
		}
	}

	if cmd := d.handleKeyPress(msg); cmd != nil {
		return d, cmd, true
	}

	return nil, nil, false
}

func (d *Screen) handleMouseWheel(msg tea.MouseWheelMsg) bool {
	lb := d.layout.LogViewBounds()
	if msg.Y < lb.Y || msg.Y >= lb.Y+lb.H {
		return false
	}

	layer, ok := d.layers[d.topLayer]
	if !ok {
		return false
	}

	switch msg.Button {
	case tea.MouseWheelUp:
		layer.WheelUp()
	case tea.MouseWheelDown:
		layer.WheelDown()
	}

	return true
}

func (d *Screen) handleServiceSelected(msg msgs.ServiceSelected) {
	if prev, ok := d.layers[d.topLayer]; ok {
		prev.SetFocused(false)
	}

	d.topLayer = msg.ServiceName
	d.nextZ++

	if layer, ok := d.layers[d.topLayer]; ok {
		layer.SetZ(d.nextZ)
		layer.SetFocused(true)
		layer.SetShowLabels(d.showLabels)
	}
}

func (d *Screen) handleProjectLoaded(msg msgs.ProjectLoaded) tea.Cmd {
	d.project = msg.Project
	d.serviceList = d.serviceList.SetProject(msg.Project)

	// Build set of service names in the updated project.
	newNames := make(map[string]struct{}, len(msg.Project.Services))
	for _, svc := range msg.Project.Services {
		newNames[svc.Name] = struct{}{}
	}

	// Close and remove layers for services no longer present.
	for name, layer := range d.layers {
		if _, ok := newNames[name]; !ok {
			layer.Close()
			delete(d.layers, name)
		}
	}

	// If the top layer was removed, pick a new one from the updated list.
	if _, ok := d.layers[d.topLayer]; !ok {
		d.topLayer = ""
		if len(msg.Project.Services) > 0 {
			d.topLayer = msg.Project.Services[0].Name
		}

		// Focus immediately if the new top layer is an existing layer; if it is
		// a newly-created service it will be focused in the creation loop below.
		if layer, layerExists := d.layers[d.topLayer]; layerExists {
			d.nextZ++
			layer.SetZ(d.nextZ)
			layer.SetFocused(true)
		}
	}

	connected := d.connection.ConnectState() == inspector.ConnectStateConnected

	var cmds []tea.Cmd

	// Create layers for services that were not in the previous project.
	for _, svc := range msg.Project.Services {
		if _, exists := d.layers[svc.Name]; exists {
			continue
		}

		layer := servicelayer.New(d.ctx, d.project.Name, svc, d.theme, logs.New(), d.logBufferCap)

		if svc.Name == d.topLayer {
			d.nextZ++
			layer.SetZ(d.nextZ)
			layer.SetFocused(true)
		}

		d.layers[svc.Name] = layer

		if connected {
			_, layerCmd := layer.Update(msgs.DaemonConnected{})
			if layerCmd != nil {
				cmds = append(cmds, layerCmd)
			}
		}
	}

	return tea.Batch(cmds...)
}

func (d *Screen) handleDaemonConnectedMsg() tea.Cmd {
	d.connection.HandleConnected()

	var cmds []tea.Cmd

	for _, layer := range d.layers {
		_, layerCmd := layer.Update(msgs.DaemonConnected{})
		if layerCmd != nil {
			cmds = append(cmds, layerCmd)
		}
	}

	return tea.Batch(cmds...)
}

func (d *Screen) handleDaemonUnavailable() tea.Cmd {
	countdownCmd := d.connection.HandleUnavailable()
	if countdownCmd == nil {
		return nil
	}

	cmds := []tea.Cmd{countdownCmd}

	for _, layer := range d.layers {
		_, layerCmd := layer.Update(msgs.DaemonUnavailable{Err: nil})
		layer.SetUnavailable(d.connection.Unavailable())

		if layerCmd != nil {
			cmds = append(cmds, layerCmd)
		}
	}

	return tea.Batch(cmds...)
}

func (d *Screen) handleGracePeriodExpired() tea.Cmd {
	countdownCmd := d.connection.HandleGracePeriodExpired()
	if countdownCmd == nil {
		return nil
	}

	for _, layer := range d.layers {
		layer.SetUnavailable(d.connection.Unavailable())
	}

	return countdownCmd
}

func (d *Screen) handleRetryTick() tea.Cmd {
	cmd := d.connection.HandleRetryTick(d.ctx)
	if cmd == nil {
		return nil
	}

	if d.connection.ConnectState() == inspector.ConnectStateConnecting {
		for _, layer := range d.layers {
			layer.SetConnectState(inspector.ConnectStateConnecting)
		}
	} else {
		for _, layer := range d.layers {
			layer.SetUnavailable(d.connection.Unavailable())
		}
	}

	return cmd
}

func (d *Screen) handleServiceActionCompleted(msg msgs.ServiceActionCompleted) {
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

func (d *Screen) handleLogLine(msg msgs.LogLine) tea.Cmd {
	layer, ok := d.layers[msg.ServiceName]
	if !ok {
		return nil
	}

	_, cmd := layer.Update(msg)

	return cmd
}

func (d *Screen) handleLogStreamError(msg msgs.LogStreamError) {
	layer, ok := d.layers[msg.ServiceName]
	if !ok {
		return
	}

	layer.Update(msg)
}

func (d *Screen) handleLogStreamContainerNotFound(msg msgs.LogStreamContainerNotFound) tea.Cmd {
	layer, ok := d.layers[msg.ServiceName]
	if !ok {
		return nil
	}

	_, cmd := layer.Update(msg)

	return cmd
}

func (d *Screen) handleSettingsApplied(msg msgs.SettingsApplied) tea.Cmd {
	th, _ := theme.Load(msg.Theme, "")
	d.theme = th
	d.themeName = msg.Theme
	d.logBufferCap = msg.LogBufferCap

	d.serviceList = servicelist.New(d.project, th, d.zm, 0, 0)
	d.layout = NewPaneLayout(th, d.zm)

	for _, layer := range d.layers {
		layer.Close()
	}

	d.nextZ = 1
	d.layers, d.topLayer = d.initLayers()

	if d.connection.ConnectState() != inspector.ConnectStateConnected {
		return nil
	}

	var cmds []tea.Cmd

	for _, layer := range d.layers {
		_, layerCmd := layer.Update(msgs.DaemonConnected{})
		if layerCmd != nil {
			cmds = append(cmds, layerCmd)
		}
	}

	return tea.Batch(cmds...)
}

// View renders the dashboard via the lipgloss Compositor with a help bar appended.
func (d *Screen) View() string {
	if d.layout.w == 0 || d.layout.h == 0 {
		return ""
	}

	// DragCoordinator.ApplyHighlight is disabled — pending compositor adaptation.
	return d.renderFull()
}

func (d *Screen) renderFull() string {
	w := d.layout.w
	h := d.layout.h
	paneH := max(h-separatorRows-helpBarHeight, 0)
	innerH := max(paneH-borderHeight, 0)

	var lyrs []*lipgloss.Layer

	if !d.layout.IsLogFullscreen() {
		leftW := min(w*servicePaneRatio/servicePaneRatioDen, servicePaneMaxW)
		leftContentW := max(leftW-borderWidth, 0)

		leftBorderStyle := d.theme.BorderBlurred
		if d.focus == focusLeft {
			leftBorderStyle = d.theme.BorderFocused
		}

		svcContent := lipgloss.NewStyle().
			Width(leftContentW).
			Height(innerH).
			Render(d.serviceList.View())
		svcContent = d.zm.Mark("pane-left", svcContent)
		leftPane := leftBorderStyle.Width(leftW).Height(paneH).Render(svcContent)

		lyrs = append(lyrs, lipgloss.NewLayer(leftPane).X(0).Y(0).Z(1).ID("service-list"))
	}

	if layer, ok := d.layers[d.topLayer]; ok {
		var rightX, rightW int

		if d.layout.IsLogFullscreen() {
			rightX = 0
			rightW = w
		} else {
			leftW := min(w*servicePaneRatio/servicePaneRatioDen, servicePaneMaxW)
			rightX = leftW
			rightW = w - leftW
		}

		rightContentW := max(rightW-borderWidth, 0)

		rightBorderStyle := d.theme.BorderBlurred
		if d.focus == focusRight {
			rightBorderStyle = d.theme.BorderFocused
		}

		inspContent := lipgloss.NewStyle().Width(rightContentW).Height(innerH).Render(layer.View())
		inspContent = d.zm.Mark("pane-right", inspContent)
		rightPane := rightBorderStyle.Width(rightW).Height(paneH).Render(inspContent)

		lyrs = append(lyrs, lipgloss.NewLayer(rightPane).X(rightX).Y(0).Z(layer.Z()).ID(d.topLayer))
	}

	if d.settings != nil {
		lyrs = append(
			lyrs,
			lipgloss.NewLayer(d.settings.View()).X(0).Y(0).Z(settingsLayerZ).ID("settings"),
		)
	}

	if len(lyrs) == 0 {
		return "\n" + d.footerView()
	}

	return lipgloss.NewCompositor(lyrs...).Render() + "\n" + d.footerView()
}

func (d *Screen) footerView() string {
	km := combinedKeyMap{
		dashboard:      d.keys,
		list:           d.serviceList.KeyMap(),
		actionBindings: d.actionBindings(),
	}

	return d.help.View(km)
}

// actionBindings returns the context-sensitive action key bindings for the
// help bar. Returns nil when Docker is unavailable or an action is in-flight.
func (d *Screen) actionBindings() []key.Binding {
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

func (d *Screen) handleZoom() {
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

	for _, layer := range d.layers {
		layer.SetSize(lb.W, lb.H)
	}
}

func (d *Screen) handleToggleLabels() {
	d.showLabels = !d.showLabels

	if layer, ok := d.layers[d.topLayer]; ok {
		layer.SetShowLabels(d.showLabels)
	}
}

func (d *Screen) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd {
	if d.serviceList.IsFiltering() {
		return nil
	}

	// Zoom and ToggleLabels are handled here; scroll keys (pgup/pgdn) fall
	// through to the servicelayer's focused key handler in the Update loop.
	for _, kb := range []keyBinding{
		{d.keys.Zoom, d.handleZoom},
		{d.keys.ToggleLabels, d.handleToggleLabels},
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

	name := d.topLayer
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

func (d *Screen) handleOrphanDiscovered(msg msgs.OrphanDiscovered) tea.Cmd {
	if _, exists := d.layers[msg.Service.Name]; exists {
		return nil
	}

	layer := servicelayer.New(
		d.ctx,
		d.project.Name,
		msg.Service,
		d.theme,
		logs.New(),
		d.logBufferCap,
	)

	if d.topLayer == "" {
		d.topLayer = msg.Service.Name
		d.nextZ++
		layer.SetZ(d.nextZ)
		layer.SetFocused(true)
	}

	lb := d.layout.LogViewBounds()
	layer.SetSize(lb.W, lb.H)

	d.layers[msg.Service.Name] = layer

	if d.connection.ConnectState() != inspector.ConnectStateConnected {
		return nil
	}

	_, cmd := layer.Update(msgs.DaemonConnected{})

	return cmd
}

func (d *Screen) handleOrphanGone(msg msgs.OrphanGone) {
	layer, ok := d.layers[msg.ServiceName]
	if !ok {
		return
	}

	layer.Close()
	delete(d.layers, msg.ServiceName)

	if d.topLayer != msg.ServiceName {
		return
	}

	d.topLayer = ""

	for name := range d.layers {
		d.topLayer = name
		d.nextZ++
		d.layers[name].SetZ(d.nextZ)
		d.layers[name].SetFocused(true)

		break
	}
}

// initLayers creates a Service Layer for every Service in the current project.
// The first service is set as focused and assigned Z=nextZ. Returns the layers
// map and the name of the top layer.
func (d *Screen) initLayers() (map[string]*servicelayer.Model, string) {
	layers := make(map[string]*servicelayer.Model, len(d.project.Services))
	topLayer := ""

	for i, svc := range d.project.Services {
		layer := servicelayer.New(d.ctx, d.project.Name, svc, d.theme, logs.New(), d.logBufferCap)

		if i == 0 {
			topLayer = svc.Name

			layer.SetZ(d.nextZ)
			layer.SetFocused(true)
		}

		layers[svc.Name] = layer
	}

	return layers, topLayer
}
