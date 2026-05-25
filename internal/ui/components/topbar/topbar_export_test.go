package topbar

import "time"

func SetNow(m *Model, t time.Time) {
	m.now = func() time.Time { return t }
}
