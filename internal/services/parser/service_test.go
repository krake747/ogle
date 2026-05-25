package parser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/services/parser"
)

func TestParse(t *testing.T) { //nolint:funlen // table-driven test with multiple cases
	t.Parallel()

	type testCase struct {
		name string
		// arrange
		readFileFn func(string) ([]byte, error)
		path       string
		// assert
		expectedProject *domain.Project
		expectedError   error
	}

	testPath := "/some/path/compose.yaml"

	cases := []testCase{
		{
			name: "valid compose file with ports",
			readFileFn: func(_ string) ([]byte, error) {
				yaml := "name: myapp\n" +
					"services:\n" +
					"  web:\n" +
					"    image: nginx\n" +
					"    ports:\n" +
					"      - \"8080:80\"\n" +
					"      - \"4443:443\"\n" +
					"  db:\n" +
					"    image: postgres\n" +
					"    ports:\n" +
					"      - \"5432\"\n"

				return []byte(yaml), nil
			},
			path: testPath,
			expectedProject: &domain.Project{
				Name: "myapp",
				File: testPath,
				Services: []domain.ServiceDef{
					{Name: "db", Image: "postgres", Ports: []string{"\u21925432/tcp"}},
					{
						Name:  "web",
						Image: "nginx",
						Ports: []string{"8080\u219280/tcp", "4443\u2192443/tcp"},
					},
				},
			},
		},
		{
			name: "invalid YAML returns parse error",
			readFileFn: func(_ string) ([]byte, error) {
				return []byte("invalid: yaml: [bad"), nil
			},
			path:          testPath,
			expectedError: parser.ErrParseComposeFile,
		},
		{
			name: "file not found returns read error",
			readFileFn: func(_ string) ([]byte, error) {
				return nil, os.ErrNotExist
			},
			path:          "/nonexistent/compose.yaml",
			expectedError: parser.ErrReadComposeFile,
		},
		{
			name: "compose with no services returns empty project",
			readFileFn: func(_ string) ([]byte, error) {
				return []byte("name: empty"), nil
			},
			path: testPath,
			expectedProject: &domain.Project{
				Name:     "empty",
				File:     testPath,
				Services: []domain.ServiceDef{},
			},
		},
		{
			name: "name from directory when compose has no name",
			readFileFn: func(_ string) ([]byte, error) {
				return []byte("services:\n  web:\n    image: nginx"), nil
			},
			path: "/myproject/compose.yaml",
			expectedProject: &domain.Project{
				Name: "myproject",
				File: "/myproject/compose.yaml",
				Services: []domain.ServiceDef{
					{Name: "web", Image: "nginx"},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := parser.New()
			s.ReadFileFn = tc.readFileFn

			project, err := s.Parse(tc.path)

			if tc.expectedError != nil {
				require.ErrorIs(t, err, tc.expectedError)
				assert.Nil(t, project)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, project)
			assert.Equal(t, tc.expectedProject.Name, project.Name)
			assert.Equal(t, tc.expectedProject.File, project.File)
			require.Len(t, project.Services, len(tc.expectedProject.Services))

			for i, expected := range tc.expectedProject.Services {
				assert.Equal(t, expected.Name, project.Services[i].Name)
				assert.Equal(t, expected.Image, project.Services[i].Image)
				assert.Equal(t, expected.Ports, project.Services[i].Ports)
			}
		})
	}
}

func TestParseDefaultReadFileFn(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")
	content := []byte("name: default-fn\nservices:\n  web:\n    image: nginx")
	require.NoError(t, os.WriteFile(path, content, 0o600))

	s := parser.New()
	project, err := s.Parse(path)
	require.NoError(t, err)
	require.NotNil(t, project)
	assert.Equal(t, "default-fn", project.Name)
}
