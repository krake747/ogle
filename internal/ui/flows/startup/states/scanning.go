package states

import tea "charm.land/bubbletea/v2"

// Scanning is the initial state: a directory scan is in flight and no view is
// rendered. It transitions as soon as scanDoneMsg arrives.
type Scanning struct {
	Scan        tea.Cmd
	HandleFiles func([]string, State) (State, tea.Cmd)
}

// Init fires the pre-built scan command.
func (s Scanning) Init() tea.Cmd {
	return s.Scan
}

// Update handles the scan result and delegates to HandleFiles for the
// 0/1/2+ dispatch. All other messages are silently dropped — the scan is
// sub-millisecond and the view is blank.
func (s Scanning) Update(msg tea.Msg) (State, tea.Cmd) {
	if done, ok := msg.(scanDoneMsg); ok {
		return s.HandleFiles(done.valid, s)
	}

	return s, nil
}

// View returns an empty string. The scan is sub-millisecond; a blank screen is
// intentional.
func (s Scanning) View() string { return "" }
