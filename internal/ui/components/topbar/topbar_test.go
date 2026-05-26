package topbar_test

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/docker/connection"
	"github.com/ma-tf/ogle/internal/services/docker/mocks"
	"github.com/ma-tf/ogle/internal/ui/components/topbar"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

//nolint:gochecknoglobals // shared test time fixtures
var (
	early = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	later = time.Date(2020, 1, 1, 0, 0, 11, 0, time.UTC)
)

func newModel(t *testing.T) topbar.Model {
	t.Helper()
	mockD := mocks.NewMockDocker(t)
	mockD.EXPECT().Connect(mock.Anything).
		RunAndReturn(func(_ context.Context) tea.Cmd {
			return func() tea.Msg { return msgs.DaemonConnected{} }
		}).Maybe()

	return topbar.New(context.Background(), connection.New(), theme.Default(), mockD, zone.New())
}

func TestInit(t *testing.T) {
	t.Parallel()

	m := newModel(t)
	cmd := m.Init()
	require.NotNil(t, cmd)
}

//nolint:funlen // table-driven test with many cases
func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(m topbar.Model) topbar.Model

		// act
		msg tea.Msg

		// assert
		expectedMsg tea.Msg
		expectCmd   bool
	}

	cases := []testCase{
		{
			name: "TopbarContext produces no command",
			msg:  msgs.TopbarContext{Phase: "dashboard", File: "docker-compose.yml"},
		},
		{
			name:      "DaemonConnected schedules poll command",
			msg:       msgs.DaemonConnected{},
			expectCmd: true,
		},
		{
			name: "DaemonUnavailable schedules retry command",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				topbar.SetNow(&m, early)

				return m
			},
			// act
			msg: msgs.DaemonUnavailable{},
			// assert
			expectCmd: true,
		},
		{
			name: "DaemonGraceExpired while connecting schedules retry command",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				topbar.SetNow(&m, early)

				return m
			},
			// act
			msg: msgs.DaemonGraceExpired{},
			// assert
			expectCmd: true,
		},
		{
			name: "DaemonGraceExpired while connected produces no command",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				topbar.SetNow(&m, early)
				m.Update(msgs.DaemonConnected{})

				return m
			},
			// act
			msg: msgs.DaemonGraceExpired{},
		},
		{
			name: "DaemonTick with retry due triggers reconnect",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				topbar.SetNow(&m, early)
				m.Update(msgs.DaemonUnavailable{})
				topbar.SetNow(&m, later)

				return m
			},
			// act
			msg: msgs.DaemonTick{},
			// assert
			expectCmd: true,
		},
		{
			name: "DaemonTick with retry not due schedules next tick",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				topbar.SetNow(&m, early)
				m.Update(msgs.DaemonUnavailable{})

				return m
			},
			// act
			msg: msgs.DaemonTick{},
			// assert
			expectCmd: true,
		},
		{
			name: "DaemonPoll while Connected fires docker.Connect",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.DaemonConnected{})

				return m
			},
			// act
			msg: msgs.DaemonPoll{},
			// assert
			expectedMsg: msgs.DaemonConnected{},
		},
		{
			name: "DaemonPoll while not Connected produces no command",
			// act
			msg: msgs.DaemonPoll{},
		},
		{
			name: "Post-switch IsRetryDue fires docker.Connect when retry is due",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				topbar.SetNow(&m, early)
				m, _ = m.Update(msgs.DaemonUnavailable{})
				topbar.SetNow(&m, later)

				return m
			},
			// act
			msg: msgs.TopbarContext{Phase: "startup"},
			// assert
			expectedMsg: msgs.DaemonConnected{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel(t)
			if tc.setup != nil {
				m = tc.setup(m)
			}

			_, cmd := m.Update(tc.msg)

			switch {
			case tc.expectedMsg != nil:
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			case tc.expectCmd:
				require.NotNil(t, cmd)
			default:
				require.Nil(t, cmd)
			}
		})
	}
}

//nolint:funlen // table-driven test with many cases
func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(m topbar.Model) topbar.Model

		// assert
		expectedResult string
		assert         func(t *testing.T, m topbar.Model)
	}

	cases := []testCase{
		{
			name:           "startup phase shows ogle brand and scanning text",
			expectedResult: "ogle",
		},
		{
			name:           "startup phase shows scanning text",
			expectedResult: "scanning for compose files",
		},
		{
			name: "dashboard phase shows project file",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.TopbarContext{Phase: "dashboard", File: "compose.yaml"})

				return m
			},
			// assert
			expectedResult: "compose.yaml",
		},
		{
			name: "watching phase shows disconnected text",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.TopbarContext{Phase: "watching", File: ""})

				return m
			},
			// assert
			expectedResult: "disconnected",
		},
		{
			name:           "connecting daemon status shows RECONNECTING",
			expectedResult: "RECONNECTING",
		},
		{
			name: "connected daemon status shows LIVE",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.DaemonConnected{})

				return m
			},
			// assert
			expectedResult: "LIVE",
		},
		{
			name: "unavailable daemon status shows DISCONNECTED with countdown",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				topbar.SetNow(&m, early)
				m, _ = m.Update(msgs.DaemonUnavailable{})

				return m
			},
			// assert
			expectedResult: "DISCONNECTED",
		},
		{
			name: "default (unknown) phase renders empty context",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				topbar.SetPhase(&m, topbar.Phase(99))

				return m
			},
			// assert
			assert: func(t *testing.T, m topbar.Model) {
				t.Helper()

				view := m.View()

				assert.NotContains(t, view.Content, "scanning for compose files")
				assert.NotContains(t, view.Content, "disconnected")
				assert.Contains(t, view.Content, "ogle")
				assert.Contains(t, view.Content, "RECONNECTING")
			},
		},
		{
			name: "default (unknown) connection state renders empty status",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				topbar.SetConnectState(&m, connection.ConnectState(99))

				return m
			},
			// assert
			assert: func(t *testing.T, m topbar.Model) {
				t.Helper()

				view := m.View()

				assert.NotContains(t, view.Content, "RECONNECTING")
				assert.NotContains(t, view.Content, "LIVE")
				assert.NotContains(t, view.Content, "DISCONNECTED")
				assert.Contains(t, view.Content, "ogle")
				assert.Contains(t, view.Content, "scanning for compose files")
			},
		},
		{
			name: "brand zone marker present in View output",
			// assert
			assert: func(t *testing.T, m topbar.Model) {
				t.Helper()

				zm := topbar.GetZM(&m)
				view := m.View()
				zm.Scan(view.Content)

				require.Eventually(t, func() bool {
					zi := zm.Get(topbar.BrandZone)

					return zi != nil && !zi.IsZero()
				}, time.Second, 10*time.Millisecond)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel(t)
			topbar.SetNow(&m, early)
			_ = m.Init()

			if tc.setup != nil {
				m = tc.setup(m)
			}

			switch {
			case tc.assert != nil:
				tc.assert(t, m)
			case tc.expectedResult == "":
				assert.Empty(t, m.View().Content)
			default:
				assert.Contains(t, m.View().Content, tc.expectedResult)
			}
		})
	}
}
