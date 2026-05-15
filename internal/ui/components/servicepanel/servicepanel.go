// Package servicepanel provides a compositor-manager component that renders
// one tile per project service as a stacked list of bordered boxes in the
// right-pane content area of the dashboard.
package servicepanel

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	tileHeight  = 4
	borderWidth = 2
)

type tile struct {
	name string
}

// Model manages a set of per-service tiles and renders them as a vertical
// stack of fixed-height bordered boxes.
type Model struct {
	tiles []tile
	theme *theme.Theme
	w, h  int
}

// New constructs a Model with one tile per project service.
func New(project *domain.Project, th *theme.Theme, w, h int) Model {
	tiles := make([]tile, len(project.Services))
	for i, svc := range project.Services {
		tiles[i] = tile{name: svc.Name}
	}

	return Model{
		tiles: tiles,
		theme: th,
		w:     w,
		h:     h,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles tea.WindowSizeMsg for dimension changes.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if wm, ok := msg.(tea.WindowSizeMsg); ok {
		m.w = wm.Width
		m.h = wm.Height
	}

	return m, nil
}

// View renders all tiles as a vertical stack of bordered boxes.
func (m Model) View() string {
	if m.w == 0 || len(m.tiles) == 0 {
		return ""
	}

	innerW := m.w - borderWidth
	borderStyle := lipgloss.NewStyle().
		Width(innerW).
		Height(tileHeight - borderWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.StateMuted)

	var rows []string

	for _, t := range m.tiles {
		content := lipgloss.NewStyle().Width(innerW).Render(t.name)
		rows = append(rows, borderStyle.Render(content))
	}

	return lipgloss.JoinVertical(lipgloss.Top, rows...)
}
