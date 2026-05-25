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

const testFile = "test.yml"

func TestInit(t *testing.T) {
	t.Parallel()

	m := watching.New(testFile, 100, 50, theme.Default(), mocks.NewMockParser(t))
	cmd := m.Init()
	require.Nil(t, cmd)
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup func(*mocks.MockParser) watching.Model

		// act
		msg tea.Msg

		// assert
		expectedMsg tea.Msg
	}

	cases := []testCase{
		{
			name: "FileAvailabilityChanged with file present and parse success emits ProjectLoaded",
			setup: func(p *mocks.MockParser) watching.Model {
				p.EXPECT().Parse(testFile).Return(&domain.Project{Name: "test"}, nil)

				return watching.New(testFile, 100, 50, theme.Default(), p)
			},
			msg:         msgs.FileAvailabilityChanged{Files: []string{testFile}},
			expectedMsg: msgs.ProjectLoaded{Project: &domain.Project{Name: "test"}},
		},
		{
			name: "FileAvailabilityChanged with parse error returns nil cmd",
			setup: func(p *mocks.MockParser) watching.Model {
				p.EXPECT().Parse(testFile).Return(nil, assert.AnError)

				return watching.New(testFile, 100, 50, theme.Default(), p)
			},
			msg:         msgs.FileAvailabilityChanged{Files: []string{testFile}},
			expectedMsg: nil,
		},
		{
			name: "FileAvailabilityChanged without tracked file resets to idle",
			setup: func(p *mocks.MockParser) watching.Model {
				return watching.New(testFile, 100, 50, theme.Default(), p)
			},
			msg:         msgs.FileAvailabilityChanged{Files: []string{"other.yml"}},
			expectedMsg: nil,
		},
		{
			name: "KeyPress q emits QuitMsg",
			setup: func(p *mocks.MockParser) watching.Model {
				return watching.New(testFile, 100, 50, theme.Default(), p)
			},
			msg:         tea.KeyPressMsg{Code: 'q'},
			expectedMsg: tea.QuitMsg{},
		},
		{
			name: "KeyPress non-q returns no cmd",
			setup: func(p *mocks.MockParser) watching.Model {
				return watching.New(testFile, 100, 50, theme.Default(), p)
			},
			msg:         tea.KeyPressMsg{Code: 'x'},
			expectedMsg: nil,
		},
		{
			name: "WindowSizeMsg stores dimensions",
			setup: func(p *mocks.MockParser) watching.Model {
				return watching.New(testFile, 0, 0, theme.Default(), p)
			},
			msg:         tea.WindowSizeMsg{Width: 100, Height: 50},
			expectedMsg: nil,
		},
		{
			name: "theme.Changed updates theme pointer",
			setup: func(p *mocks.MockParser) watching.Model {
				return watching.New(testFile, 100, 50, theme.Default(), p)
			},
			msg:         theme.Changed{Theme: theme.DefaultLight()},
			expectedMsg: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := mocks.NewMockParser(t)
			m := tc.setup(p)
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
		setup func(*mocks.MockParser) watching.Model

		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name: "idle state shows waiting message",
			setup: func(p *mocks.MockParser) watching.Model {
				return watching.New(testFile, 100, 50, theme.Default(), p)
			},
			expectedResult: "compose file unavailable",
		},
		{
			name: "parse error state shows error alongside waiting message",
			setup: func(p *mocks.MockParser) watching.Model {
				p.EXPECT().Parse(testFile).Return(nil, assert.AnError)
				m := watching.New(testFile, 100, 50, theme.Default(), p)
				m, _ = m.Update(msgs.FileAvailabilityChanged{Files: []string{testFile}})

				return m
			},
			expectedResult: "Parse error",
		},
		{
			name: "view after theme change does not panic",
			setup: func(p *mocks.MockParser) watching.Model {
				m := watching.New(testFile, 100, 50, theme.Default(), p)
				m, _ = m.Update(theme.Changed{Theme: theme.DefaultLight()})

				return m
			},
			expectedResult: "compose file unavailable",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := mocks.NewMockParser(t)
			m := tc.setup(p)

			if tc.expectedResult == "" {
				assert.Empty(t, m.View().Content)
			} else {
				assert.Contains(t, m.View().Content, tc.expectedResult)
			}
		})
	}
}
