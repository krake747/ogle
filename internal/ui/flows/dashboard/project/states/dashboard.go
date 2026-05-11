package states

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist"
)

type dashboardKeyMap struct {
	Quit key.Binding
}

func (k dashboardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit}
}

func (k dashboardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Quit}}
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
		c.dashboard.Quit,
	}
}

func (c combinedKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{c.list.CursorUp, c.list.CursorDown, c.list.NextPage, c.list.PrevPage},
		{c.list.Filter, c.list.ClearFilter, c.list.AcceptWhileFiltering, c.list.CancelWhileFiltering},
		{c.dashboard.Quit},
	}
}

//nolint:gochecknoglobals // list of key bindings should be global and immutable
var defaultDashboardKeys = dashboardKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
}

const (
	focusLeft  = 0
	focusRight = 1 // reserved for tab/focus switching (out of scope this iteration)

	servicePaneRatio    = 30
	servicePaneRatioDen = 100
	servicePaneMaxW     = 80
	borderWidth         = 2
	borderHeight        = 2
	separatorRows       = 1
	helpBarHeight       = 1
)

// Dashboard is the main project state. It renders a two-pane horizontal split:
// service list on the left, log/detail on the right.
type Dashboard struct {
	project     *domain.Project
	keys        dashboardKeyMap
	help        help.Model
	serviceList servicelist.Model
	w, h        int
	focus       int
}

// NewDashboard returns a Dashboard state initialised with the given project.
func NewDashboard(project *domain.Project) State {
	//nolint:exhaustruct // w, h set on first SetSize call
	return &Dashboard{
		project:     project,
		keys:        defaultDashboardKeys,
		help:        help.New(),
		serviceList: servicelist.New(project, 0, 0),
		focus:       focusLeft,
	}
}

// Init implements State.
func (d *Dashboard) Init() tea.Cmd { return nil }

// SetSize implements State.
func (d *Dashboard) SetSize(w, h int) {
	d.w = w
	d.h = h
	d.help.SetWidth(w)

	leftW := min(w*servicePaneRatio/servicePaneRatioDen, servicePaneMaxW)
	leftContentW := max(leftW-borderWidth, 0)
	paneH := max(h-separatorRows-helpBarHeight, 0)
	innerH := max(paneH-borderHeight, 0)

	d.serviceList = d.serviceList.SetSize(leftContentW, innerH)
}

// Update handles the quit key and forwards messages to the service list.
func (d *Dashboard) Update(msg tea.Msg) (State, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		if key.Matches(keyMsg, d.keys.Quit) && !d.serviceList.IsFiltering() {
			return d, tea.Quit
		}
	}

	if loaded, ok := msg.(msgs.ProjectLoaded); ok {
		d.project = loaded.Project
		d.serviceList = d.serviceList.SetProject(loaded.Project)
	}

	var listCmd tea.Cmd

	d.serviceList, listCmd = d.serviceList.Update(msg)

	return d, listCmd
}

// View renders the two-pane dashboard layout with a help bar on the last row.
func (d *Dashboard) View() string {
	if d.w == 0 || d.h == 0 {
		return ""
	}

	leftW := min(d.w*servicePaneRatio/servicePaneRatioDen, servicePaneMaxW)
	rightW := d.w - leftW
	leftContentW := max(leftW-borderWidth, 0)
	rightContentW := max(rightW-borderWidth, 0)
	paneH := max(d.h-separatorRows-helpBarHeight, 0)
	innerH := max(paneH-borderHeight, 0)

	highlight := lipgloss.Color("62")
	dimmed := lipgloss.Color("240")

	leftBorderColor := dimmed
	rightBorderColor := dimmed

	if d.focus == focusLeft {
		leftBorderColor = highlight
	} else {
		rightBorderColor = highlight
	}

	leftInner := lipgloss.NewStyle().
		Width(leftContentW).
		Height(innerH).
		Render(d.serviceList.View())

	rightInner := lipgloss.NewStyle().
		Width(rightContentW).
		Height(innerH).
		Align(lipgloss.Center, lipgloss.Center).
		Render("logs")

	leftPane := lipgloss.NewStyle().
		Width(leftW).
		Height(paneH).
		Border(lipgloss.NormalBorder()).
		BorderForeground(leftBorderColor).
		Render(leftInner)

	rightPane := lipgloss.NewStyle().
		Width(rightW).
		Height(paneH).
		Border(lipgloss.NormalBorder()).
		BorderForeground(rightBorderColor).
		Render(rightInner)

	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	km := combinedKeyMap{dashboard: d.keys, list: d.serviceList.KeyMap()}

	return panes + "\n" + d.help.View(km)
}
