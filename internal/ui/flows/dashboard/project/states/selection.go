package states

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
)

// dragSelection tracks an in-progress mouse drag selection.
type dragSelection struct {
	active    bool
	startX    int
	startY    int
	endY      int
	component SelectionComponent
}

// SelectionComponent identifies which component owns the active drag.
type SelectionComponent int

const (
	// SelectionNone indicates no component is selected.
	SelectionNone SelectionComponent = iota
	// SelectionServiceList indicates the service list pane is selected.
	SelectionServiceList
	// SelectionInspector indicates the inspector pane is selected.
	SelectionInspector
	// SelectionFooter indicates the footer bar is selected.
	SelectionFooter
)

// rows returns the [min, max] row range covered by the selection (inclusive).
func (s dragSelection) rows() (int, int) {
	if s.startY <= s.endY {
		return s.startY, s.endY
	}

	return s.endY, s.startY
}

// extractText strips ANSI escapes and trailing whitespace from lines[minRow..maxRow]
// and joins them with newlines. bounds is in the same coordinate space as row indices
// (origin 0 = top of the component's own View() output).
func extractText(lines []string, minRow, maxRow int, bounds Rect) string {
	var sb strings.Builder

	for row := minRow; row <= maxRow; row++ {
		if row < bounds.Y || row >= bounds.Y+bounds.H {
			continue
		}

		if row >= len(lines) {
			break
		}

		plain := strings.TrimRight(ansi.Strip(lines[row]), " ")
		sb.WriteString(plain)

		if row < maxRow {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}

// DragCoordinator manages mouse drag-to-select state and hit-testing.
// Layout and component views are passed at call time; nothing external is
// stored on the type.
type DragCoordinator struct {
	zm         *zone.Manager
	drag       dragSelection
	lastPressX int
	lastPressY int
}

// newDragCoordinator returns a DragCoordinator with all fields explicitly
// initialised to their zero values. Used by NewDashboard to satisfy exhaustruct.
func newDragCoordinator(zm *zone.Manager) DragCoordinator {
	return DragCoordinator{
		zm: zm,
		drag: dragSelection{
			active:    false,
			startX:    0,
			startY:    0,
			endY:      0,
			component: SelectionNone,
		},
		lastPressX: 0,
		lastPressY: 0,
	}
}

// DragActive reports whether a drag selection is currently in progress.
func (dc *DragCoordinator) DragActive() bool {
	return dc.drag.active
}

// DragComponent returns the component that owns the active drag.
func (dc *DragCoordinator) DragComponent() SelectionComponent {
	return dc.drag.component
}

// LastPressX returns the X coordinate of the most recent mouse press.
func (dc *DragCoordinator) LastPressX() int {
	return dc.lastPressX
}

// LastPressY returns the Y coordinate of the most recent mouse press.
func (dc *DragCoordinator) LastPressY() int {
	return dc.lastPressY
}

// SetLastPress records a press position directly. Used by tests to set up state
// without going through HandleClick.
func (dc *DragCoordinator) SetLastPress(x, y int) {
	dc.lastPressX = x
	dc.lastPressY = y
}

// SetActiveDrag sets the drag state directly. Used by tests to set up an
// in-progress drag without replaying mouse events.
func (dc *DragCoordinator) SetActiveDrag(
	active bool,
	startX, startY, endY int,
	component SelectionComponent,
) {
	dc.drag = dragSelection{
		active:    active,
		startX:    startX,
		startY:    startY,
		endY:      endY,
		component: component,
	}
}

// SetZoneManager sets the zone manager used for hit-testing. Used by tests to
// inject a fake zone manager without going through newDragCoordinator.
func (dc *DragCoordinator) SetZoneManager(zm *zone.Manager) {
	dc.zm = zm
}

// HandleClick records the press position and clears any active drag.
func (dc *DragCoordinator) HandleClick(msg tea.MouseClickMsg) {
	dc.lastPressX = msg.X
	dc.lastPressY = msg.Y
	dc.drag = dragSelection{
		active:    false,
		startX:    0,
		startY:    0,
		endY:      0,
		component: SelectionNone,
	}
}

// HandleMotion returns true when Update must short-circuit — while a drag is
// active the inspector and service list must not receive the motion event.
func (dc *DragCoordinator) HandleMotion(msg tea.MouseMotionMsg, layout PaneLayout) bool {
	if msg.Button != tea.MouseLeft {
		return false
	}

	dx := msg.X - dc.lastPressX
	dy := msg.Y - dc.lastPressY

	if dc.drag.active {
		b := dc.boundsForComponent(dc.drag.component, layout)
		dc.drag.endY = clamp(msg.Y, b.Y, b.Y+b.H-1)

		return true
	}

	if abs(dx) <= 1 && abs(dy) <= 1 {
		return false
	}

	comp := dc.hitTestComponent(dc.lastPressX, dc.lastPressY, layout)
	if comp == SelectionNone {
		return false
	}

	dc.drag = dragSelection{
		active:    true,
		startX:    dc.lastPressX,
		startY:    dc.lastPressY,
		endY:      msg.Y,
		component: comp,
	}

	return true
}

// HandleRelease returns (selectedText, handled). text is "" if the drag
// produced no selection. handled is true whenever Update must short-circuit.
func (dc *DragCoordinator) HandleRelease(
	msg tea.MouseReleaseMsg,
	layout PaneLayout,
	listView, inspView, footerView string,
) (string, bool) {
	if msg.Button == tea.MouseLeft && dc.drag.active {
		text := dc.extractSelection(layout, listView, inspView, footerView)
		dc.drag = dragSelection{
			active:    false,
			startX:    0,
			startY:    0,
			endY:      0,
			component: SelectionNone,
		}

		return text, true
	}

	dc.drag = dragSelection{
		active:    false,
		startX:    0,
		startY:    0,
		endY:      0,
		component: SelectionNone,
	}

	return "", false
}

// ApplyHighlight applies reverse-video highlight to dragged rows in the
// fully-rendered output.
func (dc *DragCoordinator) ApplyHighlight(rendered string, layout PaneLayout) string {
	lines := strings.Split(rendered, "\n")
	minRow, maxRow := dc.drag.rows()
	b := dc.boundsForComponent(dc.drag.component, layout)
	highlight := lipgloss.NewStyle().Reverse(true)

	for row := minRow; row <= maxRow; row++ {
		if row < b.Y || row >= b.Y+b.H || row >= len(lines) {
			continue
		}

		lines[row] = highlight.Render(ansi.Strip(lines[row]))
	}

	return strings.Join(lines, "\n")
}

// Active reports whether a drag is currently in progress.
func (dc *DragCoordinator) Active() bool {
	return dc.drag.active
}

// hitTestComponentByCoords uses coordinate-based boundary checking (used by tests without zones).
func (dc *DragCoordinator) hitTestComponentByCoords(
	x, y int,
	layout PaneLayout,
) SelectionComponent {
	lb := layout.ServiceListBounds()
	if x >= lb.X && x < lb.X+lb.W && y >= lb.Y && y < lb.Y+lb.H {
		return SelectionServiceList
	}

	rb := layout.LogViewBounds()
	if x >= rb.X && x < rb.X+rb.W && y >= rb.Y && y < rb.Y+rb.H {
		return SelectionInspector
	}

	paneH := layout.h - separatorRows - helpBarHeight
	if y == paneH {
		return SelectionFooter
	}

	return SelectionNone
}

func (dc *DragCoordinator) hitTestComponent(x, y int, layout PaneLayout) SelectionComponent {
	// If zone manager is available, use zone-based hit-testing.
	if dc.zm == nil {
		return dc.hitTestComponentByCoords(x, y, layout)
	}

	if dc.zm.Get("pane-left").
		InBounds(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseNone, Mod: 0}) {
		return SelectionServiceList
	}

	if dc.zm.Get("pane-right").
		InBounds(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseNone, Mod: 0}) {
		return SelectionInspector
	}

	paneH := layout.h - separatorRows - helpBarHeight
	if y == paneH {
		return SelectionFooter
	}

	return SelectionNone
}

func (dc *DragCoordinator) boundsForComponent(c SelectionComponent, layout PaneLayout) Rect {
	switch c {
	case SelectionServiceList:
		return layout.ServiceListBounds()
	case SelectionInspector:
		return layout.LogViewBounds()
	case SelectionFooter:
		paneH := layout.h - separatorRows - helpBarHeight

		return Rect{X: 0, Y: paneH, W: layout.w, H: 1}
	case SelectionNone:
		return Rect{}
	}

	return Rect{}
}

// extractSelection uses each component's own View() output to avoid x-range
// slicing across split-pane terminal rows.
func (dc *DragCoordinator) extractSelection(
	layout PaneLayout,
	listView, inspView, footerView string,
) string {
	b := dc.boundsForComponent(dc.drag.component, layout)
	minRow, maxRow := dc.drag.rows()
	localMin := minRow - b.Y
	localMax := maxRow - b.Y

	var source string

	switch dc.drag.component {
	case SelectionServiceList:
		source = listView
	case SelectionInspector:
		source = inspView
	case SelectionFooter:
		source = footerView
	case SelectionNone:
		return ""
	}

	lines := strings.Split(source, "\n")

	return extractText(lines, localMin, localMax, Rect{X: 0, Y: 0, W: b.W, H: b.H})
}
