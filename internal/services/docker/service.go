// Package docker provides the Docker daemon connectivity layer for ogle.
package docker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

// Docker interacts with the Docker daemon for connectivity, state polling,
// and service actions. All methods return tea.Cmd values that the Bubble Tea
// runtime executes asynchronously.
//
//mockery:generate: true
type Docker interface {
	Connect(ctx context.Context) tea.Cmd
	Ps(ctx context.Context, composeFile, projectName string) tea.Cmd
	Stop(ctx context.Context, composeFile, projectName, serviceName string) tea.Cmd
	Start(ctx context.Context, composeFile, projectName, serviceName string) tea.Cmd
	Restart(ctx context.Context, composeFile, projectName, serviceName string) tea.Cmd
	Rebuild(ctx context.Context, composeFile, projectName, serviceName string) tea.Cmd
}

// Service implements Docker using the Docker Unix socket and docker compose CLI.
type Service struct{}

// New returns a Service ready for use.
func New() *Service {
	return &Service{}
}

var _ Docker = (*Service)(nil)

var ErrUnexpectedPingStatus = errors.New("docker ping returned unexpected status")

const (
	socketPath     = "/var/run/docker.sock"
	pingPath       = "http://localhost/_ping"
	dialTimeout    = 2 * time.Second
	requestTimeout = 5 * time.Second
)

// Connect returns a Cmd that attempts to ping the Docker daemon by issuing an
// HTTP GET to /_ping over the Unix socket at /var/run/docker.sock. On success
// it returns msgs.DaemonConnected; on any failure it returns
// msgs.DaemonUnavailable with a wrapped error.
//
// The context passed to the returned Cmd controls the request lifetime.
// Connect itself does not start a long-running goroutine — callers are
// responsible for scheduling retries.
func (s *Service) Connect(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		transport := &http.Transport{
			DialContext: func(dialCtx context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{Timeout: dialTimeout}

				conn, err := d.DialContext(dialCtx, "unix", socketPath)
				if err != nil {
					return nil, fmt.Errorf("dial docker socket: %w", err)
				}

				return conn, nil
			},
		}

		client := &http.Client{
			Transport: transport,
			Timeout:   requestTimeout,
		}
		defer client.CloseIdleConnections()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingPath, nil)
		if err != nil {
			return msgs.DaemonUnavailable{Err: fmt.Errorf("build ping request: %w", err)}
		}

		resp, err := client.Do(req)
		if err != nil {
			return msgs.DaemonUnavailable{Err: fmt.Errorf("ping docker daemon: %w", err)}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return msgs.DaemonUnavailable{
				Err: fmt.Errorf("%w: %d", ErrUnexpectedPingStatus, resp.StatusCode),
			}
		}

		return msgs.DaemonConnected{}
	}
}
