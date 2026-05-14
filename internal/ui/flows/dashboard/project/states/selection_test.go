package states_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/ui/flows/dashboard/project/states"
)

func newTestLayout() states.PaneLayout {
	return states.NewPaneLayout(nil, nil).SetSize(200, 50)
}

func TestDragCoordinator_HandleClick_ClearsActiveDrag(t *testing.T) {
	t.Parallel()

	var dc states.DragCoordinator

	dc.SetActiveDrag(true, 5, 5, 10, states.SelectionInspector)

	dc.HandleClick(tea.MouseClickMsg{X: 3, Y: 7})

	if dc.DragActive() {
		t.Error("expected drag cleared after HandleClick")
	}

	if dc.LastPressX() != 3 || dc.LastPressY() != 7 {
		t.Errorf("expected lastPress (3,7), got (%d,%d)", dc.LastPressX(), dc.LastPressY())
	}
}

func TestDragCoordinator_HandleClick_RecordsPressPosition(t *testing.T) {
	t.Parallel()

	var dc states.DragCoordinator

	dc.HandleClick(tea.MouseClickMsg{X: 42, Y: 17})

	if dc.LastPressX() != 42 || dc.LastPressY() != 17 {
		t.Errorf("expected lastPress (42,17), got (%d,%d)", dc.LastPressX(), dc.LastPressY())
	}
}

func TestDragCoordinator_HandleMotion_BelowThreshold_ReturnsFalse(t *testing.T) {
	t.Parallel()

	var dc states.DragCoordinator

	dc.SetLastPress(10, 10)

	layout := newTestLayout()

	// Move by exactly 1 cell — at or below the ≤1 threshold.
	handled := dc.HandleMotion(tea.MouseMotionMsg{Button: tea.MouseLeft, X: 11, Y: 11}, layout)

	if handled {
		t.Error("expected false (no drag started) for motion within threshold")
	}

	if dc.DragActive() {
		t.Error("expected drag not started for motion within threshold")
	}
}

func TestDragCoordinator_HandleMotion_AboveThreshold_StartsDrag(t *testing.T) {
	t.Parallel()

	layout := newTestLayout()

	// Place press inside the inspector area (right pane). LogViewBounds for a
	// 200×50 layout: inspector occupies roughly x=61..199, y=0..47.
	lb := layout.LogViewBounds()
	pressX := lb.X + 5
	pressY := lb.Y + 5

	var dc states.DragCoordinator

	dc.SetLastPress(pressX, pressY)

	handled := dc.HandleMotion(tea.MouseMotionMsg{
		Button: tea.MouseLeft,
		X:      pressX + 5,
		Y:      pressY + 5,
	}, layout)

	if !handled {
		t.Error("expected true (drag started) for motion above threshold")
	}

	if !dc.DragActive() {
		t.Error("expected drag.active=true after threshold crossed")
	}

	if dc.DragComponent() != states.SelectionInspector {
		t.Errorf("expected SelectionInspector, got %v", dc.DragComponent())
	}
}

func TestDragCoordinator_HandleRelease_ActiveDrag_ReturnsHandled(t *testing.T) {
	t.Parallel()

	var dc states.DragCoordinator

	dc.SetActiveDrag(true, 5, 5, 10, states.SelectionInspector)

	layout := newTestLayout()

	_, handled := dc.HandleRelease(
		tea.MouseReleaseMsg{Button: tea.MouseLeft},
		layout,
		"list", "insp\nline2", "footer",
	)

	if !handled {
		t.Error("expected handled=true for active drag release")
	}

	if dc.DragActive() {
		t.Error("expected drag cleared after release")
	}
}

func TestDragCoordinator_HandleRelease_InactiveDrag_ReturnsFalse(t *testing.T) {
	t.Parallel()

	var dc states.DragCoordinator

	layout := newTestLayout()

	_, handled := dc.HandleRelease(
		tea.MouseReleaseMsg{Button: tea.MouseLeft},
		layout,
		"list", "insp", "footer",
	)

	if handled {
		t.Error("expected handled=false when no drag was active")
	}
}
