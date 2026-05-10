package states

import tea "charm.land/bubbletea/v2"

// Scanning is the initial state: a directory scan is in flight and no view is
// rendered.
type Scanning struct {
	scan    tea.Cmd
	handler fileHandler
}

// NewScanning constructs the initial Scanning state for the given directory.
func NewScanning(dir string) tea.Model {
	fh := fileHandler{dir: dir}

	return Scanning{scan: ScanCmd(dir), handler: fh}
}

// Init returns the scan command, kicking off the directory scan.
func (s Scanning) Init() tea.Cmd { return s.scan }

// Update dispatches to handler on scan completion. Other messages are
// dropped — the scan is sub-millisecond and the view is blank.
func (s Scanning) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if done, ok := msg.(scanDoneMsg); ok {
		return s.handler.handle(done.valid, s)
	}

	return s, nil
}

// View is blank — the scan is sub-millisecond and a blank screen is intentional.
func (s Scanning) View() tea.View { return tea.NewView("") }
