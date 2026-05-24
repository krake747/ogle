package statusbar_test

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/statusbar"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

func TestView(t *testing.T) {
	t.Parallel()

	const (
		errMsg  = "error text"
		infoMsg = "info"
	)

	tests := []struct {
		name           string
		setup          func(m statusbar.Model) statusbar.Model
		expectedResult string
	}{
		{
			name:           "initial state returns empty view",
			expectedResult: "",
		},
		{
			name: "display status shows message",
			setup: func(m statusbar.Model) statusbar.Model {
				m, _ = m.Update(msgs.DisplayStatus{Msg: "operation complete"})

				return m
			},
			expectedResult: "operation complete",
		},
		{
			name: "display error shows message",
			setup: func(m statusbar.Model) statusbar.Model {
				m, _ = m.Update(msgs.DisplayError{Err: "something went wrong"})

				return m
			},
			expectedResult: "something went wrong",
		},
		{
			name: "clear before expiry keeps message",
			setup: func(m statusbar.Model) statusbar.Model {
				m, _ = m.Update(msgs.DisplayStatus{Msg: "persist"})
				m, _ = m.Update(msgs.ClearStatusMsg{})

				return m
			},
			expectedResult: "persist",
		},
		{
			name:           "clear when already empty stays empty",
			expectedResult: "",
		},
		{
			name: "overwrite status with error",
			setup: func(m statusbar.Model) statusbar.Model {
				m, _ = m.Update(msgs.DisplayStatus{Msg: infoMsg})
				m, _ = m.Update(msgs.DisplayError{Err: errMsg})

				return m
			},
			expectedResult: errMsg,
		},
		{
			name: "overwrite error with status",
			setup: func(m statusbar.Model) statusbar.Model {
				m, _ = m.Update(msgs.DisplayError{Err: errMsg})
				m, _ = m.Update(msgs.DisplayStatus{Msg: infoMsg})

				return m
			},
			expectedResult: infoMsg,
		},
		{
			name: "width does not hide message",
			setup: func(m statusbar.Model) statusbar.Model {
				m, _ = m.Update(tea.WindowSizeMsg{Width: 80})
				m, _ = m.Update(msgs.DisplayStatus{Msg: "visible"})

				return m
			},
			expectedResult: "visible",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			sb := statusbar.New(theme.Default())
			_ = sb.Init()

			if tt.setup != nil {
				sb = tt.setup(sb)
			}

			if tt.expectedResult == "" {
				assert.Empty(t, sb.View().Content)
			} else {
				assert.Contains(t, sb.View().Content, tt.expectedResult)
			}
		})
	}
}
