package docker

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
)

// Stop returns a Cmd that runs `docker compose -f <file> -p <project> stop <service>`.
func Stop(ctx context.Context, file, projectName, serviceName string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(ctx, "docker", "compose", "-f", file, "-p", projectName, "stop", serviceName)

		cmd.Dir = filepath.Dir(file)
		if err := cmd.Run(); err != nil {
			return msgs.ServiceActionCompleted{
				ServiceName: serviceName,
				Action:      domain.ServiceActionStop,
				Err:         fmt.Errorf("docker compose stop: %w", err),
			}
		}

		return msgs.ServiceActionCompleted{
			ServiceName: serviceName,
			Action:      domain.ServiceActionStop,
			Err:         nil,
		}
	}
}

// Start returns a Cmd that runs `docker compose -f <file> -p <project> up -d <service>`.
// Handles both exited and not-created states.
func Start(ctx context.Context, file, projectName, serviceName string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(ctx, "docker", "compose", "-f", file, "-p", projectName, "up", "-d", serviceName)

		cmd.Dir = filepath.Dir(file)
		if err := cmd.Run(); err != nil {
			return msgs.ServiceActionCompleted{
				ServiceName: serviceName,
				Action:      domain.ServiceActionStart,
				Err:         fmt.Errorf("docker compose up -d: %w", err),
			}
		}

		return msgs.ServiceActionCompleted{
			ServiceName: serviceName,
			Action:      domain.ServiceActionStart,
			Err:         nil,
		}
	}
}

// Restart returns a Cmd that runs `docker compose -f <file> -p <project> restart <service>`.
func Restart(ctx context.Context, file, projectName, serviceName string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(
			ctx, "docker", "compose", "-f", file, "-p", projectName, "restart", serviceName,
		)

		cmd.Dir = filepath.Dir(file)
		if err := cmd.Run(); err != nil {
			return msgs.ServiceActionCompleted{
				ServiceName: serviceName,
				Action:      domain.ServiceActionRestart,
				Err:         fmt.Errorf("docker compose restart: %w", err),
			}
		}

		return msgs.ServiceActionCompleted{
			ServiceName: serviceName,
			Action:      domain.ServiceActionRestart,
			Err:         nil,
		}
	}
}

// Rebuild returns a Cmd that runs `docker compose -f <file> -p <project> up --build -d <service>`.
// Compose handles the stop/recreate lifecycle.
func Rebuild(ctx context.Context, file, projectName, serviceName string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(
			ctx, "docker", "compose", "-f", file, "-p", projectName, "up", "--build", "-d", serviceName,
		)

		cmd.Dir = filepath.Dir(file)
		if err := cmd.Run(); err != nil {
			return msgs.ServiceActionCompleted{
				ServiceName: serviceName,
				Action:      domain.ServiceActionRebuild,
				Err:         fmt.Errorf("docker compose up --build -d: %w", err),
			}
		}

		return msgs.ServiceActionCompleted{
			ServiceName: serviceName,
			Action:      domain.ServiceActionRebuild,
			Err:         nil,
		}
	}
}
