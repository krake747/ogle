package settings_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/settings"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	testW = 100
	testH = 50
)

func newModel() settings.Model {
	return settings.New(theme.Default(), config.Defaults(), testW, testH)
}

func extractSettingsApplied(t *testing.T, cmd tea.Cmd) msgs.SettingsApplied {
	t.Helper()
	require.NotNil(t, cmd)
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	require.True(t, ok)
	require.Len(t, batch, 2)
	applied, ok := batch[0]().(msgs.SettingsApplied)
	require.True(t, ok)

	return applied
}

// ---------------------------------------------------------------------------
// TestUpdate — key up/down cycles focusField
// ---------------------------------------------------------------------------

func TestUpdate_UpDown_CyclesFocusField(t *testing.T) {
	t.Parallel()

	m := newModel()

	// Down: focusField 0 (Theme) → 1 (LogBufferCap).
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Nil(t, cmd)

	// Right on cap field → increments cap, theme unchanged.
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	applied := extractSettingsApplied(t, cmd)
	assert.Equal(t, "default", applied.Theme)
	assert.Equal(t, 1500, applied.LogBufferCap)

	// Down: focusField 1 → 0.
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	require.Nil(t, cmd)

	// Right on theme field → cycles theme, cap unchanged.
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	applied = extractSettingsApplied(t, cmd)
	assert.Equal(t, "default_light", applied.Theme)

	// Up: focusField 0 → 1.
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	require.Nil(t, cmd)

	// Left on cap field → decrements cap.
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	applied = extractSettingsApplied(t, cmd)
	assert.Equal(t, 1000, applied.LogBufferCap)
}

// ---------------------------------------------------------------------------
// TestUpdate — Theme field left/right
// ---------------------------------------------------------------------------

func TestUpdate_KeyRight_OnThemeField_CyclesTheme(t *testing.T) {
	t.Parallel()

	m := newModel()
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	applied := extractSettingsApplied(t, cmd)
	assert.Equal(t, "default_light", applied.Theme)
	assert.Equal(t, 1000, applied.LogBufferCap)
}

func TestUpdate_KeyLeft_OnThemeField_CyclesThemeReverse(t *testing.T) {
	t.Parallel()

	m := newModel()
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	applied := extractSettingsApplied(t, cmd)
	assert.Equal(t, "solarized_light", applied.Theme)
	assert.Equal(t, 1000, applied.LogBufferCap)
}

// ---------------------------------------------------------------------------
// TestUpdate — Log buffer cap field left/right
// ---------------------------------------------------------------------------

func TestUpdate_KeyRight_OnLogBufferCapField_Increments(t *testing.T) {
	t.Parallel()

	m := newModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	applied := extractSettingsApplied(t, cmd)
	assert.Equal(t, "default", applied.Theme)
	assert.Equal(t, 1500, applied.LogBufferCap)
}

func TestUpdate_KeyLeft_OnLogBufferCapField_Decrements(t *testing.T) {
	t.Parallel()

	m := newModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	applied := extractSettingsApplied(t, cmd)
	assert.Equal(t, "default", applied.Theme)
	assert.Equal(t, 500, applied.LogBufferCap)
}

func TestUpdate_KeyLeft_OnLogBufferCapField_ClampMin(t *testing.T) {
	t.Parallel()

	m := newModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	for range 5 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	}

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	applied := extractSettingsApplied(t, cmd)
	assert.Equal(t, 500, applied.LogBufferCap)
}

func TestUpdate_KeyRight_OnLogBufferCapField_ClampMax(t *testing.T) {
	t.Parallel()

	m := newModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	for range 10 {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	}

	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	applied := extractSettingsApplied(t, cmd)
	assert.Equal(t, 5000, applied.LogBufferCap)
}

// ---------------------------------------------------------------------------
// TestUpdate — Close keys: esc, q, comma
// ---------------------------------------------------------------------------

func TestUpdate_CloseKeys_EmitsVisibilityChanged(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		msg  tea.KeyPressMsg
	}

	cases := []testCase{
		{name: "esc", msg: tea.KeyPressMsg{Code: tea.KeyEsc}},
		{name: "q", msg: tea.KeyPressMsg{Text: "q"}},
		{name: "comma", msg: tea.KeyPressMsg{Text: ","}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel()
			_, cmd := m.Update(tc.msg)
			require.NotNil(t, cmd)
			msg := cmd()
			vc, ok := msg.(msgs.SettingsVisibilityChanged)
			require.True(t, ok)
			assert.False(t, vc.Visible)
		})
	}
}

// ---------------------------------------------------------------------------
// TestUpdate — Non-key messages
// ---------------------------------------------------------------------------

func TestUpdate_NonKeyMsg_NoCommand(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		msg  tea.Msg
	}

	cases := []testCase{
		{name: "WindowSizeMsg", msg: tea.WindowSizeMsg{Width: 200, Height: 100}},
		{name: "theme.Changed", msg: theme.Changed{Theme: theme.DefaultLight()}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel()
			_, cmd := m.Update(tc.msg)
			require.Nil(t, cmd)
		})
	}
}

// ---------------------------------------------------------------------------
// TestView
// ---------------------------------------------------------------------------

func TestView_RendersTitleAndFields(t *testing.T) {
	t.Parallel()

	m := newModel()
	content := m.View().Content

	assert.Contains(t, content, "Settings")
	assert.Contains(t, content, "Theme")
	assert.Contains(t, content, "default")
	assert.Contains(t, content, "Log Buffer Cap")
	assert.Contains(t, content, "1000")
	assert.Contains(t, content, "esc close")
}

func TestView_FocusedField_DifferentFromBlurred(t *testing.T) {
	t.Parallel()

	m := newModel()
	blurredContent := m.View().Content

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	focusedContent := m.View().Content

	assert.NotEqual(t, blurredContent, focusedContent,
		"focus change should alter border rendering")
}

func TestView_SavedCheckmark_Shown(t *testing.T) {
	t.Parallel()

	m := newModel()
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})

	content := m.View().Content
	assert.Contains(t, content, "✓")
}
