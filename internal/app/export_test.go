package app

import (
	"github.com/ma-tf/ogle/internal/ui/components/helpbar"
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

func GetShowingAbout(m *Model) bool {
	return m.showingAbout
}

func SetShowingAbout(m *Model, v bool) {
	m.showingAbout = v
}

func GetHelpbar(m *Model) helpbar.Model {
	return m.helpbar
}
