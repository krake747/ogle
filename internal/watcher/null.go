package watcher

import (
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

// nullWatcher satisfies the Watcher interface but delivers no events.
// Next blocks until Close is called. Used when watcher creation fails,
// so the rest of the application never handles a nil Watcher.
type nullWatcher struct {
	done chan struct{}
	once sync.Once
}

// NewNull returns a Watcher that never delivers events.
func NewNull() Watcher {
	return &nullWatcher{
		done: make(chan struct{}),
		once: sync.Once{},
	}
}

func (n *nullWatcher) Dir() string {
	return ""
}

func (n *nullWatcher) Next() tea.Cmd {
	return func() tea.Msg {
		<-n.done

		return nil
	}
}

// Snapshot delivers an empty file list. In practice unreachable on a
// nullWatcher — Snapshot is only called via the watcherReadyMsg handler, which
// is only produced on successful watcher creation (i.e. never for a
// nullWatcher).
func (n *nullWatcher) Snapshot() tea.Cmd {
	return func() tea.Msg {
		return msgs.FileAvailabilityChanged{}
	}
}

func (n *nullWatcher) Close() error {
	n.once.Do(func() { close(n.done) })

	return nil
}
