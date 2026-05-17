package startup

import (
	"context"
	"fmt"
	"log/slog"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/services/watcher"
	"github.com/ma-tf/ogle/internal/ui/components/fileselect"
)

// Model is the startup flow orchestrator.
type Model struct {
	watcher watcher.Watcher
	parser  parser.Parser

	fileSelect tea.Model
}

// New constructs a startup Model.
func New(
	ctx context.Context,
	logger *slog.Logger,
	dir string,
	w, h int,
) (Model, error) {
	watcher, err := watcher.New(dir, logger) // starts goroutine that fires FileAvailabilityChanged
	if err != nil {
		return Model{}, fmt.Errorf("watcher: %w", err)
	}

	return Model{
		watcher: watcher,
		parser:  parser.New(ctx, logger),

		fileSelect: fileselect.New(nil, w, h),
	}, nil
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.watcher.Snapshot()
}

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
