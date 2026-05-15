package config

// Config holds the application configuration loaded from file, environment, or flags.
type Config struct {
	ProjectFile  string `mapstructure:"projectFile"`
	Theme        string `mapstructure:"theme"`
	LogBufferCap int    `mapstructure:"logBufferCap"`
	Log          struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
}
