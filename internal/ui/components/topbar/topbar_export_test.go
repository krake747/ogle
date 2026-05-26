package topbar

import (
	"time"

	"github.com/ma-tf/ogle/internal/ui/theme"
)

func SetNow(m *Model, t time.Time) {
	m.now = func() time.Time { return t }
}

func GetTheme(m *Model) *theme.Theme {
	return m.th
}

func GetWidth(m *Model) int {
	return m.width
}
