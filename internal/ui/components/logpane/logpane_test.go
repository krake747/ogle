package logpane_test

import (
	"testing"

	"charm.land/lipgloss/v2"

	logs_mocks "github.com/ma-tf/ogle/internal/services/docker/logs/mocks"
	"github.com/ma-tf/ogle/internal/ui/components/inspector"
	"github.com/ma-tf/ogle/internal/ui/components/logpane"
)

func noStyle() lipgloss.Style { return lipgloss.NewStyle() }

func TestLogPane_ComputeDisplayLines_EmptyBuffer(t *testing.T) {
	t.Parallel()

	lp := logpane.NewLogPane(logs_mocks.NewMockStreamer(t), 100)
	lines := lp.ComputeDisplayLines(80, 24, noStyle())

	if len(lines) != 0 {
		t.Fatalf("expected empty slice for empty buffer, got %d lines", len(lines))
	}
}

func TestLogPane_ComputeDisplayLines_WindowIsCorrect(t *testing.T) {
	t.Parallel()

	lp := logpane.NewLogPane(logs_mocks.NewMockStreamer(t), 200)

	// Fill with 20 single-character lines that won't wrap at width=80.
	for i := range 20 {
		lp.AppendLine(string(rune('A'+i%26)), false)
	}

	// height=14 → availRows = 14 - inspector.HeaderLines = 14-6 = 8
	lines := lp.ComputeDisplayLines(80, 14, noStyle())

	want := 8
	if len(lines) != want {
		t.Fatalf("expected %d lines (availRows), got %d", want, len(lines))
	}

	// Default scrollRows=0 → last 8 lines of 20 → indices 12..19 → 'M'..'T'
	if lines[0] != "M" {
		t.Errorf("expected first visible line 'M', got %q", lines[0])
	}

	if lines[7] != "T" {
		t.Errorf("expected last visible line 'T', got %q", lines[7])
	}
}

func TestLogPane_ComputeDisplayLines_ScrollRowsClamped(t *testing.T) {
	t.Parallel()

	lp := logpane.NewLogPane(logs_mocks.NewMockStreamer(t), 200)

	for i := range 5 {
		lp.AppendLine(string(rune('A'+i)), false)
	}

	// height=14 → availRows=8; only 5 rows total → maxScroll=0
	lp.SetScrollRows(999)

	lines := lp.ComputeDisplayLines(80, 14, noStyle())

	if lp.ScrollRows() != 0 {
		t.Errorf("expected scrollRows clamped to 0, got %d", lp.ScrollRows())
	}

	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}
}

func TestLogPane_ComputeDisplayLines_PausedReducesAvailRows(t *testing.T) {
	t.Parallel()

	lp := logpane.NewLogPane(logs_mocks.NewMockStreamer(t), 200)
	lp.SetPaused(true)

	for i := range 20 {
		lp.AppendLine(string(rune('A'+i%26)), false)
	}

	// height=14, paused → availRows = 14 - inspector.HeaderLines - 1 = 7
	lines := lp.ComputeDisplayLines(80, 14, noStyle())

	want := 7
	if len(lines) != want {
		t.Fatalf("expected %d lines when paused, got %d", want, len(lines))
	}
}

func TestLogPane_ComputeDisplayLines_ScrollOffsetWindow(t *testing.T) {
	t.Parallel()

	lp := logpane.NewLogPane(logs_mocks.NewMockStreamer(t), 200)

	// 10 single-char lines
	for i := range 10 {
		lp.AppendLine(string(rune('A'+i)), false)
	}

	// height=12 → availRows = 12 - 6 = 6
	// scrollRows=2 → end = max(10-2,0)=8; start = max(8-6,0)=2 → lines[2..8] = C,D,E,F,G,H
	lp.SetScrollRows(2)

	lines := lp.ComputeDisplayLines(80, 12, noStyle())

	if len(lines) != 6 {
		t.Fatalf("expected 6 lines, got %d", len(lines))
	}

	if lines[0] != "C" {
		t.Errorf("expected 'C' at index 0, got %q", lines[0])
	}

	if lines[5] != "H" {
		t.Errorf("expected 'H' at index 5, got %q", lines[5])
	}
}

func TestLogPane_HandleStreamError_SchedulesRetry(t *testing.T) {
	t.Parallel()

	streamer := logs_mocks.NewMockStreamer(t)
	streamer.EXPECT().Close().Return()

	lp := logpane.NewLogPane(streamer, 100)
	cmd := lp.HandleStreamError()

	if cmd == nil {
		t.Fatal("expected non-nil cmd (retry timer), got nil")
	}

	if lp.State() != inspector.LogAreaNotFound {
		t.Fatalf("expected LogAreaNotFound after stream error, got %v", lp.State())
	}
}
