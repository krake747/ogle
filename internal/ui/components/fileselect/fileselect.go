package fileselect

import (
	"fmt"
	"path/filepath"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/hoverlist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

//nolint:gochecknoglobals // package-level key binding
var keySelect = key.NewBinding(key.WithKeys("enter"))

type fileItem struct{ path string }

func (f fileItem) Title() string       { return filepath.Base(f.path) }
func (f fileItem) Description() string { return f.path }
func (f fileItem) FilterValue() string { return filepath.Base(f.path) }

// Model is the file selection component state.
type Model struct {
	list     list.Model
	delegate hoverlist.Delegate
	zm       *zone.Manager
	th       *theme.Theme
}

// New constructs a file selection model from the given files.
func New(
	files []string,
	w, h int,
	zm *zone.Manager,
	th *theme.Theme,
) Model {
	items := make([]list.Item, 0, len(files))
	for _, f := range files {
		items = append(items, fileItem{path: f})
	}

	base := list.NewDefaultDelegate()
	hd := hoverlist.NewDelegate(base, th, zm)

	l := list.New(items, hd, w, h)
	l.Title = "ogle"
	l.SetFilteringEnabled(false)
	l.KeyMap.ForceQuit.SetEnabled(false)
	l.InfiniteScrolling = true

	return Model{
		list:     l,
		delegate: hd,
		zm:       zm,
		th:       th,
	}
}

// Init implements tea.Model.
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

func (m Model) handleMouseMotion(
	msg tea.MouseMotionMsg,
) (Model, tea.Cmd) { //nolint:unparam // interface consistency
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

	fi, isFileItem := m.list.SelectedItem().(fileItem)
	if !isFileItem {
		return m, nil
	}

	return m, func() tea.Msg {
		return msgs.FileSelected{Path: fi.path}
	}
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, keySelect) {
			if len(m.list.Items()) == 0 {
				return m, nil
			}

			fi, isFileItem := m.list.SelectedItem().(fileItem)
			if !isFileItem {
				return m, nil
			}

			return m, tea.Batch(func() tea.Msg {
				return msgs.FileSelected{Path: fi.path}
			})
		}

	case tea.MouseMotionMsg:
		return m.handleMouseMotion(msg)

	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)

	case msgs.FileAvailabilityChanged:
		switch len(msg.Files) {
		case 1:
			return m, tea.Batch(func() tea.Msg {
				return msgs.FileSelected{Path: msg.Files[0]}
			})

		default:
			items := make([]list.Item, 0, len(msg.Files))
			for _, f := range msg.Files {
				items = append(items, fileItem{path: f})
			}

			m.list.SetItems(items)
		}

		return m, nil
	}

	var cmd tea.Cmd

	m.list, cmd = m.list.Update(msg)

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView(m.list.View())
}
