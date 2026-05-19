// Package servicelist implements a service list component.
// from its View method.
package servicelist

import (
	"fmt"
	"slices"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/hoverlist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const doubleClickThreshold = 350 * time.Millisecond

//nolint:gochecknoglobals // package-level key bindings
var (
	_ help.KeyMap = Model{} //nolint:exhaustruct // compile-time assertion that Model implements help.KeyMap

	// KeyToggleService starts or stops the selected service.
	KeyToggleService = key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "start/stop"))
	// KeyRestart is the key binding for restarting a service.
	KeyRestart = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "restart"))
	// KeyRebuild is the key binding for rebuilding a service.
	KeyRebuild = key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "rebuild"))
	// KeyPrev moves selection to the previous service.
	KeyPrev = key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "previous"))
	// KeyNext moves selection to the next service.
	KeyNext = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "next"))
)

const (
	offsetY      = 2
	listRatio    = 30
	listMaxWidth = 80
	pctDivisor   = 100
)

// Model is the service list component. It is a value type; all mutating
// methods return a new Model.
type Model struct {
	list          list.Model
	delegate      hoverlist.Delegate
	theme         *theme.Theme
	zm            *zone.Manager
	lastSelected  string
	lastClickTime time.Time
	lastClickIdx  int
}

// New returns a Model pre-loaded with the given project's services.
func New(project *domain.Project, th *theme.Theme, zm *zone.Manager, w int) Model {
	base := list.NewDefaultDelegate()
	base.SetSpacing(0)
	base.ShowDescription = false
	hd := hoverlist.NewDelegate(base, th, zm)

	items := make([]list.Item, len(project.Services))
	for i, svc := range project.Services {
		items[i] = newServiceItem(svc, th)
	}

	listW := min(w*listRatio/pctDivisor, listMaxWidth)
	l := list.New(items, hd, listW, len(items)+offsetY)
	l.DisableQuitKeybindings()
	l.KeyMap.ShowFullHelp.SetEnabled(false)
	l.KeyMap.CloseFullHelp.SetEnabled(false)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetShowStatusBar(false)
	l.SetShowTitle(false)
	l.InfiniteScrolling = true

	return Model{
		list:          l,
		delegate:      hd,
		theme:         th,
		zm:            zm,
		lastSelected:  "",
		lastClickTime: time.Time{},
		lastClickIdx:  -1,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) hitTest(mouseX, mouseY int) (int, bool) {
	for i := range m.list.Items() {
		msg := tea.MouseClickMsg{X: mouseX, Y: mouseY, Button: tea.MouseNone, Mod: 0}
		if m.zm.Get(fmt.Sprintf("item-%d", i)).InBounds(msg) {
			return i, true
		}
	}

	return 0, false
}

func (m Model) handleMouseMotion(msg tea.MouseMotionMsg) (Model, tea.Cmd) {
	idx, hit := m.hitTest(msg.X, msg.Y)
	if hit {
		m.delegate.SetHover(idx)
	} else {
		m.delegate.SetHover(-1)
	}

	return m, nil
}

func (m Model) handleMouseClick(msg tea.MouseClickMsg) (Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}

	idx, hit := m.hitTest(msg.X, msg.Y)
	if !hit {
		return m, nil
	}

	m.list.Select(idx)

	name := m.selectedName()
	if name == "" {
		return m, nil
	}

	var cmds []tea.Cmd

	if name != m.lastSelected {
		m.lastSelected = name

		cmds = append(cmds, func() tea.Msg {
			return msgs.ServiceSelected{ServiceName: name}
		})
	}

	if msg.Mod.Contains(tea.ModShift) {
		m = m.updateItem(name, msgs.ServiceRebuild{ServiceName: name})

		return m, tea.Batch(append(cmds, func() tea.Msg {
			return msgs.ServiceRebuild{ServiceName: name}
		})...)
	}

	now := time.Now()
	if idx == m.lastClickIdx && now.Sub(m.lastClickTime) < doubleClickThreshold {
		m.lastClickTime = time.Time{}
		m.lastClickIdx = -1

		rt := m.selectedRuntime(name)
		if rt != nil && rt.State == domain.ServiceStateRunning {
			m = m.updateItem(name, msgs.ServiceStop{ServiceName: name})

			return m, tea.Batch(append(cmds, func() tea.Msg {
				return msgs.ServiceStop{ServiceName: name}
			})...)
		}

		m = m.updateItem(name, msgs.ServiceStart{ServiceName: name})

		return m, tea.Batch(append(cmds, func() tea.Msg {
			return msgs.ServiceStart{ServiceName: name}
		})...)
	}

	m.lastClickTime = now
	m.lastClickIdx = idx

	return m, tea.Batch(cmds...)
}

// Update delegates to the inner list and tracks the selected service.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(
			min(msg.Width*listRatio/pctDivisor, listMaxWidth),
			len(m.list.Items())+offsetY,
		)

		return m, nil

	case msgs.ServicesPolled:
		if msg.Err != nil {
			break
		}

		m = m.updateAllItems(msg)

	case msgs.ServiceActionCompleted:
		m = m.updateItem(msg.ServiceName, msg)

	case tea.KeyPressMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		name := m.selectedName()
		if name == "" {
			break
		}

		switch {
		case key.Matches(msg, KeyToggleService):
			rt := m.selectedRuntime(name)
			if rt != nil && rt.State == domain.ServiceStateRunning {
				m = m.updateItem(name, msgs.ServiceStop{ServiceName: name})

				return m, func() tea.Msg { return msgs.ServiceStop{ServiceName: name} }
			}

			m = m.updateItem(name, msgs.ServiceStart{ServiceName: name})

			return m, func() tea.Msg { return msgs.ServiceStart{ServiceName: name} }
		case key.Matches(msg, KeyRestart):
			m = m.updateItem(name, msgs.ServiceRestart{ServiceName: name})

			return m, func() tea.Msg { return msgs.ServiceRestart{ServiceName: name} }
		case key.Matches(msg, KeyRebuild):
			m = m.updateItem(name, msgs.ServiceRebuild{ServiceName: name})

			return m, func() tea.Msg { return msgs.ServiceRebuild{ServiceName: name} }
		case key.Matches(msg, KeyPrev), key.Matches(msg, KeyNext):
			m = m.moveCursor(msg)
		}

	case tea.MouseMotionMsg:
		return m.handleMouseMotion(msg)

	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)
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

// moveCursor moves selection by one service. No-op at list boundaries.
func (m Model) moveCursor(msg tea.KeyPressMsg) Model {
	if key.Matches(msg, KeyPrev) && m.list.Index() > 0 {
		m.list.Select(m.list.Index() - 1)
	}

	if key.Matches(msg, KeyNext) && m.list.Index() < len(m.list.Items())-1 {
		m.list.Select(m.list.Index() + 1)
	}

	return m
}

// selectedName returns the name of the currently selected service, or "".
func (m Model) selectedName() string {
	selected, ok := m.list.SelectedItem().(serviceItem)
	if !ok {
		return ""
	}

	return selected.ServiceName()
}

func (m Model) selectedRuntime(name string) *domain.ServiceRuntimeData {
	items := m.list.Items()
	for _, item := range items {
		it, ok := item.(serviceItem)
		if !ok || it.ServiceName() != name {
			continue
		}

		return it.runtime
	}

	return nil
}

func (m Model) updateItem(name string, msg tea.Msg) Model {
	items := m.list.Items()
	for i, item := range items {
		it, ok := item.(serviceItem)
		if !ok || it.ServiceName() != name {
			continue
		}

		it, _ = it.Update(msg)
		items[i] = it
	}

	m.list.SetItems(items)

	return m
}

func (m Model) updateAllItems(msg tea.Msg) Model {
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

	return m
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

		if slices.Contains(helpKeys, "k") || slices.Contains(helpKeys, "j") {
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

// View renders the service list.
func (m Model) View() tea.View {
	m.list.Styles.Title = m.theme.ServiceListTitle

	return tea.NewView(m.list.View())
}
