// Package watching provides the disconnected state shown when the project
// file disappears at runtime. On FileAvailabilityChanged it parses the
// original file and emits ProjectLoaded to transition back to dashboard.
package watching

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/services/parser"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// frameHeight is the number of terminal lines consumed by the app-level chrome
// (topbar + helpbar) that this phase must subtract from its allocated height.
const frameHeight = 3

//nolint:gochecknoglobals // package-level key binding
var keyQuit = key.NewBinding(key.WithKeys("q"))

type state int

const (
	stateIdle state = iota
	stateParseError
)

// Model is the disconnected watching state.
type Model struct {
	parser   parser.Parser
	File     string
	ctx      context.Context
	log      *slog.Logger
	th       *theme.Theme
	st       state
	parseErr error

	w, h int
}

// New returns a Model watching for file to reappear.
func New(
	ctx context.Context,
	logger *slog.Logger,
	file string,
	w, h int,
	th *theme.Theme,
) Model {
	return Model{
		parser:   parser.New(ctx, logger),
		File:     file,
		ctx:      ctx,
		log:      logger,
		th:       th,
		st:       stateIdle,
		parseErr: nil,
		w:        w,
		h:        h,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height - frameHeight

		return m, nil

	case tea.KeyPressMsg:
		if key.Matches(msg, keyQuit) {
			return m, tea.Quit
		}

		return m, nil

	case msgs.ThemeChanged:
		m.th = msg.Theme

		return m, nil

	case msgs.FileAvailabilityChanged:
		if slices.Contains(msg.Files, m.File) {
			p, err := m.parser.Parse(m.File)
			if err != nil {
				m.st = stateParseError
				m.parseErr = err

				return m, nil
			}

			return m, func() tea.Msg {
				return msgs.ProjectLoaded{Project: p}
			}
		}

		m.st = stateIdle
		m.parseErr = nil

		return m, nil
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	muted := lipgloss.NewStyle().Foreground(m.th.Subtext).Render
	errStyle := lipgloss.NewStyle().Foreground(m.th.ActionError).Render

	switch m.st {
	case stateParseError:
		body := muted("compose file unavailable — waiting...") +
			"\n\n" +
			errStyle(fmt.Sprintf("Parse error: %v", m.parseErr)) +
			"\n" +
			muted("Waiting for file to change...")

		return tea.NewView(body)

	case stateIdle:
		return tea.NewView(muted("compose file unavailable — waiting..."))
	}

	return tea.NewView("")
}
