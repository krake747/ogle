// Package watcher monitors a directory for changes to known compose filenames
// and delivers snapshot messages to the Bubble Tea runtime via tea.Cmd.
package watcher

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"

	tea "charm.land/bubbletea/v2"
	"github.com/fsnotify/fsnotify"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/scanner"
)

// ErrCreateWatcher is returned when watcher initialisation fails.
var ErrCreateWatcher = errors.New("create watcher")

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

// Service monitors a directory for filesystem events on known compose
// filenames and delivers msgs.FileAvailabilityChanged snapshots to the
// Bubble Tea runtime. extraFile is an additional basename to track beyond
// the scanner's known filenames (e.g. a manually specified project file).
// When empty, only the scanner's known filenames are tracked.
type Service struct {
	fw        *fsnotify.Watcher
	dir       string
	scanner   scanner.Scanner
	logger    *slog.Logger
	extraFile string
	events    chan tea.Msg
	done      chan struct{}
	once      sync.Once
}

// New creates a Watcher that monitors dir and starts the background event
// loop. extraFile is an additional basename to track (e.g. a manually
// specified project file). Pass "" to only track the scanner's known filenames.
// On failure, nil is returned alongside the error.
func New(dir string, logger *slog.Logger, extraFile string) (Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("%w: fsnotify: %w", ErrCreateWatcher, err)
	}

	if addErr := fw.Add(dir); addErr != nil {
		closeErr := fw.Close()

		return nil, fmt.Errorf("%w: add watch: %w", ErrCreateWatcher, errors.Join(addErr, closeErr))
	}

	sc := scanner.New(logger)

	w := &Service{
		fw:      fw,
		dir:     dir,
		scanner: sc,
		logger:  logger,
		extraFile: func() string {
			if extraFile == "" {
				return ""
			}

			return filepath.Base(extraFile)
		}(),
		events: make(chan tea.Msg, 1),
		done:   make(chan struct{}),
		once:   sync.Once{},
	}

	go w.run()

	return w, nil
}

// Dir returns the directory this Service monitors.
func (w *Service) Dir() string {
	return w.dir
}

// Close stops the background event loop and releases the underlying fsnotify
// watcher. Safe to call more than once; subsequent calls are no-ops.
func (w *Service) Close() error {
	var fwErr error

	w.once.Do(func() {
		close(w.done)
		fwErr = w.fw.Close()
	})

	if fwErr != nil {
		return fmt.Errorf("close fsnotify watcher: %w", fwErr)
	}

	return nil
}

// Next returns a tea.Cmd that blocks until the next availability snapshot is
// ready and returns it as a msgs.FileAvailabilityChanged. After receiving a
// message in Update, call Next again to continue listening. Returns nil if the
// watcher is closed.
func (w *Service) Next() tea.Cmd {
	return func() tea.Msg {
		select {
		case msg := <-w.events:
			return msg
		case <-w.done:
			return nil
		}
	}
}

// scan returns all known compose files and the extra file (if set) that
// currently exist in the monitored directory.
func (w *Service) scan() []string {
	files := w.scanner.ScanAll(w.dir)

	if w.extraFile != "" {
		path := filepath.Join(w.dir, w.extraFile)

		if _, err := os.Stat(path); err == nil && !slices.Contains(files, path) {
			files = append(files, path)
		}
	}

	return files
}

// Snapshot returns a tea.Cmd that delivers the current filesystem state as a
// msgs.FileAvailabilityChanged without waiting for a change event. Extra
// files passed to New are included in the scan.
func (w *Service) Snapshot() tea.Cmd {
	return func() tea.Msg {
		return msgs.FileAvailabilityChanged{Files: w.scan()}
	}
}

// run is the background goroutine that processes fsnotify events.
func (w *Service) run() {
	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.fw.Events:
			if !ok {
				return
			}

			known := slices.Contains(w.scanner.KnownFilenames(), filepath.Base(event.Name))
			if !known && filepath.Base(event.Name) != w.extraFile {
				continue
			}

			snapshot := msgs.FileAvailabilityChanged{
				Files: w.scan(),
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
