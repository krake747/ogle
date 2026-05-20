package carousel

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/ui/components/carousel/card"
)

const (
	rows               = 2
	cols               = 2
	pageSize           = rows * cols
	chevronW           = 2
	totalSlots         = pageSize + 2
	chevronCount       = 2
	listRatio          = 30
	listMinTermWidth   = 80
	pctDivisor         = 100
	maxCardH           = 8
	terminalCellAspect = 2
)

//nolint:gochecknoglobals // package-level key binding
var keyTab = key.NewBinding(key.WithKeys("tab"))

// Model is the carousel component state.
type Model struct {
	cards []card.Model
	w, h  int
	focus int
}

// New returns a Model for the given project.
func New(project *domain.Project, w, h int) Model {
	cards := make([]card.Model, len(project.Services))
	for i, s := range project.Services {
		cards[i] = card.New(s, w, h)
	}

	return Model{
		cards: cards,
		w:     w,
		h:     h,
		focus: 0,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles key presses and window resize.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, keyTab) {
			prevFocus := m.focus
			m.focus = (m.focus + 1) % totalSlots

			if prevFocus >= 1 && prevFocus <= pageSize {
				idx := prevFocus - 1
				updated, _ := m.cards[idx].Update(card.BlurMsg{})
				m.cards[idx] = updated
			}

			if m.focus >= 1 && m.focus <= pageSize {
				idx := m.focus - 1
				updated, cmd := m.cards[idx].Update(card.FocusMsg{})
				m.cards[idx] = updated

				return m, cmd
			}

			return m, nil
		}

	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height

		var cmds []tea.Cmd

		for i := range m.cards {
			updated, cmd := m.cards[i].Update(msg)
			m.cards[i] = updated

			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)
	}

	return m, nil
}

// View renders the carousel with chevrons and card grid.
func (m Model) View() tea.View {
	carouselW := max(m.w, listMinTermWidth) * listRatio / pctDivisor
	gridW := carouselW - chevronW*chevronCount
	cardW := gridW / cols
	cardH := min(cardW/terminalCellAspect, maxCardH)

	rowStrs := make([]string, rows)

	for row := range rows {
		cells := make([]string, cols)

		for col := range cols {
			idx := row*cols + col

			if idx < len(m.cards) {
				cells[col] = m.cards[idx].View().Content
			} else {
				cells[col] = lipgloss.NewStyle().
					Width(cardW).
					Height(cardH).
					Render("")
			}
		}

		rowStrs[row] = lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	}

	grid := lipgloss.JoinVertical(lipgloss.Left, rowStrs...)
	gridH := lipgloss.Height(grid)

	return tea.NewView(lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(chevronW).Height(gridH).Align(lipgloss.Center).Render("◀"),
		grid,
		lipgloss.NewStyle().Width(chevronW).Height(gridH).Align(lipgloss.Center).Render("▶"),
	))
}
