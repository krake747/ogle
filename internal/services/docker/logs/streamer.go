package logs

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

// Streamer is the interface satisfied by any log source. It mirrors the method
// set of *LogStreamer and NullLogStreamer exactly.
type Streamer interface {
	Start(ctx context.Context, containerName string)
	Next() tea.Cmd
	Close()
}

// Compile-time assertions.
var (
	_ Streamer = (*LogStreamer)(nil)
	_ Streamer = NullLogStreamer{}
)
