package inspector

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ma-tf/ogle/internal/domain"
)

const (
	oglePrefix   = "ogle."
	labelsHeight = 8 // fixed visible height for the label section
	keyValSep    = 2 // spaces between key and value columns
)

// labelsModel is the ogle.* label section sub-component. It is a value type.
type labelsModel struct {
	pairs    []labelPair // filtered ogle.* entries, sorted by key
	offset   int         // scroll offset (first visible index)
	focused  bool
	hover    int // index under mouse, -1 if none
	ctrlHeld bool
	dragIdx  int // index being dragged, -1 if none
	dragX    int
}

type labelPair struct {
	key   string
	value string
}

func newLabelsModel(svc domain.ServiceDef) labelsModel {
	var pairs []labelPair

	for k, v := range svc.Labels {
		if strings.HasPrefix(k, oglePrefix) {
			pairs = append(pairs, labelPair{key: k, value: v})
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].key < pairs[j].key
	})

	return labelsModel{
		pairs:    pairs,
		offset:   0,
		focused:  false,
		hover:    -1,
		dragIdx:  -1,
		ctrlHeld: false,
		dragX:    0,
	}
}

// update handles keyboard and mouse events for the label section.
func (m labelsModel) update(msg tea.Msg) (labelsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKeyPress(msg), nil
	case tea.MouseMotionMsg:
		return m.handleMouseMotion(msg), nil
	case tea.MouseClickMsg:
		return m.handleMouseClick(msg), nil
	case tea.MouseReleaseMsg:
		return m.handleMouseRelease(msg)
	}

	return m, nil
}

func (m labelsModel) handleKeyPress(msg tea.KeyPressMsg) labelsModel {
	switch msg.String() {
	case "up", "k":
		if m.offset > 0 {
			m.offset--
		}
	case "down", "j":
		if m.offset < len(m.pairs)-1 {
			m.offset++
		}
	case "escape":
		m.focused = false
	}

	return m
}

func (m labelsModel) handleMouseMotion(msg tea.MouseMotionMsg) labelsModel {
	m.ctrlHeld = msg.Mod.Contains(tea.ModCtrl)
	m.hover = m.mouseRow(msg.Y)

	return m
}

func (m labelsModel) handleMouseClick(msg tea.MouseClickMsg) labelsModel {
	if msg.Button == tea.MouseLeft {
		idx := m.mouseRow(msg.Y)
		if idx >= 0 {
			m.dragIdx = idx
			m.dragX = msg.X
		}
	}

	return m
}

func (m labelsModel) handleMouseRelease(msg tea.MouseReleaseMsg) (labelsModel, tea.Cmd) {
	if msg.Button != tea.MouseLeft || m.dragIdx < 0 {
		return m, nil
	}

	pressIdx := m.dragIdx
	releaseIdx := m.mouseRow(msg.Y)

	if pressIdx == releaseIdx && pressIdx >= 0 {
		pair := m.pairs[pressIdx]

		if m.ctrlHeld && isURL(pair.value) {
			return m, openURLCmd(pair.value)
		}
	}

	if pressIdx >= 0 && abs(msg.X-m.dragX) > 0 {
		return m, copyToClipboardCmd(m.pairs[pressIdx].value)
	}

	m.dragIdx = -1

	return m, nil
}

// view renders the label section at the given dimensions.
func (m labelsModel) view(width, height int) string {
	if len(m.pairs) == 0 {
		return fmt.Sprintf("%-*s", width, "(no ogle.* labels)")
	}

	visible := min(height, labelsHeight)

	var sb strings.Builder

	for i := range visible {
		idx := m.offset + i
		if idx >= len(m.pairs) {
			break
		}

		pair := m.pairs[idx]
		key := truncate(pair.key, width/halfWidth)
		val := pair.value

		if m.hover == i && m.ctrlHeld && isURL(val) {
			val = underline(val)
		}

		val = truncate(val, width-len(key)-keyValSep)
		line := fmt.Sprintf("%-*s  %s", width/halfWidth, key, val)
		line = truncate(line, width)
		sb.WriteString(line)

		if i < visible-1 {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

// mouseRow maps an absolute Y coordinate to a visible label index.
// Returns -1 when the coordinate is outside the label section.
func (m labelsModel) mouseRow(absY int) int {
	// Row mapping is approximate; the Dashboard must offset Y by the header height.
	if absY < 0 || absY >= labelsHeight {
		return -1
	}

	idx := m.offset + absY
	if idx >= len(m.pairs) {
		return -1
	}

	return idx
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func underline(s string) string {
	return "\x1b[4m" + s + "\x1b[0m"
}

func abs(x int) int {
	if x < 0 {
		return -x
	}

	return x
}

func openURLCmd(url string) tea.Cmd {
	return func() tea.Msg {
		_ = exec.CommandContext(context.Background(), "xdg-open", url).Start()

		return nil
	}
}

func copyToClipboardCmd(_ string) tea.Cmd {
	return func() tea.Msg {
		// copy to clipboard
		return nil
	}
}
