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
)

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
	lastSelected string
}

// New returns a Model pre-loaded with the given project's services.
func New(project *domain.Project, w, h int) Model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)

	l := list.New(toItems(project.Services), delegate, w, h)
	l.Title = filepath.Base(project.File)
	l.SetShowTitle(true)
	l.Styles.TitleBar = l.Styles.TitleBar.PaddingBottom(0)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetShowPagination(false)
	l.DisableQuitKeybindings()

	//nolint:exhaustruct // lastSelected intentionally zero — no selection on construction
	return Model{list: l}
}

// SetSize propagates new terminal dimensions to the inner list.
func (m Model) SetSize(w, h int) Model {
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

// Update delegates to the inner list and emits msgs.ServiceSelected when the
// cursor moves to a different service.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)

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
	return m.list.View()
}
