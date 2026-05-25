package theme_test

import (
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

func TestColourForState(t *testing.T) {
	t.Parallel()

	th := theme.Default()

	type testCase struct {
		name          string
		state         domain.ServiceState
		expectedColor color.Color
	}

	for _, tc := range []testCase{
		{name: "running", state: domain.ServiceStateRunning, expectedColor: th.StateRunning},
		{name: "exited", state: domain.ServiceStateExited, expectedColor: th.StateExited},
		{name: "dead", state: domain.ServiceStateDead, expectedColor: th.StateExited},
		{name: "paused", state: domain.ServiceStatePaused, expectedColor: th.StatePaused},
		{name: "restarting", state: domain.ServiceStateRestarting, expectedColor: th.StateTransient},
		{name: "not created", state: domain.ServiceStateNotCreated, expectedColor: th.StateMuted},
		{name: "unknown", state: domain.ServiceStateUnknown, expectedColor: th.StateMuted},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := th.ColourForState(tc.state)
			assert.Equal(t, tc.expectedColor, got)
		})
	}
}

func TestBuiltinConstructors(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name      string
		construct func() *theme.Theme
	}

	for _, tc := range []testCase{
		{name: "Default", construct: theme.Default},
		{name: "DefaultLight", construct: theme.DefaultLight},
		{name: "CatppuccinoFrappe", construct: theme.CatppuccinoFrappe},
		{name: "CatppuccinoLatte", construct: theme.CatppuccinoLatte},
		{name: "CatppuccinoMacchiato", construct: theme.CatppuccinoMacchiato},
		{name: "CatppuccinoMocha", construct: theme.CatppuccinoMocha},
		{name: "SolarizedDark", construct: theme.SolarizedDark},
		{name: "SolarizedLight", construct: theme.SolarizedLight},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			th := tc.construct()
			require.NotNil(t, th)
		})
	}
}

func TestBuiltinNames(t *testing.T) {
	t.Parallel()

	names := theme.BuiltinNames()
	require.Len(t, names, 8)
	expected := []string{
		"default", "default_light",
		"catppuccino_frappe", "catppuccino_latte",
		"catppuccino_macchiato", "catppuccino_mocha",
		"solarized_dark", "solarized_light",
	}
	assert.ElementsMatch(t, expected, names)
}

func TestLoadBuiltin(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name      string
		themeName string
	}

	for _, tc := range []testCase{
		{name: "empty string returns default", themeName: ""},
		{name: "default", themeName: "default"},
		{name: "default_light", themeName: "default_light"},
		{name: "catppuccino_frappe", themeName: "catppuccino_frappe"},
		{name: "catppuccino_latte", themeName: "catppuccino_latte"},
		{name: "catppuccino_macchiato", themeName: "catppuccino_macchiato"},
		{name: "catppuccino_mocha", themeName: "catppuccino_mocha"},
		{name: "solarized_dark", themeName: "solarized_dark"},
		{name: "solarized_light", themeName: "solarized_light"},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			th, err := theme.Load(tc.themeName, t.TempDir())
			require.NoError(t, err)
			require.NotNil(t, th)
		})
	}
}

func TestLoadUnknownTheme(t *testing.T) {
	t.Parallel()

	th, err := theme.Load("nonexistent-theme", t.TempDir())
	require.Error(t, err)
	require.ErrorIs(t, err, theme.ErrUnknownTheme)
	require.NotNil(t, th)
}
