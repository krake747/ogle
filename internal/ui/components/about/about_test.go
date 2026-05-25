package about_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/ui/components/about"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

func TestInit(t *testing.T) {
	t.Parallel()

	m := about.New(theme.Default())
	cmd := m.Init()
	require.Nil(t, cmd)
}

func TestView_RendersVersionInfo(t *testing.T) {
	t.Parallel()

	m := about.New(theme.Default())
	content := m.View().Content

	assert.Contains(t, content, "ogle")
	assert.Contains(t, content, "dev")
	assert.Contains(t, content, "github.com/ma-tf/ogle")
	assert.Contains(t, content, "F1 / esc / q to close")
}

func TestView_DoesNotPanic(t *testing.T) {
	t.Parallel()

	m := about.New(theme.Default())

	require.NotPanics(t, func() { m.View() })
}

func TestUpdate_WindowSizeMsg_NoCommand(t *testing.T) {
	t.Parallel()

	m := about.New(theme.Default())
	_, cmd := m.Update(tea.WindowSizeMsg{Width: 200, Height: 100})
	require.Nil(t, cmd)
}

func TestUpdate_ThemeChanged_NoCommand(t *testing.T) {
	t.Parallel()

	m := about.New(theme.Default())
	_, cmd := m.Update(theme.Changed{Theme: theme.DefaultLight()})
	require.Nil(t, cmd)
}

func TestUpdate_ThemeChanged_ChangesAppearance(t *testing.T) {
	t.Parallel()

	m := about.New(theme.Default())
	viewBefore := m.View().Content

	m, _ = m.Update(theme.Changed{Theme: theme.DefaultLight()})
	viewAfter := m.View().Content

	assert.NotEqual(t, viewBefore, viewAfter,
		"different theme should produce different ANSI-rendered output")
}
