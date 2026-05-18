package startup

import (
	"context"
	"log/slog"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/ui/components/fileselect"
)

// Model is the startup flow orchestrator.
type Model struct {
	parser     parser.Parser
	fileSelect tea.Model
}

// New constructs a startup Model.
func New(
	ctx context.Context,
	logger *slog.Logger,
	w, h int,
) Model {
	return Model{
		parser:     parser.New(ctx, logger),
		fileSelect: fileselect.New(nil, w, h),
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(msgs.FileSelected); ok {
		p, err := m.parser.Parse(msg.Path)
		if err != nil {
			return m, nil
		}

		return m, tea.Batch(func() tea.Msg {
			return msgs.ProjectLoaded{
				Project: p,
			}
		})
	}

	var cmd tea.Cmd

	m.fileSelect, cmd = m.fileSelect.Update(msg)

	return m, cmd
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView(m.fileSelect.View().Content)
}
