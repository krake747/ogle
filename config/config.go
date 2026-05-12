package config

import "time"

// Config holds the application configuration loaded from file, environment, or flags.
type Config struct {
	ProjectFile  string        `mapstructure:"project-file"`
	Theme        string        `mapstructure:"theme"`
	PollInterval time.Duration `mapstructure:"poll-interval"`
	LogBufferCap int           `mapstructure:"log-buffer-cap"`
	Log          struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
}
