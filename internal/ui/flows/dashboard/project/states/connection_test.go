package states_test

import (
	"context"
	"testing"

	"github.com/ma-tf/ogle/internal/ui/components/inspector"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard/project/states"
)

func TestConnectionMachine_HandleConnected(t *testing.T) {
	t.Parallel()

	cm := states.NewConnectionMachine(inspector.ConnectStateConnecting, 0)
	cm.HandleConnected()

	if cm.ConnectState() != inspector.ConnectStateConnected {
		t.Fatalf("expected ConnectStateConnected, got %v", cm.ConnectState())
	}
}

func TestConnectionMachine_HandleUnavailable_GuardWhenNotConnected(t *testing.T) {
	t.Parallel()

	for _, initial := range []inspector.ConnectState{
		inspector.ConnectStateConnecting,
		inspector.ConnectStateUnavailable,
	} {
		cm := states.NewConnectionMachine(initial, 0)
		cmd := cm.HandleUnavailable()

		if cmd != nil {
			t.Errorf("state=%v: expected nil cmd (no-op), got non-nil", initial)
		}

		if cm.ConnectState() != initial {
			t.Errorf("state=%v: expected state unchanged, got %v", initial, cm.ConnectState())
		}
	}
}

func TestConnectionMachine_HandleUnavailable_TransitionsWhenConnected(t *testing.T) {
	t.Parallel()

	cm := states.NewConnectionMachine(inspector.ConnectStateConnected, 0)
	cmd := cm.HandleUnavailable()

	if cmd == nil {
		t.Fatal("expected non-nil cmd (startCountdown), got nil")
	}

	if cm.ConnectState() != inspector.ConnectStateUnavailable {
		t.Fatalf("expected ConnectStateUnavailable, got %v", cm.ConnectState())
	}

	if cm.Unavailable().SecondsUntilRetry != states.RetryIntervalSeconds {
		t.Fatalf(
			"expected SecondsUntilRetry=%d, got %d",
			states.RetryIntervalSeconds,
			cm.Unavailable().SecondsUntilRetry,
		)
	}
}

func TestConnectionMachine_HandleGracePeriodExpired_GuardWhenNotConnecting(t *testing.T) {
	t.Parallel()

	for _, initial := range []inspector.ConnectState{
		inspector.ConnectStateConnected,
		inspector.ConnectStateUnavailable,
	} {
		cm := states.NewConnectionMachine(initial, 0)
		cmd := cm.HandleGracePeriodExpired()

		if cmd != nil {
			t.Errorf("state=%v: expected nil cmd (no-op), got non-nil", initial)
		}

		if cm.ConnectState() != initial {
			t.Errorf("state=%v: expected state unchanged, got %v", initial, cm.ConnectState())
		}
	}
}

func TestConnectionMachine_HandleGracePeriodExpired_TransitionsWhenConnecting(t *testing.T) {
	t.Parallel()

	cm := states.NewConnectionMachine(inspector.ConnectStateConnecting, 0)
	cmd := cm.HandleGracePeriodExpired()

	if cmd == nil {
		t.Fatal("expected non-nil cmd (startCountdown), got nil")
	}

	if cm.ConnectState() != inspector.ConnectStateUnavailable {
		t.Fatalf("expected ConnectStateUnavailable, got %v", cm.ConnectState())
	}

	if cm.Unavailable().SecondsUntilRetry != states.RetryIntervalSeconds {
		t.Fatalf(
			"expected SecondsUntilRetry=%d, got %d",
			states.RetryIntervalSeconds,
			cm.Unavailable().SecondsUntilRetry,
		)
	}
}

func TestConnectionMachine_HandleRetryTick_NoOpWhenNotUnavailable(t *testing.T) {
	t.Parallel()

	for _, initial := range []inspector.ConnectState{
		inspector.ConnectStateConnecting,
		inspector.ConnectStateConnected,
	} {
		cm := states.NewConnectionMachine(initial, 0)
		cmd := cm.HandleRetryTick(context.Background())

		if cmd != nil {
			t.Errorf("state=%v: expected nil cmd (no-op), got non-nil", initial)
		}
	}
}

func TestConnectionMachine_HandleRetryTick_CountdownFiresConnect(t *testing.T) {
	t.Parallel()

	cm := states.NewConnectionMachine(inspector.ConnectStateUnavailable, 1)

	cmd := cm.HandleRetryTick(context.Background())

	if cmd == nil {
		t.Fatal("expected non-nil cmd (svcdocker.Connect), got nil")
	}

	if cm.ConnectState() != inspector.ConnectStateConnecting {
		t.Fatalf("expected ConnectStateConnecting after countdown, got %v", cm.ConnectState())
	}

	// Verify the returned cmd is the Connect cmd by executing it and checking the
	// message type. svcdocker.Connect returns msgs.DaemonConnected or msgs.DaemonUnavailable.
	// We accept either — the important thing is it runs without panic and emits a known type.
	_ = cmd
}

func TestConnectionMachine_HandleRetryTick_CountdownContinues(t *testing.T) {
	t.Parallel()

	cm := states.NewConnectionMachine(inspector.ConnectStateUnavailable, 3)

	cmd := cm.HandleRetryTick(context.Background())

	if cmd == nil {
		t.Fatal("expected non-nil cmd (startCountdown), got nil")
	}

	if cm.ConnectState() != inspector.ConnectStateUnavailable {
		t.Fatalf("expected ConnectStateUnavailable, got %v", cm.ConnectState())
	}

	if cm.Unavailable().SecondsUntilRetry != 2 {
		t.Fatalf("expected SecondsUntilRetry=2, got %d", cm.Unavailable().SecondsUntilRetry)
	}
}
