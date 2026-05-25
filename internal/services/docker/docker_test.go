package docker_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/services/docker"
)

func TestParsePsOutput(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name        string
		input       []byte
		expectedLen int
	}

	tt := []testCase{
		{
			name:        "empty input",
			input:       []byte(""),
			expectedLen: 0,
		},
		{
			name:        "whitespace only",
			input:       []byte("   \n  \n"),
			expectedLen: 0,
		},
		{
			name: "single running service",
			input: []byte(`{"id":"abc123","service":"web","state":"running",` +
				`"createdat":"2026-05-25 12:00:00 +0000 UTC","status":"Up 1h"}` + "\n"),
			expectedLen: 1,
		},
		{
			name: "multiple services",
			input: []byte(`{"id":"abc","service":"web","state":"running",` +
				`"createdat":"2026-05-25 12:00:00 +0000 UTC","status":"Up 1h"}` + "\n" +
				`{"id":"def","service":"db","state":"exited",` +
				`"createdat":"2026-05-25 10:00:00 +0000 UTC","status":"Exited (0) 2h ago"}` + "\n"),
			expectedLen: 2,
		},
		{
			name: "empty service name skipped",
			input: []byte(`{"id":"abc","service":"","state":"running",` +
				`"createdat":"2026-05-25 12:00:00 +0000 UTC","status":"Up 1h"}` + "\n"),
			expectedLen: 0,
		},
		{
			name:        "malformed json",
			input:       []byte(`{invalid json}` + "\n"),
			expectedLen: 0,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := docker.ParsePsOutput(tc.input)

			if tc.name == "malformed json" {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tc.expectedLen)
		})
	}
}

func TestParsePsOutputServiceRuntimeData(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		input    []byte
		expected map[string]*domain.ServiceRuntimeData
	}

	tt := []testCase{
		{
			name: "running service",
			input: []byte(`{"id":"abc","service":"web","state":"running",` +
				`"createdat":"2026-05-25 12:00:00 +0000 UTC","status":"Up 1h"}` + "\n"),
			expected: map[string]*domain.ServiceRuntimeData{
				"web": {
					ContainerID: "abc",
					State:       domain.ServiceStateRunning,
					Status:      "Up 1h",
				},
			},
		},
		{
			name: "exited service",
			input: []byte(`{"id":"def","service":"db","state":"exited",` +
				`"createdat":"2026-05-25 10:00:00 +0000 UTC","status":"Exited (0)"}` + "\n"),
			expected: map[string]*domain.ServiceRuntimeData{
				"db": {
					ContainerID: "def",
					State:       domain.ServiceStateExited,
					Status:      "Exited (0)",
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := docker.ParsePsOutput(tc.input)
			require.NoError(t, err)

			for serviceName, expectedRuntime := range tc.expected {
				rt, ok := result[serviceName]
				require.True(t, ok, "service %s not found in result", serviceName)
				assert.Equal(t, expectedRuntime.ContainerID, rt.ContainerID)
				assert.Equal(t, expectedRuntime.State, rt.State)
				assert.Equal(t, expectedRuntime.Status, rt.Status)
			}
		})
	}
}

func TestParseState(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		input    string
		expected domain.ServiceState
	}

	tt := []testCase{
		{name: "running", input: "running", expected: domain.ServiceStateRunning},
		{name: "exited", input: "exited", expected: domain.ServiceStateExited},
		{name: "paused", input: "paused", expected: domain.ServiceStatePaused},
		{name: "restarting", input: "restarting", expected: domain.ServiceStateRestarting},
		{name: "dead", input: "dead", expected: domain.ServiceStateDead},
		{name: "unknown state", input: "removing", expected: domain.ServiceStateUnknown},
		{name: "empty string", input: "", expected: domain.ServiceStateUnknown},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result := docker.ParseState(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
