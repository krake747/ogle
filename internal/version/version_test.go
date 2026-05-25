package version_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ma-tf/ogle/internal/version"
)

func TestDefaults(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "dev", version.Version)
	assert.Equal(t, "none", version.Commit)
	assert.Equal(t, "unknown", version.Date)
}
