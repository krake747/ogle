package fileselect

import (
	"fmt"
	"path/filepath"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/hoverlist"
)

// headerRows is the number of terminal rows occupied by the list header.
// fileselect shows a title bar and a status bar; both are always shown.
// Each default style has Padding(0,0,1,2) — 1 text row + 1 bottom-padding row
// each — so 2 + 2 = 4.
const headerRows = 4

// itemHeight is the number of rows one file item occupies (title + description).
const itemHeight = 2

// rowStride is itemHeight plus the default delegate spacing of 1.
const rowStride = 3

type fileItem struct{ path string }

func (f fileItem) Title() string       { return filepath.Base(f.path) }
func (f fileItem) Description() string { return f.path }
func (f fileItem) FilterValue() string { return filepath.Base(f.path) }

// Model is the fileselect view. It is a value type; all mutating methods
// return a new Model.
type Model struct {
	list     list.Model
	delegate hoverlist.Delegate
	layout   hoverlist.Layout
	parseErr error
	errFile  string // basename of the file that produced a parse error
	parsing  bool
	files    []string // kept for cursor-clamp and error-clear logic in SetFiles
	w, h     int
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
	hd := hoverlist.NewDelegate(list.NewDefaultDelegate())

	l := list.New(toItems(files), hd, width, height)
	l.Title = "ogle"
	l.SetFilteringEnabled(false)
	l.KeyMap.ForceQuit.SetEnabled(false)

	//nolint:exhaustruct // list.Model has many fields, but only a few are relevant to us
	return Model{
		list:     l,
		delegate: hd,
		layout:   hoverlist.Layout{HeaderRows: headerRows, ItemHeight: itemHeight, RowStride: rowStride, Width: width},
		files:    files,
		w:        width,
		h:        height,
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

// SetBounds sets the terminal position and dimensions of this view. fileselect
// is always fullscreen so x and y will typically be zero.
func (m Model) SetBounds(x, y, w, h int) Model {
	m.layout.OriginX = x
	m.layout.OriginY = y
	m.layout.Width = w
	m.w = w
	m.h = h
	m.list.SetSize(w, h)

	return m
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

// Update handles keyboard navigation, mouse clicks, and selection.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.w = sz.Width
		m.h = sz.Height
		m.layout.Width = sz.Width
		m.list.SetSize(sz.Width, sz.Height)

		return m, nil
	}

	m.list, cmd = m.list.Update(msg)

	var emit func() tea.Msg

	switch msg := msg.(type) {
	case tea.MouseMotionMsg:
		newHover := -1

		if idx, ok := m.hitTest(msg.X, msg.Y); ok {
			newHover = idx
		}

		m.delegate.SetHover(newHover)

	case tea.MouseReleaseMsg:
		if msg.Button == tea.MouseLeft {
			if idx, ok := m.hitTest(msg.X, msg.Y); ok {
				if item, isFile := m.list.VisibleItems()[idx].(fileItem); isFile {
					emit = func() tea.Msg { return msgs.FileSelected{Path: item.path} }
				}
			}
		}

	case tea.KeyPressMsg:
		if item, ok := m.list.SelectedItem().(fileItem); ok {
			if msg.String() == "enter" {
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
