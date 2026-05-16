package servicehost_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/servicehost"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const testServiceName = "api"

func newTestModel(t *testing.T) servicehost.Model {
	t.Helper()

	th := theme.Default()
	def := domain.ServiceDef{Name: testServiceName}

	return servicehost.New(th, def, "myproject", 80, 24)
}

func TestModel_DaemonConnected_ReturnsNextCmd(t *testing.T) {
	t.Parallel()

	m := newTestModel(t)

	updated, cmd := m.Update(msgs.DaemonConnected{})

	require.NotNil(t, cmd)

	_ = updated
}

func TestModel_LogLine_ForwardsToLogPane(t *testing.T) {
	t.Parallel()

	m := newTestModel(t)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(msgs.DaemonConnected{})

	updated, cmd := m.Update(msgs.LogLine{
		Text:        "hello world",
		ServiceName: testServiceName,
	})

	require.NotNil(t, cmd)
	require.Contains(t, updated.View(), "hello world")
}

func TestModel_LogStreamError_ReSubscribes(t *testing.T) {
	t.Parallel()

	m := newTestModel(t)
	m, _ = m.Update(msgs.DaemonConnected{})

	_, cmd := m.Update(msgs.LogStreamError{
		Err:         nil,
		ServiceName: testServiceName,
	})

	require.NotNil(t, cmd)
}

func TestModel_LogStreamContainerNotFound_ReSubscribes(t *testing.T) {
	t.Parallel()

	m := newTestModel(t)
	m, _ = m.Update(msgs.DaemonConnected{})

	_, cmd := m.Update(msgs.LogStreamContainerNotFound{
		ServiceName: testServiceName,
	})

	require.NotNil(t, cmd)
}

func TestModel_MultipleLogLines_Accumulate(t *testing.T) {
	t.Parallel()

	m := newTestModel(t)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(msgs.DaemonConnected{})

	m, _ = m.Update(msgs.LogLine{Text: "line 1", ServiceName: testServiceName})
	m, _ = m.Update(msgs.LogLine{Text: "line 2", ServiceName: testServiceName})
	m, _ = m.Update(msgs.LogLine{Text: "line 3", ServiceName: testServiceName})

	view := m.View()

	require.Contains(t, view, "line 1")
	require.Contains(t, view, "line 2")
	require.Contains(t, view, "line 3")
}
