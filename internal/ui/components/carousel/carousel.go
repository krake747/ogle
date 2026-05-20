package carousel

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/paginator"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/carousel/card"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	rows               = 2
	cols               = 2
	pageSize           = rows * cols
	chevronCount       = 2
	listRatio          = 30
	listMinTermWidth   = 80
	pctDivisor         = 100
	maxCardH           = 12
	terminalCellAspect = 2
)

//nolint:gochecknoglobals // package-level key bindings
var (
	keyTab   = key.NewBinding(key.WithKeys("tab"))
	keyEnter = key.NewBinding(key.WithKeys("enter"))
)

// Model is the carousel component state.
type Model struct {
	all       []domain.ServiceDef
	cards     []card.Model
	w, h      int
	focus     int
	paginator paginator.Model
	th        *theme.Theme
}

// New returns a Model for the given project.
func New(project *domain.Project, w, h int, th *theme.Theme) Model {
	p := paginator.New(paginator.WithPerPage(pageSize))
	p.Type = paginator.Dots
	p.SetTotalPages(len(project.Services))
	p.KeyMap = paginator.KeyMap{
		PrevPage: key.NewBinding(key.WithKeys("pgup")),
		NextPage: key.NewBinding(key.WithKeys("pgdown")),
	}

	_, end := p.GetSliceBounds(len(project.Services))
	n := end - p.Page*p.PerPage

	cards := make([]card.Model, n)
	for i := range n {
		cards[i] = card.New(project.Services[p.Page*p.PerPage+i], w, h, th)
	}

	focus := 0

	if n > 0 && p.TotalPages <= 1 {
		cards[0], _ = cards[0].Update(card.FocusMsg{})
	}

	return Model{
		all:       project.Services,
		cards:     cards,
		w:         w,
		h:         h,
		focus:     focus,
		paginator: p,
		th:        th,
	}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles key presses and window resize.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height

	case msgs.ThemeChanged:
		m.th = msg.Theme
	}

	for i := range m.cards {
		updated, cmd := m.cards[i].Update(msg)
		m.cards[i] = updated

		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if key.Matches(msg, keyTab) {
		prevFocus := m.focus
		m.focus = (m.focus + 1) % m.totalSlots()

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

	if key.Matches(msg, keyEnter) {
		switch m.focus {
		case 0:
			if m.paginator.OnLastPage() {
				return m, nil
			}

			m.paginator.NextPage()

			return m.rebuildCards()

		case m.totalSlots() - 1:
			if m.paginator.OnFirstPage() {
				return m, nil
			}

			m.paginator.PrevPage()

			return m.rebuildCards()
		}
	}

	prevPage := m.paginator.Page

	var pageCmd tea.Cmd

	m.paginator, pageCmd = m.paginator.Update(msg)

	if m.paginator.Page != prevPage {
		return m.rebuildCards()
	}

	return m, pageCmd
}

// View renders the carousel with card grid, and nav bar below.
func (m Model) View() tea.View {
	carouselW := max(m.w, listMinTermWidth) * listRatio / pctDivisor
	cardW := carouselW / cols
	cardH := min(cardW/terminalCellAspect, maxCardH)

	if cardH%2 == 0 {
		cardH--
	}

	cardH = max(cardH, 1)

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
					Background(m.th.CarouselBackground).
					Render("")
			}
		}

		rowStrs[row] = lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	}

	grid := lipgloss.JoinVertical(lipgloss.Left, rowStrs...)

	grid = lipgloss.NewStyle().
		Width(carouselW).
		Background(m.th.CarouselBackground).
		Render(grid)

	var navBar string

	if m.paginator.TotalPages > 1 {
		focusedFg := m.th.CarouselFocused
		unfocusedFg := m.th.CarouselBlurred
		navBg := m.th.CarouselNavBackground

		rightChevronColour := unfocusedFg
		if m.focus == 0 {
			rightChevronColour = focusedFg
		}

		leftChevronColour := unfocusedFg
		if m.focus == m.totalSlots()-1 {
			leftChevronColour = focusedFg
		}

		navContent := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Foreground(leftChevronColour).Background(navBg).Render("◀"),
			lipgloss.NewStyle().Background(navBg).Render(m.paginator.View()),
			lipgloss.NewStyle().Foreground(rightChevronColour).Background(navBg).Render("▶"),
		)

		navBar = lipgloss.NewStyle().
			Width(carouselW).
			Align(lipgloss.Center).
			Background(navBg).
			Render(navContent)
	}

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, grid, navBar))
}

func (m Model) rebuildCards() (Model, tea.Cmd) {
	start, end := m.paginator.GetSliceBounds(len(m.all))
	n := end - start
	m.cards = make([]card.Model, n)

	for i := range n {
		m.cards[i] = card.New(m.all[start+i], m.w, m.h, m.th)
	}

	if n == 0 {
		m.focus = 0

		return m, nil
	}

	m.focus = 0

	if m.paginator.TotalPages <= 1 && len(m.cards) > 0 {
		var cmd tea.Cmd

		m.cards[0], cmd = m.cards[0].Update(card.FocusMsg{})

		return m, cmd
	}

	return m, nil
}

func (m Model) totalSlots() int {
	if m.paginator.TotalPages > 1 {
		return pageSize + chevronCount
	}

	return pageSize
}
