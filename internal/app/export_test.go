package app

import (
	"github.com/ma-tf/ogle/internal/ui/components/watching"
	"github.com/ma-tf/ogle/internal/ui/flows/dashboard"
)

const (
	PhaseStartup   = 0
	PhaseDashboard = 1
	PhaseWatching  = 2
)

func GetPhase(m *Model) int {
	return int(m.phase)
}

func GetDashboard(m *Model) dashboard.Model {
	return m.dashboard
}

func GetWatching(m *Model) watching.Model {
	return m.watching
}
