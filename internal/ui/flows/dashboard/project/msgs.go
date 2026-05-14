package project

// gracePeriodExpiredMsg is fired once after the initial 5-second grace window.
// If the daemon has not connected by then, the Dashboard transitions to the
// Docker Unavailable state.
type gracePeriodExpiredMsg struct{}

// retryTickMsg is fired every second during the Docker Unavailable countdown.
type retryTickMsg struct{}
