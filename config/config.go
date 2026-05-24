// Package config holds the application configuration model, defaults, loading,
// and persistence.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

// Config holds the application configuration loaded from file, environment, or flags.
type Config struct {
	Theme        string `yaml:"theme"`
	LogBufferCap int    `yaml:"logBufferCap"`
	Log          struct {
		Level string `yaml:"level"`
	} `yaml:"log"`
}

// DefaultLogBufferCap is the default maximum number of log lines buffered per service.
const DefaultLogBufferCap = 1000

// Defaults returns a Config populated with built-in defaults.
func Defaults() Config {
	return Config{
		Theme:        "default",
		LogBufferCap: DefaultLogBufferCap,
		Log: struct {
			Level string `yaml:"level"`
		}{Level: ""},
	}
}

// Load reads config from a viper instance already configured with flag bindings,
// environment variables, and config file paths. The caller is responsible for
// setting up viper (ReadInConfig, BindPFlags, etc.) before calling Load.
//
// Load applies built-in defaults to the viper instance before unmarshalling so
// that unset fields receive their default values.
func Load(v *viper.Viper) (Config, error) {
	v.SetDefault("theme", Defaults().Theme)
	v.SetDefault("logBufferCap", DefaultLogBufferCap)

	var cfg Config

	if err := v.Unmarshal(&cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "yaml"
	}); err != nil {
		return Config{}, fmt.Errorf("unmarshal config: %w", err)
	}

	return cfg, nil
}

// Save writes cfg as YAML to path, creating the file with 0600 permissions.
func Save(path string, cfg Config) error {
	data, marshalErr := yaml.Marshal(&cfg)
	if marshalErr != nil {
		return fmt.Errorf("marshal config: %w", marshalErr)
	}

	if writeErr := os.WriteFile(path, data, 0o600); writeErr != nil {
		return fmt.Errorf("write config: %w", writeErr)
	}

	return nil
}

// ResolvePath returns the config file path. If flagPath is non-empty it is
// returned verbatim; otherwise the default path $HOME/.ogle/config.yaml is used.
func ResolvePath(flagPath string) (string, error) {
	if flagPath != "" {
		return flagPath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	return filepath.Join(home, ".ogle", "config.yaml"), nil
}
