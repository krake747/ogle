package parser_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/services/parser"
)

//nolint:funlen
func TestParse(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		yaml  string
		setup func(tc *testCase, dir string)
		path  string

		// assert
		expected      domain.Project
		expectedError error
	}

	cases := []testCase{
		{
			name: "valid file with name field",
			yaml: "name: myproject\nservices:\n  web:\n    image: nginx\n",
			setup: func(tc *testCase, dir string) {
				tc.path = filepath.Join(dir, "compose.yaml")
				tc.expected.File = tc.path
			},
			expected: domain.Project{
				Name: "myproject",
				Services: []domain.ServiceDef{
					{Name: "web", Image: "nginx"},
				},
			},
			expectedError: nil,
		},
		{
			name: "name falls back to directory name",
			yaml: "services:\n  web:\n    image: nginx\n",
			setup: func(tc *testCase, dir string) {
				full := filepath.Join(dir, "myproject")
				require.NoError(t, os.MkdirAll(full, 0o700))
				tc.path = filepath.Join(full, "compose.yaml")
				tc.expected.File = tc.path
			},
			expected: domain.Project{
				Name: "myproject",
				Services: []domain.ServiceDef{
					{Name: "web", Image: "nginx"},
				},
			},
			expectedError: nil,
		},
		{
			name: "multiple services",
			yaml: "name: multi\nservices:\n  alpha:\n    image: alpine\n  beta:\n    image: busybox\n",
			setup: func(tc *testCase, dir string) {
				tc.path = filepath.Join(dir, "compose.yaml")
				tc.expected.File = tc.path
			},
			expected: domain.Project{
				Name: "multi",
				Services: []domain.ServiceDef{
					{Name: "alpha", Image: "alpine"},
					{Name: "beta", Image: "busybox"},
				},
			},
			expectedError: nil,
		},
		{
			name: "file does not exist",
			setup: func(tc *testCase, dir string) {
				tc.path = filepath.Join(dir, "compose.yaml")
			},
			expectedError: parser.ErrReadComposeFile,
		},
		{
			name: "invalid YAML",
			yaml: "{",
			setup: func(tc *testCase, dir string) {
				tc.path = filepath.Join(dir, "compose.yaml")
			},
			expectedError: parser.ErrParseComposeFile,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.setup(&tc, t.TempDir())

			if tc.yaml != "" {
				require.NoError(t, os.WriteFile(tc.path, []byte(tc.yaml), 0o600))
			}

			svc := parser.New(t.Context(), slog.New(slog.NewTextHandler(os.Stderr, nil)))
			result, err := svc.Parse(tc.path)

			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
				require.Nil(t, result)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, *result)
		})
	}
}
