package app

import "github.com/ma-tf/ogle/internal/ui/flows/dashboard"

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

func GetWidth(m *Model) int { return m.width }

func GetHeight(m *Model) int { return m.height }
