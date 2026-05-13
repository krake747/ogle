package states

type logLine struct {
	text     string
	isStderr bool
}

type logBuffer struct {
	lines []logLine
	cap   int
}

func newLogBuffer(capacity int) logBuffer {
	return logBuffer{lines: nil, cap: capacity}
}

// Append adds a line to the buffer. When the buffer is at capacity the oldest
// line is discarded to make room.
func (b *logBuffer) Append(line logLine) {
	if len(b.lines) >= b.cap {
		b.lines = b.lines[1:]
	}

	b.lines = append(b.lines, line)
}

// Clear removes all lines from the buffer and releases the backing array.
func (b *logBuffer) Clear() {
	b.lines = nil
}

// Lines returns the current log lines in order from oldest to newest.
func (b *logBuffer) Lines() []logLine {
	return b.lines
}
