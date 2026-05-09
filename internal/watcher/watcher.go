// Package watcher monitors a directory for changes to known compose filenames
// and delivers snapshot messages to the Bubble Tea runtime via tea.Cmd.
package watcher

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/fsnotify/fsnotify"

	"github.com/ma-tf/ogle/internal/compose"
	"github.com/ma-tf/ogle/internal/msgs"
)

// Watcher monitors a directory for filesystem changes and delivers
// msgs.FileAvailabilityChanged snapshots to the Bubble Tea runtime.
type Watcher interface {
	// Dir returns the directory being monitored.
	Dir() string
	// Next returns a tea.Cmd that blocks until the next snapshot is ready.
	// Re-call after each FileAvailabilityChanged to continue listening.
	Next() tea.Cmd
	// Snapshot returns a tea.Cmd that delivers the current filesystem state
	// as a msgs.FileAvailabilityChanged without waiting for a change event.
	Snapshot() tea.Cmd
	// Close stops the background goroutine and releases resources.
	Close() error
}

// fsWatcher monitors a directory for filesystem events on known compose
// filenames and delivers msgs.FileAvailabilityChanged snapshots to the
// Bubble Tea runtime.
type fsWatcher struct {
	fw     *fsnotify.Watcher
	dir    string
	logger *slog.Logger
	events chan tea.Msg
	done   chan struct{}
	once   sync.Once
}

// New creates a Watcher that monitors dir and starts the background event
// loop. On failure, a NullWatcher is returned alongside the error so the
// caller always receives a valid Watcher.
func New(dir string, logger *slog.Logger) (Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return NewNull(), fmt.Errorf("create fsnotify watcher: %w", err)
	}

	if addErr := fw.Add(dir); addErr != nil {
		closeErr := fw.Close()

		return NewNull(), fmt.Errorf("watch directory %s: %w", dir, errors.Join(addErr, closeErr))
	}

	w := &fsWatcher{
		fw:     fw,
		dir:    dir,
		logger: logger,
		events: make(chan tea.Msg, 1),
		done:   make(chan struct{}),
	}

	go w.run()

	return w, nil
}

// Dir returns the directory this fsWatcher monitors.
func (w *fsWatcher) Dir() string {
	return w.dir
}

// Close stops the background event loop and releases the underlying fsnotify
// watcher. Safe to call more than once.
func (w *fsWatcher) Close() error {
	w.once.Do(func() {
		close(w.done)
	})

	if err := w.fw.Close(); err != nil {
		return fmt.Errorf("close fsnotify watcher: %w", err)
	}

	return nil
}

// Next returns a tea.Cmd that blocks until the next availability snapshot is
// ready and returns it as a msgs.FileAvailabilityChanged. After receiving a
// message in Update, call Next again to continue listening. Returns nil if the
// watcher is closed.
func (w *fsWatcher) Next() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-w.events:
			return msg
		case <-w.done:
			return nil
		}
	}
}

// Snapshot returns a tea.Cmd that delivers the current filesystem state as a
// msgs.FileAvailabilityChanged without waiting for a change event.
func (w *fsWatcher) Snapshot() tea.Cmd {
	return func() tea.Msg {
		return msgs.FileAvailabilityChanged{Files: compose.ScanAll(w.dir)}
	}
}

// run is the background goroutine that processes fsnotify events.
func (w *fsWatcher) run() {
	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.fw.Events:
			if !ok {
				return
			}

			if !slices.Contains(compose.KnownFilenames(), filepath.Base(event.Name)) {
				continue
			}

			snapshot := msgs.FileAvailabilityChanged{
				Files: compose.ScanAll(w.dir),
			}

			// Drain any stale snapshot before sending the current one so
			// the consumer always sees the most recent state.
			select {
			case <-w.events:
			default:
			}

			select {
			case w.events <- snapshot:
			case <-w.done:
				return
			}

		case err, ok := <-w.fw.Errors:
			if !ok {
				return
			}

			w.logger.Error("watcher: fsnotify error", "dir", w.dir, "err", err)
		}
	}
}
