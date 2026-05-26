package app_test

import (
	"context"
	"log/slog"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

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

func newModel(t *testing.T) (
	app.Model, func() error, *dockermocks.MockDocker, *watchermocks.MockWatcher,
) {
	t.Helper()

	ctx := context.Background()
	cfg := config.Defaults()
	log := slog.Default()
	th := theme.Default()

	mockDocker := dockermocks.NewMockDocker(t)
	mockParser := parsermocks.NewMockParser(t)
	mockWatcher := watchermocks.NewMockWatcher(t)

	mockWatcher.EXPECT().Close().Return(nil)

	m, cleanup, err := app.New(ctx, cfg, "", "", log, th, mockDocker, mockParser, mockWatcher)
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
			{Name: "web", Image: "nginx:latest"},
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
