// Package daemonstatus implements a Bubble Tea sub-model that tracks Docker
// daemon connectivity and renders connection status text.
package daemonstatus

import (
	"context"
	"fmt"
	"math"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
	"github.com/ma-tf/ogle/internal/services/docker/connection"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	gracePeriodDuration = 10 * time.Second
	healthCheckInterval = 2 * time.Second
)

// Model tracks Docker daemon connectivity and renders status text.
type Model struct {
	ctx  context.Context
	conn *connection.Machine
	spn  spinner.Model
	th   *theme.Theme
}

// New returns a Model that shares the given Machine.
func New(ctx context.Context, conn *connection.Machine, th *theme.Theme) Model {
	return Model{
		ctx:  ctx,
		conn: conn,
		spn:  spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		th:   th,
	}
}

// Init fires the initial Docker Connect ping, a grace-period tick, and the
// first spinner tick. The 1-second retry loop is started on-demand when the
// daemon becomes unavailable (see DaemonUnavailable and DaemonGraceExpired
// handlers).
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		svcdocker.Connect(m.ctx),
		tea.Tick(gracePeriodDuration, func(_ time.Time) tea.Msg {
			return msgs.DaemonGraceExpired{}
		}),
		m.spn.Tick,
	)
}

// Update handles connection-related messages and drives the Machine.
// DaemonTick and spinner.TickMsg handlers chain the next tick via tea.Tick
// so the loop is self-sustaining.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case msgs.DaemonConnected:
		m.conn.HandleConnected()

		cmds = append(cmds, pollDaemonCmd())

	case msgs.DaemonUnavailable:
		m.conn.HandleUnavailable(time.Now().UTC())

		cmds = append(cmds, daemonTickCmd())

	case msgs.DaemonGraceExpired:
		if m.conn.HandleGracePeriodExpired(time.Now().UTC()) {
			cmds = append(cmds, daemonTickCmd())
		}

	case msgs.DaemonTick:
		if m.conn.ConnectState() == connection.ConnectStateUnavailable {
			if m.conn.IsRetryDue(time.Now().UTC()) {
				cmds = append(cmds, svcdocker.Connect(m.ctx))
			} else {
				cmds = append(cmds, daemonTickCmd())
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd

		m.spn, cmd = m.spn.Update(msg)

		cmds = append(cmds, cmd)

	case msgs.DaemonPoll:
		if m.conn.ConnectState() == connection.ConnectStateConnected {
			cmds = append(cmds, svcdocker.Connect(m.ctx))
		}
	}

	if m.conn.IsRetryDue(time.Now().UTC()) {
		cmds = append(cmds, svcdocker.Connect(m.ctx))
	}

	return m, tea.Batch(cmds...)
}

func daemonTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return msgs.DaemonTick{}
	})
}

func pollDaemonCmd() tea.Cmd {
	return tea.Tick(healthCheckInterval, func(_ time.Time) tea.Msg {
		return msgs.DaemonPoll{}
	})
}

// View renders connection status text.
func (m Model) View() string {
	switch m.conn.ConnectState() {
	case connection.ConnectStateConnecting:
		faded := lipgloss.NewStyle().Faint(true).Render("🐳")
		label := lipgloss.NewStyle().Foreground(m.th.StateTransient).Render("○")

		return faded + " " + label + " " + m.spn.View()

	case connection.ConnectStateConnected:
		live := lipgloss.NewStyle().Foreground(m.th.StateRunning).Render("● LIVE")

		return "🐳 " + live

	case connection.ConnectStateUnavailable:
		secs := int(math.Ceil(m.conn.Remaining().Seconds()))

		countdown := "(now)"
		if secs >= 1 {
			countdown = fmt.Sprintf("(%ds)", secs)
		}

		faded := lipgloss.NewStyle().Faint(true).Render("🐳")
		label := lipgloss.NewStyle().Foreground(m.th.StateMuted).Render("○")

		return faded + " " + label + " " + countdown

	default:
		return "dashboard"
	}
}
