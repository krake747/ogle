package states

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	fieldTheme = iota
	fieldPollInterval
	fieldLogBufferCap
	fieldCount
)

const (
	pollMin    = 1 * time.Second
	pollMax    = 60 * time.Second
	pollStep   = 1 * time.Second
	pollFast   = 5 * time.Second
	bufCapMin  = 100
	bufCapMax  = 10_000
	bufCapStep = 100
	bufCapFast = 1_000
	fieldWidth = 16
)

type settingsKeyMap struct {
	Next    key.Binding
	Prev    key.Binding
	Inc     key.Binding
	Dec     key.Binding
	FastInc key.Binding
	FastDec key.Binding
	Confirm key.Binding
	Cancel  key.Binding
}

func (k settingsKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Confirm, k.Cancel, k.Next, k.Inc, k.FastInc}
}

func (k settingsKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Next, k.Prev, k.Inc, k.Dec, k.FastInc, k.FastDec, k.Confirm, k.Cancel},
	}
}

//nolint:gochecknoglobals // key bindings are package-level and immutable
var defaultSettingsKeys = settingsKeyMap{
	Next:    key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next")),
	Prev:    key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev")),
	Inc:     key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "inc")),
	Dec:     key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "dec")),
	FastInc: key.NewBinding(key.WithKeys("shift+up"), key.WithHelp("shift+↑", "fast inc")),
	FastDec: key.NewBinding(key.WithKeys("shift+down"), key.WithHelp("shift+↓", "fast dec")),
	Confirm: key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
	Cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
}

// Settings is the in-session settings overlay. It is managed as a compositor
// layer inside Dashboard rather than as a top-level State; Dashboard stays live
// underneath while the overlay is visible.
type Settings struct {
	zm *zone.Manager

	origThemeName string
	origTheme     *theme.Theme
	origPoll      time.Duration
	origCap       int

	themeIdx     int
	themeNames   []string
	themes       []*theme.Theme
	pollInterval time.Duration
	logBufferCap int

	liveTheme *theme.Theme

	focusField int
	keys       settingsKeyMap
	help       help.Model
	w, h       int
}

// NewSettings constructs a Settings overlay from the current Dashboard values.
// Themes are preloaded at construction so cycling never performs I/O during Update.
func NewSettings(
	themeName string,
	poll time.Duration,
	logBufCap int,
	th *theme.Theme,
	zm *zone.Manager,
) *Settings {
	names := theme.BuiltinNames()
	themes := make([]*theme.Theme, len(names))
	idx := 0

	for i, n := range names {
		// Load never returns nil: unknown names fall back to Default().
		// Empty configDir skips the user-file lookup; builtins always resolve.
		themes[i], _ = theme.Load(n, "")
		if n == themeName {
			idx = i
		}
	}

	return &Settings{
		zm:            zm,
		origThemeName: themeName,
		origTheme:     th,
		origPoll:      poll,
		origCap:       logBufCap,
		themeIdx:      idx,
		themeNames:    names,
		themes:        themes,
		pollInterval:  poll,
		logBufferCap:  logBufCap,
		liveTheme:     themes[idx],
		focusField:    fieldTheme,
		keys:          defaultSettingsKeys,
		help:          help.New(),
		w:             0,
		h:             0,
	}
}

// SetSize stores the terminal dimensions for View.
func (s *Settings) SetSize(w, h int) {
	s.w = w
	s.h = h
	s.help.SetWidth(w)
}

// Update handles keyboard navigation and field adjustments. Returns nil when
// the overlay should be closed (Confirm or Cancel).
func (s *Settings) Update(msg tea.Msg) (*Settings, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return s, nil
	}

	switch {
	case key.Matches(keyMsg, s.keys.Confirm):
		settingsAppliedCmd := func() tea.Msg {
			return msgs.SettingsApplied{
				Theme:        s.themeNames[s.themeIdx],
				PollInterval: s.pollInterval,
				LogBufferCap: s.logBufferCap,
			}
		}

		return nil, settingsAppliedCmd

	case key.Matches(keyMsg, s.keys.Cancel):
		return nil, nil

	case key.Matches(keyMsg, s.keys.Next):
		s.focusField = (s.focusField + 1) % fieldCount

	case key.Matches(keyMsg, s.keys.Prev):
		s.focusField = (s.focusField + fieldCount - 1) % fieldCount

	case key.Matches(keyMsg, s.keys.Inc):
		s.adjustField(1, false)

	case key.Matches(keyMsg, s.keys.Dec):
		s.adjustField(-1, false)

	case key.Matches(keyMsg, s.keys.FastInc):
		s.adjustField(1, true)

	case key.Matches(keyMsg, s.keys.FastDec):
		s.adjustField(-1, true)
	}

	return s, nil
}

func (s *Settings) adjustField(dir int, fast bool) {
	switch s.focusField {
	case fieldTheme:
		s.themeIdx = (s.themeIdx + dir + len(s.themeNames)) % len(s.themeNames)
		s.liveTheme = s.themes[s.themeIdx]

	case fieldPollInterval:
		step := pollStep
		if fast {
			step = pollFast
		}

		if dir > 0 {
			s.pollInterval = min(s.pollInterval+step, pollMax)
		} else {
			s.pollInterval = max(s.pollInterval-step, pollMin)
		}

	case fieldLogBufferCap:
		step := bufCapStep
		if fast {
			step = bufCapFast
		}

		if dir > 0 {
			s.logBufferCap = min(s.logBufferCap+step, bufCapMax)
		} else {
			s.logBufferCap = max(s.logBufferCap-step, bufCapMin)
		}
	}
}

// View renders the centred settings form styled with the live theme.
func (s *Settings) View() string {
	if s.w == 0 || s.h == 0 {
		return ""
	}

	focusStyle := lipgloss.NewStyle().
		Foreground(s.liveTheme.BorderFocused.GetBorderTopForeground())

	labelW := 15
	rows := []string{
		"Settings",
		"",
		s.renderRow(fieldTheme, "Theme", s.themeNames[s.themeIdx], labelW, focusStyle),
		s.renderRow(
			fieldPollInterval,
			"Poll Interval",
			fmt.Sprintf("%ds", int(s.pollInterval.Seconds())),
			labelW,
			focusStyle,
		),
		s.renderRow(
			fieldLogBufferCap,
			"Log Buffer Cap",
			strconv.Itoa(s.logBufferCap),
			labelW,
			focusStyle,
		),
		"",
		s.help.View(s.keys),
	}

	form := strings.Join(rows, "\n")

	return lipgloss.NewStyle().
		Width(s.w).
		Height(s.h).
		Align(lipgloss.Center, lipgloss.Center).
		Render(form)
}

func (s *Settings) renderRow(
	field int,
	label, value string,
	labelW int,
	focusStyle lipgloss.Style,
) string {
	paddedLabel := lipgloss.NewStyle().Width(labelW).Render(label)
	valueBlock := lipgloss.NewStyle().
		Width(fieldWidth).
		MaxWidth(fieldWidth).
		Inline(true).
		Render(value)
	cell := "[" + valueBlock + "]"

	if s.focusField == field {
		cell = focusStyle.Render(cell)
	}

	return paddedLabel + cell
}
