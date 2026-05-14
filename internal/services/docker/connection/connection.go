// Package connection provides Docker daemon connectivity state tracking.
// Machine is a pure state machine with no bubbletea dependencies.
package connection

import "time"

// ConnectState represents Docker daemon connectivity.
type ConnectState int

// ConnectState values.
const (
	ConnectStateConnecting ConnectState = iota
	ConnectStateConnected
	ConnectStateUnavailable
)

// RetryInterval is the number of seconds to wait before retrying a connection.
const RetryInterval = 10

// Machine tracks Docker daemon connectivity state.
// All methods are pure state transitions — no tea imports, no command returns.
type Machine struct {
	state   ConnectState
	retryAt *time.Time // non-nil when Unavailable
}

// New returns a Machine in the Connecting state.
func New() *Machine {
	return &Machine{
		state:   ConnectStateConnecting,
		retryAt: nil,
	}
}

// HandleConnected transitions to Connected and clears the retry deadline.
func (cm *Machine) HandleConnected() {
	cm.state = ConnectStateConnected
	cm.retryAt = nil
}

// HandleUnavailable transitions to Unavailable and sets the retry deadline
// to now + RetryInterval seconds. Idempotent — calling it when already
// Unavailable updates the deadline but returns false.
func (cm *Machine) HandleUnavailable(now time.Time) bool {
	already := cm.state == ConnectStateUnavailable
	cm.state = ConnectStateUnavailable
	t := now.Add(RetryInterval * time.Second)
	cm.retryAt = &t

	return !already
}

// HandleGracePeriodExpired transitions from Connecting to Unavailable and
// sets the retry deadline to now + RetryInterval seconds.
// Returns false if not in the Connecting state.
func (cm *Machine) HandleGracePeriodExpired(now time.Time) bool {
	if cm.state != ConnectStateConnecting {
		return false
	}

	cm.state = ConnectStateUnavailable
	t := now.Add(RetryInterval * time.Second)
	cm.retryAt = &t

	return true
}

// IsRetryDue checks whether the retry deadline has passed. If so, transitions
// to Connecting, clears the deadline, and returns true. Otherwise returns false.
func (cm *Machine) IsRetryDue(now time.Time) bool {
	if cm.state != ConnectStateUnavailable || cm.retryAt == nil {
		return false
	}

	if now.Equal(*cm.retryAt) || now.After(*cm.retryAt) {
		cm.state = ConnectStateConnecting
		cm.retryAt = nil

		return true
	}

	return false
}

// ConnectState returns the current connection state.
func (cm *Machine) ConnectState() ConnectState {
	return cm.state
}

// RetryAt returns the UTC retry deadline, or nil if not unavailable.
func (cm *Machine) RetryAt() *time.Time {
	return cm.retryAt
}

// Remaining returns the duration until the next retry, or 0 if not Unavailable.
func (cm *Machine) Remaining() time.Duration {
	if cm.retryAt == nil {
		return 0
	}

	d := time.Until(*cm.retryAt)
	if d < 0 {
		return 0
	}

	return d
}
