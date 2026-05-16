// Package logs implements the LogStreamer service, which streams Docker
// container log output over the Docker Unix socket using raw net/http.
package logs

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

const (
	socketPath     = "/var/run/docker.sock"
	channelCap     = 64
	lineBufferCap  = 5000
	tailLines      = "1000"
	errorBodyLimit = 512
)

var errUnexpectedStatus = errors.New("logs: unexpected status")

// ContainerName resolves the Docker container name for a service.
// If containerNameOverride is non-empty it is returned verbatim; otherwise the
// Compose v2 convention "<project>-<service>-1" is used.
//
// Compose v1 used underscores (project_service_1); v2 uses dashes. Only v2 is supported.
func ContainerName(project, service, containerNameOverride string) string {
	if containerNameOverride != "" {
		return containerNameOverride
	}

	return project + "-" + service + "-1"
}

// LogStreamer streams Docker container logs over the Unix socket. Normal log
// lines flow through lineCh; errors flow through ch.
type LogStreamer struct {
	cancel      context.CancelFunc
	ch          chan tea.Msg
	lineCh      chan string
	done        chan struct{}
	wg          sync.WaitGroup
	serviceName string
}

// New returns an idle LogStreamer. Call Start before calling Next.
func New(serviceName string) *LogStreamer {
	return &LogStreamer{
		cancel:      nil,
		ch:          make(chan tea.Msg, channelCap),
		lineCh:      make(chan string, lineBufferCap),
		done:        make(chan struct{}),
		wg:          sync.WaitGroup{},
		serviceName: serviceName,
	}
}

// Start begins streaming logs for the named container. Close must be called
// before calling Start again on a reused LogStreamer. It returns immediately;
// the HTTP connection and all I/O run in a single background goroutine. On
// 404 or non-200 the goroutine writes a single error message to ch and exits.
// On 200 it writes log line strings to lineCh until the stream ends or Close
// is called.
func (s *LogStreamer) Start(appCtx context.Context, containerName string) {
	ctx, cancel := context.WithCancel(appCtx)
	s.cancel = cancel

	s.wg.Go(func() {
		transport := &http.Transport{
			DialContext: func(dialCtx context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{}

				return d.DialContext(dialCtx, "unix", socketPath)
			},
		}
		client := &http.Client{Transport: transport}

		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			fmt.Sprintf(
				"http://localhost/containers/%s/logs?follow=true&stdout=1&stderr=1&tail=%s",
				url.PathEscape(containerName), tailLines,
			),
			nil,
		)
		if err != nil {
			select {
			case s.ch <- msgs.LogStreamError{Err: err, ServiceName: s.serviceName}:
			case <-ctx.Done():
			}

			return
		}

		resp, err := client.Do(req)
		if err != nil {
			select {
			case s.ch <- msgs.LogStreamError{Err: err, ServiceName: s.serviceName}:
			case <-ctx.Done():
			}

			return
		}

		if resp.StatusCode == http.StatusNotFound {
			_ = resp.Body.Close()

			client.CloseIdleConnections()

			select {
			case s.ch <- msgs.LogStreamContainerNotFound{ServiceName: s.serviceName}:
			case <-ctx.Done():
			}

			return
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
			_ = resp.Body.Close()

			client.CloseIdleConnections()

			select {
			case s.ch <- msgs.LogStreamError{
				Err:         fmt.Errorf("%w %d: %s", errUnexpectedStatus, resp.StatusCode, body),
				ServiceName: s.serviceName,
			}:
			case <-ctx.Done():
			}

			return
		}

		s.readFrames(ctx, resp.Body, client)
	})
}

// readFrames reads the Docker multiplexed log stream until the context is
// cancelled or the connection closes. It owns body and client: both are closed
// before it returns.
func (s *LogStreamer) readFrames(ctx context.Context, body io.ReadCloser, client *http.Client) {
	defer body.Close()
	defer client.CloseIdleConnections()

	for {
		var header [8]byte

		if _, readErr := io.ReadFull(body, header[:]); readErr != nil {
			select {
			case <-ctx.Done():
				// normal shutdown — do not emit an error
			case s.ch <- msgs.LogStreamError{Err: readErr, ServiceName: s.serviceName}:
			}

			return
		}

		size := binary.BigEndian.Uint32(header[4:])
		payload := make([]byte, size)

		if _, readErr := io.ReadFull(body, payload); readErr != nil {
			select {
			case <-ctx.Done():
			case s.ch <- msgs.LogStreamError{Err: readErr, ServiceName: s.serviceName}:
			}

			return
		}

		select {
		case s.lineCh <- strings.TrimRight(string(payload), "\n"):
			select {
			case s.ch <- msgs.LogLinesAvailable{}:
			default:
			}
		case <-ctx.Done():
			return
		}
	}
}

// Lines returns a read-only channel that delivers log line strings. The
// channel is closed in Close.
func (s *LogStreamer) Lines() <-chan string {
	return s.lineCh
}

// Next returns a tea.Cmd that blocks until the next message arrives on the
// internal channel. The caller must call Next again after each received message
// to re-subscribe — there is no automatic re-subscription. If the streamer is
// closed while Next is in-flight, the cmd returns nil and the goroutine exits.
func (s *LogStreamer) Next() tea.Cmd {
	done := s.done

	return func() tea.Msg {
		select {
		case msg := <-s.ch:
			return msg
		case <-done:
			return nil
		}
	}
}

// Close cancels the streaming context, waits for the reader goroutine to exit,
// rotates the done channel to unblock any in-flight Next cmd, then drains any
// buffered error messages. Close is a no-op when no stream is active.
func (s *LogStreamer) Close() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil

		s.wg.Wait()

		close(s.lineCh)

		old := s.done
		s.done = make(chan struct{})

		close(old)

		for {
			select {
			case <-s.ch:
			default:
				return
			}
		}
	}
}
