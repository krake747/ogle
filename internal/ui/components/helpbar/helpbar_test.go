package helpbar_test

import (
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/helpbar"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

type testKeymap struct{}

func (k testKeymap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

func (k testKeymap) FullHelp() [][]key.Binding { return nil }

var _ help.KeyMap = testKeymap{}

func TestView(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setup          func(m helpbar.Model) helpbar.Model
		expectedResult string
	}{
		{
			name:           "initial state returns empty view",
			expectedResult: "",
		},
		{
			name: "bindings msg populates view",
			setup: func(m helpbar.Model) helpbar.Model {
				m, _ = m.Update(tea.WindowSizeMsg{Width: 100})
				m, _ = m.Update(msgs.BindingsMsg{Keymap: testKeymap{}})

				return m
			},
			expectedResult: "quit",
		},
		{
			name: "window resize does not affect empty view",
			setup: func(m helpbar.Model) helpbar.Model {
				m, _ = m.Update(tea.WindowSizeMsg{Width: 80})

				return m
			},
			expectedResult: "",
		},
		{
			name: "keymap persists across window resize",
			setup: func(m helpbar.Model) helpbar.Model {
				m, _ = m.Update(msgs.BindingsMsg{Keymap: testKeymap{}})
				m, _ = m.Update(tea.WindowSizeMsg{Width: 80})

				return m
			},
			expectedResult: "q",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := helpbar.New(theme.Default())
			_ = m.Init()

			if tt.setup != nil {
				m = tt.setup(m)
			}

			if tt.expectedResult == "" {
				assert.Empty(t, m.View().Content)
			} else {
				assert.Contains(t, m.View().Content, tt.expectedResult)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string

		msg tea.Msg

		expectedMsg tea.Msg
	}

	cases := []testCase{
		{
			name:        "WindowSizeMsg returns no command",
			msg:         tea.WindowSizeMsg{Width: 80},
			expectedMsg: nil,
		},
		{
			name:        "BindingsMsg returns no command",
			msg:         msgs.BindingsMsg{Keymap: testKeymap{}},
			expectedMsg: nil,
		},
		{
			name:        "theme.Changed returns no command",
			msg:         theme.Changed{Theme: theme.Default()},
			expectedMsg: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := helpbar.New(theme.Default())
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
