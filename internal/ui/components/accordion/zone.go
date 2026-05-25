package accordion

import (
	zone "github.com/lrstanley/bubblezone/v2"
)

// ZoneManager handles mouse interaction zones.
type ZoneManager interface {
	Mark(id, v string) string
	Get(id string) *zone.ZoneInfo
}

// Compile-time check that *zone.Manager satisfies ZoneManager.
var _ ZoneManager = (*zone.Manager)(nil)
