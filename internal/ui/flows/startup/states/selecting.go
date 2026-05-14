package states

import (
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/views/fileselect"
)

// Selecting is the state rendered when two or more valid compose files are
// found. The fileselect view (Project Selector) is active.
type Selecting struct {
	model   fileselect.Model
	handler fileHandler
	zm      *zone.Manager
}

// withError returns a copy of s with an error set on the underlying view for path.
func (s Selecting) withError(path string, err error) Selecting {
	return Selecting{model: s.model.SetError(path, err), handler: s.handler, zm: s.zm}
}

// withParsing returns a copy of s with the parsing indicator set.
func (s Selecting) withParsing(v bool) Selecting {
	return Selecting{model: s.model.SetParsing(v), handler: s.handler, zm: s.zm}
}

// Init implements tea.Model.
func (s Selecting) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (s Selecting) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.FileAvailabilityChanged:
		switch valid := validateFiles(msg.Files, s.handler.parser); len(valid) {
		case 0, 1:
			return s.handler.handle(valid, s)
		default:
			return Selecting{model: s.model.SetFiles(valid), handler: s.handler, zm: s.zm}, nil
		}

	case msgs.FileSelected:
		parse := ParseCmd(msg.Path, s.handler.parser)
		display := s.withParsing(true)

		return Parsing{path: msg.Path, parse: parse, display: display}, parse

	default:
		updated, cmd := s.model.Update(msg)

		return Selecting{model: updated, handler: s.handler, zm: s.zm}, cmd
	}
}

// View implements tea.Model.
func (s Selecting) View() tea.View { return tea.NewView(s.model.View()) }
