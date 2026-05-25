package servicehost_test

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	logsmocks "github.com/ma-tf/ogle/internal/services/docker/logs/mocks"
	"github.com/ma-tf/ogle/internal/ui/components/servicehost"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	testProject = "testproj"
	svcName     = "web"
)

var svcDef = domain.ServiceDef{Name: svcName} //nolint:gochecknoglobals // shared test fixture

func newModel(t *testing.T) (servicehost.Model, *logsmocks.MockStreamer) {
	t.Helper()

	s := logsmocks.NewMockStreamer(t)
	s.EXPECT().Lines().Return((<-chan string)(make(chan string)))

	return servicehost.New(theme.Default(), svcDef, testProject, 120, 100, 100, s), s
}

//nolint:funlen
func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(*testing.T) servicehost.Model

		// act
		msg tea.Msg

		// assert
		expectedMsg tea.Msg
		expectCmd   bool
		check       func(*testing.T, servicehost.Model)
	}

	cases := []testCase{
		{
			name: "ServiceSelected matching name sets selected",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			msg:         msgs.ServiceSelected{ServiceName: svcName},
			expectedMsg: nil,
			check: func(t *testing.T, m servicehost.Model) {
				t.Helper()
				assert.Contains(t, m.View().Content, "╭")
			},
		},

		{
			name: "ServiceSelected non-matching name clears selected",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)
				m, _ = m.Update(msgs.ServiceSelected{ServiceName: svcName})

				return m
			},
			msg:         msgs.ServiceSelected{ServiceName: "db"},
			expectedMsg: nil,
			check: func(t *testing.T, m servicehost.Model) {
				t.Helper()
				assert.Empty(t, m.View().Content)
			},
		},

		{
			name: "DaemonConnected starts streamer and emits Next cmd",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)
				s.EXPECT().Start(context.Background(), testProject+"-"+svcName+"-1").Return()
				s.EXPECT().Next().Return(func() tea.Msg {
					return msgs.LogLinesAvailable{}
				})

				return m
			},
			msg:         msgs.DaemonConnected{},
			expectedMsg: msgs.LogLinesAvailable{},
		},

		{
			name: "DaemonConnected when already started is no-op",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)
				s.EXPECT().Start(context.Background(), testProject+"-"+svcName+"-1").Return()
				s.EXPECT().Next().Return(func() tea.Msg { return nil })

				m, _ = m.Update(msgs.DaemonConnected{})

				return m
			},
			msg:         msgs.DaemonConnected{},
			expectedMsg: nil,
		},

		{
			name: "LogLinesAvailable emits streamer.Next",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)
				s.EXPECT().Next().Return(func() tea.Msg {
					return msgs.LogLinesAvailable{}
				})

				return m
			},
			msg:         msgs.LogLinesAvailable{},
			expectedMsg: msgs.LogLinesAvailable{},
		},

		{
			name: "LogStreamError closes streamer and schedules retry",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)
				s.EXPECT().Close().Return()

				return m
			},
			msg:       msgs.LogStreamError{Err: nil, ServiceName: svcName},
			expectCmd: true,
		},

		{
			name: "LogStreamContainerNotFound closes streamer and schedules retry",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)
				s.EXPECT().Close().Return()

				return m
			},
			msg:       msgs.LogStreamContainerNotFound{ServiceName: svcName},
			expectCmd: true,
		},

		{
			name: "LogStreamRetryTick restarts streamer when streamerStarted is false",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)

				// Simulate a previous error that reset the flag
				s.EXPECT().Close().Return()

				m, _ = m.Update(msgs.LogStreamContainerNotFound{ServiceName: svcName})

				// Expect Start + Next on retry tick

				s.EXPECT().Start(mock.Anything, testProject+"-"+svcName+"-1").Return()
				s.EXPECT().Next().Return(func() tea.Msg {
					return msgs.LogLinesAvailable{}
				})

				return m
			},
			msg:         msgs.LogStreamRetryTick{},
			expectedMsg: msgs.LogLinesAvailable{},
		},

		{
			name: "LogStreamRetryTick is no-op when streamer is already started",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, s := newModel(t)

				// Start the streamer normally
				s.EXPECT().Start(mock.Anything, testProject+"-"+svcName+"-1").Return()
				s.EXPECT().Next().Return(func() tea.Msg { return nil })

				m, _ = m.Update(msgs.DaemonConnected{})

				return m
			},
			msg:       msgs.LogStreamRetryTick{},
			expectCmd: false,
		},

		{
			name: "KeyPressMsg when not selected is no-op",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			msg:         tea.KeyPressMsg{},
			expectedMsg: nil,
		},

		{
			name: "theme.Changed updates stored theme",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			msg:         theme.Changed{Theme: theme.DefaultLight()},
			expectedMsg: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup(t)
			m, cmd := m.Update(tc.msg)

			switch {
			case tc.expectedMsg != nil:
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			case tc.expectCmd:
				require.NotNil(t, cmd)
			default:
				require.Nil(t, cmd)
			}

			if tc.check != nil {
				tc.check(t, m)
			}
		})
	}
}

func TestUpdate_LogStreamErrorRecoveryCycle(t *testing.T) {
	t.Parallel()

	t.Run("error to retry to restart cycle via mock", func(t *testing.T) {
		t.Parallel()

		m, s := newModel(t)

		// Step 1: DaemonConnected starts the streamer
		s.EXPECT().Start(mock.Anything, testProject+"-"+svcName+"-1").Return().Once()
		s.EXPECT().Next().Return(func() tea.Msg {
			return msgs.LogLinesAvailable{}
		}).Once()

		m, cmd := m.Update(msgs.DaemonConnected{})
		require.NotNil(t, cmd)
		require.Equal(t, msgs.LogLinesAvailable{}, cmd())

		// Step 2: LogStreamContainerNotFound → Close + retry tick
		s.EXPECT().Close().Return().Once()

		m, cmd = m.Update(msgs.LogStreamContainerNotFound{ServiceName: svcName})
		require.NotNil(t, cmd) // tea.Tick cmd — non-deterministic, skip calling it

		// Step 3: LogStreamRetryTick → Start + Next
		s.EXPECT().Start(mock.Anything, testProject+"-"+svcName+"-1").Return().Once()
		s.EXPECT().Next().Return(func() tea.Msg {
			return msgs.LogLinesAvailable{}
		}).Once()

		m, cmd = m.Update(msgs.LogStreamRetryTick{})
		require.NotNil(t, cmd)
		result := cmd()
		require.Equal(t, msgs.LogLinesAvailable{}, result)

		// Streamer started again — verify LogLinesAvailable re-subscribes
		s.EXPECT().Next().Return(func() tea.Msg {
			return msgs.LogLinesAvailable{}
		}).Once()

		_, cmd = m.Update(msgs.LogLinesAvailable{})
		require.NotNil(t, cmd)
		require.Equal(t, msgs.LogLinesAvailable{}, cmd())
	})
}

func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(*testing.T) servicehost.Model

		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name: "empty when not selected",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)

				return m
			},
			expectedResult: "",
		},

		{
			name: "log pane when selected",
			setup: func(t *testing.T) servicehost.Model {
				t.Helper()
				m, _ := newModel(t)
				m, _ = m.Update(msgs.ServiceSelected{ServiceName: svcName})

				return m
			},
			expectedResult: "╭",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup(t)

			if tc.expectedResult == "" {
				assert.Empty(t, m.View().Content)
			} else {
				assert.Contains(t, m.View().Content, tc.expectedResult)
			}
		})
	}
}
