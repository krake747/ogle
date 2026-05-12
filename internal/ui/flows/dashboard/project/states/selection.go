package states

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// dragSelection tracks an in-progress mouse drag selection.
type dragSelection struct {
	active    bool
	startX    int
	startY    int
	endY      int
	component selectionComponent
}

// selectionComponent identifies which component owns the active drag.
type selectionComponent int

const (
	selectionNone selectionComponent = iota
	selectionServiceList
	selectionInspector
	selectionFooter
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
func extractText(lines []string, minRow, maxRow int, bounds rect) string {
	var sb strings.Builder

	for row := minRow; row <= maxRow; row++ {
		if row < bounds.y || row >= bounds.y+bounds.h {
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
