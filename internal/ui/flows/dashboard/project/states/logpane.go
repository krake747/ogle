package states

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ma-tf/ogle/internal/msgs"
	logs "github.com/ma-tf/ogle/internal/services/docker/logs"
	"github.com/ma-tf/ogle/internal/ui/components/inspector"
)

// LogPane manages a streaming log view: the streamer, buffer, scroll offset,
// and pause state.
type LogPane struct {
	streamer   logs.Streamer
	buffer     logBuffer
	scrollRows int
	paused     bool
	state      inspector.LogAreaState
}

// NewLogPane returns a LogPane ready for use in tests. The streamer is
// injected by the caller; the buffer is pre-allocated to bufCap entries.
func NewLogPane(streamer logs.Streamer, bufCap int) *LogPane {
	return &LogPane{
		streamer:   streamer,
		buffer:     newLogBuffer(bufCap),
		scrollRows: 0,
		paused:     false,
		state:      inspector.LogAreaStreaming,
	}
}

// AppendLine adds a log line to the buffer. Exported for use by tests.
func (lp *LogPane) AppendLine(text string, isStderr bool) {
	lp.buffer.Append(logLine{text: text, isStderr: isStderr})
}

// SetScrollRows sets the scroll offset directly. Exported for use by tests.
func (lp *LogPane) SetScrollRows(n int) {
	lp.scrollRows = n
}

// ScrollRows returns the current scroll offset. Exported for use by tests.
func (lp *LogPane) ScrollRows() int {
	return lp.scrollRows
}

// SetPaused sets the paused flag directly. Exported for use by tests.
func (lp *LogPane) SetPaused(paused bool) {
	lp.paused = paused
}

// StartStream closes any existing stream, starts a new one for the given
// container name, and returns the first Next() cmd. Sets state to LogAreaStreaming.
// containerName is pre-computed by the caller via logs.ContainerName.
func (lp *LogPane) StartStream(ctx context.Context, containerName string) tea.Cmd {
	lp.streamer.Start(ctx, containerName)
	lp.state = inspector.LogAreaStreaming

	return lp.streamer.Next()
}

// HandleLogLine appends the line to the buffer and, if not paused, resets the
// scroll offset to follow the tail.
func (lp *LogPane) HandleLogLine(msg msgs.LogLine) tea.Cmd {
	lp.buffer.Append(logLine{text: msg.Text, isStderr: msg.IsStderr})

	if !lp.paused {
		lp.scrollRows = 0
	}

	return lp.streamer.Next()
}

// HandleStreamError closes the streamer and marks the log area unavailable.
func (lp *LogPane) HandleStreamError() {
	lp.streamer.Close()
	lp.state = inspector.LogAreaUnavailable
}

// HandleContainerNotFound marks the log area not-found and schedules a retry.
func (lp *LogPane) HandleContainerNotFound() tea.Cmd {
	lp.state = inspector.LogAreaNotFound

	return tea.Tick(logStreamRetryDelay, func(_ time.Time) tea.Msg {
		return logStreamRetryMsg{}
	})
}

// HandleRetry starts a new stream for the given container. containerName is
// pre-computed by the caller via logs.ContainerName.
func (lp *LogPane) HandleRetry(ctx context.Context, containerName string) tea.Cmd {
	return lp.StartStream(ctx, containerName)
}

// ComputeDisplayLines builds the slice of pre-styled rows for the current
// scroll position. scrollRows is clamped in-place.
func (lp *LogPane) ComputeDisplayLines(width, height int, stderrStyle lipgloss.Style) []string {
	if width <= 0 || height <= 0 {
		return nil
	}

	var displayRows []string

	for _, line := range lp.buffer.Lines() {
		for part := range strings.SplitSeq(line.text, "\n") {
			for row := range strings.SplitSeq(ansi.Hardwrap(part, width, true), "\n") {
				if line.isStderr {
					row = stderrStyle.Render(row)
				}

				displayRows = append(displayRows, row)
			}
		}
	}

	totalRows := len(displayRows)

	pausedRows := 0
	if lp.paused {
		pausedRows = 1
	}

	availRows := height - inspector.HeaderLines - pausedRows
	if availRows <= 0 {
		return nil
	}

	maxScroll := max(totalRows-availRows, 0)

	lp.scrollRows = clamp(lp.scrollRows, 0, maxScroll)

	end := max(totalRows-lp.scrollRows, 0)
	start := max(end-availRows, 0)

	return displayRows[start:end]
}

// ScrollUp increases the scroll offset by paneHeight/2 and sets paused.
func (lp *LogPane) ScrollUp(paneHeight int) {
	halfPane := max(paneHeight/halfPaneDivisor, 1)
	lp.scrollRows += halfPane
	lp.paused = true
}

// ScrollDown decreases the scroll offset by paneHeight/2. Clears paused when
// scrollRows reaches zero.
func (lp *LogPane) ScrollDown(paneHeight int) {
	halfPane := max(paneHeight/halfPaneDivisor, 1)
	lp.scrollRows -= halfPane

	if lp.scrollRows < 0 {
		lp.scrollRows = 0
	}

	if lp.scrollRows == 0 {
		lp.paused = false
	}
}

// Clear resets the buffer and all scroll state.
func (lp *LogPane) Clear() {
	lp.buffer.Clear()
	lp.scrollRows = 0
	lp.paused = false
}

// Close shuts down the underlying log streamer.
func (lp *LogPane) Close() {
	lp.streamer.Close()
}

// State returns the current log area display state.
func (lp *LogPane) State() inspector.LogAreaState {
	return lp.state
}

// Paused reports whether the log view is currently paused.
func (lp *LogPane) Paused() bool {
	return lp.paused
}
