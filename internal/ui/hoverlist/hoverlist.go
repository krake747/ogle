// Package hoverlist provides shared hover-rendering and hit-test infrastructure
// for list components. It has no Bubble Tea model; it is list infrastructure,
// not a sub-model.
package hoverlist

import (
	"io"

	"charm.land/bubbles/v2/list"

	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Delegate extends list.ItemDelegate with hover state management.
// The concrete implementation is unexported; obtain one via NewDelegate.
type Delegate interface {
	list.ItemDelegate
	// SetHover updates the hovered VisibleItems index (-1 = none).
	SetHover(index int)
}

// delegate is the single unexported implementation of Delegate.
// All methods use pointer receivers — no mixed-receiver issue.
type delegate struct {
	list.DefaultDelegate

	hoverIndex int
	theme      *theme.Theme
}

// NewDelegate returns a Delegate wrapping base with no item hovered.
func NewDelegate(base list.DefaultDelegate, th *theme.Theme) Delegate {
	return &delegate{DefaultDelegate: base, hoverIndex: -1, theme: th}
}

func (d *delegate) SetHover(index int) {
	d.hoverIndex = index
}

// Render implements list.ItemDelegate. It applies a background tint to the
// hovered item and delegates rendering to DefaultDelegate for all others.
func (d *delegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	if index == d.hoverIndex {
		dd := d.DefaultDelegate
		bg := d.theme.HoverBackground
		dd.Styles.NormalTitle = dd.Styles.NormalTitle.Background(bg)
		dd.Styles.NormalDesc = dd.Styles.NormalDesc.Background(bg)
		dd.Render(w, m, index, item)

		return
	}

	d.DefaultDelegate.Render(w, m, index, item)
}

// Layout describes the static and positional geometry of a list component.
// OriginX, OriginY, and Width are updated on resize; the remaining fields
// are constants for a given component.
type Layout struct {
	OriginX    int
	OriginY    int
	Width      int
	HeaderRows int
	ItemHeight int
	RowStride  int
}

// HitTest maps absolute terminal coordinates to an index into list.VisibleItems().
// pageOffset is Paginator.Page*Paginator.PerPage; itemCount is len(list.VisibleItems()).
// Returns (index, true) on a hit; (0, false) on a miss or gap row.
func (l Layout) HitTest(mouseX, mouseY, pageOffset, itemCount int) (int, bool) {
	if mouseX < l.OriginX || mouseX >= l.OriginX+l.Width {
		return 0, false
	}

	localY := mouseY - (l.OriginY + l.HeaderRows)
	if localY < 0 {
		return 0, false
	}

	if localY%l.RowStride >= l.ItemHeight {
		return 0, false
	}

	localIndex := localY / l.RowStride
	globalIndex := pageOffset + localIndex

	if globalIndex >= itemCount {
		return 0, false
	}

	return globalIndex, true
}
