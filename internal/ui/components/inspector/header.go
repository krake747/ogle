package inspector

import (
	"fmt"
	"strings"
	"time"

	"github.com/ma-tf/ogle/internal/domain"
)

// headerLines is the number of rendered lines the detail header occupies.
const headerLines = 2

// dash is the placeholder rendered for Docker fields when runtime data is absent.
const dash = "—"

const (
	halfWidth     = 2  // divisor: a column occupies half the available width
	row1ImagePad  = 3  // chars reserved for the gap between name and image in row 1
	shortIDLen    = 12 // Docker conventional short-hash length
	secsPerMinute = 60
	secsPerHour   = 3600
)

// renderHeader returns the compact detail header for the given service.
// Compose File fields (name, image) are always rendered. Docker fields
// (container ID, state, health, age) show dash when runtime is nil.
func renderHeader(svc domain.ServiceDef, rt *domain.ServiceRuntimeData, width int) string {
	// Row 1: name | image
	name := truncate(svc.Name, width/halfWidth)

	image := svc.Image
	if image == "" {
		image = dash
	}

	image = truncate(image, width-len(name)-row1ImagePad)
	row1 := padColumns(name, image, width)

	// Row 2: container hash | state | health | age
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

	row2 := truncate(
		fmt.Sprintf("%s  %s  %s  %s", containerID, state, health, age),
		width,
	)

	return row1 + "\n" + row2
}

// padColumns places left and right strings separated by spaces to fill width.
func padColumns(left, right string, width int) string {
	gap := max(width-len(left)-len(right), 1)

	return left + strings.Repeat(" ", gap) + right
}

// shortID returns the first 12 characters of a container ID, or the full
// string if shorter, matching Docker's conventional short-hash display.
func shortID(id string) string {
	if len(id) <= shortIDLen {
		return id
	}

	return id[:shortIDLen]
}

// truncate clips s to at most limit runes, appending "…" if clipped.
func truncate(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}

	if limit <= 1 {
		return "…"
	}

	return string(runes[:limit-1]) + "…"
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
