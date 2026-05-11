package watcher_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	scannermocks "github.com/ma-tf/ogle/internal/services/scanner/mocks"
	"github.com/ma-tf/ogle/internal/services/watcher"
)

const composeYML = "compose.yml"

func newTestLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestNewNull(t *testing.T) {
	t.Parallel()

	t.Run("Dir returns empty string", func(t *testing.T) {
		t.Parallel()

		w := watcher.NewNull()
		defer w.Close()

		require.Empty(t, w.Dir())
	})

	t.Run("Snapshot delivers empty FileAvailabilityChanged", func(t *testing.T) {
		t.Parallel()

		w := watcher.NewNull()
		defer w.Close()

		msg := w.Snapshot()()
		require.Equal(t, msgs.FileAvailabilityChanged{}, msg)
	})

	t.Run("Next blocks until Close then returns nil", func(t *testing.T) {
		t.Parallel()

		w := watcher.NewNull()

		got := make(chan any, 1)
		go func() { got <- w.Next()() }()

		require.NoError(t, w.Close())

		select {
		case msg := <-got:
			require.Nil(t, msg)
		case <-time.After(2 * time.Second):
			t.Fatal("Next did not return after Close")
		}
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		t.Parallel()

		w := watcher.NewNull()
		require.NoError(t, w.Close())
		require.NoError(t, w.Close())
		require.NoError(t, w.Close())
	})
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("Dir returns the watched directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		sc := scannermocks.NewMockScanner(t)

		w, err := watcher.New(dir, sc, newTestLogger())
		require.NoError(t, err)

		defer w.Close()

		require.Equal(t, dir, w.Dir())
	})

	t.Run("non-existent directory returns error and valid Watcher", func(t *testing.T) {
		t.Parallel()
		sc := scannermocks.NewMockScanner(t)

		w, err := watcher.New("/nonexistent/path/that/cannot/exist", sc, newTestLogger())
		require.Error(t, err)
		require.NotNil(t, w)
		require.NoError(t, w.Close())
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		sc := scannermocks.NewMockScanner(t)

		w, err := watcher.New(dir, sc, newTestLogger())
		require.NoError(t, err)

		require.NoError(t, w.Close())
		require.NoError(t, w.Close())
		require.NoError(t, w.Close())
	})
}

func TestSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("delivers current files without waiting for a filesystem event", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		expected := []string{filepath.Join(dir, composeYML)}

		sc := scannermocks.NewMockScanner(t)
		sc.On("ScanAll", dir).Return(expected).Once()

		w, err := watcher.New(dir, sc, newTestLogger())
		require.NoError(t, err)

		defer w.Close()

		msg := w.Snapshot()()

		require.Equal(t, msgs.FileAvailabilityChanged{Files: expected}, msg)
	})
}

//nolint:funlen
func TestNext(t *testing.T) {
	t.Parallel()

	t.Run("known file created triggers FileAvailabilityChanged", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		composePath := filepath.Join(dir, composeYML)

		sc := scannermocks.NewMockScanner(t)
		sc.On("KnownFilenames").Return([]string{composeYML})
		sc.On("ScanAll", dir).Return([]string{composePath})

		w, err := watcher.New(dir, sc, newTestLogger())
		require.NoError(t, err)

		defer w.Close()

		got := make(chan any, 1)
		go func() { got <- w.Next()() }()

		require.NoError(t, os.WriteFile(composePath, []byte{}, 0o600))

		select {
		case msg := <-got:
			require.Equal(t, msgs.FileAvailabilityChanged{Files: []string{composePath}}, msg)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for FileAvailabilityChanged")
		}
	})

	t.Run("unknown file created does not trigger an event", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		sc := scannermocks.NewMockScanner(t)
		sc.On("KnownFilenames").Return([]string{composeYML}).Maybe()
		// ScanAll is intentionally NOT set up. If called, the mock panics —
		// acting as a hard assertion that unknown files never reach ScanAll.
		// KnownFilenames is Maybe() because Close() may race the fsnotify event:
		// if run() processes the event it calls KnownFilenames; if Close() wins
		// first it does not. Either way ScanAll must never be called.

		w, err := watcher.New(dir, sc, newTestLogger())
		require.NoError(t, err)

		require.NoError(t, os.WriteFile(filepath.Join(dir, "unknown.txt"), []byte{}, 0o600))

		// Close before calling Next so Next returns nil (not a FileAvailabilityChanged).
		require.NoError(t, w.Close())

		got := make(chan any, 1)
		go func() { got <- w.Next()() }()

		select {
		case msg := <-got:
			require.Nil(t, msg)
		case <-time.After(2 * time.Second):
			t.Fatal("Next did not return nil after Close")
		}
	})

	t.Run("known file deleted triggers FileAvailabilityChanged", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		composePath := filepath.Join(dir, composeYML)

		// Pre-create so the watcher sees a REMOVE event.
		require.NoError(t, os.WriteFile(composePath, []byte{}, 0o600))

		sc := scannermocks.NewMockScanner(t)
		sc.On("KnownFilenames").Return([]string{composeYML})
		sc.On("ScanAll", dir).Return([]string{})

		w, err := watcher.New(dir, sc, newTestLogger())
		require.NoError(t, err)

		defer w.Close()

		got := make(chan any, 1)
		go func() { got <- w.Next()() }()

		require.NoError(t, os.Remove(composePath))

		select {
		case msg := <-got:
			fac, ok := msg.(msgs.FileAvailabilityChanged)
			require.True(t, ok)
			require.Empty(t, fac.Files)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for FileAvailabilityChanged")
		}
	})

	t.Run("rapid events deliver only the latest snapshot", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		composePath := filepath.Join(dir, composeYML)

		// Pre-create the file before the watcher starts so subsequent writes
		// generate only WRITE events (not CREATE+WRITE).
		require.NoError(t, os.WriteFile(composePath, []byte("initial"), 0o600))

		sc := scannermocks.NewMockScanner(t)
		sc.On("KnownFilenames").Return([]string{composeYML})

		release := make(chan struct{})
		ready := make(chan struct{}, 10)

		var n atomic.Int32

		sc.On("ScanAll", dir).Return(func(string) []string {
			callNum := int(n.Add(1))

			ready <- struct{}{}

			<-release

			if callNum == 1 {
				return []string{"file_v1"}
			}

			return []string{"file_v2"}
		})

		w, err := watcher.New(dir, sc, newTestLogger())
		require.NoError(t, err)

		defer w.Close()

		// Event 1: run() calls ScanAll and blocks.
		require.NoError(t, os.WriteFile(composePath, []byte("v1"), 0o600))

		select {
		case <-ready:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for first ScanAll call")
		}

		// Event 2: written while run() is blocked on event 1's ScanAll.
		require.NoError(t, os.WriteFile(composePath, []byte("v2"), 0o600))

		// Unblock: event 1 ScanAll returns "file_v1" → run() sends snapshot_1
		// → loops → processes event 2 → drains snapshot_1 → sends snapshot_2.
		close(release)

		select {
		case <-ready:
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for second ScanAll call")
		}

		got := make(chan any, 1)
		go func() { got <- w.Next()() }()

		select {
		case msg := <-got:
			fac, ok := msg.(msgs.FileAvailabilityChanged)
			require.True(t, ok)
			require.Equal(t, []string{"file_v2"}, fac.Files)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for message from Next()")
		}
	})
}
