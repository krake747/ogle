// Package servicelist implements the service list component for the Dashboard's
// left pane. It renders a navigable, filterable list of Services declared in
// the current Project.
package servicelist

import (
	"path/filepath"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/hoverlist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// headerRows is the number of terminal rows occupied by the list header.
// servicelist shows a title bar only; status bar and help are disabled.
const headerRows = 1

// serviceItem is a single entry in the list component.
type serviceItem struct {
	def            domain.ServiceDef
	runtime        *domain.ServiceRuntimeData
	actionInFlight bool
	actionLabel    string // e.g. "stopping…"
	actionError    string // e.g. "stop failed"
	displayTitle   string // precomputed ANSI-styled string; returned by Title()
}

func (s serviceItem) Title() string { return s.displayTitle }

func (s serviceItem) Description() string {
	if s.runtime == nil {
		return "—"
	}

	return string(s.runtime.State)
}

func (s serviceItem) FilterValue() string { return s.def.Name }

// buildTitle computes the ANSI-styled display title for a service list item.
// It picks the icon and colour from the state-icon table, appends action labels
// or error suffixes as needed, and pre-renders the result.
func buildTitle(
	name string,
	rt *domain.ServiceRuntimeData,
	inFlight bool,
	actionLabel, actionError string,
	th *theme.Theme,
) string {
	icon := "●"
	colour := th.StateMuted

	switch {
	case inFlight:
		icon = "◌"
		colour = th.StateTransient
	case rt == nil:
		// icon and colour already set to defaults above
	default:
		switch rt.State {
		case domain.ServiceStateRunning:
			icon = "●"
			colour = th.StateRunning
		case domain.ServiceStateExited, domain.ServiceStateDead:
			icon = "●"
			colour = th.StateExited
		case domain.ServiceStateNotCreated:
			icon = "○"
		case domain.ServiceStatePaused:
			icon = "●"
			colour = th.StatePaused
		case domain.ServiceStateRestarting:
			icon = "●"
			colour = th.StateTransient
		case domain.ServiceStateUnknown:
			icon = "●"
		}
	}

	rendered := lipgloss.NewStyle().Foreground(colour).Render(icon) + " " + name

	if inFlight && actionLabel != "" {
		rendered += "  " + actionLabel
	}

	if !inFlight && actionError != "" {
		rendered += "  " + lipgloss.NewStyle().Foreground(th.ActionError).Render(actionError)
	}

	return rendered
}

func toItems(
	services []domain.ServiceDef,
	runtimes map[string]*domain.ServiceRuntimeData,
	th *theme.Theme,
) []list.Item {
	items := make([]list.Item, len(services))
	for i, svc := range services {
		rt := runtimes[svc.Name]
		items[i] = serviceItem{
			def:            svc,
			runtime:        rt,
			actionInFlight: false,
			actionLabel:    "",
			actionError:    "",
			displayTitle:   buildTitle(svc.Name, rt, false, "", "", th),
		}
	}

	return items
}

// Model is the service list component. It is a value type; all mutating
// methods return a new Model.
type Model struct {
	list         list.Model
	delegate     hoverlist.Delegate
	layout       hoverlist.Layout
	theme        *theme.Theme
	zm           *zone.Manager
	lastSelected string
	runtimes     map[string]*domain.ServiceRuntimeData
}

// New returns a Model pre-loaded with the given project's services.
func New(project *domain.Project, th *theme.Theme, zm *zone.Manager, w, h int) Model {
	base := list.NewDefaultDelegate()
	base.ShowDescription = false
	base.SetSpacing(0)
	hd := hoverlist.NewDelegate(base, th, zm)

	l := list.New(toItems(project.Services, nil, th), hd, w, h)
	l.Title = filepath.Base(project.File)
	l.SetShowTitle(true)
	l.Styles.TitleBar = l.Styles.TitleBar.PaddingBottom(0).PaddingLeft(0)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()
	l.InfiniteScrolling = true

	//nolint:exhaustruct // lastSelected intentionally zero — no selection on construction
	return Model{
		list:     l,
		delegate: hd,
		layout:   hoverlist.Layout{HeaderRows: headerRows, ItemHeight: 1, RowStride: 1, Width: w},
		theme:    th,
		zm:       zm,
	}
}

// SetBounds propagates new terminal position and dimensions to the inner list.
func (m Model) SetBounds(x, y, w, h int) Model {
	m.layout.OriginX = x
	m.layout.OriginY = y
	m.layout.Width = w
	m.list.SetSize(w, h)

	return m
}

// SetProject replaces the service items and updates the title. Called on Live Reload.
func (m Model) SetProject(project *domain.Project) Model {
	m.list.SetItems(toItems(project.Services, m.runtimes, m.theme))
	m.list.Title = filepath.Base(project.File)
	m.lastSelected = ""

	return m
}

// SetRuntimes updates the service state data shown in each item's description.
// Called after each State Poll cycle. A nil map resets all descriptions to "—".
// Preserves any in-flight action state on each item.
func (m Model) SetRuntimes(runtimes map[string]*domain.ServiceRuntimeData) Model {
	m.runtimes = runtimes

	items := m.list.Items()
	for i, item := range items {
		if si, ok := item.(serviceItem); ok {
			si.runtime = runtimes[si.def.Name]
			si.displayTitle = buildTitle(
				si.def.Name,
				si.runtime,
				si.actionInFlight,
				si.actionLabel,
				si.actionError,
				m.theme,
			)
			items[i] = si
		}
	}

	m.list.SetItems(items)

	return m
}

// SetActionInFlight marks the named service as in-flight and rebuilds its displayTitle.
func (m Model) SetActionInFlight(name, label string) Model {
	items := m.list.Items()
	for i, item := range items {
		if si, ok := item.(serviceItem); ok && si.def.Name == name {
			si.actionInFlight = true
			si.actionLabel = label
			si.actionError = ""
			si.displayTitle = buildTitle(si.def.Name, si.runtime, true, label, "", m.theme)
			items[i] = si

			break
		}
	}

	m.list.SetItems(items)

	return m
}

// SetActionSuccess clears action state and applies an optimistic ServiceState.
// If runtime is nil a minimal ServiceRuntimeData is created with the optimistic state.
func (m Model) SetActionSuccess(name string, optimisticState domain.ServiceState) Model {
	items := m.list.Items()
	for i, item := range items {
		if si, ok := item.(serviceItem); ok && si.def.Name == name {
			si.actionInFlight = false
			si.actionLabel = ""
			si.actionError = ""

			if si.runtime == nil {
				si.runtime = &domain.ServiceRuntimeData{
					ContainerID: "",
					State:       optimisticState,
					Health:      "",
					StateAge:    0,
				}
			} else {
				rt := *si.runtime
				rt.State = optimisticState
				si.runtime = &rt
			}

			si.displayTitle = buildTitle(si.def.Name, si.runtime, false, "", "", m.theme)
			items[i] = si

			break
		}
	}

	m.list.SetItems(items)

	return m
}

// SetActionError clears in-flight state and sets an error suffix on the named service.
func (m Model) SetActionError(name, errMsg string) Model {
	items := m.list.Items()
	for i, item := range items {
		if si, ok := item.(serviceItem); ok && si.def.Name == name {
			si.actionInFlight = false
			si.actionLabel = ""
			si.actionError = errMsg
			si.displayTitle = buildTitle(si.def.Name, si.runtime, false, "", errMsg, m.theme)
			items[i] = si

			break
		}
	}

	m.list.SetItems(items)

	return m
}

// SelectedEffectiveState returns state info for the currently selected item.
// hasState is false when runtime is nil and no optimistic state has been applied.
func (m Model) SelectedEffectiveState() (domain.ServiceState, bool, bool) {
	si, ok := m.list.SelectedItem().(serviceItem)
	if !ok {
		return "", false, false
	}

	if si.actionInFlight {
		return "", false, true
	}

	if si.runtime == nil {
		return "", false, false
	}

	return si.runtime.State, true, false
}

// KeyMap returns the component's key map so Dashboard can merge it into the
// help bar.
func (m Model) KeyMap() list.KeyMap {
	return m.list.KeyMap
}

// IsFiltering reports whether the inner list is currently in filter-input mode.
func (m Model) IsFiltering() bool {
	return m.list.FilterState() == list.Filtering
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// hitTest maps absolute terminal coordinates to a visible-item index.
// Returns (index, true) when the cursor is over a valid item row; (0, false) otherwise.
func (m Model) hitTest(mouseX, mouseY int) (int, bool) {
	return m.layout.HitTest(
		mouseX, mouseY,
		m.list.Paginator.Page*m.list.Paginator.PerPage,
		len(m.list.VisibleItems()),
	)
}

// Update delegates to the inner list and emits msgs.ServiceSelected when the
// cursor moves to a different service. Mouse motion updates the hover highlight;
// a left click moves the cursor to the clicked item.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)

	switch msg := msg.(type) {
	case tea.MouseMotionMsg:
		idx, ok := m.hitTest(msg.X, msg.Y)
		if !ok {
			idx = -1
		}

		m.delegate.SetHover(idx)

	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft {
			if idx, ok := m.hitTest(msg.X, msg.Y); ok {
				m.list.Select(idx)
			}
		}
	}

	selected, ok := m.list.SelectedItem().(serviceItem)
	if !ok {
		// Filter produced no matches or list is empty — no emission.
		return m, cmd
	}

	if selected.def.Name == m.lastSelected {
		return m, cmd
	}

	m.lastSelected = selected.def.Name

	emit := func() tea.Msg {
		return msgs.ServiceSelected{Service: selected.def}
	}

	return m, tea.Batch(cmd, emit)
}

// View renders the service list.
func (m Model) View() string {
	m.list.Styles.Title = m.theme.ServiceListTitle

	return m.list.View()
}
