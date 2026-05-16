package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
)

// psLine maps a single JSON line from "docker compose ps --format json".
type psLine struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Service   string `json:"service"`
	State     string `json:"state"`
	Health    string `json:"health"`
	CreatedAt string `json:"createdat"`
}

// Ps returns a Cmd that runs docker compose ps and returns a ServicesPolled
// message with the parsed runtime data for every service.
func Ps(ctx context.Context, composeFile, projectName string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(
			ctx,
			"docker", "compose",
			"-f", composeFile,
			"-p", projectName,
			"ps", "--format", "json",
		)
		cmd.Dir = filepath.Dir(composeFile)

		out, err := cmd.Output()
		if err != nil {
			return msgs.ServicesPolled{
				Runtimes: nil,
				Err:      fmt.Errorf("docker compose ps: %w", err),
			}
		}

		runtimes, err := parsePsOutput(out)
		if err != nil {
			return msgs.ServicesPolled{
				Runtimes: nil,
				Err:      fmt.Errorf("parse compose ps output: %w", err),
			}
		}

		return msgs.ServicesPolled{Runtimes: runtimes, Err: nil}
	}
}

// parsePsOutput parses the JSON-lines output of "docker compose ps --format json"
// into a map keyed by service name.
func parsePsOutput(data []byte) (map[string]*domain.ServiceRuntimeData, error) {
	runtimes := make(map[string]*domain.ServiceRuntimeData)

	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	if len(lines) == 1 && len(lines[0]) == 0 {
		return runtimes, nil
	}

	for _, line := range lines {
		var entry psLine
		if err := json.Unmarshal(line, &entry); err != nil {
			return nil, fmt.Errorf("decode json line: %w", err)
		}

		name := strings.TrimSpace(entry.Service)
		if name == "" {
			continue
		}

		createdAt, _ := time.Parse("2006-01-02 15:04:05 -0700 MST", entry.CreatedAt)

		stateAge := time.Duration(0)
		if !createdAt.IsZero() {
			stateAge = time.Since(createdAt)
		}

		runtimes[name] = &domain.ServiceRuntimeData{
			ContainerID: entry.ID,
			State:       parseState(entry.State),
			Health:      parseHealth(entry.Health),
			StateAge:    stateAge,
		}
	}

	return runtimes, nil
}

func parseState(s string) domain.ServiceState {
	switch s {
	case "running":
		return domain.ServiceStateRunning
	case "exited":
		return domain.ServiceStateExited
	case "paused":
		return domain.ServiceStatePaused
	case "restarting":
		return domain.ServiceStateRestarting
	case "dead":
		return domain.ServiceStateDead
	default:
		return domain.ServiceStateUnknown
	}
}

func parseHealth(s string) domain.ServiceHealth {
	switch s {
	case "healthy":
		return domain.ServiceHealthHealthy
	case "unhealthy":
		return domain.ServiceHealthUnhealthy
	case "starting":
		return domain.ServiceHealthStarting
	case "", "none", "no healthcheck":
		return domain.ServiceHealthNoHealthcheck
	default:
		return domain.ServiceHealthUnknown
	}
}
