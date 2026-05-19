// Package theme defines the Theme type, built-in themes, and user theme loading.
// lipgloss must not be imported outside the UI layer; this package is the
// single source of all style definitions.
package theme

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"

	"charm.land/lipgloss/v2"
	"go.yaml.in/yaml/v3"
)

// ErrUnknownTheme is returned by Load when the name does not match any
// built-in theme and no user file exists for it.
var ErrUnknownTheme = errors.New("unknown theme")

// Theme holds the complete set of themeable style values for the UI layer.
// BorderFocused and BorderBlurred pre-compose lipgloss.NormalBorder(); call
// sites extend with Width/Height only.
type Theme struct {
	BorderFocused         lipgloss.Style
	BorderBlurred         lipgloss.Style
	ServiceListTitle      lipgloss.Style
	ServiceListBackground color.Color
	SelectedBackground    color.Color
	HoverBackground       color.Color
	StateRunning          color.Color // running
	StateExited           color.Color // exited / dead
	StatePaused           color.Color // paused
	StateTransient        color.Color // restarting, action in-flight
	StateMuted            color.Color // not created, unknown, nil runtime
	ActionError           color.Color // error suffix text
}

// userThemeFile is the YAML schema for a user-defined theme override file.
type userThemeFile struct {
	Base                       string `yaml:"base"`
	BorderFocusedColor         string `yaml:"borderFocusedColor"`
	BorderBlurredColor         string `yaml:"borderBlurredColor"`
	ServiceListTitleColor      string `yaml:"serviceListTitleColor"`
	ServiceListBackgroundColor string `yaml:"serviceListBackgroundColor"`
	SelectedBackgroundColor    string `yaml:"selectedBackgroundColor"`
	HoverBackgroundColor       string `yaml:"hoverBackgroundColor"`
	StateRunningColor          string `yaml:"stateRunningColor"`
	StateExitedColor           string `yaml:"stateExitedColor"`
	StatePausedColor           string `yaml:"statePausedColor"`
	StateTransientColor        string `yaml:"stateTransientColor"`
	StateMutedColor            string `yaml:"stateMutedColor"`
	ActionErrorColor           string `yaml:"actionErrorColor"`
}

// Load resolves a theme by name. configDir is the directory containing
// config.yaml (typically ~/.ogle).
//
// Resolution order:
//  1. configDir/themes/<name>.yaml — user-defined theme file
//  2. Built-in theme with the given name
//
// On any resolution failure Load returns Default() and a descriptive error.
// Callers should log the error at Warn level and continue.
func Load(name, configDir string) (*Theme, error) {
	path := filepath.Join(configDir, "themes", name+".yaml")

	data, err := os.ReadFile(path)
	if err == nil {
		var f userThemeFile
		if yamlErr := yaml.Unmarshal(data, &f); yamlErr != nil {
			return Default(), fmt.Errorf("parse theme file %q: %w", path, yamlErr)
		}

		base := builtinByName(f.Base)
		if base == nil {
			base = Default()
		}

		return applyOverrides(base, f), nil
	}

	t := builtinByName(name)
	if t != nil {
		return t, nil
	}

	return Default(), fmt.Errorf("%q: %w", name, ErrUnknownTheme)
}

func builtinByName(name string) *Theme {
	switch name {
	case "default", "":
		return Default()
	case "catppuccino_mocha":
		return CatppuccinoMocha()
	default:
		return nil
	}
}

func applyOverrides(t *Theme, f userThemeFile) *Theme {
	result := *t

	if f.BorderFocusedColor != "" {
		result.BorderFocused = result.BorderFocused.BorderForeground(
			lipgloss.Color(f.BorderFocusedColor),
		)
	}

	if f.BorderBlurredColor != "" {
		result.BorderBlurred = result.BorderBlurred.BorderForeground(
			lipgloss.Color(f.BorderBlurredColor),
		)
	}

	if f.ServiceListTitleColor != "" {
		result.ServiceListTitle = result.ServiceListTitle.Foreground(
			lipgloss.Color(f.ServiceListTitleColor),
		)
	}

	if f.ServiceListBackgroundColor != "" {
		result.ServiceListBackground = lipgloss.Color(f.ServiceListBackgroundColor)
	}

	if f.SelectedBackgroundColor != "" {
		result.SelectedBackground = lipgloss.Color(f.SelectedBackgroundColor)
	}

	if f.HoverBackgroundColor != "" {
		result.HoverBackground = lipgloss.Color(f.HoverBackgroundColor)
	}

	if f.StateRunningColor != "" {
		result.StateRunning = lipgloss.Color(f.StateRunningColor)
	}

	if f.StateExitedColor != "" {
		result.StateExited = lipgloss.Color(f.StateExitedColor)
	}

	if f.StatePausedColor != "" {
		result.StatePaused = lipgloss.Color(f.StatePausedColor)
	}

	if f.StateTransientColor != "" {
		result.StateTransient = lipgloss.Color(f.StateTransientColor)
	}

	if f.StateMutedColor != "" {
		result.StateMuted = lipgloss.Color(f.StateMutedColor)
	}

	if f.ActionErrorColor != "" {
		result.ActionError = lipgloss.Color(f.ActionErrorColor)
	}

	return &result
}
