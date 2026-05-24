package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/config"
)

func TestDefaults(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()

	assert.Equal(t, "default", cfg.Theme)
	assert.Equal(t, config.DefaultLogBufferCap, cfg.LogBufferCap)
	assert.Equal(t, 1000, config.DefaultLogBufferCap)
}

func TestSave(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := config.Defaults()
	cfg.Theme = "catppuccin_mocha"

	err := config.Save(path, cfg)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	content := string(data)
	assert.Contains(t, content, "catppuccin_mocha")
	assert.Contains(t, content, "theme")
	assert.Contains(t, content, "logBufferCap")
}

func TestResolvePath(t *testing.T) {
	t.Parallel()

	t.Run("custom path returned verbatim", func(t *testing.T) {
		t.Parallel()

		path, err := config.ResolvePath("/tmp/my-config.yaml")
		require.NoError(t, err)
		assert.Equal(t, "/tmp/my-config.yaml", path)
	})

	t.Run("empty path falls back to home dir", func(t *testing.T) {
		t.Parallel()

		path, err := config.ResolvePath("")
		require.NoError(t, err)
		assert.True(t, strings.HasSuffix(path, ".ogle/config.yaml"))
	})
}
