package topbar

import (
	"context"
	"fmt"
	"math"
	"strings"
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

// Phase identifies the active UI phase for context text rendering.
type Phase int

// Phase values.
const (
	PhaseStartup Phase = iota
	PhaseDashboard
	PhaseWatching
)

// Model holds top bar state: the active phase, project file, daemon connection
// machine, spinner, theme, and terminal width.
type Model struct {
	phase       Phase
	projectFile string
	conn        *connection.Machine
	spn         spinner.Model
	th          *theme.Theme
	width       int
	ctx         context.Context
}

// New returns a Model in PhaseStartup with no project file.
func New(ctx context.Context, conn *connection.Machine, th *theme.Theme) Model {
	return Model{
		phase:       PhaseStartup,
		projectFile: "",
		ctx:         ctx,
		conn:        conn,
		spn:         spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		th:          th,
		width:       0,
	}
}

// Init fires the initial Docker connect, grace-period tick, and spinner tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		svcdocker.Connect(m.ctx),
		tea.Tick(gracePeriodDuration, func(_ time.Time) tea.Msg {
			return msgs.DaemonGraceExpired{}
		}),
		m.spn.Tick,
	)
}

// Update handles daemon connectivity messages, spinner ticks, window
// resize events, and topbar context changes.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case msgs.TopbarContext:
		switch msg.Phase {
		case "startup":
			m.phase = PhaseStartup
		case "dashboard":
			m.phase = PhaseDashboard
		case "watching":
			m.phase = PhaseWatching
		}

		m.projectFile = msg.File

	case msgs.ThemeChanged:
		m.th = msg.Theme

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

func (m Model) contextText() string {
	switch m.phase {
	case PhaseStartup:
		return "scanning for compose files"
	case PhaseDashboard:
		return m.projectFile
	case PhaseWatching:
		return "disconnected"
	default:
		return ""
	}
}

func (m Model) renderDaemonStatus() string {
	switch m.conn.ConnectState() {
	case connection.ConnectStateConnecting:
		label := lipgloss.NewStyle().
			Foreground(m.th.StateTransient).
			Background(m.th.TopbarBackground).
			Render("🐳 ○")

		return label + " " + m.spn.View()

	case connection.ConnectStateConnected:
		live := lipgloss.NewStyle().
			Foreground(m.th.StateRunning).
			Background(m.th.TopbarBackground).
			Render("🐳 ● LIVE")

		return live

	case connection.ConnectStateUnavailable:
		secs := int(math.Ceil(m.conn.Remaining().Seconds()))
		countdown := "(now)"

		if secs >= 1 {
			countdown = fmt.Sprintf("(%ds)", secs)
		}

		label := lipgloss.NewStyle().
			Foreground(m.th.StateMuted).
			Background(m.th.TopbarBackground).
			Render("🐳 ○")

		return label + " " + countdown
	default:
		return ""
	}
}

// View renders the top bar: faint "ogle" prefix + phase context on the left,
// Docker daemon status on the right, right-aligned via padding.
func (m Model) View() tea.View {
	bg := m.th.TopbarBackground
	brandStyle := lipgloss.NewStyle().Foreground(m.th.Subtext).Background(bg)
	contextStyle := lipgloss.NewStyle().Foreground(m.th.Subtext).Background(bg)
	spacerStyle := lipgloss.NewStyle().Background(bg)

	left := brandStyle.Render("ogle") + contextStyle.Render("  "+m.contextText())
	right := m.renderDaemonStatus()

	leftW := lipgloss.Width(left)
	rightW := lipgloss.Width(right)
	pad := max(m.width-leftW-rightW, 0)

	return tea.NewView(left + spacerStyle.Render(strings.Repeat(" ", pad)) + right)
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
