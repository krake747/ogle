package fileselect

import (
	"path/filepath"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

type fileItem struct{ path string }

func (f fileItem) Title() string       { return filepath.Base(f.path) }
func (f fileItem) Description() string { return f.path }
func (f fileItem) FilterValue() string { return filepath.Base(f.path) }

// Model is the file selection component state.
type Model struct {
	list list.Model
}

// New constructs a file selection model from the given files.
func New(
	files []string,
	w, h int,
) Model {
	items := make([]list.Item, 0, len(files))
	for _, f := range files {
		items = append(items, fileItem{path: f})
	}

	l := list.New(items, list.NewDefaultDelegate(), w, h)
	l.Title = "ogle"
	l.SetFilteringEnabled(false)
	l.KeyMap.ForceQuit.SetEnabled(false)
	l.InfiniteScrolling = true

	return Model{
		list: l,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if kp := msg.String(); kp == "enter" {
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
	case msgs.FileAvailabilityChanged:
		switch len(msg.Files) {
		// case 0:
		// 	output = "No compose files found. Waiting for files to appear..."
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
