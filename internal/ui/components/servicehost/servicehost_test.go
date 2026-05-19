package servicehost_test

import (
	"testing"

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

	return servicehost.New(th, def, "myproject", 80, 24, 1000)
}

func TestModel_DaemonConnected_ReturnsNextCmd(t *testing.T) {
	t.Parallel()

	m := newTestModel(t)

	updated, cmd := m.Update(msgs.DaemonConnected{})

	require.NotNil(t, cmd)

	_ = updated
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
