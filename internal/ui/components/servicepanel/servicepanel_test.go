package servicepanel_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/servicepanel"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

func TestUpdate(t *testing.T) {
	t.Parallel()

	th := theme.Default()

	type testCase struct {
		name string
		// arrange
		setup func(m servicepanel.Model) servicepanel.Model
		// act
		msg tea.Msg
		// assert
		expectedCmdNonNil bool
	}

	cases := []testCase{
		{
			name:              "daemon connected starts poller",
			msg:               msgs.DaemonConnected{},
			expectedCmdNonNil: true,
		},
		{
			name: "daemon connected when already started is no-op",
			setup: func(m servicepanel.Model) servicepanel.Model {
				m, _ = m.Update(msgs.DaemonConnected{})

				return m
			},
			msg:               msgs.DaemonConnected{},
			expectedCmdNonNil: false,
		},
		{
			name:              "state poll tick emits next tick",
			msg:               msgs.StatePollTick{},
			expectedCmdNonNil: true,
		},
		{
			name:              "theme changed updates stored theme",
			msg:               theme.Changed{Theme: th},
			expectedCmdNonNil: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := servicepanel.New(&domain.Project{}, th, 80, 24, 1000)
			_ = m.Init()

			if tc.setup != nil {
				m = tc.setup(m)
			}

			_, cmd := m.Update(tc.msg)

			if tc.expectedCmdNonNil {
				require.NotNil(t, cmd)
			} else {
				require.Nil(t, cmd)
			}
		})
	}
}

func TestView(t *testing.T) {
	t.Parallel()

	th := theme.Default()

	type testCase struct {
		name string
		// arrange
		project *domain.Project
		setup   func(m servicepanel.Model) servicepanel.Model
		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name:           "empty project renders empty view",
			project:        &domain.Project{Name: "test"},
			expectedResult: "",
		},
		{
			name: "project with services renders compositor layers",
			project: &domain.Project{
				Name: "test",
				Services: []domain.ServiceDef{
					{Name: "web"},
				},
			},
			setup: func(m servicepanel.Model) servicepanel.Model {
				m, _ = m.Update(msgs.ServiceSelected{ServiceName: "web"})

				return m
			},
			expectedResult: "╭",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := servicepanel.New(tc.project, th, 80, 24, 1000)
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
