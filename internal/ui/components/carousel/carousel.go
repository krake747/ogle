package carousel

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/paginator"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/carousel/card"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	rows                 = 2
	cols                 = 3
	pageSize             = rows * cols
	chevronCount         = 2
	listRatio            = 30
	listMinTermWidth     = 80
	pctDivisor           = 100
	maxCardH             = 12
	terminalCellAspect   = 2
	doubleClickThreshold = 350 * time.Millisecond
	zonePrev             = "carousel-prev"
	zoneNext             = "carousel-next"
	zoneDotFmt           = "carousel-dot-%d"
	zoneCardFmt          = "carousel-card-%d"
)

// Model is the carousel component state.
type Model struct {
	all           []domain.ServiceDef
	cards         []card.Model
	w, h          int
	focus         int
	hovered       int
	hoveredDot    int
	paginator     paginator.Model
	th            *theme.Theme
	zm            *zone.Manager
	runtimeData   map[string]*domain.ServiceRuntimeData
	lastClickTime time.Time
	lastClickIdx  int
}

// New returns a Model for the given project.
func New(project *domain.Project, w, h int, th *theme.Theme, zm *zone.Manager) Model {
	p := paginator.New(paginator.WithPerPage(pageSize))
	p.Type = paginator.Dots
	p.SetTotalPages(len(project.Services))
	p.KeyMap = paginator.KeyMap{
		PrevPage: KeyPgUp,
		NextPage: KeyPgDown,
	}
	p.ActiveDot = lipgloss.NewStyle().Foreground(th.CarouselFocused).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(th.CarouselBlurred).Render("○")

	_, end := p.GetSliceBounds(len(project.Services))
	n := end - p.Page*p.PerPage

	cards := make([]card.Model, n)
	for i := range n {
		cards[i] = card.New(project.Services[p.Page*p.PerPage+i], w, h, th)
	}

	focus := 0

	if n > 0 {
		focus = 1
		cards[0], _ = cards[0].Update(card.FocusMsg{})
	}

	return Model{
		all:           project.Services,
		cards:         cards,
		w:             w,
		h:             h,
		focus:         focus,
		hovered:       -1,
		hoveredDot:    -1,
		paginator:     p,
		th:            th,
		zm:            zm,
		runtimeData:   nil,
		lastClickTime: time.Time{},
		lastClickIdx:  -1,
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
		m.paginator.ActiveDot = lipgloss.NewStyle().Foreground(m.th.CarouselFocused).Render("•")
		m.paginator.InactiveDot = lipgloss.NewStyle().Foreground(m.th.CarouselBlurred).Render("○")

	case msgs.ServicesPolled:
		if msg.Err == nil {
			m.runtimeData = msg.Runtimes
		}

	case tea.MouseClickMsg:
		return m.handleMouseClick(msg)

	case tea.MouseMotionMsg:
		return m.handleMouseMotion(msg)
	}

	for i := range m.cards {
		updated, cmd := m.cards[i].Update(msg)
		m.cards[i] = updated

		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleKeyPress(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if key.Matches(msg, KeyTab) {
		return m.handleTab()
	}

	if key.Matches(msg, KeyEnter) {
		return m.handleEnter()
	}

	prevPage := m.paginator.Page

	var pageCmd tea.Cmd

	m.paginator, pageCmd = m.paginator.Update(msg)

	if m.paginator.Page != prevPage {
		return m.rebuildCards()
	}

	return m, pageCmd
}

func (m Model) handleTab() (Model, tea.Cmd) {
	prevFocus := m.focus
	total := m.totalSlots()
	m.focus = (m.focus + 1) % total

	// skip empty grid slots that have no card
	for m.focus >= 1 && m.focus <= pageSize && !m.slotHasCard(m.focus) {
		m.focus = (m.focus + 1) % total
	}

	if prevFocus >= 1 && prevFocus <= pageSize && m.slotHasCard(prevFocus) {
		idx := prevFocus - 1
		updated, _ := m.cards[idx].Update(card.BlurMsg{})
		m.cards[idx] = updated
	}

	if m.focus >= 1 && m.focus <= pageSize && m.slotHasCard(m.focus) {
		idx := m.focus - 1
		updated, cmd := m.cards[idx].Update(card.FocusMsg{})
		m.cards[idx] = updated

		return m, tea.Batch(cmd, func() tea.Msg {
			return msgs.ServiceSelected{ServiceName: m.cardServiceName(idx)}
		})
	}

	return m, nil
}

func (m Model) handleEnter() (Model, tea.Cmd) {
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

	// enter on an empty slot is a no-op
	if m.focus >= 1 && m.focus <= pageSize && !m.slotHasCard(m.focus) {
		return m, nil
	}

	// enter on a card toggles start/stop
	if m.focus >= 1 && m.focus <= pageSize {
		idx := m.focus - 1
		name := m.cardServiceName(idx)

		return m, m.toggleServiceCmd(name)
	}

	return m, nil
}

func (m Model) handleMouseClick(msg tea.MouseClickMsg) (Model, tea.Cmd) {
	switch {
	case m.zm.Get(zonePrev).InBounds(msg):
		if m.paginator.OnFirstPage() {
			return m, nil
		}

		m.paginator.PrevPage()

		return m.rebuildCards()

	case m.zm.Get(zoneNext).InBounds(msg):
		if m.paginator.OnLastPage() {
			return m, nil
		}

		m.paginator.NextPage()

		return m.rebuildCards()
	}

	for i := range m.paginator.TotalPages {
		if m.zm.Get(fmt.Sprintf(zoneDotFmt, i)).InBounds(msg) {
			if m.paginator.Page == i {
				return m, nil
			}

			m.paginator.Page = i

			return m.rebuildCards()
		}
	}

	for i := range m.cards {
		if !m.zm.Get(fmt.Sprintf(zoneCardFmt, i)).InBounds(msg) {
			continue
		}

		return m.handleCardClick(i)
	}

	return m, nil
}

func (m Model) handleCardClick(i int) (Model, tea.Cmd) {
	newFocus := i + 1

	if m.focus == newFocus {
		if i == m.lastClickIdx && time.Since(m.lastClickTime) < doubleClickThreshold {
			return m, m.toggleServiceCmd(m.cardServiceName(i))
		}

		m.lastClickTime = time.Now()
		m.lastClickIdx = i

		return m, nil
	}

	if m.focus >= 1 && m.focus <= pageSize {
		idx := m.focus - 1
		updated, _ := m.cards[idx].Update(card.BlurMsg{})
		m.cards[idx] = updated
	}

	m.focus = newFocus
	m.lastClickTime = time.Now()
	m.lastClickIdx = i

	updated, focusCmd := m.cards[i].Update(card.FocusMsg{})
	m.cards[i] = updated

	return m, tea.Batch(focusCmd, func() tea.Msg {
		return msgs.ServiceSelected{ServiceName: m.cardServiceName(i)}
	})
}

func (m Model) handleMouseMotion(msg tea.MouseMotionMsg) (Model, tea.Cmd) {
	hit := -1

	switch {
	case m.zm.Get(zonePrev).InBounds(msg):
		hit = m.totalSlots() - 1

	case m.zm.Get(zoneNext).InBounds(msg):
		hit = 0

	default:
		for i := range m.paginator.TotalPages {
			if m.zm.Get(fmt.Sprintf(zoneDotFmt, i)).InBounds(msg) {
				if m.hoveredDot == i {
					return m, nil
				}

				m.hoveredDot = i
				m = m.unhoverCard()
				m.hovered = -1

				return m, nil
			}
		}

		for i := range m.cards {
			if m.zm.Get(fmt.Sprintf(zoneCardFmt, i)).InBounds(msg) {
				hit = i + 1

				break
			}
		}
	}

	if m.hoveredDot >= 0 {
		m.hoveredDot = -1
	}

	if hit == m.hovered {
		return m, nil
	}

	m = m.unhoverCard()
	m.hovered = hit

	if m.hovered >= 1 && m.hovered <= pageSize {
		idx := m.hovered - 1
		updated, cmd := m.cards[idx].Update(card.HoverMsg{})
		m.cards[idx] = updated

		return m, cmd
	}

	return m, nil
}

func (m Model) unhoverCard() Model {
	if m.hovered >= 1 && m.hovered <= pageSize {
		idx := m.hovered - 1
		updated, _ := m.cards[idx].Update(card.UnhoverMsg{})
		m.cards[idx] = updated
	}

	return m
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
				cells[col] = m.zm.Mark(
					fmt.Sprintf(zoneCardFmt, idx),
					m.cards[idx].View().Content,
				)
			} else {
				innerW := cardW - card.BorderW
				innerH := cardH - card.BorderW
				emptyLabel := lipgloss.NewStyle().Foreground(m.th.Subtext).Render("-")
				padded := lipgloss.Place(
					innerW,
					innerH,
					lipgloss.Center,
					lipgloss.Center,
					emptyLabel,
				)
				cells[col] = lipgloss.NewStyle().
					Width(cardW).
					Height(cardH).
					Border(lipgloss.RoundedBorder()).
					BorderForeground(m.th.CarouselEmpty).
					BorderBackground(m.th.CarouselBackground).
					Background(m.th.CarouselBackground).
					Render(padded)
			}
		}

		rowStrs[row] = lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	}

	grid := lipgloss.JoinVertical(lipgloss.Left, rowStrs...)

	grid = lipgloss.NewStyle().
		Width(carouselW).
		Background(m.th.CarouselBackground).
		Render(grid)

	navBar := m.renderNavBar(carouselW)

	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, grid, navBar))
}

func (m Model) renderNavBar(carouselW int) string {
	if m.paginator.TotalPages <= 1 {
		return ""
	}

	focusedFg := m.th.CarouselFocused
	unfocusedFg := m.th.CarouselBlurred
	hoverFg := m.th.CarouselHover
	navBg := m.th.CarouselNavBackground

	rightChevronColour := unfocusedFg
	if m.focus == 0 {
		rightChevronColour = focusedFg
	} else if m.hovered == 0 {
		rightChevronColour = hoverFg
	}

	leftChevronColour := unfocusedFg
	if m.focus == m.totalSlots()-1 {
		leftChevronColour = focusedFg
	} else if m.hovered == m.totalSlots()-1 {
		leftChevronColour = hoverFg
	}

	leftChevron := lipgloss.NewStyle().
		Foreground(leftChevronColour).
		Background(navBg).
		Render("◀")

	rightChevron := lipgloss.NewStyle().
		Foreground(rightChevronColour).
		Background(navBg).
		Render("▶")

	totalPages := m.paginator.TotalPages
	dots := make([]string, totalPages)

	for i := range totalPages {
		dotChar := "○"
		dotColour := unfocusedFg

		switch i {
		case m.paginator.Page:
			dotChar = "•"
			dotColour = focusedFg
		case m.hoveredDot:
			dotColour = hoverFg
		}

		dots[i] = m.zm.Mark(
			fmt.Sprintf(zoneDotFmt, i),
			lipgloss.NewStyle().
				Foreground(dotColour).
				Background(m.th.CarouselBackground).
				Render(dotChar),
		)
	}

	paginatorView := strings.Join(dots, "")

	navContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		m.zm.Mark(zonePrev, leftChevron),
		paginatorView,
		m.zm.Mark(zoneNext, rightChevron),
	)

	return lipgloss.NewStyle().
		Width(carouselW).
		Align(lipgloss.Center).
		Background(navBg).
		Render(navContent)
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
		m.hovered = -1
		m.hoveredDot = -1
		m.lastClickTime = time.Time{}
		m.lastClickIdx = -1

		return m, nil
	}

	m.focus = 1
	m.hovered = -1
	m.hoveredDot = -1
	m.lastClickTime = time.Time{}
	m.lastClickIdx = -1

	if len(m.cards) > 0 {
		updated, cmd := m.cards[0].Update(card.FocusMsg{})
		m.cards[0] = updated

		return m, tea.Batch(cmd, func() tea.Msg {
			return msgs.ServiceSelected{ServiceName: m.all[start].Name}
		})
	}

	return m, nil
}

func (m Model) slotHasCard(slot int) bool {
	if slot < 1 || slot > pageSize {
		return false
	}

	return slot-1 < len(m.cards)
}

func (m Model) cardServiceName(idx int) string {
	start, _ := m.paginator.GetSliceBounds(len(m.all))

	return m.all[start+idx].Name
}

func (m Model) toggleServiceCmd(name string) tea.Cmd {
	rt := m.runtimeData[name]
	if rt != nil && rt.State == domain.ServiceStateRunning {
		return func() tea.Msg { return msgs.ServiceStop{ServiceName: name} }
	}

	return func() tea.Msg { return msgs.ServiceStart{ServiceName: name} }
}

func (m Model) totalSlots() int {
	if m.paginator.TotalPages > 1 {
		return pageSize + chevronCount
	}

	return pageSize
}
