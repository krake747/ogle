package fileselect

import (
	"fmt"
	"path/filepath"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/hoverlist"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

type fileItem struct{ path string }

func (f fileItem) Title() string       { return filepath.Base(f.path) }
func (f fileItem) Description() string { return f.path }
func (f fileItem) FilterValue() string { return filepath.Base(f.path) }

// Model is the fileselect view. It is a value type; all mutating methods
// return a new Model.
type Model struct {
	list       list.Model
	delegate   hoverlist.Delegate
	parseErr   error
	errFile    string // basename of the file that produced a parse error
	parsing    bool
	files      []string // kept for cursor-clamp and error-clear logic in SetFiles
	zm         *zone.Manager
	w, h       int
	pressedIdx int // index of the item pressed on MouseClick; -1 if none
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
func New(files []string, th *theme.Theme, zm *zone.Manager, width, height int) Model {
	hd := hoverlist.NewDelegate(list.NewDefaultDelegate(), th, zm)

	l := list.New(toItems(files), hd, width, height)
	l.Title = "ogle"
	l.SetFilteringEnabled(false)
	l.KeyMap.ForceQuit.SetEnabled(false)
	l.InfiniteScrolling = true

	//nolint:exhaustruct // list.Model has many fields, but only a few are relevant to us
	return Model{
		list:       l,
		delegate:   hd,
		files:      files,
		zm:         zm,
		w:          width,
		h:          height,
		pressedIdx: -1,
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
func (m Model) SetBounds(_, _, w, h int) Model {
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
	for i := range m.list.VisibleItems() {
		msg := tea.MouseClickMsg{X: mouseX, Y: mouseY, Button: tea.MouseNone, Mod: 0}
		if m.zm.Get(fmt.Sprintf("item-%d", i)).InBounds(msg) {
			return i, true
		}
	}

	return 0, false
}

// handleMouseRelease processes mouse release events for file selection.
// Returns (updated model, command) if a file was selected; otherwise returns
// the original model and command with pressedIdx cleared.
func (m Model) handleMouseRelease(msg tea.MouseReleaseMsg, cmd tea.Cmd) (Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, cmd
	}

	idx, ok := m.hitTest(msg.X, msg.Y)
	if ok && idx == m.pressedIdx && m.pressedIdx >= 0 {
		if item, isFile := m.list.VisibleItems()[idx].(fileItem); isFile {
			m.pressedIdx = -1

			return m, tea.Batch(
				cmd,
				func() tea.Msg { return msgs.FileSelected{Path: item.path} },
			)
		}
	}

	m.pressedIdx = -1

	return m, cmd
}

// Update handles keyboard navigation, mouse clicks, and selection.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	// WindowSizeMsg is handled before m.list.Update to avoid double-resizing:
	// bubbles list.Update also handles this message internally. We call
	// SetSize directly and return early so the list never sees the message.
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		m.w = sz.Width
		m.h = sz.Height
		m.list.SetSize(sz.Width, sz.Height)

		return m, nil
	}

	m.list, cmd = m.list.Update(msg)

	switch msg := msg.(type) {
	case tea.MouseMotionMsg:
		idx, ok := m.hitTest(msg.X, msg.Y)
		if !ok {
			idx = -1
		}

		m.delegate.SetHover(idx)

	case tea.MouseClickMsg:
		if msg.Button == tea.MouseLeft {
			if idx, ok := m.hitTest(msg.X, msg.Y); ok {
				m.pressedIdx = idx
			} else {
				m.pressedIdx = -1
			}
		}

	case tea.MouseReleaseMsg:
		return m.handleMouseRelease(msg, cmd)

	case tea.KeyPressMsg:
		if item, ok := m.list.SelectedItem().(fileItem); ok {
			if msg.String() == "enter" {
				return m, tea.Batch(
					cmd,
					func() tea.Msg { return msgs.FileSelected{Path: item.path} },
				)
			}
		}
	}

	return m, cmd
}

// View renders the Project Selector screen.
func (m Model) View() string {
	return m.list.View()
}
