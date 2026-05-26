package app

import (
	"github.com/ma-tf/ogle/internal/ui/components/helpbar"
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

func GetShowingAbout(m *Model) bool {
	return m.showingAbout
}

func SetShowingAbout(m *Model, v bool) {
	m.showingAbout = v
}

func GetHelpbar(m *Model) helpbar.Model {
	return m.helpbar
}

func GetWidth(m *Model) int { return m.width }

func GetHeight(m *Model) int { return m.height }
