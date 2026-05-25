package accordion_test

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/accordion"
	"github.com/ma-tf/ogle/internal/ui/components/accordion/mocks"
	"github.com/ma-tf/ogle/internal/ui/components/accordion/value"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	dash            = "—"
	nginxImage      = "nginx:latest"
	testContainerID = "abc123def4567890"
	up2Hours        = "Up 2 hours"
	svcWeb          = "web"
)

//nolint:gochecknoglobals // shared test fixtures
var testProject = &domain.Project{
	Name: "testproj",
	File: "/path/to/compose.yaml",
	Services: []domain.ServiceDef{
		{Name: svcWeb, Image: nginxImage, Ports: []string{"80:80", "443:443"}},
	},
}

//nolint:gochecknoglobals // shared test fixtures
var multiServiceProject = &domain.Project{
	Name: "testproj",
	File: "/path/to/compose.yaml",
	Services: []domain.ServiceDef{
		{Name: svcWeb, Image: nginxImage, Ports: []string{"80:80"}},
		{Name: "api", Image: "api:latest", Ports: []string{"8080:8080"}},
	},
}

//nolint:funlen
func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name           string
		setup          func(*mocks.MockZoneManager) accordion.Model
		expectedResult string
	}

	cases := []testCase{
		{
			name: "empty when no services",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				return accordion.New(&domain.Project{}, 100, 24, theme.Default(), zm)
			},
			expectedResult: "",
		},
		{
			name: "empty when width is zero",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				return accordion.New(testProject, 0, 24, theme.Default(), zm)
			},
			expectedResult: "",
		},
		{
			name: "expanded header indicator",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				return accordion.New(testProject, 100, 24, theme.Default(), zm)
			},
			expectedResult: "▼",
		},
		{
			name: "image label rendered",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				return accordion.New(testProject, 100, 24, theme.Default(), zm)
			},
			expectedResult: "Image:",
		},
		{
			name: "container id label rendered",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				return accordion.New(testProject, 100, 24, theme.Default(), zm)
			},
			expectedResult: "Container ID:",
		},
		{
			name: "created label rendered",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				return accordion.New(testProject, 100, 24, theme.Default(), zm)
			},
			expectedResult: "Created:",
		},
		{
			name: "state label rendered",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				return accordion.New(testProject, 100, 24, theme.Default(), zm)
			},
			expectedResult: "State:",
		},
		{
			name: "ports label rendered",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				return accordion.New(testProject, 100, 24, theme.Default(), zm)
			},
			expectedResult: "Ports:",
		},
		{
			name: "image value from service def",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				return accordion.New(testProject, 100, 24, theme.Default(), zm)
			},
			expectedResult: nginxImage,
		},
		{
			name: "ports value from service def",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				return accordion.New(testProject, 100, 24, theme.Default(), zm)
			},
			expectedResult: "80:80, 443:443",
		},
		{
			name: "placeholders when runtime is nil",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				return accordion.New(testProject, 100, 24, theme.Default(), zm)
			},
			expectedResult: dash,
		},
		{
			name: "container id truncated from runtime",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				m := accordion.New(testProject, 100, 24, theme.Default(), zm)
				m, _ = m.Update(msgs.ServicesPolled{
					Runtimes: map[string]*domain.ServiceRuntimeData{
						svcWeb: {
							ContainerID: testContainerID,
							State:       domain.ServiceStateRunning,
							Status:      up2Hours,
							CreatedAt:   time.Now().Add(-2 * time.Hour),
						},
					},
				})

				return m
			},
			expectedResult: "abc123def456",
		},
		{
			name: "state string from runtime",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				m := accordion.New(testProject, 100, 24, theme.Default(), zm)
				m, _ = m.Update(msgs.ServicesPolled{
					Runtimes: map[string]*domain.ServiceRuntimeData{
						svcWeb: {
							ContainerID: testContainerID,
							State:       domain.ServiceStateRunning,
							Status:      up2Hours,
							CreatedAt:   time.Now().Add(-2 * time.Hour),
						},
					},
				})

				return m
			},
			expectedResult: up2Hours,
		},
		{
			name: "created age from runtime",
			setup: func(zm *mocks.MockZoneManager) accordion.Model {
				zm.EXPECT().
					Mark(mock.Anything, mock.Anything).
					RunAndReturn(func(_, v string) string { return v }).
					Maybe()
				zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

				m := accordion.New(testProject, 100, 24, theme.Default(), zm)
				m, _ = m.Update(msgs.ServicesPolled{
					Runtimes: map[string]*domain.ServiceRuntimeData{
						svcWeb: {
							ContainerID: testContainerID,
							State:       domain.ServiceStateRunning,
							Status:      up2Hours,
							CreatedAt:   time.Now().Add(-2 * time.Hour),
						},
					},
				})

				return m
			},
			expectedResult: "h ago",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			zm := mocks.NewMockZoneManager(t)

			m := tc.setup(zm)
			_ = m.Init()

			if tc.expectedResult == "" {
				assert.Empty(t, m.View().Content)
			} else {
				assert.Contains(t, m.View().Content, tc.expectedResult)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name        string
		msg         tea.Msg
		expectedMsg tea.Msg
	}

	cases := []testCase{
		{
			name:        "ServiceSelected triggers sync",
			msg:         msgs.ServiceSelected{ServiceName: "api"},
			expectedMsg: value.StartMsg{Gen: 2},
		},
		{
			name: "ServicesPolled stores runtime and triggers sync",
			msg: msgs.ServicesPolled{
				Runtimes: map[string]*domain.ServiceRuntimeData{
					svcWeb: {
						ContainerID: "abc123def456",
						State:       domain.ServiceStateRunning,
						Status:      up2Hours,
					},
				},
			},
			expectedMsg: value.StartMsg{Gen: 2},
		},
		{
			name: "ServicesPolled error does not update runtime",
			msg: msgs.ServicesPolled{
				Err: assert.AnError,
			},
			expectedMsg: nil,
		},
		{
			name:        "WindowSizeMsg updates dimensions and triggers sync",
			msg:         tea.WindowSizeMsg{Width: 200, Height: 50},
			expectedMsg: value.StartMsg{Gen: 2},
		},
		{
			name:        "theme.Changed updates theme and triggers sync",
			msg:         theme.Changed{Theme: theme.DefaultLight()},
			expectedMsg: value.StartMsg{Gen: 2},
		},
		{
			name:        "MouseClickMsg no-op",
			msg:         tea.MouseClickMsg{},
			expectedMsg: nil,
		},
		{
			name:        "MouseMotionMsg no-op",
			msg:         tea.MouseMotionMsg{},
			expectedMsg: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			zm := mocks.NewMockZoneManager(t)
			zm.EXPECT().
				Mark(mock.Anything, mock.Anything).
				RunAndReturn(func(_, v string) string { return v }).
				Maybe()
			zm.EXPECT().Get(mock.Anything).Return(nil).Maybe()

			m := accordion.New(multiServiceProject, 100, 24, theme.Default(), zm)
			_ = m.Init()

			_, cmd := m.Update(tc.msg)

			if tc.expectedMsg != nil {
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			} else {
				require.Nil(t, cmd)
			}
		})
	}
}
