package states

import tea "charm.land/bubbletea/v2"

// Scanning is the initial state: a directory scan is in flight and no view is
// rendered.
type Scanning struct {
	Scan        tea.Cmd
	HandleFiles func([]string, tea.Model) (tea.Model, tea.Cmd)
}

func (s Scanning) Init() tea.Cmd {
	return s.Scan
}

// Update dispatches to HandleFiles on scan completion. Other messages are
// dropped — the scan is sub-millisecond and the view is blank.
func (s Scanning) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if done, ok := msg.(scanDoneMsg); ok {
		return s.HandleFiles(done.valid, s)
	}

	return s, nil
}

// View is blank — the scan is sub-millisecond and a blank screen is intentional.
func (s Scanning) View() tea.View { return tea.NewView("") }
