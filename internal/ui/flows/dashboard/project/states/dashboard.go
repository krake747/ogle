package states

import (
	"context"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
	logs "github.com/ma-tf/ogle/internal/services/docker/logs"
	"github.com/ma-tf/ogle/internal/ui/components/inspector"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	gracePeriodDuration  = 5 * time.Second
	retryIntervalSeconds = 60
	logStreamRetryDelay  = 5 * time.Second
	halfPaneDivisor      = 2
)

type dashboardKeyMap struct {
	Quit          key.Binding
	Zoom          key.Binding
	ToggleLabels  key.Binding
	Settings      key.Binding
	ActionStop    key.Binding
	ActionStart   key.Binding
	ActionRestart key.Binding
	ActionRebuild key.Binding
	ScrollUp      key.Binding
	ScrollDown    key.Binding
}

func (k dashboardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Zoom, k.ToggleLabels}
}

func (k dashboardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Quit, k.Zoom, k.ToggleLabels}}
}

// combinedKeyMap merges the dashboard-level bindings with the service list
// bindings for the help bar.
type combinedKeyMap struct {
	dashboard      dashboardKeyMap
	list           list.KeyMap
	actionBindings []key.Binding
}

// shortHelpBaseCount is the number of fixed bindings in combinedKeyMap.ShortHelp.
const shortHelpBaseCount = 8

func (c combinedKeyMap) ShortHelp() []key.Binding {
	bindings := make([]key.Binding, 0, shortHelpBaseCount+len(c.actionBindings))
	bindings = append(bindings,
		c.list.CursorUp,
		c.list.CursorDown,
		c.list.Filter,
		c.list.ClearFilter,
		c.dashboard.Zoom,
		c.dashboard.ToggleLabels,
		c.dashboard.Settings,
		c.dashboard.Quit,
	)

	return append(bindings, c.actionBindings...)
}

func (c combinedKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{c.list.CursorUp, c.list.CursorDown, c.list.NextPage, c.list.PrevPage},
		{c.list.Filter, c.list.ClearFilter, c.list.AcceptWhileFiltering, c.list.CancelWhileFiltering},
		{c.dashboard.Zoom, c.dashboard.ToggleLabels, c.dashboard.Quit},
	}
}

//nolint:gochecknoglobals // list of key bindings should be global and immutable
var defaultDashboardKeys = dashboardKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	Zoom: key.NewBinding(
		key.WithKeys("z"),
		key.WithHelp("z", "fullscreen"),
	),
	ToggleLabels: key.NewBinding(
		key.WithKeys("l"),
		key.WithHelp("l", "labels"),
	),
	Settings: key.NewBinding(
		key.WithKeys(","),
		key.WithHelp(",", "settings"),
	),
	ActionStop: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "stop"),
	),
	ActionStart: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "start"),
	),
	ActionRestart: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "restart"),
	),
	ActionRebuild: key.NewBinding(
		key.WithKeys("b"),
		key.WithHelp("b", "rebuild"),
	),
	ScrollUp: key.NewBinding(
		key.WithKeys("pgup"),
		key.WithHelp("pgup", "scroll up"),
	),
	ScrollDown: key.NewBinding(
		key.WithKeys("pgdn"),
		key.WithHelp("pgdn", "scroll down"),
	),
}

type keyBinding struct {
	binding key.Binding
	handle  func()
}

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
	layout          paneLayout
	focus           int
	selectedService domain.ServiceDef
	connectState    inspector.ConnectState
	unavailable     inspector.UnavailableState
	showLabels      bool
	drag            dragSelection
	lastPressX      int
	lastPressY      int
	theme           *theme.Theme
	themeName       string
	pollInterval    time.Duration
	logBufferCap    int
	logStreamer     *logs.LogStreamer
	logBuffer       logBuffer
	logScrollRows   int
	logPaused       bool
	logState        inspector.LogAreaState
}

// NewDashboard returns a Dashboard state initialised with the given project.
func NewDashboard(
	ctx context.Context,
	project *domain.Project,
	th *theme.Theme,
	themeName string,
	poll time.Duration,
	logBufCap int,
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
		layout:          newPaneLayout(th),
		focus:           focusLeft,
		selectedService: first,
		connectState:    inspector.ConnectStateConnecting,
		unavailable:     inspector.UnavailableState{SecondsUntilRetry: 0},
		showLabels:      false,
		drag:            dragSelection{active: false, startX: 0, startY: 0, endY: 0, component: selectionNone},
		lastPressX:      0,
		lastPressY:      0,
		theme:           th,
		themeName:       themeName,
		pollInterval:    poll,
		logBufferCap:    logBufCap,
		logStreamer:     logs.New(),
		logBuffer:       newLogBuffer(logBufCap),
		logScrollRows:   0,
		logPaused:       false,
		logState:        inspector.LogAreaConnecting,
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
	d.serviceList = d.serviceList.SetBounds(b.x, b.y, b.w, b.h)

	lb := d.layout.LogViewBounds()
	d.inspector = d.inspector.SetBounds(lb.w, lb.h, lb.y)
}

// Update handles all Dashboard messages.
func (d *Dashboard) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if s, cmd, handled := d.handleKeyPressMsg(msg); handled {
			return s, cmd
		}
	case tea.MouseClickMsg:
		d.handleMouseClick(msg)
	case tea.MouseMotionMsg:
		if d.handleMouseMotion(msg) {
			return d, nil
		}
	case tea.MouseReleaseMsg:
		if cmd, handled := d.handleMouseRelease(msg); handled {
			return d, cmd
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
		d.logStreamer.Close()

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
	if msg.Y < lb.y || msg.Y >= lb.y+lb.h {
		return false
	}

	switch msg.Button {
	case tea.MouseWheelUp:
		d.logScrollRows++
		d.logPaused = true
	case tea.MouseWheelDown:
		if d.logScrollRows > 0 {
			d.logScrollRows--
		}

		if d.logScrollRows == 0 {
			d.logPaused = false
		}
	}

	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logPaused, d.logState)

	return true
}

func (d *Dashboard) handleServiceSelected(msg msgs.ServiceSelected) tea.Cmd {
	d.selectedService = msg.Service
	d.inspector = d.inspector.SetService(msg.Service)
	d.logStreamer.Close()
	d.logBuffer.Clear()
	d.logScrollRows = 0
	d.logPaused = false

	var cmd tea.Cmd

	if d.connectState == inspector.ConnectStateConnected {
		cmd = d.startLogStream(msg.Service)
	} else {
		d.logState = inspector.LogAreaConnecting
	}

	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logPaused, d.logState)

	return cmd
}

func (d *Dashboard) handleProjectLoaded(msg msgs.ProjectLoaded) tea.Cmd {
	d.project = msg.Project
	d.serviceList = d.serviceList.SetProject(msg.Project)
	d.logStreamer.Close()
	d.logBuffer.Clear()
	d.logScrollRows = 0
	d.logPaused = false

	if len(msg.Project.Services) > 0 {
		d.selectedService = msg.Project.Services[0]
		d.inspector = d.inspector.SetService(d.selectedService)
	}

	var cmd tea.Cmd

	if d.connectState == inspector.ConnectStateConnected && len(msg.Project.Services) > 0 {
		cmd = d.startLogStream(d.selectedService)
	} else {
		d.logState = inspector.LogAreaConnecting
	}

	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logPaused, d.logState)

	return cmd
}

func (d *Dashboard) handleDaemonConnectedMsg() tea.Cmd {
	d.handleDaemonConnected()

	cmd := d.startLogStream(d.selectedService)

	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logPaused, d.logState)

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
	d.logBuffer.Append(logLine{text: msg.Text, isStderr: msg.IsStderr})

	if !d.logPaused {
		d.logScrollRows = 0
	}

	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logPaused, d.logState)

	return d.logStreamer.Next()
}

func (d *Dashboard) handleLogStreamError() {
	d.logStreamer.Close()
	d.logState = inspector.LogAreaUnavailable
	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logPaused, d.logState)
}

func (d *Dashboard) handleLogStreamContainerNotFound() tea.Cmd {
	d.logState = inspector.LogAreaNotFound
	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logPaused, d.logState)

	return tea.Tick(logStreamRetryDelay, func(_ time.Time) tea.Msg {
		return logStreamRetryMsg{}
	})
}

func (d *Dashboard) handleLogStreamRetry() tea.Cmd {
	cmd := d.startLogStream(d.selectedService)
	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logPaused, d.logState)

	return cmd
}

// View renders the two-pane dashboard layout with a help bar on the last row.
func (d *Dashboard) View() string {
	if d.layout.w == 0 || d.layout.h == 0 {
		return ""
	}

	full := d.renderFull()
	if d.drag.active {
		full = d.applySelectionHighlight(full)
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
	if d.connectState != inspector.ConnectStateConnected {
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

// startCountdown returns a one-shot one-second timer that fires retryTickMsg.
func startCountdown() tea.Cmd {
	return tea.Every(time.Second, func(_ time.Time) tea.Msg {
		return retryTickMsg{}
	})
}

func (d *Dashboard) handleDaemonConnected() {
	d.connectState = inspector.ConnectStateConnected
	d.inspector = d.inspector.SetConnectState(inspector.ConnectStateConnected)
}

func (d *Dashboard) handleDaemonUnavailable() tea.Cmd {
	if d.connectState != inspector.ConnectStateConnected {
		return nil
	}

	d.connectState = inspector.ConnectStateUnavailable
	d.unavailable = inspector.UnavailableState{SecondsUntilRetry: retryIntervalSeconds}
	d.inspector = d.inspector.SetUnavailable(d.unavailable)

	return startCountdown()
}

func (d *Dashboard) handleGracePeriodExpired() tea.Cmd {
	if d.connectState != inspector.ConnectStateConnecting {
		return nil
	}

	d.connectState = inspector.ConnectStateUnavailable
	d.unavailable = inspector.UnavailableState{SecondsUntilRetry: retryIntervalSeconds}
	d.inspector = d.inspector.SetUnavailable(d.unavailable)

	return startCountdown()
}

func (d *Dashboard) handleRetryTick() tea.Cmd {
	if d.connectState != inspector.ConnectStateUnavailable {
		return nil
	}

	d.unavailable.SecondsUntilRetry--

	if d.unavailable.SecondsUntilRetry <= 0 {
		d.connectState = inspector.ConnectStateConnecting
		d.inspector = d.inspector.SetConnectState(inspector.ConnectStateConnecting)

		return svcdocker.Connect(d.ctx)
	}

	d.inspector = d.inspector.SetUnavailable(d.unavailable)

	return startCountdown()
}

// startLogStream closes any existing stream, starts a new one for svc, and
// returns a Next() cmd. Sets logState to LogAreaStreaming.
// If svc has no name (empty project) it is a no-op.
func (d *Dashboard) startLogStream(svc domain.ServiceDef) tea.Cmd {
	if svc.Name == "" {
		return nil
	}

	name := logs.ContainerName(d.project.Name, svc.Name, svc.ContainerName)
	d.logStreamer.Start(d.ctx, name)
	d.logState = inspector.LogAreaStreaming

	return d.logStreamer.Next()
}

// computeDisplayLines builds the slice of pre-styled display rows that the
// inspector should render for the current scroll position. logScrollRows is
// clamped in-place.
func (d *Dashboard) computeDisplayLines() []string {
	lb := d.layout.LogViewBounds()
	if lb.w <= 0 || lb.h <= 0 {
		return nil
	}

	stderrStyle := lipgloss.NewStyle().Foreground(d.theme.LogStderr)

	var displayRows []string

	for _, line := range d.logBuffer.Lines() {
		for part := range strings.SplitSeq(line.text, "\n") {
			for _, row := range wrapLine(part, lb.w) {
				if line.isStderr {
					row = stderrStyle.Render(row)
				}

				displayRows = append(displayRows, row)
			}
		}
	}

	totalRows := len(displayRows)

	pausedRows := 0
	if d.logPaused {
		pausedRows = 1
	}

	availRows := lb.h - inspector.HeaderLines - pausedRows
	if availRows <= 0 {
		return nil
	}

	maxScroll := max(totalRows-availRows, 0)

	d.logScrollRows = clamp(d.logScrollRows, 0, maxScroll)

	end := max(totalRows-d.logScrollRows, 0)

	start := max(end-availRows, 0)

	return displayRows[start:end]
}

// wrapLine breaks text into display rows of at most width visible characters.
// ANSI CSI escape sequences are passed through without contributing to width.
//
//nolint:gocognit // three-state ANSI CSI parser; extracting transitions would obscure the state machine
func wrapLine(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	const (
		stNormal = iota
		stEsc
		stCSI
	)

	var (
		lines   []string
		current strings.Builder
	)

	currentWidth := 0
	state := stNormal

	for _, r := range text {
		switch state {
		case stNormal:
			if r == '\x1b' {
				state = stEsc

				current.WriteRune(r)

				continue
			}

			rw := ansi.StringWidth(string(r))
			if currentWidth+rw > width && currentWidth > 0 {
				lines = append(lines, current.String())
				current.Reset()

				currentWidth = 0
			}

			current.WriteRune(r)

			currentWidth += rw

		case stEsc:
			current.WriteRune(r)

			if r == '[' {
				state = stCSI
			} else {
				state = stNormal
			}

		case stCSI:
			current.WriteRune(r)

			if r >= 0x40 && r <= 0x7e {
				state = stNormal
			}
		}
	}

	if current.Len() > 0 || len(lines) == 0 {
		lines = append(lines, current.String())
	}

	return lines
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
	d.serviceList = d.serviceList.SetBounds(b.x, b.y, b.w, b.h)
	lb := d.layout.LogViewBounds()
	d.inspector = d.inspector.SetBounds(lb.w, lb.h, lb.y)
}

func (d *Dashboard) handleToggleLabels() {
	d.showLabels = !d.showLabels
	d.inspector = d.inspector.SetShowLabels(d.showLabels)
}

func (d *Dashboard) handleScrollUp() {
	lb := d.layout.LogViewBounds()

	halfPane := max(lb.h/halfPaneDivisor, 1)

	d.logScrollRows += halfPane
	d.logPaused = true
	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logPaused, d.logState)
}

func (d *Dashboard) handleScrollDown() {
	lb := d.layout.LogViewBounds()

	halfPane := max(lb.h/halfPaneDivisor, 1)

	d.logScrollRows -= halfPane
	if d.logScrollRows < 0 {
		d.logScrollRows = 0
	}

	if d.logScrollRows == 0 {
		d.logPaused = false
	}

	d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logPaused, d.logState)
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

	if d.connectState != inspector.ConnectStateConnected {
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

func (d *Dashboard) handleMouseClick(msg tea.MouseClickMsg) {
	if msg.Button != tea.MouseLeft {
		return
	}

	d.lastPressX = msg.X
	d.lastPressY = msg.Y
	d.drag = dragSelection{active: false, startX: 0, startY: 0, endY: 0, component: selectionNone}
}

// handleMouseMotion returns true when Update must short-circuit: while a drag
// is active the inspector and service list must not receive the motion event.
func (d *Dashboard) handleMouseMotion(msg tea.MouseMotionMsg) bool {
	if msg.Button != tea.MouseLeft {
		return false
	}

	dx := msg.X - d.lastPressX
	dy := msg.Y - d.lastPressY

	if d.drag.active {
		b := d.boundsForComponent(d.drag.component)
		d.drag.endY = clamp(msg.Y, b.y, b.y+b.h-1)

		return true
	}

	if abs(dx) <= 1 && abs(dy) <= 1 {
		return false
	}

	comp := d.hitTestComponent(d.lastPressX, d.lastPressY)
	if comp == selectionNone {
		return false
	}

	d.drag = dragSelection{
		active:    true,
		startX:    d.lastPressX,
		startY:    d.lastPressY,
		endY:      msg.Y,
		component: comp,
	}

	return true
}

// handleMouseRelease returns (cmd, true) when Update must short-circuit: a
// completed drag must not propagate to the inspector or service list.
func (d *Dashboard) handleMouseRelease(msg tea.MouseReleaseMsg) (tea.Cmd, bool) {
	if msg.Button == tea.MouseLeft && d.drag.active {
		text := d.extractSelection()
		d.drag = dragSelection{active: false, startX: 0, startY: 0, endY: 0, component: selectionNone}

		if text != "" {
			return tea.SetClipboard(text), true
		}

		return nil, true
	}

	d.drag = dragSelection{active: false, startX: 0, startY: 0, endY: 0, component: selectionNone}

	return nil, false
}

func (d *Dashboard) hitTestComponent(x, y int) selectionComponent {
	lb := d.layout.ServiceListBounds()
	if x >= lb.x && x < lb.x+lb.w && y >= lb.y && y < lb.y+lb.h {
		return selectionServiceList
	}

	rb := d.layout.LogViewBounds()
	if x >= rb.x && x < rb.x+rb.w && y >= rb.y && y < rb.y+rb.h {
		return selectionInspector
	}

	paneH := d.layout.h - separatorRows - helpBarHeight
	if y == paneH {
		return selectionFooter
	}

	return selectionNone
}

func (d *Dashboard) boundsForComponent(c selectionComponent) rect {
	switch c {
	case selectionServiceList:
		return d.layout.ServiceListBounds()
	case selectionInspector:
		return d.layout.LogViewBounds()
	case selectionFooter:
		paneH := d.layout.h - separatorRows - helpBarHeight

		return rect{x: 0, y: paneH, w: d.layout.w, h: 1}
	case selectionNone:
		return rect{}
	}

	return rect{}
}

// extractSelection uses each component's own View() output to avoid x-range
// slicing across split-pane terminal rows (Decision 14).
func (d *Dashboard) extractSelection() string {
	b := d.boundsForComponent(d.drag.component)
	minRow, maxRow := d.drag.rows()
	localMin := minRow - b.y
	localMax := maxRow - b.y

	var source string

	switch d.drag.component {
	case selectionServiceList:
		source = d.serviceList.View()
	case selectionInspector:
		source = d.inspector.View()
	case selectionFooter:
		source = d.footerView()
	case selectionNone:
		return ""
	}

	lines := strings.Split(source, "\n")

	return extractText(lines, localMin, localMax, rect{x: 0, y: 0, w: b.w, h: b.h})
}

func (d *Dashboard) applySelectionHighlight(rendered string) string {
	lines := strings.Split(rendered, "\n")
	minRow, maxRow := d.drag.rows()
	b := d.boundsForComponent(d.drag.component)
	highlight := lipgloss.NewStyle().Reverse(true)

	for row := minRow; row <= maxRow; row++ {
		if row < b.y || row >= b.y+b.h || row >= len(lines) {
			continue
		}

		lines[row] = highlight.Render(ansi.Strip(lines[row]))
	}

	return strings.Join(lines, "\n")
}
