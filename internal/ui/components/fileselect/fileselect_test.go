package fileselect_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	zone "github.com/lrstanley/bubblezone/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/fileselect"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const testFile = "test.yml"

func newModel(t *testing.T, files []string) fileselect.Model {
	t.Helper()

	return fileselect.New(files, 100, 40, zone.New(), theme.Default())
}

func TestInit(t *testing.T) {
	t.Parallel()

	m := newModel(t, nil)
	cmd := m.Init()

	require.Nil(t, cmd)
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		files []string
		setup func(m fileselect.Model) fileselect.Model

		// act
		msg tea.Msg

		// assert
		expectedMsg tea.Msg
	}

	cases := []testCase{
		{
			name:        "KeyPress Enter with items emits FileSelected",
			files:       []string{testFile},
			msg:         tea.KeyPressMsg{Code: tea.KeyEnter},
			expectedMsg: msgs.FileSelected{Path: testFile},
		},
		{
			name:        "KeyPress Enter with empty list no-op",
			msg:         tea.KeyPressMsg{Code: tea.KeyEnter},
			expectedMsg: nil,
		},
		{
			name:        "FileAvailabilityChanged single file auto-selects",
			msg:         msgs.FileAvailabilityChanged{Files: []string{testFile}},
			expectedMsg: msgs.FileSelected{Path: testFile},
		},
		{
			name:  "FileAvailabilityChanged multiple files updates list items",
			files: []string{"original.yml"},
			setup: func(m fileselect.Model) fileselect.Model {
				m, _ = m.Update(msgs.FileAvailabilityChanged{Files: []string{"a.yml", "b.yml"}})

				return m
			},
			msg:         tea.KeyPressMsg{Code: tea.KeyEnter},
			expectedMsg: msgs.FileSelected{Path: "a.yml"},
		},
		{
			name:        "theme.Changed updates delegate and list styles",
			files:       []string{testFile},
			msg:         theme.Changed{Theme: theme.DefaultLight()},
			expectedMsg: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel(t, tc.files)

			if tc.setup != nil {
				m = tc.setup(m)
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
		files []string

		// assert
		expectedResult string
	}

	cases := []testCase{
		{
			name:           "renders file list with filenames",
			files:          []string{testFile},
			expectedResult: testFile,
		},
		{
			name:           "renders status bar item count",
			files:          []string{testFile},
			expectedResult: "1 file",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := newModel(t, tc.files)

			assert.Contains(t, m.View().Content, tc.expectedResult)
		})
	}
}
