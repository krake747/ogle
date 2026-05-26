package app_test

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"

	"github.com/ma-tf/ogle/config"
	"github.com/ma-tf/ogle/internal/app"
	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	dockermocks "github.com/ma-tf/ogle/internal/services/docker/mocks"
	parsermocks "github.com/ma-tf/ogle/internal/services/parser/mocks"
	watchermocks "github.com/ma-tf/ogle/internal/services/watcher/mocks"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	testServiceName = "web"
	testStatusMsg   = "hello"
)

func newModel(t *testing.T) (
	app.Model, func() error, *dockermocks.MockDocker, *watchermocks.MockWatcher,
) {
	t.Helper()

	ctx := context.Background()
	cfg := config.Defaults()
	logger := slog.Default()
	th := theme.Default()

	mockDocker := dockermocks.NewMockDocker(t)
	mockParser := parsermocks.NewMockParser(t)
	mockWatcher := watchermocks.NewMockWatcher(t)

	mockWatcher.EXPECT().Close().Return(nil)

	m, cleanup, err := app.New(ctx, cfg, "", "", logger, th, mockDocker, mockParser, mockWatcher)
	require.NoError(t, err)

	return m, cleanup, mockDocker, mockWatcher
}

func TestInit(t *testing.T) {
	t.Parallel()

	m, cleanup, mockDocker, mockWatcher := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	mockDocker.EXPECT().Connect(mock.Anything).Return(func() tea.Msg { return nil }).Maybe()
	mockWatcher.EXPECT().Snapshot().Return(nil)

	cmd := m.Init()
	require.NotNil(t, cmd)
}

func TestUpdateQuit(t *testing.T) {
	t.Parallel()

	m, cleanup, _, _ := newModel(t) // only model and cleanup needed
	defer func() {
		require.NoError(t, cleanup())
	}()

	result, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	msg := cmd()
	require.NotNil(t, msg)
}

func TestUpdateFileAvailabilityChanged(t *testing.T) {
	t.Parallel()

	m, cleanup, _, watcherMock := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	watcherMock.EXPECT().Next().Return(func() tea.Msg { return nil })

	msg := msgs.FileAvailabilityChanged{Files: []string{"/tmp/compose.yaml"}}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)
	require.NotNil(t, cmd)
}

func TestUpdateProjectLoaded(t *testing.T) {
	t.Parallel()

	m, cleanup, _, _ := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	assert.Equal(t, app.PhaseStartup, app.GetPhase(&m), "should start in startup phase")

	project := &domain.Project{
		Name: "myapp",
		File: "/path/to/compose.yaml",
		Services: []domain.ServiceDef{
			{Name: testServiceName, Image: "nginx:latest"},
		},
	}

	result, cmd := m.Update(msgs.ProjectLoaded{Project: project})
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	appModel, ok := result.(app.Model)
	require.True(t, ok, "expected app.Model, got %T", result)

	assert.Equal(t, app.PhaseDashboard, app.GetPhase(&appModel),
		"should transition to dashboard phase")

	dash := app.GetDashboard(&appModel)
	assert.NotEqual(t, dashboard.Model{}, dash, "dashboard sub-model should be created")

	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	require.True(t, ok, "expected BatchMsg, got %T", msg)

	found := false

	for _, entry := range batch {
		if tc, tcOk := entry().(msgs.TopbarContext); tcOk {
			assert.Equal(t, "dashboard", tc.Phase)
			assert.Equal(t, "compose.yaml", tc.File)

			found = true

			break
		}
	}

	require.True(t, found, "expected TopbarContext in BatchMsg")
}

func TestView(t *testing.T) {
	t.Parallel()

	m, cleanup, _, _ := newModel(t) // only model and cleanup needed
	defer func() {
		require.NoError(t, cleanup())
	}()

	v := m.View()
	require.NotNil(t, v)
	assert.NotEmpty(t, v.Content)
}

func newModelWithConfig(t *testing.T, configPath string) (
	app.Model, func() error, *theme.Theme,
) {
	t.Helper()

	ctx := context.Background()
	cfg := config.Defaults()
	logger := slog.Default()
	th := theme.Default()

	mockDocker := dockermocks.NewMockDocker(t)
	mockParser := parsermocks.NewMockParser(t)
	mockWatcher := watchermocks.NewMockWatcher(t)
	mockWatcher.EXPECT().Close().Return(nil)

	m, cleanup, err := app.New(
		ctx, cfg, configPath, "", logger, th, mockDocker, mockParser, mockWatcher,
	)
	require.NoError(t, err)

	return m, cleanup, th
}

func TestUpdateSettingsApplied(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name              string
		themeName         string
		logBufferCap      int
		expectThemeLoaded bool
	}

	for _, tc := range []testCase{
		{
			name:              "known theme loads and persists",
			themeName:         "solarized_dark",
			logBufferCap:      2000,
			expectThemeLoaded: true,
		},
		{
			name:              "unknown theme keeps existing theme",
			themeName:         "not-a-theme",
			logBufferCap:      2000,
			expectThemeLoaded: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			configPath := filepath.Join(dir, "config.yaml")

			require.NoError(t, config.Save(configPath, config.Defaults()))

			m, cleanup, originalTheme := newModelWithConfig(t, configPath)
			defer func() {
				require.NoError(t, cleanup())
			}()

			msg := msgs.SettingsApplied{Theme: tc.themeName, LogBufferCap: tc.logBufferCap}
			result, cmd := m.Update(msg)
			require.NotNil(t, result)
			require.NotNil(t, cmd)

			resultMsg := cmd()
			changed, ok := resultMsg.(theme.Changed)
			require.True(t, ok)

			if tc.expectThemeLoaded {
				assert.NotSame(t, originalTheme, changed.Theme)
			} else {
				assert.Same(t, originalTheme, changed.Theme)
			}

			data, err := os.ReadFile(configPath)
			require.NoError(t, err, "config should be persisted")

			var savedCfg config.Config
			require.NoError(t, yaml.Unmarshal(data, &savedCfg))
			assert.Equal(t, tc.themeName, savedCfg.Theme)
			assert.Equal(t, tc.logBufferCap, savedCfg.LogBufferCap)
		})
	}
}

func TestUpdateSettingsAppliedConfigSaveFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "nonexistent", "config.yaml")

	m, cleanup, originalTheme := newModelWithConfig(t, configPath)
	defer func() {
		require.NoError(t, cleanup())
	}()

	msg := msgs.SettingsApplied{Theme: "solarized_dark", LogBufferCap: 2000}
	result, cmd := m.Update(msg)
	require.NotNil(t, result)
	require.NotNil(t, cmd)

	resultMsg := cmd()
	changed, ok := resultMsg.(theme.Changed)
	require.True(t, ok)

	assert.NotSame(t, originalTheme, changed.Theme)

	_, err := os.Stat(configPath)
	require.True(t, os.IsNotExist(err))
}

func TestUpdateWindowSize(t *testing.T) {
	t.Parallel()

	m, cleanup, _, _ := newModel(t)
	defer func() {
		require.NoError(t, cleanup())
	}()

	result, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.NotNil(t, result)
	require.Nil(t, cmd)

	m2, ok := result.(app.Model)
	require.True(t, ok)
	assert.Equal(t, 120, app.GetWidth(&m2))
	assert.Equal(t, 40, app.GetHeight(&m2))
}

func TestViewPhaseContent(t *testing.T) {
	t.Parallel()

	project := &domain.Project{
		Name: "myapp",
		File: "/path/to/compose.yaml",
		Services: []domain.ServiceDef{
			{Name: testServiceName, Image: "nginx:latest"},
		},
	}

	type testCase struct {
		name string
		// arrange
		setup func(m tea.Model) tea.Model
		// assert
		expectedContains string
	}

	for _, tc := range []testCase{
		{
			name: "startup phase shows startup body",
			setup: func(m tea.Model) tea.Model {
				return m
			},
			expectedContains: "scanning for compose files",
		},
		{
			name: "dashboard phase shows dashboard body",
			setup: func(m tea.Model) tea.Model {
				r, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
				m = r

				r, _ = m.Update(msgs.ProjectLoaded{Project: project})

				return r
			},
			expectedContains: testServiceName,
		},
		{
			name: "watching phase shows watching body",
			setup: func(m tea.Model) tea.Model {
				r, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
				m = r

				r, _ = m.Update(msgs.FileRemoved{File: "test.yaml"})

				return r
			},
			expectedContains: "compose file unavailable",
		},
		{
			name: "about overlay composes on top",
			setup: func(m tea.Model) tea.Model {
				r, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
				m = r

				r, _ = m.Update(msgs.AboutVisibilityChanged{Visible: true})

				return r
			},
			expectedContains: "github.com/ma-tf/ogle",
		},
		{
			name: "display error routes to statusbar",
			setup: func(m tea.Model) tea.Model {
				r, _ := m.Update(msgs.DisplayError{Err: "oops"})

				return r
			},
			expectedContains: "oops",
		},
		{
			name: "display status routes to statusbar",
			setup: func(m tea.Model) tea.Model {
				r, _ := m.Update(msgs.DisplayStatus{Msg: testStatusMsg})

				return r
			},
			expectedContains: testStatusMsg,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, cleanup, _, _ := newModel(t)
			defer func() {
				require.NoError(t, cleanup())
			}()

			r := tc.setup(m)
			m2, ok := r.(app.Model)
			require.True(t, ok, "expected app.Model, got %T", r)

			v := m2.View()
			require.NotNil(t, v)
			assert.Contains(t, v.Content, tc.expectedContains)
		})
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name string
		// act
		msg tea.Msg
		// assert
		expectedMsg tea.Msg
		expectCmd   bool
	}

	cases := []testCase{
		{
			name:      "DaemonConnected produces command",
			msg:       msgs.DaemonConnected{},
			expectCmd: true,
		},
		{
			name:      "DaemonUnavailable produces command",
			msg:       msgs.DaemonUnavailable{Err: assert.AnError},
			expectCmd: true,
		},
		{
			name:      "DisplayError produces command",
			msg:       msgs.DisplayError{Err: "test error"},
			expectCmd: true,
		},
		{
			name:      "DisplayStatus produces command",
			msg:       msgs.DisplayStatus{Msg: testStatusMsg},
			expectCmd: true,
		},
		{
			name:      "ClearStatusMsg produces no command",
			msg:       msgs.ClearStatusMsg{},
			expectCmd: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			m, cleanup, _, _ := newModel(t)
			defer func() {
				require.NoError(t, cleanup())
			}()

			_, cmd := m.Update(tc.msg)

			switch {
			case tc.expectedMsg != nil:
				require.NotNil(t, cmd)
				require.Equal(t, tc.expectedMsg, cmd())
			case tc.expectCmd:
				require.NotNil(t, cmd)
			default:
				require.Nil(t, cmd)
			}
		})
	}
}
