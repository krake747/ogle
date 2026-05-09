// Package fileselect provides the file-picker view: displayed during startup
// when two or more valid compose files are found in the watched directory.
package fileselect

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

type state int

const (
	stateBrowsing state = iota // navigating the list
	stateError                 // last confirmed selection failed to parse
)

// Model is the fileselect view. It is a value type; all mutating methods
// return a new Model.
type Model struct {
	files    []string // absolute paths of valid compose files
	cursor   int
	state    state
	parseErr error
	errFile  string // basename of the file that failed to parse
}

// New returns a Model pre-loaded with the given file paths. files must be
// non-empty; callers should not construct a fileselect model with 0 files.
func New(files []string) Model {
	return Model{
		files:    files,
		cursor:   0,
		state:    stateBrowsing,
		parseErr: nil,
		errFile:  "",
	}
}

// SetFiles refreshes the list. The cursor is clamped to the new last index if
// it would otherwise be out of bounds. If the previously-errored file is no
// longer in the list the error is cleared.
func (m Model) SetFiles(files []string) Model {
	m.files = files
	if m.cursor >= len(files) && len(files) > 0 {
		m.cursor = len(files) - 1
	}
	// Clear error if the offending file is gone.
	if m.state == stateError {
		found := false

		for _, f := range files {
			if filepath.Base(f) == m.errFile {
				found = true

				break
			}
		}

		if !found {
			m.state = stateBrowsing
			m.parseErr = nil
			m.errFile = ""
		}
	}

	return m
}

// SetError enters stateError with an inline parse-failure notice. path is the
// absolute path of the file that failed.
func (m Model) SetError(path string, err error) Model {
	m.state = stateError
	m.parseErr = err
	m.errFile = filepath.Base(path)

	return m
}

// Init satisfies tea.Model. The fileselect view has no startup commands.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles keyboard navigation and selection.
//   - ↑ / k   move cursor up
//   - ↓ / j   move cursor down
//   - enter    emit msgs.FileSelected for the highlighted file
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.files)-1 {
			m.cursor++
		}
	case "enter":
		if len(m.files) > 0 {
			selected := m.files[m.cursor]

			return m, func() tea.Msg { return msgs.FileSelected{Path: selected} }
		}
	}

	return m, nil
}

// View renders the fileselect screen. All output fits within 80 columns.
func (m Model) View() string {
	var sb strings.Builder

	sb.WriteString("ogle\n\n")
	sb.WriteString("Multiple compose files found. Select one:\n\n")

	for i, f := range m.files {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		fmt.Fprintf(&sb, "  %s%s\n", cursor, filepath.Base(f))
	}

	if m.state == stateError {
		fmt.Fprintf(&sb, "\nnotice: %s could not be parsed: %v\n",
			m.errFile, m.parseErr)
	}

	sb.WriteString("\n↑/↓ navigate   enter select   ctrl+c quit\n")

	return sb.String()
}
