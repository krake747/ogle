package card

import (
	"image/color"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

// colourForState maps a service state to its theme colour.
func colourForState(s domain.ServiceState, th *theme.Theme) color.Color {
	switch s {
	case domain.ServiceStateRunning:
		return th.StateRunning
	case domain.ServiceStateExited, domain.ServiceStateDead:
		return th.StateExited
	case domain.ServiceStatePaused:
		return th.StatePaused
	case domain.ServiceStateRestarting:
		return th.StateTransient
	case domain.ServiceStateNotCreated, domain.ServiceStateUnknown:
		return th.StateMuted
	default:
		return th.StateMuted
	}
}
