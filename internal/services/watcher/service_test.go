package watcher_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/watcher"
)

const composeYML = "compose.yml"

func newTestLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("Dir returns the watched directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		w, err := watcher.New(dir, newTestLogger())
		require.NoError(t, err)

		defer w.Close()

		require.Equal(t, dir, w.Dir())
	})

	t.Run("non-existent directory returns error and nil", func(t *testing.T) {
		t.Parallel()

		w, err := watcher.New("/nonexistent/path/that/cannot/exist", newTestLogger())
		require.Error(t, err)
		require.Nil(t, w)
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		w, err := watcher.New(dir, newTestLogger())
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
		composePath := filepath.Join(dir, composeYML)
		require.NoError(t, os.WriteFile(composePath, []byte{}, 0o600))

		w, err := watcher.New(dir, newTestLogger())
		require.NoError(t, err)

		defer w.Close()

		msg := w.Snapshot()()

		require.Equal(t, msgs.FileAvailabilityChanged{Files: []string{composePath}}, msg)
	})
}

//nolint:funlen
func TestNext(t *testing.T) {
	t.Parallel()

	t.Run("known file created triggers FileAvailabilityChanged", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		composePath := filepath.Join(dir, composeYML)

		w, err := watcher.New(dir, newTestLogger())
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

		w, err := watcher.New(dir, newTestLogger())
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

		w, err := watcher.New(dir, newTestLogger())
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
		composePath1 := filepath.Join(dir, composeYML)
		composePath2 := filepath.Join(dir, "compose.yaml")

		require.NoError(t, os.WriteFile(composePath1, []byte("v1"), 0o600))

		w, err := watcher.New(dir, newTestLogger())
		require.NoError(t, err)

		defer w.Close()

		got := make(chan any, 1)
		go func() { got <- w.Next()() }()

		require.NoError(t, os.WriteFile(composePath2, []byte("v2"), 0o600))

		select {
		case msg := <-got:
			fac, ok := msg.(msgs.FileAvailabilityChanged)
			require.True(t, ok)
			require.Contains(t, fac.Files, composePath1)
			require.Contains(t, fac.Files, composePath2)
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for FileAvailabilityChanged")
		}
	})
}
