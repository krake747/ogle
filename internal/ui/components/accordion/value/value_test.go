package value_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/ui/components/accordion/value"
)

func TestView(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		input string
		width int

		// assert
		expectedResult string
	}

	const shortContent = "hello"

	cases := []testCase{
		{
			name:           "empty view on zero width",
			input:          shortContent,
			width:          0,
			expectedResult: "",
		},
		{
			name:           "empty view on negative width",
			input:          shortContent,
			width:          -1,
			expectedResult: "",
		},
		{
			name:           "full text when content fits",
			input:          shortContent,
			width:          20,
			expectedResult: shortContent,
		},
		{
			name:           "first N chars when overflowing",
			input:          "hello world",
			width:          5,
			expectedResult: shortContent,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := value.New(tc.input, lipgloss.Color("white"), lipgloss.Color("black"), tc.width)
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
		name string
		// arrange
		input string
		width int

		// act
		msg tea.Msg

		// assert
		expectedMsg tea.Msg
	}

	cases := []testCase{
		{
			name:        "no cmd when content fits",
			input:       "hi",
			width:       20,
			msg:         value.StartMsg{Gen: 1},
			expectedMsg: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := value.New(tc.input, lipgloss.Color("white"), lipgloss.Color("black"), tc.width)
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
