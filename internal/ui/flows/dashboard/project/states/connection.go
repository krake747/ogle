package states

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	svcdocker "github.com/ma-tf/ogle/internal/services/docker"
	"github.com/ma-tf/ogle/internal/ui/components/inspector"
)

// RetryIntervalSeconds is the countdown value set when the daemon becomes
// unavailable. Exported so tests can assert the initial countdown.
const RetryIntervalSeconds = 60

// ConnectionMachine is a state machine that tracks Docker daemon connectivity.
type ConnectionMachine struct {
	state       inspector.ConnectState
	unavailable inspector.UnavailableState
}

// NewConnectionMachine returns a ConnectionMachine with the given initial state
// and seconds-until-retry value. Pass 0 for secondsUntilRetry when the initial
// state is not ConnectStateUnavailable.
func NewConnectionMachine(state inspector.ConnectState, secondsUntilRetry int) *ConnectionMachine {
	return &ConnectionMachine{
		state:       state,
		unavailable: inspector.UnavailableState{SecondsUntilRetry: secondsUntilRetry},
	}
}

// HandleConnected transitions to ConnectStateConnected.
func (cm *ConnectionMachine) HandleConnected() {
	cm.state = inspector.ConnectStateConnected
}

// HandleUnavailable transitions to Unavailable and starts the retry countdown.
// No-op if the current state is not ConnectStateConnected.
func (cm *ConnectionMachine) HandleUnavailable() tea.Cmd {
	if cm.state != inspector.ConnectStateConnected {
		return nil
	}

	cm.state = inspector.ConnectStateUnavailable
	cm.unavailable = inspector.UnavailableState{SecondsUntilRetry: RetryIntervalSeconds}

	return startCountdown()
}

// HandleGracePeriodExpired transitions to Unavailable when the initial grace
// period elapses without a successful connect. No-op if not ConnectStateConnecting.
func (cm *ConnectionMachine) HandleGracePeriodExpired() tea.Cmd {
	if cm.state != inspector.ConnectStateConnecting {
		return nil
	}

	cm.state = inspector.ConnectStateUnavailable
	cm.unavailable = inspector.UnavailableState{SecondsUntilRetry: RetryIntervalSeconds}

	return startCountdown()
}

// HandleRetryTick decrements the countdown. When it reaches zero the machine
// transitions back to ConnectStateConnecting and fires svcdocker.Connect.
// No-op if not ConnectStateUnavailable.
func (cm *ConnectionMachine) HandleRetryTick(ctx context.Context) tea.Cmd {
	if cm.state != inspector.ConnectStateUnavailable {
		return nil
	}

	cm.unavailable.SecondsUntilRetry--

	if cm.unavailable.SecondsUntilRetry <= 0 {
		cm.state = inspector.ConnectStateConnecting

		return svcdocker.Connect(ctx)
	}

	return startCountdown()
}

// ConnectState returns the current connection state.
func (cm *ConnectionMachine) ConnectState() inspector.ConnectState {
	return cm.state
}

// Unavailable returns the current unavailable state (countdown, etc.).
func (cm *ConnectionMachine) Unavailable() inspector.UnavailableState {
	return cm.unavailable
}

// startCountdown returns a one-shot one-second timer that fires retryTickMsg.
func startCountdown() tea.Cmd {
	return tea.Every(time.Second, func(_ time.Time) tea.Msg {
		return retryTickMsg{}
	})
}
