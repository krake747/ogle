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
	HandleFiles func([]string, tea.Model) (tea.Model, tea.Cmd)
}

func (s Selecting) Init() tea.Cmd {
	return nil
}

func (s Selecting) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (s Selecting) View() tea.View { return tea.NewView(s.Model.View()) }
