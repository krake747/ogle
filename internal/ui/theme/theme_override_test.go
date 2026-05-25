package theme

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

func TestApplyOverrides(t *testing.T) {
	t.Parallel()

	original := Default()
	base := Default()

	overrides := userThemeFile{
		TextColor:            "#ff0000",
		StateRunningColor:    "#00ff00",
		HelpBackgroundColor:  "#0000ff",
	}

	result := applyOverrides(base, overrides)

	assert.Equal(t, lipgloss.Color("#ff0000"), result.Text)
	assert.Equal(t, lipgloss.Color("#00ff00"), result.StateRunning)
	assert.Equal(t, lipgloss.Color("#0000ff"), result.HelpBackground)

	assert.Equal(t, original.StateExited, result.StateExited)
	assert.Equal(t, original.StatePaused, result.StatePaused)
	assert.Equal(t, original.Subtext, result.Subtext)
}

func TestApplyOverridesEmptyFieldsDoNotOverride(t *testing.T) {
	t.Parallel()

	base := Default()

	overrides := userThemeFile{
		TextColor: "#ff0000",
	}

	result := applyOverrides(base, overrides)

	assert.Equal(t, lipgloss.Color("#ff0000"), result.Text)
	assert.Equal(t, base.StateRunning, result.StateRunning)
}

func TestApplyColorOverrides(t *testing.T) {
	t.Parallel()

	base := Default()

	overrides := userThemeFile{
		TextColor:           "#ff0000",
		SubtextColor:        "#00ff00",
		StateRunningColor:   "#0000ff",
		StateExitedColor:    "#ffff00",
		StatePausedColor:    "#ff00ff",
		StateTransientColor: "#00ffff",
		StateMutedColor:     "#ffffff",
	}

	result := *base
	applyColorOverrides(&result, overrides)

	assert.Equal(t, lipgloss.Color("#ff0000"), result.Text)
	assert.Equal(t, lipgloss.Color("#00ff00"), result.Subtext)
	assert.Equal(t, lipgloss.Color("#0000ff"), result.StateRunning)
	assert.Equal(t, lipgloss.Color("#ffff00"), result.StateExited)
	assert.Equal(t, lipgloss.Color("#ff00ff"), result.StatePaused)
	assert.Equal(t, lipgloss.Color("#00ffff"), result.StateTransient)
	assert.Equal(t, lipgloss.Color("#ffffff"), result.StateMuted)
}

func TestApplyStyleOverrides(t *testing.T) {
	t.Parallel()

	base := Default()

	overrides := userThemeFile{
		BorderFocusedColor:    "#ff0000",
		BorderBlurredColor:    "#00ff00",
		ServiceListTitleColor: "#0000ff",
		HelpKeyColor:          "#ffff00",
		HelpDescColor:         "#ff00ff",
		HelpSepColor:          "#00ffff",
	}

	result := *base
	applyStyleOverrides(&result, overrides)

	assert.Equal(t, lipgloss.Color("#ff0000"), result.BorderFocused.GetBorderTopForeground())
	assert.Equal(t, lipgloss.Color("#00ff00"), result.BorderBlurred.GetBorderTopForeground())
	assert.Equal(t, lipgloss.Color("#0000ff"), result.ServiceListTitle.GetForeground())
	assert.Equal(t, lipgloss.Color("#ffff00"), result.HelpKey.GetForeground())
	assert.Equal(t, lipgloss.Color("#ff00ff"), result.HelpDesc.GetForeground())
	assert.Equal(t, lipgloss.Color("#00ffff"), result.HelpSep.GetForeground())
}
