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

const svcAlpha = "svc-alpha"

func testServices3() []domain.ServiceDef {
	return []domain.ServiceDef{{Name: svcAlpha}, {Name: "svc-beta"}, {Name: "svc-gamma"}}
}

func testServices8() []domain.ServiceDef {
	return []domain.ServiceDef{
		{Name: "svc-a"},
		{Name: "svc-b"},
		{Name: "svc-c"},
		{Name: "svc-d"},
		{Name: "svc-e"},
		{Name: "svc-f"},
		{Name: "svc-g"},
		{Name: "svc-h"},
	}
}

func newModel(t *testing.T, services []domain.ServiceDef) carousel.Model {
	t.Helper()

	return carousel.New(&domain.Project{Services: services}, 100, 50, theme.Default(), zone.New())
}

// ---------------------------------------------------------------------------
// TestInit
// ---------------------------------------------------------------------------

func TestInit(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		services []domain.ServiceDef
		// assert
		expectCmd    bool
		expectedName string
	}

	cases := []testCase{
		{
			name:         "first card focused and ServiceSelected emitted",
			services:     testServices3(),
			expectCmd:    true,
			expectedName: svcAlpha,
		},
		{
			name:      "no cards returns nil cmd",
			services:  nil,
			expectCmd: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel(t, tc.services)
			cmd := m.Init()

			if !tc.expectCmd {
				require.Nil(t, cmd)

				return
			}

			require.NotNil(t, cmd)

			msg := cmd()
			batch, ok := msg.(tea.BatchMsg)
			require.True(t, ok)
			require.Len(t, batch, 2)

			focusMsg, ok := batch[0]().(card.FocusMsg)
			require.True(t, ok)
			assert.Equal(t, tc.expectedName, focusMsg.ServiceName)

			selMsg, ok := batch[1]().(msgs.ServiceSelected)
			require.True(t, ok)
			assert.Equal(t, tc.expectedName, selMsg.ServiceName)
		})
	}
}

// ---------------------------------------------------------------------------
// TestUpdate
// ---------------------------------------------------------------------------

func TestUpdate(t *testing.T) { //nolint:funlen // long table-driven test cases
	t.Parallel()

	type testCase struct {
		name     string
		services []domain.ServiceDef
		setup    func(m carousel.Model) carousel.Model
		msg      tea.Msg
		// assert
		assert func(t *testing.T, m carousel.Model, cmd tea.Cmd)
	}

	cases := []testCase{
		{
			name:     "Tab cycles focus to next card slot",
			services: testServices3(),
			setup: func(m carousel.Model) carousel.Model {
				_ = m.Init()

				return m
			},
			msg: tea.KeyPressMsg{Code: tea.KeyTab},
			assert: func(t *testing.T, _ carousel.Model, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)

				msg := cmd()
				batch, ok := msg.(tea.BatchMsg)
				require.True(t, ok)
				require.Len(t, batch, 3)

				blurMsg, ok := batch[0]().(card.BlurMsg)
				require.True(t, ok)
				assert.Equal(t, svcAlpha, blurMsg.ServiceName)

				focusMsg, ok := batch[1]().(card.FocusMsg)
				require.True(t, ok)
				assert.Equal(t, "svc-beta", focusMsg.ServiceName)

				selMsg, ok := batch[2]().(msgs.ServiceSelected)
				require.True(t, ok)
				assert.Equal(t, "svc-beta", selMsg.ServiceName)
			},
		},
		{
			name:     "Enter on card without runtime emits ServiceStart",
			services: testServices3(),
			setup: func(m carousel.Model) carousel.Model {
				_ = m.Init()

				return m
			},
			msg: tea.KeyPressMsg{Code: tea.KeyEnter},
			assert: func(t *testing.T, _ carousel.Model, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)

				msg := cmd()
				startMsg, ok := msg.(msgs.ServiceStart)
				require.True(t, ok)
				assert.Equal(t, svcAlpha, startMsg.ServiceName)
			},
		},
		{
			name:     "Enter on running card emits ServiceStop",
			services: testServices3(),
			setup: func(m carousel.Model) carousel.Model {
				_ = m.Init()

				m, _ = m.Update(msgs.ServicesPolled{
					Runtimes: map[string]*domain.ServiceRuntimeData{
						svcAlpha: {State: domain.ServiceStateRunning},
					},
				})

				return m
			},
			msg: tea.KeyPressMsg{Code: tea.KeyEnter},
			assert: func(t *testing.T, _ carousel.Model, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)

				msg := cmd()
				stopMsg, ok := msg.(msgs.ServiceStop)
				require.True(t, ok)
				assert.Equal(t, svcAlpha, stopMsg.ServiceName)
			},
		},
		{
			name:     "PgDown changes page and focuses first card on new page",
			services: testServices8(),
			setup: func(m carousel.Model) carousel.Model {
				_ = m.Init()

				return m
			},
			msg: tea.KeyPressMsg{Code: tea.KeyPgDown},
			assert: func(t *testing.T, _ carousel.Model, cmd tea.Cmd) {
				t.Helper()
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
			},
		},
		{
			name:     "PgUp on first page returns no cmd",
			services: testServices8(),
			setup: func(m carousel.Model) carousel.Model {
				_ = m.Init()

				return m
			},
			msg:    tea.KeyPressMsg{Code: tea.KeyPgUp},
			assert: func(t *testing.T, _ carousel.Model, cmd tea.Cmd) { t.Helper(); require.Nil(t, cmd) },
		},
		{
			name:     "WindowSizeMsg returns no cmd",
			services: testServices3(),
			setup:    nil,
			msg:      tea.WindowSizeMsg{Width: 200, Height: 100},
			assert:   func(t *testing.T, _ carousel.Model, cmd tea.Cmd) { t.Helper(); require.Nil(t, cmd) },
		},
		{
			name:     "theme.Changed returns no cmd",
			services: testServices3(),
			setup:    nil,
			msg:      theme.Changed{Theme: theme.DefaultLight()},
			assert:   func(t *testing.T, _ carousel.Model, cmd tea.Cmd) { t.Helper(); require.Nil(t, cmd) },
		},
		{
			name:     "Enter on empty slot no-ops",
			services: nil,
			setup: func(m carousel.Model) carousel.Model {
				_ = m.Init()

				return m
			},
			msg:    tea.KeyPressMsg{Code: tea.KeyEnter},
			assert: func(t *testing.T, _ carousel.Model, cmd tea.Cmd) { t.Helper(); require.Nil(t, cmd) },
		},
		{
			name:     "ServicesPolled stores runtime data",
			services: testServices3(),
			setup:    nil,
			msg: msgs.ServicesPolled{
				Runtimes: map[string]*domain.ServiceRuntimeData{
					svcAlpha: {State: domain.ServiceStateRunning, ContainerID: "abc123"},
				},
			},
			assert: func(t *testing.T, _ carousel.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
			},
		},
		{
			name:     "Enter on dot changes page",
			services: testServices8(),
			setup: func(m carousel.Model) carousel.Model {
				_ = m.Init()

				// Navigate from focus=2 to dot 1 (slot 1, inactive).
				// With 2 pages dotCount=2, totalSlots=8.
				// Tab from 2->3->4->5->6->7->0(skip)->1.
				for range 6 {
					m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
				}

				return m
			},
			msg: tea.KeyPressMsg{Code: tea.KeyEnter},
			assert: func(t *testing.T, _ carousel.Model, cmd tea.Cmd) {
				t.Helper()
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
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel(t, tc.services)

			if tc.setup != nil {
				m = tc.setup(m)
			}

			m, cmd := m.Update(tc.msg)
			_ = m

			tc.assert(t, m, cmd)
		})
	}
}

// ---------------------------------------------------------------------------
// TestView
// ---------------------------------------------------------------------------

func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name           string
		services       []domain.ServiceDef
		setup          func(m carousel.Model) carousel.Model
		expectedResult string
	}

	cases := []testCase{
		{
			name:           "card grid shows service names",
			services:       testServices3(),
			setup:          nil,
			expectedResult: "svc-beta",
		},
		{
			name:           "nav bar hidden when single page",
			services:       testServices3(),
			setup:          nil,
			expectedResult: "",
		},
		{
			name:           "nav bar shown when multiple pages",
			services:       testServices8(),
			setup:          nil,
			expectedResult: "•",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel(t, tc.services)

			if tc.setup != nil {
				m = tc.setup(m)
			}

			if tc.expectedResult == "" {
				assert.NotContains(t, m.View().Content, "•")
				assert.NotContains(t, m.View().Content, "○")
			} else {
				assert.Contains(t, m.View().Content, tc.expectedResult)
			}
		})
	}
}
