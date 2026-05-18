package config

// Config holds the application configuration loaded from file, environment, or flags.
type Config struct {
	Theme        string `mapstructure:"theme"`
	LogBufferCap int    `mapstructure:"logBufferCap"`
	Log          struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
}
