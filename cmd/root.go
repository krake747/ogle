package cmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds the application configuration loaded from file, environment, or flags.
type Config struct {
	Log struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
	Timeout time.Duration `mapstructure:"timeout"`
}

var (
	cfgFile       string
	config        Config
	logger        *slog.Logger
	logLevel      = new(slog.LevelVar)
	cancelTimeout context.CancelFunc
	buildVersion  string
	buildCommit   string
	buildDate     string
	rootCmd       = &cobra.Command{
		Use:   "ogle",
		Short: "A brief description of your application",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			err := initialiseConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to initialise configuration: %w", err)
			}

			level := slog.LevelWarn

			switch strings.ToLower(config.Log.Level) {
			case "debug":
				level = slog.LevelDebug
			case "info":
				level = slog.LevelInfo
			case "warn", "warning":
				level = slog.LevelWarn
			case "error":
				level = slog.LevelError
			}

			logLevel.Set(level)

			//nolint:sloglint // global logger is fine here
			logger.DebugContext(
				cmd.Context(),
				"Configuration initialised. Using config file:",
				slog.String("cfgFile", viper.ConfigFileUsed()),
			)

			ctx, cancel := context.WithTimeout(cmd.Context(), config.Timeout)
			cancelTimeout = cancel

			cmd.SetContext(ctx)

			return nil
		},
		PersistentPostRunE: func(_ *cobra.Command, _ []string) error {
			if cancelTimeout != nil {
				cancelTimeout()
			}

			return nil
		},
	}
)

// Execute runs the root command and handles any errors.
// This is called by main.main() and should only be called once.
func Execute(version, commit, date string) {
	buildVersion = version
	buildCommit = commit
	buildDate = date

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level: logLevel,
	})
	logger = slog.New(handler)

	const defaultTimeout = 3 * time.Minute
	viper.SetDefault("timeout", defaultTimeout)

	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ogle/config)")

	rootCmd.AddCommand(newVersionCommand())
}

func initialiseConfig(cmd *cobra.Command) error {
	viper.SetEnvPrefix("OGLE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for a config file in default locations.
		home, err := os.UserHomeDir()
		// Only panic if we can't get the home directory.
		cobra.CheckErr(err)

		// Search config in home directory with name "config" (without extension).
		viper.AddConfigPath(".")
		viper.AddConfigPath(home + "/.ogle")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("failed to initialise config: %w", err)
		}
	}

	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		return fmt.Errorf("failed to bind config flags: %w", err)
	}

	if err = viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// Root exposes the root command for tools like doc generators.
func Root() *cobra.Command { return rootCmd }
