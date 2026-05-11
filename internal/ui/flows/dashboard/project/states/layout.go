package states

import "charm.land/lipgloss/v2"

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

type rect struct{ x, y, w, h int }

type paneLayout struct {
	mode layoutMode
	w, h int
}

func newPaneLayout() paneLayout {
	return paneLayout{}
}

func (p paneLayout) SetSize(w, h int) paneLayout {
	p.w = w
	p.h = h

	return p
}

func (p paneLayout) ToggleMode() paneLayout {
	if p.mode == modeSplit {
		p.mode = modeLogFullscreen
	} else {
		p.mode = modeSplit
	}

	return p
}

func (p paneLayout) IsLogFullscreen() bool {
	return p.mode == modeLogFullscreen
}

// ServiceListBounds returns the content area of the left pane in split mode,
// accounting for the border (x=1, y=1). Returns rect{} in log-fullscreen mode
// since the service list is not visible.
func (p paneLayout) ServiceListBounds() rect {
	if p.mode == modeLogFullscreen {
		return rect{}
	}

	leftW := min(p.w*servicePaneRatio/servicePaneRatioDen, servicePaneMaxW)
	leftContentW := max(leftW-borderWidth, 0)
	paneH := max(p.h-separatorRows-helpBarHeight, 0)
	innerH := max(paneH-borderHeight, 0)

	return rect{x: 1, y: 1, w: leftContentW, h: innerH}
}

// LogViewBounds returns the content area of the right pane. In split mode the
// x offset accounts for the left pane's outer width plus the right border. In
// log-fullscreen mode the pane spans the full terminal width.
func (p paneLayout) LogViewBounds() rect {
	paneH := max(p.h-separatorRows-helpBarHeight, 0)
	innerH := max(paneH-borderHeight, 0)

	if p.mode == modeLogFullscreen {
		contentW := max(p.w-borderWidth, 0)

		return rect{x: 1, y: 1, w: contentW, h: innerH}
	}

	leftW := min(p.w*servicePaneRatio/servicePaneRatioDen, servicePaneMaxW)
	rightW := p.w - leftW
	rightContentW := max(rightW-borderWidth, 0)

	return rect{x: leftW + 1, y: 1, w: rightContentW, h: innerH}
}

// View renders both panes with NormalBorder, applying the highlight colour to
// the focused pane and dimmed to the other, then joins them horizontally. In
// log-fullscreen mode only the right pane is rendered at full terminal width.
// The help bar is not rendered here — that remains in Dashboard.View.
func (p paneLayout) View(serviceListStr, logViewStr string, leftFocused bool) string {
	highlight := lipgloss.Color("62")
	dimmed := lipgloss.Color("240")

	paneH := max(p.h-separatorRows-helpBarHeight, 0)
	innerH := max(paneH-borderHeight, 0)

	rightBorderColor := dimmed
	if !leftFocused {
		rightBorderColor = highlight
	}

	if p.mode == modeLogFullscreen {
		contentW := max(p.w-borderWidth, 0)

		rightInner := lipgloss.NewStyle().
			Width(contentW).
			Height(innerH).
			Align(lipgloss.Center, lipgloss.Center).
			Render(logViewStr)

		return lipgloss.NewStyle().
			Width(p.w).
			Height(paneH).
			Border(lipgloss.NormalBorder()).
			BorderForeground(rightBorderColor).
			Render(rightInner)
	}

	leftW := min(p.w*servicePaneRatio/servicePaneRatioDen, servicePaneMaxW)
	rightW := p.w - leftW
	leftContentW := max(leftW-borderWidth, 0)
	rightContentW := max(rightW-borderWidth, 0)

	leftBorderColor := dimmed
	if leftFocused {
		leftBorderColor = highlight
	}

	leftInner := lipgloss.NewStyle().
		Width(leftContentW).
		Height(innerH).
		Render(serviceListStr)

	rightInner := lipgloss.NewStyle().
		Width(rightContentW).
		Height(innerH).
		Align(lipgloss.Center, lipgloss.Center).
		Render(logViewStr)

	leftPane := lipgloss.NewStyle().
		Width(leftW).
		Height(paneH).
		Border(lipgloss.NormalBorder()).
		BorderForeground(leftBorderColor).
		Render(leftInner)

	rightPane := lipgloss.NewStyle().
		Width(rightW).
		Height(paneH).
		Border(lipgloss.NormalBorder()).
		BorderForeground(rightBorderColor).
		Render(rightInner)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}
