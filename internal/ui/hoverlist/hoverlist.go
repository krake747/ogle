// Package hoverlist provides shared hover-rendering and hit-test infrastructure
// for list components. It has no Bubble Tea model; it is list infrastructure,
// not a sub-model.
package hoverlist

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	zone "github.com/lrstanley/bubblezone/v2"

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
	zm         *zone.Manager
}

// NewDelegate returns a Delegate wrapping base with no item hovered.
func NewDelegate(base list.DefaultDelegate, th *theme.Theme, zm *zone.Manager) Delegate {
	return &delegate{DefaultDelegate: base, hoverIndex: -1, theme: th, zm: zm}
}

func (d *delegate) SetHover(index int) {
	d.hoverIndex = index
}

// Render implements list.ItemDelegate. It applies a background tint to the
// hovered item and delegates rendering to DefaultDelegate for all others.
func (d *delegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var buf strings.Builder

	if index == d.hoverIndex {
		dd := d.DefaultDelegate
		bg := d.theme.HoverBackground
		dd.Styles.NormalTitle = dd.Styles.NormalTitle.Background(bg)
		dd.Styles.NormalDesc = dd.Styles.NormalDesc.Background(bg)
		dd.Render(&buf, m, index, item)
	} else {
		d.DefaultDelegate.Render(&buf, m, index, item)
	}

	_, _ = io.WriteString(w, d.zm.Mark(fmt.Sprintf("item-%d", index), buf.String()))
}
