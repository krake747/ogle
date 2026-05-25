package topbar

import "time"

func (m *Model) SetNow(t time.Time) {
	m.now = func() time.Time { return t }
}
