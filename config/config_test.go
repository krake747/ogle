package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
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

func TestLoad(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		configYAML string
		writeFile  bool // false → no file written (missing config)

		// assert
		expectedConfig config.Config
		expectedError  error
	}

	defaultCfg := config.Defaults()

	for _, tc := range []testCase{
		{
			name:       "missing config file returns defaults",
			writeFile:  false,
			expectedConfig: defaultCfg,
		},
		{
			name:       "empty config returns defaults",
			writeFile:  true,
			configYAML: "",
			expectedConfig: defaultCfg,
		},
		{
			name:      "partial config reflects set fields with defaults for rest",
			writeFile: true,
			configYAML: "theme: catppuccino_mocha\n",
			expectedConfig: config.Config{
				Theme:        "catppuccino_mocha",
				LogBufferCap: config.DefaultLogBufferCap,
			},
		},
		{
			name:      "malformed YAML fails read",
			writeFile: true,
			configYAML: "invalid: yaml: [\nbroken",
			expectedError: fmt.Errorf("read"), // any read error
		},
		{
			name:      "valid config with all fields",
			writeFile: true,
			configYAML: "theme: solarized_light\nlogBufferCap: 2000\nlog:\n  level: debug\n",
			expectedConfig: config.Config{
				Theme:        "solarized_light",
				LogBufferCap: 2000,
				Log: struct {
					Level string `yaml:"level"`
				}{Level: "debug"},
			},
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			v := viper.New()
			v.SetConfigType("yaml")

			if tc.writeFile {
				dir := t.TempDir()
				path := filepath.Join(dir, "config.yaml")
				require.NoError(t, os.WriteFile(path, []byte(tc.configYAML), 0o600))
				v.SetConfigFile(path)

				if tc.expectedError != nil {
					err := v.ReadInConfig()
					require.Error(t, err)
					return
				}
				require.NoError(t, v.ReadInConfig())
			}

			cfg, err := config.Load(v)
			require.NoError(t, err)
			require.Equal(t, tc.expectedConfig, cfg)
		})
	}
}
