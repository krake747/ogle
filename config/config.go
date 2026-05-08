package config

import "time"

// Config holds the application configuration loaded from file, environment, or flags.
type Config struct {
	ProjectFile string `mapstructure:"project-file"`
	Log         struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
	Timeout time.Duration `mapstructure:"timeout"`
}
