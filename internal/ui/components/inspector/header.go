package inspector

import (
	"fmt"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
)

// HeaderLines is the number of rendered lines the detail header occupies.
const HeaderLines = 2

// dash is the placeholder rendered for Docker fields when runtime data is absent.
const dash = "—"

const (
	halfWidth     = 2  // divisor: a column occupies half the available width
	shortIDLen    = 12 // Docker conventional short-hash length
	secsPerMinute = 60
	secsPerHour   = 3600
)

// renderHeader returns the compact detail header for the given service.
// Compose File fields (name, image) are always rendered. Docker fields
// (container ID, state, health, age) show dash when runtime is nil.
func renderHeader(svc domain.ServiceDef, rt *domain.ServiceRuntimeData, width int) string {
	// Row 1: name (left) | image (right-aligned)
	image := svc.Image
	if image == "" {
		image = dash
	}

	leftW := width / halfWidth
	rightW := width - leftW
	left := lipgloss.NewStyle().Width(leftW).MaxWidth(leftW).Inline(true).Render(svc.Name)
	right := lipgloss.NewStyle().
		Width(rightW).
		MaxWidth(rightW).
		Inline(true).
		Align(lipgloss.Right).
		Render(image)
	row1 := left + right

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

	row2 := lipgloss.NewStyle().MaxWidth(width).Inline(true).Render(
		fmt.Sprintf("%s  %s  %s  %s", containerID, state, health, age),
	)

	return row1 + "\n" + row2
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
