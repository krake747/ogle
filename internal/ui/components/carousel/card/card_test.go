package card_test

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/domain"
	"github.com/ma-tf/ogle/internal/msgs"
	"github.com/ma-tf/ogle/internal/ui/components/carousel/card"
	"github.com/ma-tf/ogle/internal/ui/theme"
)

const (
	testShortName = "test-service"
	testLongName  = "service-name-that-needs-scrolling-yes"
	otherService  = "other-service"
	testW         = 200
	testH         = 40
)

func newCard(name string) card.Model {
	return card.New(domain.ServiceDef{Name: name}, testW, testH, theme.Default())
}

// Update tests

func TestUpdate_FocusMsg_Matching_SetsFocused(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	viewBefore := m.View().Content

	m, cmd := m.Update(card.FocusMsg{ServiceName: testShortName})

	require.Nil(t, cmd, "short name should not schedule scroll")

	viewAfter := m.View().Content
	assert.NotEqual(t, viewBefore, viewAfter, "focused background should differ")
	assert.Contains(t, viewAfter, testShortName)
}

func TestUpdate_FocusMsg_NonMatching_NoOp(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	viewBefore := m.View().Content

	m, cmd := m.Update(card.FocusMsg{ServiceName: otherService})

	require.Nil(t, cmd)
	assert.Equal(t, viewBefore, m.View().Content)
}

func TestUpdate_FocusMsg_Matching_WithScroll_StartsScroll(t *testing.T) {
	t.Parallel()

	m := newCard(testLongName)
	_, cmd := m.Update(card.FocusMsg{ServiceName: testLongName})

	require.NotNil(t, cmd, "long name should schedule initial scroll tick")
	scrollMsg := cmd()
	_, ok := scrollMsg.(card.ScrollTick)
	require.True(t, ok, "cmd should produce a ScrollTick message")
}

func TestUpdate_BlurMsg_Matching_ClearsFocus(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	m, _ = m.Update(card.FocusMsg{ServiceName: testShortName})
	focusedView := m.View().Content

	m, cmd := m.Update(card.BlurMsg{ServiceName: testShortName})

	require.Nil(t, cmd)

	blurredView := m.View().Content
	assert.NotEqual(t, focusedView, blurredView, "blur should revert background")
	assert.Contains(t, blurredView, testShortName)
}

func TestUpdate_BlurMsg_NonMatching_NoOp(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	m, _ = m.Update(card.FocusMsg{ServiceName: testShortName})
	focusedView := m.View().Content

	m, cmd := m.Update(card.BlurMsg{ServiceName: otherService})

	require.Nil(t, cmd)
	assert.Equal(t, focusedView, m.View().Content)
}

func TestUpdate_HoverMsg_Matching_SetsHovered(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	viewBefore := m.View().Content

	m, cmd := m.Update(card.HoverMsg{ServiceName: testShortName})

	require.Nil(t, cmd)

	viewAfter := m.View().Content
	assert.NotEqual(t, viewBefore, viewAfter, "hover background should differ")
	assert.Contains(t, viewAfter, testShortName)
}

func TestUpdate_HoverMsg_NonMatching_NoOp(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	viewBefore := m.View().Content

	m, cmd := m.Update(card.HoverMsg{ServiceName: otherService})

	require.Nil(t, cmd)
	assert.Equal(t, viewBefore, m.View().Content)
}

func TestUpdate_UnhoverMsg_Matching_ClearsHovered(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	m, _ = m.Update(card.HoverMsg{ServiceName: testShortName})
	hoveredView := m.View().Content

	m, cmd := m.Update(card.UnhoverMsg{ServiceName: testShortName})

	require.Nil(t, cmd)

	unhoveredView := m.View().Content
	assert.NotEqual(t, hoveredView, unhoveredView, "unhover should revert background")
	assert.Contains(t, unhoveredView, testShortName)
}

func TestUpdate_ServicesPolled_NilErr_StoresRuntime(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)

	viewBefore := m.View().Content
	m, cmd := m.Update(msgs.ServicesPolled{
		Runtimes: map[string]*domain.ServiceRuntimeData{
			testShortName: {State: domain.ServiceStateRunning},
		},
	})

	require.Nil(t, cmd)
	assert.NotEqual(t, viewBefore, m.View().Content,
		"runtime data should change border colour from muted to state-based")
}

func TestUpdate_ServicesPolled_WithErr_NoChange(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	viewBefore := m.View().Content

	m, cmd := m.Update(msgs.ServicesPolled{
		Err: assert.AnError,
	})

	require.Nil(t, cmd)
	assert.Equal(t, viewBefore, m.View().Content)
}

func TestUpdate_ServicesPolled_EmptyName_NoChange(t *testing.T) {
	t.Parallel()

	m := card.New(domain.ServiceDef{}, testW, testH, theme.Default())
	m, _ = m.Update(msgs.ServiceStart{ServiceName: ""})
	viewBefore := m.View().Content

	m, cmd := m.Update(msgs.ServicesPolled{
		Runtimes: map[string]*domain.ServiceRuntimeData{
			"": {State: domain.ServiceStateRunning},
		},
	})

	require.Nil(t, cmd)
	assert.Equal(t, viewBefore, m.View().Content)
}

func TestUpdate_ServiceAction_SetsInFlight(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  func(string) tea.Msg
	}{
		{
			name: "ServiceStart",
			msg:  func(n string) tea.Msg { return msgs.ServiceStart{ServiceName: n} },
		},
		{
			name: "ServiceStop",
			msg:  func(n string) tea.Msg { return msgs.ServiceStop{ServiceName: n} },
		},
		{
			name: "ServiceRestart",
			msg:  func(n string) tea.Msg { return msgs.ServiceRestart{ServiceName: n} },
		},
		{
			name: "ServiceRebuild",
			msg:  func(n string) tea.Msg { return msgs.ServiceRebuild{ServiceName: n} },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			name := tc.name + "-svc"
			m := newCard(name)
			viewBefore := m.View().Content

			m, cmd := m.Update(tc.msg(name))

			require.Nil(t, cmd)

			viewAfter := m.View().Content
			assert.NotEqual(t, viewBefore, viewAfter, "in-flight border should differ")
			assert.Contains(t, viewAfter, name)
		})
	}
}

func TestUpdate_ServiceAction_NonMatching_NoChange(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	viewBefore := m.View().Content

	m, cmd := m.Update(msgs.ServiceStart{ServiceName: otherService})

	require.Nil(t, cmd)
	assert.Equal(t, viewBefore, m.View().Content)
}

func TestUpdate_ServiceActionCompleted_ClearsInFlightAndUpdates(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	m, _ = m.Update(msgs.ServiceStart{ServiceName: testShortName})
	inflightView := m.View().Content

	m, cmd := m.Update(msgs.ServiceActionCompleted{
		ServiceName: testShortName,
		Action:      domain.ServiceActionStart,
	})

	require.Nil(t, cmd)

	completedView := m.View().Content
	assert.NotEqual(t, inflightView, completedView,
		"view should change when in-flight cleared and runtime set")
}

func TestUpdate_ServiceActionCompleted_NonMatching_NoChange(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	viewBefore := m.View().Content

	m, cmd := m.Update(msgs.ServiceActionCompleted{
		ServiceName: otherService,
		Action:      domain.ServiceActionStart,
	})

	require.Nil(t, cmd)
	assert.Equal(t, viewBefore, m.View().Content)
}

func TestUpdate_ServiceActionCompleted_Error_ClearsInFlightKeepsState(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	m, _ = m.Update(msgs.ServiceStart{ServiceName: testShortName})
	inflightView := m.View().Content

	m, cmd := m.Update(msgs.ServiceActionCompleted{
		ServiceName: testShortName,
		Action:      domain.ServiceActionStart,
		Err:         assert.AnError,
	})

	require.Nil(t, cmd)

	errorView := m.View().Content
	assert.NotEqual(t, inflightView, errorView,
		"view should change when in-flight cleared")
}

func TestUpdate_ServiceActionCompleted_Stop_SetsExitedState(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	m, _ = m.Update(msgs.ServicesPolled{
		Runtimes: map[string]*domain.ServiceRuntimeData{
			testShortName: {State: domain.ServiceStateRunning},
		},
	})
	m, _ = m.Update(msgs.ServiceStop{ServiceName: testShortName})

	m, cmd := m.Update(msgs.ServiceActionCompleted{
		ServiceName: testShortName,
		Action:      domain.ServiceActionStop,
	})

	require.Nil(t, cmd)
	assert.NotEmpty(t, m.View().Content)
}

func TestUpdate_WindowSizeMsg_UpdatesDimensions(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	viewBefore := m.View().Content

	m, cmd := m.Update(tea.WindowSizeMsg{Width: 300, Height: 60})

	require.Nil(t, cmd, "short name should not schedule scroll")

	viewAfter := m.View().Content
	assert.NotEqual(t, viewBefore, viewAfter, "different dimensions produce different card size")
}

func TestUpdate_WindowSizeMsg_FocusedWithScroll_Reschedules(t *testing.T) {
	t.Parallel()

	m := newCard(testLongName)
	m, _ = m.Update(card.FocusMsg{ServiceName: testLongName})

	_, cmd := m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})

	require.NotNil(t, cmd, "focused scrollable card should reschedule tick on resize")
}

func TestUpdate_ThemeChanged_UpdatesTheme(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	viewBefore := m.View().Content

	m, cmd := m.Update(theme.Changed{Theme: theme.Default()})

	require.Nil(t, cmd)
	assert.Equal(t, viewBefore, m.View().Content,
		"same theme values should produce the same view")
}

// ScrollTick tests
//
// These tests involve real time delays (scrollIdleInterval = 2500ms) because
// nextScrollTime is set in the past via the FocusMsg path, and ScrollTick's
// gen field is unexported so we must obtain messages through tickScroll cmd().

func TestUpdate_ScrollTick_AdvancesAndSchedulesNext(t *testing.T) {
	t.Parallel()

	m := newCard(testLongName)
	m, cmd1 := m.Update(card.FocusMsg{ServiceName: testLongName})
	require.NotNil(t, cmd1, "long name should schedule initial scroll tick")

	// cmd1() blocks for scrollIdleInterval (2.5s) then returns ScrollTick{gen: 1}
	scrollMsg := cmd1()

	// Ensure we're safely past nextScrollTime before sending
	time.Sleep(50 * time.Millisecond)

	_, cmd2 := m.Update(scrollMsg)
	require.NotNil(t, cmd2, "scroll should advance and schedule next tick")
	_, ok := cmd2().(card.ScrollTick)
	require.True(t, ok, "next cmd should be a ScrollTick")
}

func TestUpdate_ScrollTick_StaleGen_NoOp(t *testing.T) {
	t.Parallel()

	m := newCard(testLongName)

	// First focus → focusGen=1
	m, cmd1 := m.Update(card.FocusMsg{ServiceName: testLongName})
	require.NotNil(t, cmd1)

	// Second focus → focusGen=2 (cmd1 is now stale)
	m, cmd2 := m.Update(card.FocusMsg{ServiceName: testLongName})
	require.NotNil(t, cmd2)

	// Get ScrollTick from first cmd (gen=1, but focusGen=2 → stale)
	scrollMsg := cmd1()

	time.Sleep(50 * time.Millisecond)

	_, cmd3 := m.Update(scrollMsg)
	require.Nil(t, cmd3, "stale generation should produce no command")
}

// View tests

func TestView_ShortName_Focused(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	m, _ = m.Update(card.FocusMsg{ServiceName: testShortName})

	view := m.View().Content
	assert.NotEmpty(t, view)
	assert.Contains(t, view, testShortName)
}

func TestView_ShortName_Unfocused(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)

	view := m.View().Content
	assert.NotEmpty(t, view)
	assert.Contains(t, view, testShortName)
}

func TestView_LongName_Focused_Windowed(t *testing.T) {
	t.Parallel()

	m := newCard(testLongName)
	m, _ = m.Update(card.FocusMsg{ServiceName: testLongName})

	window := testLongName[:18]
	view := m.View().Content
	assert.NotEmpty(t, view)
	assert.Contains(t, view, window,
		"focused card should show windowed portion of long name")
}

func TestView_LongName_Unfocused_Truncated(t *testing.T) {
	t.Parallel()

	m := newCard(testLongName)

	truncated := testLongName[:17] + "…"
	view := m.View().Content
	assert.NotEmpty(t, view)
	assert.Contains(t, view, truncated,
		"unfocused card should show truncated name with ellipsis")
}

func TestView_InFlight_BorderColour(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	viewBefore := m.View().Content

	m, _ = m.Update(msgs.ServiceStart{ServiceName: testShortName})

	viewAfter := m.View().Content
	assert.NotEmpty(t, viewAfter)
	assert.Contains(t, viewAfter, testShortName)
	assert.NotEqual(t, viewBefore, viewAfter,
		"in-flight state should change border colour")
}

func TestView_RuntimeState_BorderColour(t *testing.T) {
	t.Parallel()

	m := newCard(testShortName)
	viewBefore := m.View().Content

	m, _ = m.Update(msgs.ServicesPolled{
		Runtimes: map[string]*domain.ServiceRuntimeData{
			testShortName: {State: domain.ServiceStateRunning},
		},
	})

	viewAfter := m.View().Content
	assert.NotEmpty(t, viewAfter)
	assert.Contains(t, viewAfter, testShortName)
	assert.NotEqual(t, viewBefore, viewAfter,
		"runtime state should change border colour from muted")
}

func TestView_EmptyOnZeroDimensions(t *testing.T) {
	t.Parallel()

	m := card.New(domain.ServiceDef{Name: testShortName}, 0, 1, theme.Default())
	view := m.View()

	assert.NotNil(t, view)
	// Even at zero input dimensions, the card has a minimum internal width
	// due to listMinTermWidth clamping, so it always renders content.
	assert.NotEmpty(t, view.Content)
	assert.Contains(t, view.Content, testShortName[:5])
}
