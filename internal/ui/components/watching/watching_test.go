package watching_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/parser/mocks"
	"github.com/ma-tf/ogle/internal/ui/components/watching"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

func TestInit(t *testing.T) {
	t.Parallel()

	m := watching.New("test.yml", 100, 50, theme.Default(), mocks.NewMockParser(t))
	cmd := m.Init()
	require.Nil(t, cmd)
}

//nolint:funlen
func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(*testing.T) watching.Model

		// act
		msg tea.Msg

		// assert
		expectedMsg tea.Msg
		check       func(*testing.T, watching.Model)
	}

	const testFile = "test.yml"

	errParse := assert.AnError

	cases := []testCase{
		{
			name: "FileAvailabilityChanged with file present and parse success emits ProjectLoaded",
			setup: func(t *testing.T) watching.Model {
				mockParser := mocks.NewMockParser(t)
				mockParser.EXPECT().Parse(testFile).Return(&domain.Project{Name: "test"}, nil)
				return watching.New(testFile, 100, 50, theme.Default(), mockParser)
			},
			msg:         msgs.FileAvailabilityChanged{Files: []string{testFile}},
			expectedMsg: msgs.ProjectLoaded{Project: &domain.Project{Name: "test"}},
		},

		{
			name: "FileAvailabilityChanged with file present and parse error sets stateParseError",
			setup: func(t *testing.T) watching.Model {
				mockParser := mocks.NewMockParser(t)
				mockParser.EXPECT().Parse(testFile).Return(nil, errParse)
				return watching.New(testFile, 100, 50, theme.Default(), mockParser)
			},
			msg:         msgs.FileAvailabilityChanged{Files: []string{testFile}},
			expectedMsg: nil,
			check: func(t *testing.T, m watching.Model) {
				t.Helper()
				v := m.View().Content
				assert.Contains(t, v, "compose file unavailable")
				assert.Contains(t, v, errParse.Error())
				assert.Contains(t, v, "Waiting for file to change")
			},
		},

		{
			name: "FileAvailabilityChanged without tracked file resets to idle",
			setup: func(t *testing.T) watching.Model {
				return watching.New(testFile, 100, 50, theme.Default(), mocks.NewMockParser(t))
			},
			msg:         msgs.FileAvailabilityChanged{Files: []string{"other.yml"}},
			expectedMsg: nil,
		},

		{
			name: "KeyPress q emits QuitMsg",
			setup: func(t *testing.T) watching.Model {
				return watching.New(testFile, 100, 50, theme.Default(), mocks.NewMockParser(t))
			},
			msg:         tea.KeyPressMsg{Code: 'q'},
			expectedMsg: tea.QuitMsg{},
		},

		{
			name: "KeyPress non-q returns no cmd",
			setup: func(t *testing.T) watching.Model {
				return watching.New(testFile, 100, 50, theme.Default(), mocks.NewMockParser(t))
			},
			msg:         tea.KeyPressMsg{Code: 'x'},
			expectedMsg: nil,
		},

		{
			name: "WindowSizeMsg stores dimensions",
			setup: func(t *testing.T) watching.Model {
				return watching.New(testFile, 0, 0, theme.Default(), mocks.NewMockParser(t))
			},
			msg:         tea.WindowSizeMsg{Width: 100, Height: 50},
			expectedMsg: nil,
		},

		{
			name: "theme.Changed updates theme pointer",
			setup: func(t *testing.T) watching.Model {
				return watching.New(testFile, 100, 50, theme.Default(), mocks.NewMockParser(t))
			},
			msg:         theme.Changed{Theme: theme.DefaultLight()},
			expectedMsg: nil,
			check: func(t *testing.T, m watching.Model) {
				t.Helper()
				assert.NotPanics(t, func() { m.View() })
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup(t)
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
		setup func(*testing.T) watching.Model

		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name: "idle state shows waiting message",
			setup: func(t *testing.T) watching.Model {
				return watching.New("test.yml", 100, 50, theme.Default(), mocks.NewMockParser(t))
			},
			expectedResult: "compose file unavailable",
		},

		{
			name: "parse error state shows error alongside waiting message",
			setup: func(t *testing.T) watching.Model {
				mockParser := mocks.NewMockParser(t)
				mockParser.EXPECT().Parse("test.yml").Return(nil, assert.AnError)
				m := watching.New("test.yml", 100, 50, theme.Default(), mockParser)
				m, _ = m.Update(msgs.FileAvailabilityChanged{Files: []string{"test.yml"}})
				return m
			},
			expectedResult: "Parse error",
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
