// Package servicelist2 implements a service list component.
// from its View method.
package servicelist2

import (
	"path/filepath"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/hoverlist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	listRatio    = 30
	listMaxWidth = 80
	pctDivisor   = 100
	frameChrome  = 2
)

// ListWidth returns the allocated width for the service list
// based on the total window width.
func ListWidth(totalW int) int {
	return min(totalW*listRatio/pctDivisor, listMaxWidth)
}

// Model is the service list component. It is a value type; all mutating
// methods return a new Model.
type Model struct {
	list         list.Model
	delegate     hoverlist.Delegate
	theme        *theme.Theme
	lastSelected string
}

// New returns a Model pre-loaded with the given project's services.
func New(project *domain.Project, th *theme.Theme, zm *zone.Manager, w, h int) Model {
	base := list.NewDefaultDelegate()
	base.SetSpacing(0)
	base.ShowDescription = false
	hd := hoverlist.NewDelegate(base, th, zm)

	items := make([]list.Item, len(project.Services))
	for i, svc := range project.Services {
		items[i] = newServiceItem(svc, th)
	}

	l := list.New(items, hd, w, h)
	l.DisableQuitKeybindings()
	l.KeyMap.ShowFullHelp.SetEnabled(false)
	l.KeyMap.CloseFullHelp.SetEnabled(false)
	l.SetFilteringEnabled(false)
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
		lastSelected: "",
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update delegates to the inner list and tracks the selected service.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(ListWidth(msg.Width), msg.Height-frameChrome)

		return m, nil

	case msgs.ServicesPolled:
		if msg.Err != nil {
			break
		}

		items := m.list.Items()
		for i, item := range items {
			it, isItem := item.(serviceItem)
			if !isItem {
				continue
			}

			it, _ = it.Update(msg)
			items[i] = it
		}

		m.list.SetItems(items)
	}

	m.list, cmd = m.list.Update(msg)

	selected, ok := m.list.SelectedItem().(serviceItem)
	if !ok {
		return m, cmd
	}

	if selected.def.Name == m.lastSelected {
		return m, cmd
	}

	m.lastSelected = selected.def.Name

	return m, tea.Batch(cmd, func() tea.Msg {
		return msgs.ServiceSelected{ServiceName: selected.def.Name}
	})
}

// ShortHelp returns the inner list's short help bindings, excluding the help
// toggle. Implements help.KeyMap.
func (m Model) ShortHelp() []key.Binding {
	all := m.list.ShortHelp()
	out := make([]key.Binding, 0, len(all))

	for _, b := range all {
		helpKeys := b.Keys()
		if len(helpKeys) == 1 && helpKeys[0] == "?" {
			continue
		}

		if b.Enabled() {
			out = append(out, b)
		}
	}

	return out
}

// FullHelp returns the inner list's full help bindings. Implements help.KeyMap.
func (m Model) FullHelp() [][]key.Binding {
	return m.list.FullHelp()
}

// IsFiltering reports whether the inner list is currently in filter-input mode.
func (m Model) IsFiltering() bool {
	return m.list.FilterState() == list.Filtering
}

// View renders the service list.
func (m Model) View() string {
	m.list.Styles.Title = m.theme.ServiceListTitle

	return m.list.View()
}
