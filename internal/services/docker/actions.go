package docker

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
)

func runAction(
	cmd *exec.Cmd,
	action domain.ServiceAction,
	serviceName string,
) msgs.ServiceActionCompleted {
	var stderrBuf strings.Builder

	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		stderr := strings.TrimSpace(strings.ReplaceAll(stderrBuf.String(), "\n", " "))
		if stderr != "" {
			err = fmt.Errorf("%w: %s", err, stderr)
		}

		return msgs.ServiceActionCompleted{
			ServiceName: serviceName,
			Action:      action,
			Err:         err,
		}
	}

	return msgs.ServiceActionCompleted{
		ServiceName: serviceName,
		Action:      action,
		Err:         nil,
	}
}

// Stop returns a Cmd that runs `docker compose -f <file> -p <project> stop <service>`.
func (s *Service) Stop(ctx context.Context, file, projectName, serviceName string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(
			ctx, "docker", "compose", "-f", file, "-p", projectName, "stop", serviceName,
		)
		cmd.Dir = filepath.Dir(file)

		return runAction(cmd, domain.ServiceActionStop, serviceName)
	}
}

// Start returns a Cmd that runs `docker compose -f <file> -p <project> up -d <service>`.
// Handles both exited and not-created states.
func (s *Service) Start(ctx context.Context, file, projectName, serviceName string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(
			ctx, "docker", "compose", "-f", file, "-p", projectName, "up", "-d", serviceName,
		)
		cmd.Dir = filepath.Dir(file)

		return runAction(cmd, domain.ServiceActionStart, serviceName)
	}
}

// Restart returns a Cmd that runs `docker compose -f <file> -p <project> restart <service>`.
func (s *Service) Restart(ctx context.Context, file, projectName, serviceName string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(
			ctx, "docker", "compose", "-f", file, "-p", projectName, "restart", serviceName,
		)
		cmd.Dir = filepath.Dir(file)

		return runAction(cmd, domain.ServiceActionRestart, serviceName)
	}
}

// Rebuild returns a Cmd that runs `docker compose -f <file> -p <project> up --build -d <service>`.
// Compose handles the stop/recreate lifecycle.
func (s *Service) Rebuild(ctx context.Context, file, projectName, serviceName string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(
			ctx,
			"docker", "compose",
			"-f", file,
			"-p", projectName,
			"up", "--build", "-d",
			serviceName,
		)
		cmd.Dir = filepath.Dir(file)

		return runAction(cmd, domain.ServiceActionRebuild, serviceName)
	}
}
