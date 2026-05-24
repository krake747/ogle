package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/app"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	pprofReadHeaderTimeout = 5 * time.Second
	pprofReadTimeout       = 60 * time.Second
	pprofWriteTimeout      = 60 * time.Second
)

var (
	cfgFile      string
	pprofAddr    string
	projectFile  string
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

			if projectFile != "" {
				abs, err := filepath.Abs(projectFile)
				if err != nil {
					return fmt.Errorf("resolve project file path: %w", err)
				}

				projectFile = abs
			}

			configPath, err := config.ResolvePath(viper.ConfigFileUsed())
			if err != nil {
				logger.WarnContext(
					ctx,
					"could not determine config file path",
					slog.Any("err", err),
				)
			}

			th, themeErr := theme.Load(cfg.Theme, filepath.Dir(configPath))
			if themeErr != nil {
				logger.WarnContext(
					ctx,
					"theme load failed, using default",
					slog.Any("err", themeErr),
				)
			}

			if pprofAddr != "" {
				mux := http.NewServeMux()
				mux.HandleFunc("/debug/pprof/", pprof.Index)
				mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
				mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
				mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
				mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

				srv := &http.Server{
					Addr:              pprofAddr,
					Handler:           mux,
					ReadHeaderTimeout: pprofReadHeaderTimeout,
					ReadTimeout:       pprofReadTimeout,
					WriteTimeout:      pprofWriteTimeout,
				}

				go func() {
					logger.InfoContext(ctx, "pprof listening", slog.String("addr", pprofAddr))

					if srvErr := srv.ListenAndServe(); srvErr != nil {
						logger.WarnContext(ctx, "pprof server stopped", slog.Any("err", srvErr))
					}
				}()
			}

			model, cleanup, err := app.New(ctx, cfg, configPath, projectFile, logger, th)
			if err != nil {
				return fmt.Errorf("app init: %w", err)
			}

			defer func() {
				if cleanErr := cleanup(); cleanErr != nil {
					logger.WarnContext(ctx, "cleanup on exit", slog.Any("err", cleanErr))
				}
			}()

			program := tea.NewProgram(
				model,
				tea.WithContext(ctx),
			)

			_, runErr := program.Run()
			if runErr != nil {
				return fmt.Errorf("run program: %w", runErr)
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
		StringVarP(&projectFile, "project-file", "f", "", "path to docker compose file")

	rootCmd.PersistentFlags().
		StringVar(&pprofAddr, "pprof-addr", "", "pprof HTTP server address (e.g. localhost:6060)")

	rootCmd.Flags().
		String("theme", "", fmt.Sprintf(
			`theme name; built-ins: "%s" (env: OGLE_THEME)`,
			strings.Join(theme.BuiltinNames(), `", "`),
		))

	rootCmd.Flags().
		Int("log-buffer-cap", 0, "maximum log lines buffered per service (env: OGLE_LOG_BUFFER_CAP; default 1000)")

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

	// BindPFlags maps --log-buffer-cap to Viper key "log-buffer-cap", which
	// mapstructure cannot match against the yaml tag "logBufferCap". Bind the
	// flag explicitly to the correct key.
	if f := cmd.Flags().Lookup("log-buffer-cap"); f != nil {
		if err := viper.BindPFlag("logBufferCap", f); err != nil {
			return fmt.Errorf("bind log-buffer-cap flag: %w", err)
		}
	}

	// AutomaticEnv generates OGLE_LOGBUFFERCAP from the camelCase key. Also
	// accept the more readable OGLE_LOG_BUFFER_CAP.
	if err := viper.BindEnv("logBufferCap", "OGLE_LOG_BUFFER_CAP"); err != nil {
		return fmt.Errorf("bind OGLE_LOG_BUFFER_CAP env: %w", err)
	}

	loaded, err := config.Load(viper.GetViper())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg = loaded

	return nil
}

// Root exposes the root command for the docgen tool.
func Root() *cobra.Command {
	return rootCmd
}
