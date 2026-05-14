// Package servicelist2 implements a service list component that returns tea.View
// from its View method.
package servicelist2

import (
	"fmt"
	"path/filepath"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/hoverlist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// serviceItem is a single entry in the list component.
type serviceItem struct {
	def            domain.ServiceDef
	runtime        *domain.ServiceRuntimeData
	actionInFlight bool
	actionLabel    string
	actionError    string
	displayTitle   string
}

func (s serviceItem) Title() string {
	return s.displayTitle
}

func (s serviceItem) Description() string {
	if s.runtime == nil {
		return "—"
	}

	return string(s.runtime.State)
}

func (s serviceItem) FilterValue() string { return s.def.Name }

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
			displayTitle:   Build(svc.Name, rt, false, "", "", th),
		}
	}

	return items
}

// Model is the service list component. It is a value type; all mutating
// methods return a new Model.
type Model struct {
	list         list.Model
	delegate     hoverlist.Delegate
	theme        *theme.Theme
	zm           *zone.Manager
	lastSelected string
	runtimes     map[string]*domain.ServiceRuntimeData
}

// New returns a Model pre-loaded with the given project's services.
func New(project *domain.Project, th *theme.Theme, zm *zone.Manager, w, h int) Model {
	base := list.NewDefaultDelegate()
	base.SetSpacing(0)
	base.ShowDescription = false
	hd := hoverlist.NewDelegate(base, th, zm)

	l := list.New(toItems(project.Services, nil, th), hd, w, h)
	l.DisableQuitKeybindings()
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(true)
	l.InfiniteScrolling = true
	l.Styles.TitleBar = l.Styles.TitleBar.PaddingBottom(0).PaddingLeft(0)
	l.Title = filepath.Base(project.File)

	return Model{
		list:         l,
		delegate:     hd,
		theme:        th,
		zm:           zm,
		lastSelected: "",
		runtimes:     nil,
	}
}

// SetBounds propagates new terminal position and dimensions to the inner list.
func (m Model) SetBounds(_, _, w, h int) Model {
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
			si.displayTitle = Build(
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
			si.displayTitle = Build(si.def.Name, si.runtime, true, label, "", m.theme)
			items[i] = si

			break
		}
	}

	m.list.SetItems(items)

	return m
}

// SetActionSuccess clears action state and applies an optimistic ServiceState.
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

			si.displayTitle = Build(si.def.Name, si.runtime, false, "", "", m.theme)
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
			si.displayTitle = Build(si.def.Name, si.runtime, false, "", errMsg, m.theme)
			items[i] = si

			break
		}
	}

	m.list.SetItems(items)

	return m
}

// SelectedEffectiveState returns state info for the currently selected item.
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
func (m Model) hitTest(mouseX, mouseY int) (int, bool) {
	for i := range m.list.VisibleItems() {
		msg := tea.MouseClickMsg{X: mouseX, Y: mouseY, Button: tea.MouseNone, Mod: 0}
		if m.zm.Get(fmt.Sprintf("item-%d", i)).InBounds(msg) {
			return i, true
		}
	}

	return 0, false
}

// Update delegates to the inner list and emits msgs.ServiceSelected when the
// cursor moves to a different service.
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
func (m Model) View() tea.View {
	m.list.Styles.Title = m.theme.ServiceListTitle

	return tea.NewView(m.list.View())
}
