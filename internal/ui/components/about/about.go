// Package about implements a read-only About overlay displaying version info,
// ASCII art branding, and a clickable GitHub URL.
package about

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ma-tf/ogle/internal/ui/theme"
	"github.com/ma-tf/ogle/internal/version"
)

const ascii = `                         _               __         
       , ·. ,.-·~·.,   ‘              ,.-·^*ª'' ·,                 ,.  '                      _,.,  °    
      /  ·'´,.-·-.,   ','‚           .·´ ,·'´:¯''·,  '\‘            /   ';\               ,.·'´  ,. ,  ';\ '  
     /  .'´\:::::::'\   '\ °       ,´  ,'\:::::::::\,.·\'         ,'   ,'::'\            .´   ;´:::::\''´ \'\  
  ,·'  ,'::::\:;:-·-:';  ';\‚      /   /:::\;·'´¯''·;\:::\°      ,'    ;:::';'          /   ,'::\::::::\:::\:' 
 ;.   ';:::;´       ,'  ,':'\‚    ;   ;:::;'          '\;:·´      ';   ,':::;'          ;   ;:;:-·'~^ª*';\'´   
  ';   ;::;       ,'´ .'´\::';‚  ';   ;::/      ,·´¯';  °        ;  ,':::;' '          ;  ,.-·:*'´¨''*´\::\ '  
  ';   ':;:   ,.·´,.·´::::\;'°  ';   '·;'   ,.·´,    ;'\         ,'  ,'::;'            ;   ;\::::::::::::'\;'   
   \·,   '*´,.·'´::::::;·´     \'·.    ''´,.·:´';   ;::\'       ;  ';_:,.-·´';\‘     ;  ;'_\_:;:: -·^*';\   
    \\:¯::\:::::::;:·´         '\::\¯::::::::';   ;::'; ‘     ',   _,.-·'´:\:\‘    ';    ,  ,. -·:*'´:\:'\° 
     '\:::::\;::·'´  °            '·:\:::;:·´';.·´\::;'         \¨:::::::::::\';     \'*´ ¯\:::::::::::\;' '
         ¯                           ¯      \::::\;'‚          '\;::_;:-·'´‘         \:::::\;::-·^*'´     
          ‘                                    '\:·´'              '¨                    '*´¯              `

// Model is a read-only about overlay.
type Model struct {
	th *theme.Theme
	w  int
	h  int
}

// New returns a Model.
func New(th *theme.Theme) Model {
	return Model{th: th, w: 0, h: 0}
}

// Init satisfies tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update satisfies tea.Model.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w = msg.Width
		m.h = msg.Height
	case theme.Changed:
		m.th = msg.Theme
	}

	return m, nil
}

// View renders the about overlay.
func (m Model) View() tea.View {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.th.Text).
		Render("ogle")

	artStyle := lipgloss.NewStyle().
		Foreground(m.th.Subtext).
		Render(ascii)

	versionLine := lipgloss.NewStyle().
		Foreground(m.th.Text).
		Render(version.Version + " (commit: " + version.Commit + ", built: " + version.Date + ")")

	url := ansi.SetHyperlink("https://github.com/ma-tf/ogle") +
		"github.com/ma-tf/ogle" +
		ansi.ResetHyperlink()

	urlStyle := lipgloss.NewStyle().
		Foreground(m.th.StateRunning).
		Render(url)

	closeHint := lipgloss.NewStyle().
		Foreground(m.th.StateMuted).
		Render("F1 / esc / q to close")

	content := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		artStyle,
		"",
		versionLine,
		"",
		urlStyle,
		"",
		closeHint,
	)

	boxW := lipgloss.Width(content)

	return tea.NewView(lipgloss.NewStyle().
		Width(boxW).
		Padding(0, 2). //nolint:mnd // horizontal padding for overlay box
		Background(m.th.BodyBackground).
		Render(content))
}
