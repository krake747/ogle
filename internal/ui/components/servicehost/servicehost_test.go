package servicehost_test

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	logsmocks "github.com/ma-tf/ogle/internal/services/docker/logs/mocks"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/servicehost"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

//nolint:funlen
func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(*testing.T) (servicehost.Model, *logsmocks.MockStreamer)

		// act
		msg tea.Msg

		// assert
		expectedMsg tea.Msg
		check       func(*testing.T, servicehost.Model)
	}

	svcDef := domain.ServiceDef{Name: "web"}

	cases := []testCase{
		{
			name: "ServiceSelected matching name sets selected",
			setup: func(t *testing.T) (servicehost.Model, *logsmocks.MockStreamer) {
				ch := make(chan string)
				s := logsmocks.NewMockStreamer(t)
				s.EXPECT().Lines().Return((<-chan string)(ch))
				m := servicehost.New(theme.Default(), svcDef, "testproj", 120, 100, 100, s)
				return m, s
			},
			msg:         msgs.ServiceSelected{ServiceName: "web"},
			expectedMsg: nil,
			check: func(t *testing.T, m servicehost.Model) {
				t.Helper()
				assert.Contains(t, m.View().Content, "╭")
			},
		},

		{
			name: "ServiceSelected non-matching name clears selected",
			setup: func(t *testing.T) (servicehost.Model, *logsmocks.MockStreamer) {
				ch := make(chan string)
				s := logsmocks.NewMockStreamer(t)
				s.EXPECT().Lines().Return((<-chan string)(ch))
				m := servicehost.New(theme.Default(), svcDef, "testproj", 120, 100, 100, s)
				m, _ = m.Update(msgs.ServiceSelected{ServiceName: "web"})
				return m, s
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
			setup: func(t *testing.T) (servicehost.Model, *logsmocks.MockStreamer) {
				ch := make(chan string)
				s := logsmocks.NewMockStreamer(t)
				s.EXPECT().Lines().Return((<-chan string)(ch))
				s.EXPECT().Start(context.Background(), "testproj-web-1").Return()
				s.EXPECT().Next().Return(func() tea.Msg {
					return msgs.LogLinesAvailable{}
				})
				m := servicehost.New(theme.Default(), svcDef, "testproj", 120, 100, 100, s)
				return m, s
			},
			msg:         msgs.DaemonConnected{},
			expectedMsg: msgs.LogLinesAvailable{},
		},

		{
			name: "DaemonConnected when already started is no-op",
			setup: func(t *testing.T) (servicehost.Model, *logsmocks.MockStreamer) {
				ch := make(chan string)
				s := logsmocks.NewMockStreamer(t)
				s.EXPECT().Lines().Return((<-chan string)(ch))
				s.EXPECT().Start(context.Background(), "testproj-web-1").Return()
				s.EXPECT().Next().Return(func() tea.Msg { return nil })
				m := servicehost.New(theme.Default(), svcDef, "testproj", 120, 100, 100, s)
				m, _ = m.Update(msgs.DaemonConnected{})
				return m, s
			},
			msg:         msgs.DaemonConnected{},
			expectedMsg: nil,
		},

		{
			name: "LogLinesAvailable emits streamer.Next",
			setup: func(t *testing.T) (servicehost.Model, *logsmocks.MockStreamer) {
				ch := make(chan string)
				s := logsmocks.NewMockStreamer(t)
				s.EXPECT().Lines().Return((<-chan string)(ch))
				s.EXPECT().Next().Return(func() tea.Msg {
					return msgs.LogLinesAvailable{}
				})
				m := servicehost.New(theme.Default(), svcDef, "testproj", 120, 100, 100, s)
				return m, s
			},
			msg:         msgs.LogLinesAvailable{},
			expectedMsg: msgs.LogLinesAvailable{},
		},

		{
			name: "LogStreamError emits streamer.Next",
			setup: func(t *testing.T) (servicehost.Model, *logsmocks.MockStreamer) {
				ch := make(chan string)
				s := logsmocks.NewMockStreamer(t)
				s.EXPECT().Lines().Return((<-chan string)(ch))
				s.EXPECT().Next().Return(func() tea.Msg {
					return msgs.LogStreamError{Err: nil, ServiceName: "web"}
				})
				m := servicehost.New(theme.Default(), svcDef, "testproj", 120, 100, 100, s)
				return m, s
			},
			msg:         msgs.LogStreamError{Err: nil, ServiceName: "web"},
			expectedMsg: msgs.LogStreamError{Err: nil, ServiceName: "web"},
		},

		{
			name: "KeyPressMsg when not selected is no-op",
			setup: func(t *testing.T) (servicehost.Model, *logsmocks.MockStreamer) {
				ch := make(chan string)
				s := logsmocks.NewMockStreamer(t)
				s.EXPECT().Lines().Return((<-chan string)(ch))
				m := servicehost.New(theme.Default(), svcDef, "testproj", 120, 100, 100, s)
				return m, s
			},
			msg:         tea.KeyPressMsg{},
			expectedMsg: nil,
		},

		{
			name: "theme.Changed updates stored theme",
			setup: func(t *testing.T) (servicehost.Model, *logsmocks.MockStreamer) {
				ch := make(chan string)
				s := logsmocks.NewMockStreamer(t)
				s.EXPECT().Lines().Return((<-chan string)(ch))
				m := servicehost.New(theme.Default(), svcDef, "testproj", 120, 100, 100, s)
				return m, s
			},
			msg:         theme.Changed{Theme: theme.DefaultLight()},
			expectedMsg: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, _ := tc.setup(t)
			m, cmd := m.Update(tc.msg)

			if tc.expectedMsg != nil {
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			} else {
				require.Nil(t, cmd)
			}

			if tc.check != nil {
				tc.check(t, m)
			}
		})
	}
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

	svcDef := domain.ServiceDef{Name: "web"}

	cases := []testCase{
		{
			name: "empty when not selected",
			setup: func(t *testing.T) servicehost.Model {
				ch := make(chan string)
				s := logsmocks.NewMockStreamer(t)
				s.EXPECT().Lines().Return((<-chan string)(ch))
				return servicehost.New(theme.Default(), svcDef, "testproj", 120, 100, 100, s)
			},
			expectedResult: "",
		},

		{
			name: "log pane when selected",
			setup: func(t *testing.T) servicehost.Model {
				ch := make(chan string)
				s := logsmocks.NewMockStreamer(t)
				s.EXPECT().Lines().Return((<-chan string)(ch))
				m := servicehost.New(theme.Default(), svcDef, "testproj", 120, 100, 100, s)
				m, _ = m.Update(msgs.ServiceSelected{ServiceName: "web"})
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
