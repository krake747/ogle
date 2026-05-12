package config

// Config holds the application configuration loaded from file, environment, or flags.
type Config struct {
	ProjectFile string `mapstructure:"project-file"`
	Theme       string `mapstructure:"theme"`
	Log         struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
}
