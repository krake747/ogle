package topbar_test

import (
	"context"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
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

func newModel(t *testing.T) (topbar.Model, *mocks.MockDocker) {
	t.Helper()
	mockD := mocks.NewMockDocker(t)
	mockD.EXPECT().Connect(mock.Anything).
		RunAndReturn(func(_ context.Context) tea.Cmd {
			return func() tea.Msg { return msgs.DaemonConnected{} }
		}).Maybe()
	return topbar.New(context.Background(), connection.New(), theme.Default(), mockD), mockD
}

func cmdRequired(t *testing.T, _ topbar.Model, cmd tea.Cmd) {
	t.Helper()
	require.NotNil(t, cmd, "expected a command to be returned")
}

//nolint:funlen
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
		check       func(*testing.T, topbar.Model, tea.Cmd)
	}

	cases := []testCase{
		{
			name: "TopbarContext produces no command",
			msg:  msgs.TopbarContext{Phase: "dashboard", File: "docker-compose.yml"},
		},
		{
			name:  "DaemonConnected schedules poll command",
			msg:   msgs.DaemonConnected{},
			check: cmdRequired,
		},
		{
			name: "DaemonUnavailable schedules retry command",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				m.SetNow(early)
				return m
			},
			// act
			msg: msgs.DaemonUnavailable{},
			// assert
			check: cmdRequired,
		},
		{
			name: "DaemonGraceExpired while connecting schedules retry command",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				m.SetNow(early)
				return m
			},
			// act
			msg: msgs.DaemonGraceExpired{},
			// assert
			check: cmdRequired,
		},
		{
			name: "DaemonGraceExpired while connected produces no command",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				m.SetNow(early)
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
				m.SetNow(early)
				m.Update(msgs.DaemonUnavailable{})
				m.SetNow(later)
				return m
			},
			// act
			msg: msgs.DaemonTick{},
			// assert
			check: cmdRequired,
		},
		{
			name: "DaemonTick with retry not due schedules next tick",
			// arrange
			setup: func(m topbar.Model) topbar.Model {
				m.SetNow(early)
				m.Update(msgs.DaemonUnavailable{})
				return m
			},
			// act
			msg: msgs.DaemonTick{},
			// assert
			check: cmdRequired,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, _ := newModel(t)
			if tc.setup != nil {
				m = tc.setup(m)
			}

			m, cmd := m.Update(tc.msg)

			if tc.expectedMsg != nil {
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			} else if tc.check != nil {
				tc.check(t, m, cmd)
			} else {
				require.Nil(t, cmd)
			}
		})
	}
}

//nolint:funlen
func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(m topbar.Model) topbar.Model

		// assert
		expectedResult string
	}

	cases := []testCase{
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
				m.SetNow(early)
				m, _ = m.Update(msgs.DaemonUnavailable{})
				return m
			},
			// assert
			expectedResult: "DISCONNECTED",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, _ := newModel(t)
			m.SetNow(early)
			_ = m.Init()

			if tc.setup != nil {
				m = tc.setup(m)
			}

			if tc.expectedResult == "" {
				assert.Empty(t, m.View().Content)
			} else {
				assert.Contains(t, m.View().Content, tc.expectedResult)
			}
		})
	}
}
