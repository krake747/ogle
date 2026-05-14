package inspector

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
)

// HeaderLines is the number of rendered lines the detail header occupies.
const HeaderLines = 6

// dash is the placeholder rendered for Docker fields when runtime data is absent.
const dash = "—"

const (
	shortIDLen    = 12 // Docker conventional short-hash length
	secsPerMinute = 60
	secsPerHour   = 3600
	halfWidth     = 2 // divisor: a column occupies half the available width
)

// renderHeader returns the detail header for the given service as a vertical
// list of label+value pairs. Compose File fields (name, image, ports) are
// always rendered. Docker fields (id, state, health) show dash when runtime
// is nil.
func renderHeader(svc domain.ServiceDef, rt *domain.ServiceRuntimeData, width int) string {
	// Label column width is fixed at 6 characters (length of "health")
	labelWidth := 6

	// Prepare field values
	name := svc.Name

	image := svc.Image
	if image == "" {
		image = dash
	}

	var containerID, state, health, age string
	if rt == nil {
		containerID = dash
		state = dash
		health = dash
		age = dash
	} else {
		containerID = shortID(rt.ContainerID)
		state = string(rt.State)
		health = string(rt.Health)
		age = formatAge(rt.StateAge)
	}

	// Format state with age: "running · 2h"
	stateWithAge := state + " · " + age

	// Format ports: space-separated normalized ports, or dash if none
	portsStr := dash
	if len(svc.Ports) > 0 {
		portsStr = strings.Join(svc.Ports, " ")
	}

	// Build the 6 rows
	rows := []struct {
		label string
		value string
	}{
		{"name", name},
		{"id", containerID},
		{"state", stateWithAge},
		{"health", health},
		{"image", image},
		{"ports", portsStr},
	}

	// Format each row with label and value
	lines := make([]string, 0, len(rows))

	for _, row := range rows {
		// Right-pad label to labelWidth
		paddedLabel := row.label
		if len(paddedLabel) < labelWidth {
			paddedLabel += strings.Repeat(" ", labelWidth-len(paddedLabel))
		}

		// Style the row to respect width constraint
		formattedRow := lipgloss.NewStyle().
			MaxWidth(width).
			Inline(true).
			Render(paddedLabel + "  " + row.value)

		lines = append(lines, formattedRow)
	}

	return strings.Join(lines, "\n")
}

// shortID returns the first 12 characters of a container ID, or the full
// string if shorter, matching Docker's conventional short-hash display.
func shortID(id string) string {
	if len(id) <= shortIDLen {
		return id
	}

	return id[:shortIDLen]
}

// formatAge formats a [time.Duration] as a short human-readable string.
func formatAge(d time.Duration) string {
	secs := max(int(d.Seconds()), 0)

	switch {
	case secs < secsPerMinute:
		return fmt.Sprintf("%ds", secs)
	case secs < secsPerHour:
		return fmt.Sprintf("%dm", secs/secsPerMinute)
	default:
		return fmt.Sprintf("%dh", secs/secsPerHour)
	}
}
