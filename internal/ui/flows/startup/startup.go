package startup

import (
	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/ui/components/fileselect"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Model is the startup flow orchestrator.
type Model struct {
	parser     parser.Parser
	fileSelect tea.Model
	zm         *zone.Manager
	th         *theme.Theme
	w, h       int
}

// New constructs a startup Model.
func New(
	w, h int,
	zm *zone.Manager,
	th *theme.Theme,
	p parser.Parser,
) Model {
	return Model{
		parser:     p,
		fileSelect: fileselect.New(nil, w, h, zm, th),
		zm:         zm,
		th:         th,
		w:          w,
		h:          h,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		return msgs.BindingsMsg{Keymap: startupKeymap{}}
	}
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case msgs.FileSelected:
		p, err := m.parser.Parse(msg.Path)
		if err != nil {
			return m, nil
		}

		return m, tea.Batch(func() tea.Msg {
			return msgs.ProjectLoaded{
				Project: p,
			}
		})

	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height

		var cmd tea.Cmd

		m.fileSelect, cmd = m.fileSelect.Update(msg)

		return m, cmd
	}

	var cmd tea.Cmd

	m.fileSelect, cmd = m.fileSelect.Update(msg)

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView(m.fileSelect.View().Content)
}
