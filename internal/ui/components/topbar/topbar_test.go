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

func newModel(t *testing.T) (topbar.Model, *mocks.MockDocker) {
	t.Helper()
	mockD := mocks.NewMockDocker(t)
	mockD.EXPECT().Connect(mock.Anything).
		RunAndReturn(func(_ context.Context) tea.Cmd {
			return func() tea.Msg { return msgs.DaemonConnected{} }
		}).Maybe()
	return topbar.New(context.Background(), connection.New(), theme.Default(), mockD), mockD
}

//nolint:funlen
func TestUpdate(t *testing.T) {
	t.Parallel()

	early := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	later := time.Date(2020, 1, 1, 0, 0, 11, 0, time.UTC)

	type testCase struct {
		name      string
		msg       tea.Msg
		setup     func(m topbar.Model) topbar.Model
		assertion func(t *testing.T, m topbar.Model, cmd tea.Cmd)
	}

	cases := []testCase{
		{
			name: "TopbarContext sets phase and project file",
			msg:  msgs.TopbarContext{Phase: "dashboard", File: "docker-compose.yml"},
			assertion: func(t *testing.T, _ topbar.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
			},
		},
		{
			name: "DaemonConnected emits DaemonPoll tick",
			msg:  msgs.DaemonConnected{},
			assertion: func(t *testing.T, m topbar.Model, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd, "DaemonConnected should schedule a poll tick")
				assert.Contains(t, m.View().Content, "LIVE")
			},
		},
		{
			name: "DaemonUnavailable emits retry tick",
			setup: func(m topbar.Model) topbar.Model {
				m.SetNow(early)
				return m
			},
			msg: msgs.DaemonUnavailable{},
			assertion: func(t *testing.T, m topbar.Model, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd, "DaemonUnavailable should schedule a retry tick")
				assert.Contains(t, m.View().Content, "DISCONNECTED")
			},
		},
		{
			name: "DaemonGraceExpired in connecting state transitions to unavailable and emits retry tick",
			setup: func(m topbar.Model) topbar.Model {
				m.SetNow(early)
				return m
			},
			msg: msgs.DaemonGraceExpired{},
			assertion: func(t *testing.T, _ topbar.Model, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd, "Grace expired in connecting should schedule a retry tick")
			},
		},
		{
			name: "DaemonGraceExpired in connected state is no-op",
			setup: func(m topbar.Model) topbar.Model {
				m.SetNow(early)
				m.Update(msgs.DaemonConnected{})
				return m
			},
			msg: msgs.DaemonGraceExpired{},
			assertion: func(t *testing.T, _ topbar.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd, "Grace expired in connected should produce no command")
			},
		},
		{
			name: "DaemonTick when retry due calls docker.Connect",
			setup: func(m topbar.Model) topbar.Model {
				m.SetNow(early)
				m.Update(msgs.DaemonUnavailable{})
				m.SetNow(later)
				return m
			},
			msg: msgs.DaemonTick{},
			assertion: func(t *testing.T, m topbar.Model, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd, "retry-due DaemonTick should call docker.Connect")
				assert.Contains(t, m.View().Content, "RECONNECTING",
					"after retry, state should transition back to connecting")
			},
		},
		{
			name: "DaemonTick when retry NOT due emits next tick",
			setup: func(m topbar.Model) topbar.Model {
				m.SetNow(early)
				m.Update(msgs.DaemonUnavailable{})
				return m
			},
			msg: msgs.DaemonTick{},
			assertion: func(t *testing.T, _ topbar.Model, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd, "retry-not-due DaemonTick should schedule next tick")
			},
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

			tc.assertion(t, m, cmd)
		})
	}
}

//nolint:funlen
func TestView(t *testing.T) {
	t.Parallel()

	early := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	type testCase struct {
		name           string
		setup          func(m topbar.Model) topbar.Model
		expectedResult string
	}

	cases := []testCase{
		{
			name:           "startup phase shows scanning text",
			expectedResult: "scanning for compose files",
		},
		{
			name: "dashboard phase shows project file",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.TopbarContext{Phase: "dashboard", File: "compose.yaml"})
				return m
			},
			expectedResult: "compose.yaml",
		},
		{
			name: "watching phase shows disconnected text",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.TopbarContext{Phase: "watching", File: ""})
				return m
			},
			expectedResult: "disconnected",
		},
		{
			name:           "connecting daemon status shows RECONNECTING",
			expectedResult: "RECONNECTING",
		},
		{
			name: "connected daemon status shows LIVE",
			setup: func(m topbar.Model) topbar.Model {
				m, _ = m.Update(msgs.DaemonConnected{})
				return m
			},
			expectedResult: "LIVE",
		},
		{
			name: "unavailable daemon status shows DISCONNECTED with countdown",
			setup: func(m topbar.Model) topbar.Model {
				m.SetNow(early)
				m, _ = m.Update(msgs.DaemonUnavailable{})
				return m
			},
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
