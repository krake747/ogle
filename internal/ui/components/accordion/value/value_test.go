package value_test

import (
	"testing"
	"time"

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

func TestUpdateStart(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name      string
		input     string
		width     int
		msg       value.StartMsg
		expectCmd bool
	}

	cases := []testCase{
		{
			name:      "cmd when content overflows",
			input:     "this is a long text that requires scrolling",
			width:     10,
			msg:       value.StartMsg{Gen: 1},
			expectCmd: true,
		},
		{
			name:      "no cmd when content fits",
			input:     "hi",
			width:     20,
			msg:       value.StartMsg{Gen: 1},
			expectCmd: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := value.New(tc.input, lipgloss.Color("white"), lipgloss.Color("black"), tc.width)
			_, cmd := m.Update(tc.msg)

			if tc.expectCmd {
				require.NotNil(t, cmd)
			} else {
				require.Nil(t, cmd)
			}
		})
	}
}

func TestUpdateStartStaleGen(t *testing.T) {
	t.Parallel()

	m := value.New("this is a long text that requires scrolling",
		lipgloss.Color("white"), lipgloss.Color("black"), 10)
	m, cmd := m.Update(value.StartMsg{Gen: 1})
	require.NotNil(t, cmd)

	msg := cmd()

	m, _ = m.Update(msg)

	_, cmd = m.Update(value.StartMsg{Gen: 2})
	require.NotNil(t, cmd)

	result := cmd()
	ss, ok := result.(value.ScrollState)
	require.True(t, ok)
	require.Equal(t, 2, ss.Gen)
}

func TestHandleTick(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name             string
		state            value.ScrollState
		msg              value.ScrollState
		rawValue         string
		width            int
		expectedState    value.ScrollState
		expectedInterval time.Duration
		expectedOK       bool
	}

	const rawValue = "hello world"

	const rawWidth = 5

	cases := []testCase{
		{
			name:             "advances offset by dir",
			state:            value.ScrollState{Offset: 0, Dir: 1, Gen: 1, InstanceID: 1},
			msg:              value.ScrollState{Offset: 0, Dir: 0, Gen: 1, InstanceID: 1},
			rawValue:         rawValue,
			width:            rawWidth,
			expectedState:    value.ScrollState{Offset: 1, Dir: 1, Gen: 1, InstanceID: 1},
			expectedInterval: 300 * time.Millisecond,
			expectedOK:       true,
		},
		{
			name:             "completion at max offset sets dir -1",
			state:            value.ScrollState{Offset: 5, Dir: 1, Gen: 1, InstanceID: 1},
			msg:              value.ScrollState{Offset: 0, Dir: 0, Gen: 1, InstanceID: 1},
			rawValue:         rawValue,
			width:            rawWidth,
			expectedState:    value.ScrollState{Offset: 6, Dir: -1, Gen: 1, InstanceID: 1},
			expectedInterval: 2500 * time.Millisecond,
			expectedOK:       true,
		},
		{
			name:             "overflow wraps dir and sets idle interval",
			state:            value.ScrollState{Offset: 1, Dir: -1, Gen: 1, InstanceID: 1},
			msg:              value.ScrollState{Offset: 0, Dir: 0, Gen: 1, InstanceID: 1},
			rawValue:         rawValue,
			width:            rawWidth,
			expectedState:    value.ScrollState{Offset: 0, Dir: 1, Gen: 1, InstanceID: 1},
			expectedInterval: 2500 * time.Millisecond,
			expectedOK:       true,
		},
		{
			name:             "mid-scroll returns step interval",
			state:            value.ScrollState{Offset: 2, Dir: 1, Gen: 1, InstanceID: 1},
			msg:              value.ScrollState{Offset: 0, Dir: 0, Gen: 1, InstanceID: 1},
			rawValue:         rawValue,
			width:            rawWidth,
			expectedState:    value.ScrollState{Offset: 3, Dir: 1, Gen: 1, InstanceID: 1},
			expectedInterval: 300 * time.Millisecond,
			expectedOK:       true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			newState, interval, ok := tc.state.HandleTick(tc.msg, tc.rawValue, tc.width)

			require.Equal(t, tc.expectedState, newState)
			require.Equal(t, tc.expectedInterval, interval)
			require.Equal(t, tc.expectedOK, ok)
		})
	}
}

func TestHandleTickGuard(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name             string
		state            value.ScrollState
		msg              value.ScrollState
		rawValue         string
		width            int
		expectedState    value.ScrollState
		expectedInterval time.Duration
		expectedOK       bool
	}

	const rawValue = "hello world"

	const rawWidth = 5

	cases := []testCase{
		{
			name:             "stale gen returns ok false",
			state:            value.ScrollState{Offset: 0, Dir: 1, Gen: 1, InstanceID: 1},
			msg:              value.ScrollState{Offset: 0, Dir: 0, Gen: 2, InstanceID: 1},
			rawValue:         rawValue,
			width:            rawWidth,
			expectedState:    value.ScrollState{Offset: 0, Dir: 1, Gen: 1, InstanceID: 1},
			expectedInterval: 0,
			expectedOK:       false,
		},
		{
			name:             "stale instance ID returns ok false",
			state:            value.ScrollState{Offset: 0, Dir: 1, Gen: 1, InstanceID: 1},
			msg:              value.ScrollState{Offset: 0, Dir: 0, Gen: 1, InstanceID: 2},
			rawValue:         rawValue,
			width:            rawWidth,
			expectedState:    value.ScrollState{Offset: 0, Dir: 1, Gen: 1, InstanceID: 1},
			expectedInterval: 0,
			expectedOK:       false,
		},
		{
			name:             "zero width returns ok false",
			state:            value.ScrollState{Offset: 0, Dir: 1, Gen: 1, InstanceID: 1},
			msg:              value.ScrollState{Offset: 0, Dir: 0, Gen: 1, InstanceID: 1},
			rawValue:         rawValue,
			width:            0,
			expectedState:    value.ScrollState{Offset: 0, Dir: 1, Gen: 1, InstanceID: 1},
			expectedInterval: 0,
			expectedOK:       false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			newState, interval, ok := tc.state.HandleTick(tc.msg, tc.rawValue, tc.width)

			require.Equal(t, tc.expectedState, newState)
			require.Equal(t, tc.expectedInterval, interval)
			require.Equal(t, tc.expectedOK, ok)
		})
	}
}
