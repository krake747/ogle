package states

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist"
)

type dashboardKeyMap struct {
	Quit key.Binding
	Zoom key.Binding
}

func (k dashboardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Zoom}
}

func (k dashboardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Quit, k.Zoom}}
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
		c.dashboard.Quit,
	}
}

func (c combinedKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{c.list.CursorUp, c.list.CursorDown, c.list.NextPage, c.list.PrevPage},
		{c.list.Filter, c.list.ClearFilter, c.list.AcceptWhileFiltering, c.list.CancelWhileFiltering},
		{c.dashboard.Zoom, c.dashboard.Quit},
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
}

const (
	focusLeft  = 0
	focusRight = 1 // reserved for tab/focus switching (out of scope this iteration)
)

// Dashboard is the main project state. It renders a two-pane horizontal split:
// service list on the left, log/detail on the right.
type Dashboard struct {
	project     *domain.Project
	keys        dashboardKeyMap
	help        help.Model
	serviceList servicelist.Model
	layout      paneLayout
	focus       int
}

// NewDashboard returns a Dashboard state initialised with the given project.
func NewDashboard(project *domain.Project) State {
	return &Dashboard{
		project:     project,
		keys:        defaultDashboardKeys,
		help:        help.New(),
		serviceList: servicelist.New(project, 0, 0),
		layout:      newPaneLayout(),
		focus:       focusLeft,
	}
}

// Init implements State.
func (d *Dashboard) Init() tea.Cmd { return nil }

// SetSize implements State.
func (d *Dashboard) SetSize(w, h int) {
	d.help.SetWidth(w)
	d.layout = d.layout.SetSize(w, h)

	b := d.layout.ServiceListBounds()
	d.serviceList = d.serviceList.SetBounds(b.x, b.y, b.w, b.h)
}

// Update handles the quit key and forwards messages to the service list.
func (d *Dashboard) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, d.keys.Quit) && !d.serviceList.IsFiltering() {
			return d, tea.Quit
		}

		if key.Matches(msg, d.keys.Zoom) && !d.serviceList.IsFiltering() {
			d.layout = d.layout.ToggleMode()
			if d.layout.IsLogFullscreen() {
				d.focus = focusRight
			} else {
				d.focus = focusLeft
			}

			b := d.layout.ServiceListBounds()
			d.serviceList = d.serviceList.SetBounds(b.x, b.y, b.w, b.h)
		}

	case msgs.ProjectLoaded:
		d.project = msg.Project
		d.serviceList = d.serviceList.SetProject(msg.Project)
	}

	var listCmd tea.Cmd

	d.serviceList, listCmd = d.serviceList.Update(msg)

	return d, listCmd
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

	return d.layout.View(d.serviceList.View(), "logs", d.focus == focusLeft) +
		"\n" + d.help.View(km)
}
