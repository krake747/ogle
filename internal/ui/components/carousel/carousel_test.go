package carousel_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/carousel"
	"github.com/ma-tf/ogle/internal/ui/components/carousel/card"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const serviceAlpha = "svc-alpha"

//nolint:gochecknoglobals // shared test fixtures
var (
	services3 = []domain.ServiceDef{
		{Name: serviceAlpha},
		{Name: "svc-beta"},
		{Name: "svc-gamma"},
	}

	services8 = []domain.ServiceDef{
		{Name: "svc-a"},
		{Name: "svc-b"},
		{Name: "svc-c"},
		{Name: "svc-d"},
		{Name: "svc-e"},
		{Name: "svc-f"},
		{Name: "svc-g"},
		{Name: "svc-h"},
	}
)

func newModel(t *testing.T, services []domain.ServiceDef) carousel.Model {
	t.Helper()

	return carousel.New(&domain.Project{Services: services}, 100, 50, theme.Default(), zone.New())
}

// ---------------------------------------------------------------------------
// TestInit
// ---------------------------------------------------------------------------

func TestInit(t *testing.T) {
	t.Parallel()

	t.Run("first card focused and ServiceSelected emitted", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services3)
		cmd := m.Init()
		require.NotNil(t, cmd)

		msg := cmd()
		batch, ok := msg.(tea.BatchMsg)
		require.True(t, ok)
		require.Len(t, batch, 2)

		focusMsg, ok := batch[0]().(card.FocusMsg)
		require.True(t, ok)
		assert.Equal(t, serviceAlpha, focusMsg.ServiceName)

		selMsg, ok := batch[1]().(msgs.ServiceSelected)
		require.True(t, ok)
		assert.Equal(t, serviceAlpha, selMsg.ServiceName)
	})

	t.Run("no cards returns nil cmd", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, nil)
		cmd := m.Init()
		require.Nil(t, cmd)
	})
}

// ---------------------------------------------------------------------------
// TestUpdate
// ---------------------------------------------------------------------------

//nolint:funlen
func TestUpdate(t *testing.T) {
	t.Parallel()

	t.Run("Tab cycles focus to next card slot", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services3)
		_ = m.Init()

		// Tab from focus=0 (svc-alpha) to focus=1 (svc-beta).
		_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		require.NotNil(t, cmd)

		msg := cmd()
		batch, ok := msg.(tea.BatchMsg)
		require.True(t, ok)
		require.Len(t, batch, 3)

		blurMsg, ok := batch[0]().(card.BlurMsg)
		require.True(t, ok)
		assert.Equal(t, serviceAlpha, blurMsg.ServiceName)

		focusMsg, ok := batch[1]().(card.FocusMsg)
		require.True(t, ok)
		assert.Equal(t, "svc-beta", focusMsg.ServiceName)

		selMsg, ok := batch[2]().(msgs.ServiceSelected)
		require.True(t, ok)
		assert.Equal(t, "svc-beta", selMsg.ServiceName)
	})

	t.Run("Enter on card without runtime emits ServiceStart", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services3)
		_ = m.Init()

		_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		require.NotNil(t, cmd)

		msg := cmd()
		startMsg, ok := msg.(msgs.ServiceStart)
		require.True(t, ok)
		assert.Equal(t, serviceAlpha, startMsg.ServiceName)
	})

	t.Run("Enter on running card emits ServiceStop", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services3)
		_ = m.Init()

		m, _ = m.Update(msgs.ServicesPolled{
			Runtimes: map[string]*domain.ServiceRuntimeData{
				serviceAlpha: {State: domain.ServiceStateRunning},
			},
		})

		_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		require.NotNil(t, cmd)

		msg := cmd()
		stopMsg, ok := msg.(msgs.ServiceStop)
		require.True(t, ok)
		assert.Equal(t, serviceAlpha, stopMsg.ServiceName)
	})

	t.Run("PgDown changes page and focuses first card on new page", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services8)
		_ = m.Init()

		_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
		require.NotNil(t, cmd)

		msg := cmd()
		batch, ok := msg.(tea.BatchMsg)
		require.True(t, ok)
		require.Len(t, batch, 2)

		focusMsg, ok := batch[0]().(card.FocusMsg)
		require.True(t, ok)
		assert.Equal(t, "svc-g", focusMsg.ServiceName)

		selMsg, ok := batch[1]().(msgs.ServiceSelected)
		require.True(t, ok)
		assert.Equal(t, "svc-g", selMsg.ServiceName)
	})

	t.Run("PgUp on first page returns no cmd", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services8)
		_ = m.Init()

		_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
		require.Nil(t, cmd)
	})

	t.Run("WindowSizeMsg returns no cmd", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services3)
		_, cmd := m.Update(tea.WindowSizeMsg{Width: 200, Height: 100})
		require.Nil(t, cmd)
	})

	t.Run("theme.Changed returns no cmd", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services3)
		_, cmd := m.Update(theme.Changed{Theme: theme.DefaultLight()})
		require.Nil(t, cmd)
	})

	t.Run("Enter on empty slot no-ops", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, nil)
		_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		require.Nil(t, cmd)
	})

	t.Run("ServicesPolled stores runtime data", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services3)
		m.Update(msgs.ServicesPolled{
			Runtimes: map[string]*domain.ServiceRuntimeData{
				serviceAlpha: {State: domain.ServiceStateRunning, ContainerID: "abc123"},
			},
		})
	})

	t.Run("Enter on dot changes page", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services8)
		_ = m.Init()

		// Navigate from focus=2 to dot 1 (slot 1, inactive).
		// With 2 pages dotCount=2, totalSlots=8. Tab from 2→3→4→5→6→7→0(skip)→1.
		for range 6 {
			m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		}

		_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		require.NotNil(t, cmd)

		msg := cmd()
		batch, ok := msg.(tea.BatchMsg)
		require.True(t, ok)
		require.Len(t, batch, 2)

		focusMsg, ok := batch[0]().(card.FocusMsg)
		require.True(t, ok)
		assert.Equal(t, "svc-g", focusMsg.ServiceName)

		selMsg, ok := batch[1]().(msgs.ServiceSelected)
		require.True(t, ok)
		assert.Equal(t, "svc-g", selMsg.ServiceName)
	})
}

// ---------------------------------------------------------------------------
// TestView
// ---------------------------------------------------------------------------

func TestView(t *testing.T) {
	t.Parallel()

	t.Run("card grid shows service names", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services3)
		_ = m.Init()

		content := m.View().Content
		assert.Contains(t, content, serviceAlpha[:7])
		assert.Contains(t, content, "svc-beta")
		assert.Contains(t, content, "svc-gamma"[:7])
	})

	t.Run("nav bar hidden when single page", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services3)
		content := m.View().Content
		assert.NotContains(t, content, "•")
		assert.NotContains(t, content, "○")
	})

	t.Run("nav bar shown when multiple pages", func(t *testing.T) {
		t.Parallel()

		m := newModel(t, services8)
		content := m.View().Content
		assert.Contains(t, content, "•")
		assert.Contains(t, content, "○")
	})
}
