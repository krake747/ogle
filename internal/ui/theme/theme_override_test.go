package theme //nolint:testpackage // internal function tests require access to unexported helpers

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

const (
	testColorRed   = "#ff0000"
	testColorGreen = "#00ff00"
	testColorBlue  = "#0000ff"
)

func TestApplyOverrides(t *testing.T) {
	t.Parallel()

	original := Default()
	base := Default()

	overrides := userThemeFile{
		TextColor:           testColorRed,
		StateRunningColor:   testColorGreen,
		HelpBackgroundColor: testColorBlue,
	}

	result := applyOverrides(base, overrides)

	assert.Equal(t, lipgloss.Color(testColorRed), result.Text)
	assert.Equal(t, lipgloss.Color(testColorGreen), result.StateRunning)
	assert.Equal(t, lipgloss.Color(testColorBlue), result.HelpBackground)

	assert.Equal(t, original.StateExited, result.StateExited)
	assert.Equal(t, original.StatePaused, result.StatePaused)
	assert.Equal(t, original.Subtext, result.Subtext)
}

func TestApplyOverridesEmptyFieldsDoNotOverride(t *testing.T) {
	t.Parallel()

	base := Default()

	overrides := userThemeFile{
		TextColor: testColorRed,
	}

	result := applyOverrides(base, overrides)

	assert.Equal(t, lipgloss.Color(testColorRed), result.Text)
	assert.Equal(t, base.StateRunning, result.StateRunning)
}

func TestApplyColorOverrides(t *testing.T) {
	t.Parallel()

	base := Default()

	overrides := userThemeFile{
		TextColor:           testColorRed,
		SubtextColor:        testColorGreen,
		StateRunningColor:   testColorBlue,
		StateExitedColor:    "#ffff00",
		StatePausedColor:    "#ff00ff",
		StateTransientColor: "#00ffff",
		StateMutedColor:     "#ffffff",
	}

	result := *base
	applyColorOverrides(&result, overrides)

	assert.Equal(t, lipgloss.Color(testColorRed), result.Text)
	assert.Equal(t, lipgloss.Color(testColorGreen), result.Subtext)
	assert.Equal(t, lipgloss.Color(testColorBlue), result.StateRunning)
	assert.Equal(t, lipgloss.Color("#ffff00"), result.StateExited)
	assert.Equal(t, lipgloss.Color("#ff00ff"), result.StatePaused)
	assert.Equal(t, lipgloss.Color("#00ffff"), result.StateTransient)
	assert.Equal(t, lipgloss.Color("#ffffff"), result.StateMuted)
}

func TestApplyStyleOverrides(t *testing.T) {
	t.Parallel()

	base := Default()

	overrides := userThemeFile{
		BorderFocusedColor:    testColorRed,
		BorderBlurredColor:    testColorGreen,
		ServiceListTitleColor: testColorBlue,
		HelpKeyColor:          "#ffff00",
		HelpDescColor:         "#ff00ff",
		HelpSepColor:          "#00ffff",
	}

	result := *base
	applyStyleOverrides(&result, overrides)

	assert.Equal(t, lipgloss.Color(testColorRed), result.BorderFocused.GetBorderTopForeground())
	assert.Equal(t, lipgloss.Color(testColorGreen), result.BorderBlurred.GetBorderTopForeground())
	assert.Equal(t, lipgloss.Color(testColorBlue), result.ServiceListTitle.GetForeground())
	assert.Equal(t, lipgloss.Color("#ffff00"), result.HelpKey.GetForeground())
	assert.Equal(t, lipgloss.Color("#ff00ff"), result.HelpDesc.GetForeground())
	assert.Equal(t, lipgloss.Color("#00ffff"), result.HelpSep.GetForeground())
}
