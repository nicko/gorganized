package tui

import tea "github.com/charmbracelet/bubbletea"

// Run starts the Bubble Tea application.
func Run(gorganDir string) error {
	m := newApp(gorganDir)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

type app struct {
	gorganDir string
}

func newApp(gorganDir string) app {
	return app{gorganDir: gorganDir}
}

func (a app) Init() tea.Cmd {
	return nil
}

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return a, tea.Quit
		}
	}
	return a, nil
}

func (a app) View() string {
	return "gorganized — press q to quit\n"
}
