package states

import (
	"context"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
	"github.com/ma-tf/ogle/internal/ui/components/inspector"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	gracePeriodDuration  = 5 * time.Second
	retryIntervalSeconds = 60
)

type dashboardKeyMap struct {
	Quit         key.Binding
	Zoom         key.Binding
	ToggleLabels key.Binding
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
	dashboard dashboardKeyMap
	list      list.KeyMap
}

func (c combinedKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		c.list.CursorUp,
		c.list.CursorDown,
		c.list.Filter,
		c.list.ClearFilter,
		c.dashboard.Zoom,
		c.dashboard.ToggleLabels,
		c.dashboard.Quit,
	}
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
}

// NewDashboard returns a Dashboard state initialised with the given project.
func NewDashboard(ctx context.Context, project *domain.Project, th *theme.Theme) State {
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
		if key.Matches(msg, d.keys.Quit) && !d.serviceList.IsFiltering() {
			return d, tea.Quit
		}

		d.handleKeyPress(msg)

	case msgs.ServiceSelected:
		d.selectedService = msg.Service
		d.inspector = d.inspector.SetService(msg.Service)

		return d, nil

	case msgs.ProjectLoaded:
		d.project = msg.Project
		d.serviceList = d.serviceList.SetProject(msg.Project)

		// Reset selected service to first in the reloaded project.
		if len(msg.Project.Services) > 0 {
			d.selectedService = msg.Project.Services[0]
			d.inspector = d.inspector.SetService(d.selectedService)
		}

	case msgs.DaemonConnected:
		d.handleDaemonConnected()

		return d, nil

	case msgs.DaemonUnavailable:
		return d, d.handleDaemonUnavailable()

	case gracePeriodExpiredMsg:
		return d, d.handleGracePeriodExpired()

	case retryTickMsg:
		return d, d.handleRetryTick()
	}

	var inspectorCmd tea.Cmd

	d.inspector, inspectorCmd = d.inspector.Update(msg)

	var listCmd tea.Cmd

	d.serviceList, listCmd = d.serviceList.Update(msg)

	return d, tea.Batch(inspectorCmd, listCmd)
}

// View renders the two-pane dashboard layout with a help bar on the last row.
func (d *Dashboard) View() string {
	if d.layout.w == 0 || d.layout.h == 0 {
		return ""
	}

	zoomHelp := "fullscreen"
	if d.layout.IsLogFullscreen() {
		zoomHelp = "split"
	}

	d.keys.Zoom = key.NewBinding(key.WithKeys("z"), key.WithHelp("z", zoomHelp))

	km := combinedKeyMap{dashboard: d.keys, list: d.serviceList.KeyMap()}

	return d.layout.View(d.serviceList.View(), d.inspector.View(), d.focus == focusLeft) +
		"\n" + d.help.View(km)
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

func (d *Dashboard) handleZoom() {
	d.layout = d.layout.ToggleMode()
	if d.layout.IsLogFullscreen() {
		d.focus = focusRight
	} else {
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

func (d *Dashboard) handleKeyPress(msg tea.KeyPressMsg) {
	if d.serviceList.IsFiltering() {
		return
	}

	for _, kb := range []keyBinding{
		{d.keys.Zoom, d.handleZoom},
		{d.keys.ToggleLabels, d.handleToggleLabels},
	} {
		if key.Matches(msg, kb.binding) {
			kb.handle()

			return
		}
	}
}
