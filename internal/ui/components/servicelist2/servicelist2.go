// Package servicelist2 implements a service list component that returns tea.View
// from its View method.
package servicelist2

import (
	"path/filepath"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/servicelist2/servicetitle"
	"github.com/ma-tf/ogle/internal/ui/hoverlist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is the service list component. It is a value type; all mutating
// methods return a new Model.
type Model struct {
	list     list.Model
	delegate hoverlist.Delegate
	theme    *theme.Theme
	// zm           *zone.Manager
	lastSelected string
	runtimes     map[string]*domain.ServiceRuntimeData
}

// New returns a Model pre-loaded with the given project's services.
func New(project *domain.Project, th *theme.Theme, zm *zone.Manager, w, h int) Model {
	base := list.NewDefaultDelegate()
	base.SetSpacing(0)
	base.ShowDescription = false
	hd := hoverlist.NewDelegate(base, th, zm)

	items := make([]list.Item, len(project.Services))
	for i, svc := range project.Services {
		items[i] = servicetitle.New(svc, th)
	}

	l := list.New(items, hd, w, h)
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
		list:     l,
		delegate: hd,
		theme:    th,
		// zm:           zm,
		lastSelected: "",
		runtimes:     nil,
	}
}

// SetBounds propagates new terminal position and dimensions to the inner list.
func (m Model) SetBounds(_, _, w, h int) Model {
	m.list.SetSize(w, h)

	return m
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

	selected, ok := m.list.SelectedItem().(servicetitle.Model)
	if !ok {
		return m, cmd
	}

	if selected.ServiceDef().Name == m.lastSelected {
		return m, cmd
	}

	m.lastSelected = selected.ServiceDef().Name

	return m, tea.Batch(cmd, func() tea.Msg {
		return msgs.ServiceSelected{Service: selected.ServiceDef()}
	})
}

// View renders the service list.
func (m Model) View() tea.View {
	m.list.Styles.Title = m.theme.ServiceListTitle

	return tea.NewView(m.list.View())
}
