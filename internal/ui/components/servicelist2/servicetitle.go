package servicelist2

import (
	"charm.land/lipgloss/v2"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// Build computes the display title for a service list item.
func Build(
	name string,
	rt *domain.ServiceRuntimeData,
	inFlight bool,
	actionLabel, actionError string,
	th *theme.Theme,
) string {
	icon := "●"
	colour := th.StateMuted

	switch {
	case inFlight:
		icon = "◌"
		colour = th.StateTransient
	case rt == nil:

	default:
		switch rt.State {
		case domain.ServiceStateRunning:
			icon = "●"
			colour = th.StateRunning
		case domain.ServiceStateExited, domain.ServiceStateDead:
			icon = "●"
			colour = th.StateExited
		case domain.ServiceStateNotCreated:
			icon = "○"
		case domain.ServiceStatePaused:
			icon = "●"
			colour = th.StatePaused
		case domain.ServiceStateRestarting:
			icon = "●"
			colour = th.StateTransient
		case domain.ServiceStateUnknown:
			icon = "●"
		}
	}

	rendered := lipgloss.NewStyle().Foreground(colour).Render(icon) + " " + name

	if inFlight && actionLabel != "" {
		rendered += "  " + actionLabel
	}

	if !inFlight && actionError != "" {
		rendered += "  " + lipgloss.NewStyle().Foreground(th.ActionError).Render(actionError)
	}

	return rendered
}
