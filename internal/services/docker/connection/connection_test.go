package connection_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ma-tf/ogle/internal/services/docker/connection"
)

func TestNew(t *testing.T) {
	t.Parallel()

	m := connection.New()

	assert.Equal(t, connection.ConnectStateConnecting, m.ConnectState())
	assert.Equal(t, time.Duration(0), m.Remaining())
}

func TestHandleConnected(t *testing.T) {
	t.Parallel()

	m := connection.New()
	m.HandleConnected()

	assert.Equal(t, connection.ConnectStateConnected, m.ConnectState())
	assert.Equal(t, time.Duration(0), m.Remaining())
}

func TestHandleUnavailable(t *testing.T) {
	t.Parallel()

	now := time.Now()

	type testCase struct {
		name string

		// arrange
		setup func() *connection.Machine

		// assert
		expectedState   connection.ConnectState
		expectedAlready bool
		expectRemaining bool
	}

	tt := []testCase{
		{
			name:            "from connecting",
			setup:           connection.New,
			expectedState:   connection.ConnectStateUnavailable,
			expectedAlready: true,
			expectRemaining: true,
		},
		{
			name: "from already unavailable",
			setup: func() *connection.Machine {
				m := connection.New()
				m.HandleUnavailable(now)

				return m
			},
			expectedState:   connection.ConnectStateUnavailable,
			expectedAlready: false,
			expectRemaining: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup()
			already := m.HandleUnavailable(now)

			assert.Equal(t, tc.expectedAlready, already)
			assert.Equal(t, tc.expectedState, m.ConnectState())

			if tc.expectRemaining {
				assert.NotZero(t, m.Remaining())
			} else {
				assert.Zero(t, m.Remaining())
			}
		})
	}
}

func TestHandleGracePeriodExpired(t *testing.T) {
	t.Parallel()

	now := time.Now()

	type testCase struct {
		name string

		// arrange
		setup func() *connection.Machine

		// assert
		expectedResult  bool
		expectedState   connection.ConnectState
		expectRemaining bool
	}

	tt := []testCase{
		{
			name:            "from connecting transitions to unavailable",
			setup:           connection.New,
			expectedResult:  true,
			expectedState:   connection.ConnectStateUnavailable,
			expectRemaining: true,
		},
		{
			name: "from connected returns false",
			setup: func() *connection.Machine {
				m := connection.New()
				m.HandleConnected()

				return m
			},
			expectedResult:  false,
			expectedState:   connection.ConnectStateConnected,
			expectRemaining: false,
		},
		{
			name: "from unavailable returns false",
			setup: func() *connection.Machine {
				m := connection.New()
				m.HandleUnavailable(now)

				return m
			},
			expectedResult:  false,
			expectedState:   connection.ConnectStateUnavailable,
			expectRemaining: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup()
			result := m.HandleGracePeriodExpired(now)

			assert.Equal(t, tc.expectedResult, result)
			assert.Equal(t, tc.expectedState, m.ConnectState())

			if tc.expectRemaining {
				assert.NotZero(t, m.Remaining())
			} else {
				assert.Zero(t, m.Remaining())
			}
		})
	}
}

func TestIsRetryDue(t *testing.T) {
	t.Parallel()

	now := time.Now()

	type testCase struct {
		name string

		// arrange
		setup   func() *connection.Machine
		checkAt time.Time

		// assert
		expectedResult  bool
		expectedState   connection.ConnectState
		expectRemaining bool
	}

	tt := []testCase{
		{
			name: "not unavailable returns false",
			setup: func() *connection.Machine {
				m := connection.New()
				m.HandleConnected()

				return m
			},
			checkAt:         now,
			expectedResult:  false,
			expectedState:   connection.ConnectStateConnected,
			expectRemaining: false,
		},
		{
			name: "before deadline returns false",
			setup: func() *connection.Machine {
				m := connection.New()
				m.HandleUnavailable(now)

				return m
			},
			checkAt:         now.Add(5 * time.Second),
			expectedResult:  false,
			expectedState:   connection.ConnectStateUnavailable,
			expectRemaining: true,
		},
		{
			name: "at deadline transitions to connecting",
			setup: func() *connection.Machine {
				m := connection.New()
				m.HandleUnavailable(now)

				return m
			},
			checkAt:         now.Add(10 * time.Second),
			expectedResult:  true,
			expectedState:   connection.ConnectStateConnecting,
			expectRemaining: false,
		},
		{
			name: "after deadline transitions to connecting",
			setup: func() *connection.Machine {
				m := connection.New()
				m.HandleUnavailable(now)

				return m
			},
			checkAt:         now.Add(15 * time.Second),
			expectedResult:  true,
			expectedState:   connection.ConnectStateConnecting,
			expectRemaining: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup()
			result := m.IsRetryDue(tc.checkAt)

			assert.Equal(t, tc.expectedResult, result)
			assert.Equal(t, tc.expectedState, m.ConnectState())

			if tc.expectRemaining {
				assert.NotZero(t, m.Remaining())
			} else {
				assert.Zero(t, m.Remaining())
			}
		})
	}
}

func TestRemaining(t *testing.T) {
	t.Parallel()

	now := time.Now()

	type testCase struct {
		name string

		// arrange
		setup func() *connection.Machine

		// assert
		expectZero bool
	}

	tt := []testCase{
		{
			name:       "initial connecting returns 0",
			setup:      connection.New,
			expectZero: true,
		},
		{
			name: "connected returns 0",
			setup: func() *connection.Machine {
				m := connection.New()
				m.HandleConnected()

				return m
			},
			expectZero: true,
		},
		{
			name: "unavailable returns positive",
			setup: func() *connection.Machine {
				m := connection.New()
				m.HandleUnavailable(now)

				return m
			},
			expectZero: false,
		},
		{
			name: "after retry due returns 0",
			setup: func() *connection.Machine {
				m := connection.New()
				m.HandleUnavailable(now)
				m.IsRetryDue(now.Add(10 * time.Second))

				return m
			},
			expectZero: true,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup()

			if tc.expectZero {
				assert.Zero(t, m.Remaining())
			} else {
				assert.NotZero(t, m.Remaining())
			}
		})
	}
}

func TestConnectState(t *testing.T) {
	t.Parallel()

	now := time.Now()

	type testCase struct {
		name string

		// arrange
		setup func() *connection.Machine

		// assert
		expected connection.ConnectState
	}

	tt := []testCase{
		{
			name:     "initial",
			setup:    connection.New,
			expected: connection.ConnectStateConnecting,
		},
		{
			name: "after connected",
			setup: func() *connection.Machine {
				m := connection.New()
				m.HandleConnected()

				return m
			},
			expected: connection.ConnectStateConnected,
		},
		{
			name: "after unavailable",
			setup: func() *connection.Machine {
				m := connection.New()
				m.HandleUnavailable(now)

				return m
			},
			expected: connection.ConnectStateUnavailable,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m := tc.setup()
			assert.Equal(t, tc.expected, m.ConnectState())
		})
	}
}
