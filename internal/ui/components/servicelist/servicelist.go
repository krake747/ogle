// Package servicelist implements the service list component for the Dashboard's
// left pane. It renders a navigable, filterable list of Services declared in
// the current Project.
package servicelist

import (
	"path/filepath"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/hoverlist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// headerRows is the number of terminal rows occupied by the list header.
// servicelist shows a title bar only; status bar and help are disabled.
const headerRows = 1

// serviceItem is a single entry in the list component.
type serviceItem struct{ def domain.ServiceDef }

func (s serviceItem) Title() string       { return s.def.Name }
func (s serviceItem) Description() string { return "" }
func (s serviceItem) FilterValue() string { return s.def.Name }

func toItems(services []domain.ServiceDef) []list.Item {
	items := make([]list.Item, len(services))
	for i, svc := range services {
		items[i] = serviceItem{def: svc}
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
	lastSelected string
}

// New returns a Model pre-loaded with the given project's services.
func New(project *domain.Project, th *theme.Theme, w, h int) Model {
	base := list.NewDefaultDelegate()
	base.ShowDescription = false
	base.SetSpacing(0)
	hd := hoverlist.NewDelegate(base, th)

	l := list.New(toItems(project.Services), hd, w, h)
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
	m.list.SetItems(toItems(project.Services))
	m.list.Title = filepath.Base(project.File)
	m.lastSelected = ""

	return m
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
