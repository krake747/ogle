package states

import (
	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/views/fileselect"
)

// Selecting is the state rendered when two or more valid compose files are
// found. The fileselect view (Project Selector) is active.
//
// HandleFiles is injected by startup.makeHandleFiles so this state carries no
// ambient dir string.
type Selecting struct {
	Model       fileselect.Model
	HandleFiles func([]string, State) (State, tea.Cmd)
}

// Init has no startup command.
func (s Selecting) Init() tea.Cmd {
	return nil
}

// Update handles file availability changes, file selection confirmation, and
// forwards all other messages to the fileselect sub-model.
func (s Selecting) Update(msg tea.Msg) (State, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.FileAvailabilityChanged:
		switch valid := validateFiles(msg.Files); len(valid) {
		case 0, 1:
			return s.HandleFiles(valid, s)
		default:
			return Selecting{Model: s.Model.SetFiles(valid), HandleFiles: s.HandleFiles}, nil
		}

	case msgs.FileSelected:
		parse := ParseCmd(msg.Path)

		return Parsing{Path: msg.Path, Parse: parse, Display: s}, parse

	default:
		updated, cmd := s.Model.Update(msg)

		return Selecting{Model: updated, HandleFiles: s.HandleFiles}, cmd
	}
}

// View renders the fileselect screen.
func (s Selecting) View() string {
	return s.Model.View()
}
