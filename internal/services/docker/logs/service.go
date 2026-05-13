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
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/msgs"
)

const (
	socketPath     = "/var/run/docker.sock"
	channelCap     = 64
	tailLines      = "1000"
	errorBodyLimit = 512
	// stdoutStream and stderrStream are the Docker multiplexed stream type
	// bytes (header[0]) as defined by the Docker logs API.
	stdoutStream = 1
	stderrStream = 2
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

// LogStreamer streams Docker container logs over the Unix socket. It exposes a
// Next/Close contract that mirrors the watcher pattern: the caller re-subscribes
// after each received message.
type LogStreamer struct {
	cancel context.CancelFunc
	ch     chan tea.Msg
	done   chan struct{}
	wg     sync.WaitGroup
}

// New returns an idle LogStreamer. Call Start before calling Next.
func New() *LogStreamer {
	return &LogStreamer{
		cancel: nil,
		ch:     make(chan tea.Msg, channelCap),
		done:   make(chan struct{}),
		wg:     sync.WaitGroup{},
	}
}

// Start begins streaming logs for the named container. Close must be called
// before calling Start again on a reused LogStreamer. It issues the HTTP
// request synchronously; on 404 or non-200 it writes a single error message
// to the channel and returns without spawning a goroutine. On 200 a reader
// goroutine is started; it writes msgs.LogLine values until the stream ends
// or Close is called.
func (s *LogStreamer) Start(appCtx context.Context, containerName string) {
	ctx, cancel := context.WithCancel(appCtx)
	s.cancel = cancel

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
		s.ch <- msgs.LogStreamError{Err: err}

		return
	}

	resp, err := client.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
		default:
			s.ch <- msgs.LogStreamError{Err: err}
		}

		return
	}

	if resp.StatusCode == http.StatusNotFound {
		_ = resp.Body.Close()

		client.CloseIdleConnections()

		s.ch <- msgs.LogStreamContainerNotFound{}

		return
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
		_ = resp.Body.Close()

		client.CloseIdleConnections()

		s.ch <- msgs.LogStreamError{Err: fmt.Errorf("%w %d: %s", errUnexpectedStatus, resp.StatusCode, body)}

		return
	}

	s.wg.Go(func() { s.readFrames(ctx, resp.Body, client) })
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
			default:
				s.ch <- msgs.LogStreamError{Err: readErr}
			}

			return
		}

		size := binary.BigEndian.Uint32(header[4:])
		payload := make([]byte, size)

		if _, readErr := io.ReadFull(body, payload); readErr != nil {
			select {
			case <-ctx.Done():
			default:
				s.ch <- msgs.LogStreamError{Err: readErr}
			}

			return
		}

		select {
		case s.ch <- msgs.LogLine{Text: string(payload), IsStderr: header[0] == stderrStream}:
		case <-ctx.Done():
			return
		}
	}
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
// buffered messages. Close is a no-op when no stream is active.
func (s *LogStreamer) Close() {
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil

		s.wg.Wait()

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
