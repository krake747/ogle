package fileselect

import (
	"fmt"
	"path/filepath"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

// fileItem is a single entry in the list component.
type fileItem struct{ path string }

func (f fileItem) Title() string       { return filepath.Base(f.path) }
func (f fileItem) Description() string { return f.path }
func (f fileItem) FilterValue() string { return filepath.Base(f.path) }

// Model is the fileselect view. It is a value type; all mutating methods
// return a new Model.
type Model struct {
	list     list.Model
	parseErr error
	errFile  string // basename of the file that produced a parse error
	parsing  bool
	files    []string // kept for cursor-clamp and error-clear logic in SetFiles
}

func toItems(files []string) []list.Item {
	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = fileItem{path: f}
	}

	return items
}

// New returns a Model pre-loaded with the given file paths. files must be
// non-empty; callers should not construct a fileselect model with 0 files.
func New(files []string, width, height int) Model {
	l := list.New(toItems(files), list.NewDefaultDelegate(), width, height)
	l.Title = "ogle"
	l.SetFilteringEnabled(false)
	l.KeyMap.ForceQuit.SetEnabled(false)

	//nolint:exhaustruct // list.Model has many fields, but only a few are relevant to us
	return Model{
		list:  l,
		files: files,
	}
}

// SetFiles refreshes the list. If the previously-errored file is no longer
// present the error is cleared.
func (m Model) SetFiles(files []string) Model {
	m.files = files
	m.list.SetItems(toItems(files))

	if m.errFile != "" {
		found := false

		for _, f := range files {
			if filepath.Base(f) == m.errFile {
				found = true

				break
			}
		}

		if !found {
			m.parseErr = nil
			m.errFile = ""
		}
	}

	return m
}

// SetError surfaces a parse-failure notice in the list status bar. path is the
// absolute path of the file that failed.
func (m Model) SetError(path string, err error) Model {
	m.parseErr = err
	m.errFile = filepath.Base(path)
	m.list.NewStatusMessage(fmt.Sprintf("notice: %s could not be parsed: %v", m.errFile, err))

	return m
}

// SetParsing sets the parsing indicator. When true, a "Parsing..." notice is
// shown in the list status bar.
func (m Model) SetParsing(v bool) Model {
	m.parsing = v
	if v {
		m.list.NewStatusMessage("Parsing...")
	}

	return m
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles keyboard navigation, mouse clicks, and selection.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.list.SetSize(sz.Width, sz.Height)

		return m, nil
	}

	m.list, cmd = m.list.Update(msg)

	var emit func() tea.Msg

	switch msg.(type) {
	case tea.MouseReleaseMsg, tea.KeyPressMsg:
		if item, ok := m.list.SelectedItem().(fileItem); ok {
			// Only emit on enter key, not on arbitrary mouse releases.
			if _, isKey := msg.(tea.MouseReleaseMsg); isKey {
				emit = func() tea.Msg { return msgs.FileSelected{Path: item.path} }
			}

			if kp, isKey := msg.(tea.KeyPressMsg); isKey && kp.String() == "enter" {
				emit = func() tea.Msg { return msgs.FileSelected{Path: item.path} }
			}
		}
	}

	if emit != nil {
		return m, tea.Batch(cmd, emit)
	}

	return m, cmd
}

// View renders the Project Selector screen.
func (m Model) View() string {
	return m.list.View()
}
