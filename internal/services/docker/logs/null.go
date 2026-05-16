package logs

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

// NullLogStreamer is a no-op LogStreamer that never delivers events. It
// satisfies the same method set as LogStreamer and follows ADR-0006 (Null
// Object pattern): callers do not need to nil-check the streamer field.
type NullLogStreamer struct{}

// Start is a no-op.
func (NullLogStreamer) Start(_ context.Context, _ string) {}

// Lines returns a nil channel — no lines will ever arrive.
func (NullLogStreamer) Lines() <-chan string { return nil }

// Next returns a cmd that blocks forever — no events will ever arrive.
func (NullLogStreamer) Next() tea.Cmd {
	return func() tea.Msg { return (<-make(chan tea.Msg)) }
}

// Close is a no-op.
func (NullLogStreamer) Close() {}
