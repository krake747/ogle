package startup_test

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/parser/mocks"
	"github.com/ma-tf/ogle/internal/ui/flows/startup"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

//nolint:gochecknoglobals // shared test fixtures
var (
	project  = &domain.Project{Name: "myapp", File: "/path/to/compose.yml"}
	errParse = errors.New("parse error")
)

func newModel(t *testing.T) (startup.Model, *mocks.MockParser) {
	t.Helper()
	mockP := mocks.NewMockParser(t)

	return startup.New(100, 50, zone.New(), theme.Default(), mockP), mockP
}

func TestInit(t *testing.T) {
	t.Parallel()

	m, _ := newModel(t)
	cmd := m.Init()
	require.NotNil(t, cmd)
	require.IsType(t, msgs.BindingsMsg{}, cmd())
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(m startup.Model, p *mocks.MockParser) startup.Model

		// act
		msg tea.Msg

		// assert
		expectedMsg tea.Msg
	}

	cases := []testCase{
		{
			name: "FileSelected emits ProjectLoaded",
			// arrange
			setup: func(m startup.Model, p *mocks.MockParser) startup.Model {
				p.EXPECT().Parse("test/path/file.yml").Return(project, nil)

				return m
			},
			// act
			msg: msgs.FileSelected{Path: "test/path/file.yml"},
			// assert
			expectedMsg: msgs.ProjectLoaded{Project: project},
		},
		{
			name: "FileSelected with parse error returns no command",
			// arrange
			setup: func(m startup.Model, p *mocks.MockParser) startup.Model {
				p.EXPECT().Parse("test/path/file.yml").Return(nil, errParse)

				return m
			},
			// act
			msg: msgs.FileSelected{Path: "test/path/file.yml"},
		},
		{
			name: "WindowSizeMsg forwards to fileselect",
			// act
			msg: tea.WindowSizeMsg{Width: 120, Height: 80},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, mockP := newModel(t)
			if tc.setup != nil {
				m = tc.setup(m, mockP)
			}

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

func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(m startup.Model) startup.Model

		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name:           "delegates to fileselect view",
			expectedResult: "file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, _ := newModel(t)
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
