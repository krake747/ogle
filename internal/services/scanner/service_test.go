package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/services/scanner"
)

const (
	filenameComposeYML        = "compose.yml"
	filenameComposeYAML       = "compose.yaml"
	filenameDockerComposeYML  = "docker-compose.yml"
	filenameDockerComposeYAML = "docker-compose.yaml"
)

func TestKnownFilenames(t *testing.T) {
	t.Parallel()

	svc := scanner.New()

	t.Run("returns canonical filenames in priority order", func(t *testing.T) {
		t.Parallel()

		result := svc.KnownFilenames()

		require.Equal(t, []string{
			filenameComposeYML,
			filenameComposeYAML,
			filenameDockerComposeYML,
			filenameDockerComposeYAML,
		}, result)
	})

	t.Run("mutating returned slice does not affect subsequent call", func(t *testing.T) {
		t.Parallel()

		result := svc.KnownFilenames()
		result[0] = "mutated"

		second := svc.KnownFilenames()

		require.Equal(t, []string{
			filenameComposeYML,
			filenameComposeYAML,
			filenameDockerComposeYML,
			filenameDockerComposeYAML,
		}, second)
	})
}

func TestScanAll(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		setup    func(tc *testCase, dir string)
		dir      string
		filename string

		// assert
		expected []string
	}

	cases := []testCase{
		{
			name:     "compose.yml present",
			filename: filenameComposeYML,
			setup: func(tc *testCase, dir string) {
				tc.dir = dir
				tc.expected = []string{filepath.Join(dir, filenameComposeYML)}
			},
		},
		{
			name:     "compose.yaml present",
			filename: filenameComposeYAML,
			setup: func(tc *testCase, dir string) {
				tc.dir = dir
				tc.expected = []string{filepath.Join(dir, filenameComposeYAML)}
			},
		},
		{
			name:     "docker-compose.yml present",
			filename: filenameDockerComposeYML,
			setup: func(tc *testCase, dir string) {
				tc.dir = dir
				tc.expected = []string{filepath.Join(dir, filenameDockerComposeYML)}
			},
		},
		{
			name:     "docker-compose.yaml present",
			filename: filenameDockerComposeYAML,
			setup: func(tc *testCase, dir string) {
				tc.dir = dir
				tc.expected = []string{filepath.Join(dir, filenameDockerComposeYAML)}
			},
		},
		{
			name: "all four present returns full ordered slice",
			setup: func(tc *testCase, dir string) {
				tc.dir = dir
				tc.expected = []string{
					filepath.Join(dir, filenameComposeYML),
					filepath.Join(dir, filenameComposeYAML),
					filepath.Join(dir, filenameDockerComposeYML),
					filepath.Join(dir, filenameDockerComposeYAML),
				}
			},
		},
		{
			name: "directory does not exist returns empty",
			setup: func(tc *testCase, dir string) {
				tc.dir = filepath.Join(dir, "nonexistent")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tc.setup(&tc, t.TempDir())

			if tc.filename != "" {
				require.NoError(
					t,
					os.WriteFile(filepath.Join(tc.dir, tc.filename), []byte{}, 0o600),
				)
			} else if tc.expected != nil {
				for _, path := range tc.expected {
					require.NoError(t, os.WriteFile(path, []byte{}, 0o600))
				}
			}

			svc := scanner.New()
			result := svc.ScanAll(tc.dir)

			if tc.expected == nil {
				require.Empty(t, result)
			} else {
				require.Equal(t, tc.expected, result)
			}
		})
	}
}
