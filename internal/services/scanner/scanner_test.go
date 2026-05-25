package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/services/scanner"
)

const (
	knownComposeYml  = "compose.yml"
	knownComposeYaml = "compose.yaml"
	knownDCYml       = "docker-compose.yml"
	knownDCYaml      = "docker-compose.yaml"
)

//nolint:gochecknoglobals // test-only constant slice
var knownFilenames = []string{knownComposeYml, knownComposeYaml, knownDCYml, knownDCYaml}

func TestKnownFilenames(t *testing.T) {
	t.Parallel()

	s := scanner.New()
	names := s.KnownFilenames()

	assert.Equal(t, knownFilenames, names)
}

//nolint:funlen // table-driven with multiple file-setup cases
func TestScanAll(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name     string
		setup    func(t *testing.T) string
		expected []string
	}

	tt := []testCase{
		{
			name: "no files found in empty dir",
			setup: func(t *testing.T) string {
				t.Helper()

				return t.TempDir()
			},
			expected: nil,
		},
		{
			name: "finds compose.yml",
			setup: func(t *testing.T) string {
				t.Helper()

				dir := t.TempDir()
				require.NoError(
					t,
					os.WriteFile(filepath.Join(dir, knownComposeYml), []byte(""), 0o600),
				)

				return dir
			},
			expected: []string{knownComposeYml},
		},
		{
			name: "finds compose.yml and compose.yaml in priority order",
			setup: func(t *testing.T) string {
				t.Helper()

				dir := t.TempDir()
				require.NoError(
					t,
					os.WriteFile(filepath.Join(dir, knownComposeYml), []byte(""), 0o600),
				)
				require.NoError(
					t,
					os.WriteFile(filepath.Join(dir, knownComposeYaml), []byte(""), 0o600),
				)

				return dir
			},
			expected: []string{knownComposeYml, knownComposeYaml},
		},
		{
			name: "skips missing files",
			setup: func(t *testing.T) string {
				t.Helper()

				dir := t.TempDir()
				require.NoError(t, os.WriteFile(filepath.Join(dir, knownDCYml), []byte(""), 0o600))

				return dir
			},
			expected: []string{knownDCYml},
		},
		{
			name: "all four files found in priority order",
			setup: func(t *testing.T) string {
				t.Helper()

				dir := t.TempDir()
				for _, name := range knownFilenames {
					require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(""), 0o600))
				}

				return dir
			},
			expected: knownFilenames,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := tc.setup(t)
			s := scanner.New()
			result := s.ScanAll(dir)

			if tc.expected == nil {
				assert.Empty(t, result)

				return
			}

			require.Len(t, result, len(tc.expected))

			for i, path := range result {
				assert.True(t, filepath.IsAbs(path), "path %q should be absolute", path)
				assert.Equal(t, tc.expected[i], filepath.Base(path))
			}
		})
	}
}
