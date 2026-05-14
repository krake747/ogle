// Package dashboard2 implements the project dashboard flow.
package dashboard2

import (
	"context"
	"fmt"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
	"github.com/ma-tf/ogle/internal/services/docker/connection"
)

const gracePeriodDuration = 5 * time.Second

// Model is the dashboard flow orchestrator.
type Model struct {
	ctx  context.Context
	conn *connection.Machine
	w, h int
}

// New returns a Model in the Connecting state.
func New(ctx context.Context) tea.Model {
	return Model{
		ctx:  ctx,
		conn: connection.New(),
		w:    0,
		h:    0,
	}
}

// Init fires the initial Docker Connect ping, a grace-period tick (5s), and a
// 1-second retry tick that drives the retry loop when the daemon is unavailable.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		svcdocker.Connect(m.ctx),
		tea.Tick(gracePeriodDuration, func(_ time.Time) tea.Msg {
			return gracePeriodExpiredMsg{}
		}),
		tea.Every(time.Second, func(_ time.Time) tea.Msg {
			return retryTickMsg{}
		}),
	)
}

// Update drives the connection state machine and handles window resize.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height

	case msgs.DaemonConnected:
		m.conn.HandleConnected()

	case msgs.DaemonUnavailable:
		m.conn.HandleUnavailable(time.Now().UTC())

	case gracePeriodExpiredMsg:
		m.conn.HandleGracePeriodExpired(time.Now().UTC())

	case retryTickMsg:
		if m.conn.IsRetryDue(time.Now().UTC()) {
			return m, svcdocker.Connect(m.ctx)
		}
	}

	return m, nil
}

// View renders connection status.
func (m Model) View() tea.View {
	switch m.conn.ConnectState() {
	case connection.ConnectStateConnecting:
		return tea.NewView("Connecting to Docker…")
	case connection.ConnectStateConnected:
		return tea.NewView("Connected")
	case connection.ConnectStateUnavailable:
		secs := int(m.conn.Remaining().Seconds())

		return tea.NewView(fmt.Sprintf("Docker unavailable — retrying in %ds…", secs))
	default:
		return tea.NewView("dashboard2")
	}
}

type (
	gracePeriodExpiredMsg struct{}
	retryTickMsg          struct{}
)
