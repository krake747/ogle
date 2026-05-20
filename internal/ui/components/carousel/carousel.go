package carousel

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
)

const (
	rows             = 2
	cols             = 2
	pageSize         = rows * cols
	chevronW         = 2
	totalSlots       = pageSize + 2
	chevronCount     = 2
	listRatio        = 30
	listMinTermWidth = 80
	pctDivisor       = 100
)

//nolint:gochecknoglobals // package-level key binding
var keyTab = key.NewBinding(key.WithKeys("tab"))

// Model is the carousel component state.
type Model struct {
	services []domain.ServiceDef
	w, h     int
	focus    int
}

// New returns a Model for the given project.
func New(project *domain.Project, w, h int) Model {
	return Model{
		services: project.Services,
		w:        w,
		h:        h,
		focus:    -1,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update handles key presses and window resize.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, keyTab) {
			m.focus = (m.focus + 1) % totalSlots
		}

	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height
	}

	return m, nil
}

// View renders the carousel with chevrons and card grid.
func (m Model) View() tea.View {
	carouselW := max(m.w, listMinTermWidth) * listRatio / pctDivisor
	gridW := carouselW - chevronW*chevronCount
	cardW := gridW / cols

	const (
		maxCardH           = 8
		terminalCellAspect = 2
	)

	cardH := min(cardW/terminalCellAspect, maxCardH)

	focusedFg := lipgloss.Color("#ffffff")
	unfocusedFg := lipgloss.Color("#444444")

	rowStrs := make([]string, rows)

	for row := range rows {
		cells := make([]string, cols)

		for col := range cols {
			idx := row*cols + col

			if idx < len(m.services) {
				borderColour := unfocusedFg
				if idx == m.focus {
					borderColour = focusedFg
				}

				cells[col] = lipgloss.NewStyle().
					Width(cardW).
					Height(cardH).
					Border(lipgloss.RoundedBorder()).
					BorderForeground(borderColour).
					Render("")
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

	leftStyle := lipgloss.NewStyle().Width(chevronW).Height(gridH).Align(lipgloss.Center)
	if m.focus == -1 {
		leftStyle = leftStyle.Foreground(focusedFg)
	} else {
		leftStyle = leftStyle.Foreground(unfocusedFg)
	}

	leftCol := leftStyle.Render("◀")

	rightStyle := lipgloss.NewStyle().Width(chevronW).Height(gridH).Align(lipgloss.Center)
	if m.focus == pageSize {
		rightStyle = rightStyle.Foreground(focusedFg)
	} else {
		rightStyle = rightStyle.Foreground(unfocusedFg)
	}

	rightCol := rightStyle.Render("▶")

	return tea.NewView(lipgloss.JoinHorizontal(lipgloss.Top, leftCol, grid, rightCol))
}
