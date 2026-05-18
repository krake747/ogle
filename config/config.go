package config

// Config holds the application configuration loaded from file, environment, or flags.
type Config struct {
	Theme        string `yaml:"theme"`
	LogBufferCap int    `yaml:"logBufferCap"`
	Log          struct {
		Level string `yaml:"level"`
	} `yaml:"log"`
}
