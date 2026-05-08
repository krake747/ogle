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
	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile       string
	cfg           config.Config
	logger        *slog.Logger
	logLevel      = new(slog.LevelVar)
	cancelTimeout context.CancelFunc
	buildVersion  string
	buildCommit   string
	buildDate     string
	rootCmd       = &cobra.Command{
		Use:   "ogle",
		Short: "A TUI for monitoring Docker Compose projects.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			err := initialiseConfig(cmd)
			if err != nil {
				return fmt.Errorf("failed to initialise configuration: %w", err)
			}

			level := slog.LevelWarn

			switch strings.ToLower(cfg.Log.Level) {
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

			ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
			cancelTimeout = cancel

			cmd.SetContext(ctx)

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			program := internal.Start()

			_, err := program.Run()
			return err
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

	rootCmd.PersistentFlags().
		StringVarP(&cfg.ProjectFile, "project-file", "f", "", "path to docker compose file (default is ./docker-compose.yml)")

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
		if err != nil {
			return fmt.Errorf("get home directory: %w", err)
		}

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

	if err := viper.BindPFlags(cmd.Flags()); err != nil {
		return fmt.Errorf("failed to bind config flags: %w", err)
	}

	if err := viper.BindPFlags(cmd.InheritedFlags()); err != nil {
		return fmt.Errorf("failed to bind inherited config flags: %w", err)
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// Root exposes the root command for tools like doc generators.
func Root() *cobra.Command { return rootCmd }
