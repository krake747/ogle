package project

import (
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	servicePaneRatio    = 30
	servicePaneRatioDen = 100
	servicePaneMaxW     = 80
	borderWidth         = 2
	borderHeight        = 2
	separatorRows       = 1
	helpBarHeight       = 1
)

type layoutMode int

const (
	modeSplit layoutMode = iota
	modeLogFullscreen
)

// Rect is an axis-aligned bounding rectangle in terminal cell coordinates.
type Rect struct{ X, Y, W, H int }

// PaneLayout holds the current terminal dimensions and split mode for the
// dashboard's two-pane layout.
type PaneLayout struct {
	mode  layoutMode
	theme *theme.Theme
	zm    *zone.Manager
	w, h  int
}

// NewPaneLayout returns a PaneLayout in split mode with no size set.
func NewPaneLayout(th *theme.Theme, zm *zone.Manager) PaneLayout {
	return PaneLayout{mode: modeSplit, theme: th, zm: zm, w: 0, h: 0}
}

// SetSize returns a copy of the layout with new terminal dimensions.
func (p PaneLayout) SetSize(w, h int) PaneLayout {
	p.w = w
	p.h = h

	return p
}

// ToggleMode switches between split and log-fullscreen modes.
func (p PaneLayout) ToggleMode() PaneLayout {
	if p.mode == modeSplit {
		p.mode = modeLogFullscreen
	} else {
		p.mode = modeSplit
	}

	return p
}

// IsLogFullscreen reports whether the layout is in log-fullscreen mode.
func (p PaneLayout) IsLogFullscreen() bool {
	return p.mode == modeLogFullscreen
}

// ServiceListBounds returns the content area of the left pane in split mode,
// accounting for the border (X=1, Y=1). Returns Rect{} in log-fullscreen mode
// since the service list is not visible.
func (p PaneLayout) ServiceListBounds() Rect {
	if p.mode == modeLogFullscreen {
		return Rect{}
	}

	leftW := min(p.w*servicePaneRatio/servicePaneRatioDen, servicePaneMaxW)
	leftContentW := max(leftW-borderWidth, 0)
	paneH := max(p.h-separatorRows-helpBarHeight, 0)
	innerH := max(paneH-borderHeight, 0)

	return Rect{X: 1, Y: 1, W: leftContentW, H: innerH}
}

// LogViewBounds returns the content area of the right pane. In split mode the
// X offset accounts for the left pane's outer width plus the right border. In
// log-fullscreen mode the pane spans the full terminal width.
func (p PaneLayout) LogViewBounds() Rect {
	paneH := max(p.h-separatorRows-helpBarHeight, 0)
	innerH := max(paneH-borderHeight, 0)

	if p.mode == modeLogFullscreen {
		contentW := max(p.w-borderWidth, 0)

		return Rect{X: 1, Y: 1, W: contentW, H: innerH}
	}

	leftW := min(p.w*servicePaneRatio/servicePaneRatioDen, servicePaneMaxW)
	rightW := p.w - leftW
	rightContentW := max(rightW-borderWidth, 0)

	return Rect{X: leftW + 1, Y: 1, W: rightContentW, H: innerH}
}

// View renders both panes with NormalBorder, applying the highlight colour to
// the focused pane and dimmed to the other, then joins them horizontally. In
// log-fullscreen mode only the right pane is rendered at full terminal width.
// The help bar is not rendered here — that remains in Dashboard.View.
func (p PaneLayout) View(serviceListStr, logViewStr string, leftFocused bool) string {
	paneH := max(p.h-separatorRows-helpBarHeight, 0)
	innerH := max(paneH-borderHeight, 0)

	rightBorderStyle := p.theme.BorderBlurred
	if !leftFocused {
		rightBorderStyle = p.theme.BorderFocused
	}

	if p.mode == modeLogFullscreen {
		contentW := max(p.w-borderWidth, 0)

		rightInner := lipgloss.NewStyle().
			Width(contentW).
			Height(innerH).
			Render(logViewStr)
		rightInner = p.zm.Mark("pane-right", rightInner)

		return rightBorderStyle.
			Width(p.w).
			Height(paneH).
			Render(rightInner)
	}

	leftW := min(p.w*servicePaneRatio/servicePaneRatioDen, servicePaneMaxW)
	rightW := p.w - leftW
	leftContentW := max(leftW-borderWidth, 0)
	rightContentW := max(rightW-borderWidth, 0)

	leftBorderStyle := p.theme.BorderBlurred
	if leftFocused {
		leftBorderStyle = p.theme.BorderFocused
	}

	leftInner := lipgloss.NewStyle().Width(leftContentW).Height(innerH).Render(serviceListStr)
	leftInner = p.zm.Mark("pane-left", leftInner)

	rightInner := lipgloss.NewStyle().
		Width(rightContentW).
		Height(innerH).
		Render(logViewStr)
	rightInner = p.zm.Mark("pane-right", rightInner)

	leftPane := leftBorderStyle.
		Width(leftW).
		Height(paneH).
		Render(leftInner)

	rightPane := rightBorderStyle.
		Width(rightW).
		Height(paneH).
		Render(rightInner)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}
