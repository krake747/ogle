package fileselect_test

import (
	"testing"
	"time"

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

func TestUpdateMouseClick(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name   string
		files  []string
		msg    tea.Msg
		assert func(*testing.T, tea.Cmd)
	}

	cases := []testCase{
		{
			name:  "MouseClickMsg on item hit emits FileSelected",
			files: []string{testFile},
			assert: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				fsel, ok := msg.(msgs.FileSelected)
				require.True(t, ok)
				assert.Equal(t, testFile, fsel.Path)
			},
		},
		{
			name:  "MouseClickMsg on miss returns nil cmd",
			files: []string{testFile},
			msg:   tea.MouseClickMsg{X: -1, Y: -1, Button: tea.MouseLeft},
			assert: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
			},
		},
		{
			name:  "MouseClickMsg with non-left button returns nil cmd",
			files: []string{testFile},
			msg:   tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseRight},
			assert: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			zm := zone.New()
			m := fileselect.New(tc.files, 100, 40, zm, theme.Default())

			view := m.View()
			zm.Scan(view.Content)

			msg := tc.msg
			if msg == nil {
				require.Eventually(t, func() bool {
					zi := zm.Get("item-0")

					return zi != nil && !zi.IsZero()
				}, time.Second, 10*time.Millisecond)

				zi := zm.Get("item-0")
				msg = tea.MouseClickMsg{
					X: zi.StartX, Y: zi.StartY,
					Button: tea.MouseLeft,
				}
			}

			_, cmd := m.Update(msg)
			tc.assert(t, cmd)
		})
	}
}

func TestUpdateMouseMotion(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name   string
		files  []string
		setup  func(zm *zone.Manager, m fileselect.Model) (fileselect.Model, tea.Cmd)
		msg    tea.Msg
		assert func(t *testing.T, before string, m fileselect.Model, cmd tea.Cmd)
	}

	// Use a theme where HoverBackground differs from both
	// ServiceListBackground and SelectedBackground so hover rendering changes are detectable.
	th := theme.DefaultLight()

	cases := []testCase{
		{
			name:  "MouseMotionMsg on item hit changes hover rendering",
			files: []string{testFile, "other.yml"},
			assert: func(t *testing.T, before string, m fileselect.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)

				after := m.View().Content
				assert.NotEqual(t, before, after,
					"hover background should differ from normal background")
			},
		},
		{
			name:  "MouseMotionMsg on miss clears hover rendering",
			files: []string{testFile, "other.yml"},
			setup: func(zm *zone.Manager, m fileselect.Model) (fileselect.Model, tea.Cmd) {
				require.Eventually(t, func() bool {
					zi := zm.Get("item-0")

					return zi != nil && !zi.IsZero()
				}, time.Second, 10*time.Millisecond)

				zi := zm.Get("item-0")

				return m.Update(tea.MouseMotionMsg{X: zi.StartX, Y: zi.StartY})
			},
			msg: tea.MouseMotionMsg{X: -1, Y: -1},
			assert: func(t *testing.T, before string, m fileselect.Model, cmd tea.Cmd) {
				t.Helper()
				require.Nil(t, cmd)

				after := m.View().Content
				assert.Equal(t, before, after,
					"hover should be cleared, returning to baseline rendering")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			zm := zone.New()
			m := fileselect.New(tc.files, 100, 40, zm, th)

			view := m.View()
			zm.Scan(view.Content)
			before := view.Content

			if tc.setup != nil {
				var setupCmd tea.Cmd

				m, setupCmd = tc.setup(zm, m)
				require.Nil(t, setupCmd)
			}

			msg := tc.msg
			if msg == nil {
				require.Eventually(t, func() bool {
					zi := zm.Get("item-0")

					return zi != nil && !zi.IsZero()
				}, time.Second, 10*time.Millisecond)

				zi := zm.Get("item-0")
				msg = tea.MouseMotionMsg{X: zi.StartX, Y: zi.StartY}
			}

			m, cmd := m.Update(msg)
			tc.assert(t, before, m, cmd)
		})
	}
}
