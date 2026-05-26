package topbar

import (
	"time"

	zone "github.com/lrstanley/bubblezone/v2"

	"github.com/ma-tf/ogle/internal/services/docker/connection"
)

func SetNow(m *Model, t time.Time) {
	m.now = func() time.Time { return t }
}

func SetPhase(m *Model, p Phase) {
	m.phase = p
}

func SetConnectState(m *Model, s connection.ConnectState) {
	connection.SetConnectState(m.conn, s)
}

func GetZM(m *Model) *zone.Manager {
	return m.zm
}
