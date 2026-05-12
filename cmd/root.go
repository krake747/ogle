package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/services/scanner"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

var (
	cfgFile      string
	cfg          config.Config
	logger       *slog.Logger
	logLevel     = new(slog.LevelVar)
	buildVersion string
	buildCommit  string
	buildDate    string
	rootCmd      = &cobra.Command{
		Use:   "ogle",
		Short: "A TUI for monitoring Docker Compose projects.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

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

			logger.DebugContext(
				ctx,
				"Configuration initialised. Using config file:",
				slog.String("cfgFile", viper.ConfigFileUsed()),
			)

			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			var (
				sc scanner.Scanner = scanner.New(logger)
				p  parser.Parser   = parser.New(logger)
			)

			if cfg.ProjectFile != "" {
				if err := validateProjectFile(cfg.ProjectFile, p); err != nil {
					return err
				}
			}

			configDir := ""
			if cf := viper.ConfigFileUsed(); cf != "" {
				configDir = filepath.Dir(cf)
			}

			if configDir == "" || configDir == "." {
				home, err := os.UserHomeDir()
				if err != nil {
					logger.WarnContext(
						ctx,
						"could not determine home directory for theme loading",
						slog.Any("err", err),
					)
				} else {
					configDir = home + "/.ogle"
				}
			}

			th, themeErr := theme.Load(cfg.Theme, configDir)
			if themeErr != nil {
				logger.WarnContext(ctx, "theme load failed, using default", slog.Any("err", themeErr))
			}

			model := dashboard.New(ctx, cfg, logger, sc, p, th)
			program := tea.NewProgram(
				model,
				tea.WithContext(ctx),
			)

			final, err := program.Run()
			if m, ok := final.(dashboard.Model); ok {
				if closeErr := m.Close(); closeErr != nil {
					logger.ErrorContext(ctx, "close watcher", "err", closeErr)
				}
			}

			if err != nil {
				return fmt.Errorf("run program: %w", err)
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

	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ogle/config)")

	rootCmd.PersistentFlags().
		StringVarP(&cfg.ProjectFile, "project-file", "f", "", "path to docker compose file")

	rootCmd.AddCommand(newVersionCommand())
}

func initialiseConfig(cmd *cobra.Command) error {
	viper.SetEnvPrefix("OGLE")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()
	viper.SetDefault("theme", "default")

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

// validateProjectFile checks that path is a valid, parseable compose file.
// It is called only when the -f flag is explicitly provided.
func validateProjectFile(path string, p parser.Parser) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("project file not found: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("project file %q is a directory, expected a compose file", path)
	}

	if validateErr := p.Validate(path); validateErr != nil {
		return fmt.Errorf("invalid compose file: %w", validateErr)
	}

	return nil
}

// Root exposes the root command for tools like doc generators.
func Root() *cobra.Command { return rootCmd }
