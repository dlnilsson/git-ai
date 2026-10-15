package ui

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	menuTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("205")).
			Padding(0, 1)
	menuCursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	menuSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	menuChoiceStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	menuHelpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	menuKeyStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Bold(true)
)

type selectModel struct {
	title    string
	choices  []string
	cursor   int
	selected string
	done     bool
}

// SelectMenu renders a single-choice menu on the terminal and returns the
// picked entry. Output goes to /dev/tty so stdout stays clean for the commit
// message when called through the git-ai wrapper.
func SelectMenu(title string, choices []string) (string, error) {
	if len(choices) == 0 {
		return "", errors.New("no choices available for selection")
	}
	m := selectModel{title: title, choices: choices}
	p := tea.NewProgram(m, tea.WithOutput(getTerminalOutput()))
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	selected := final.(selectModel).selected
	if strings.TrimSpace(selected) == "" {
		return "", errors.New("nothing selected")
	}
	return selected, nil
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		case "enter":
			m.selected = m.choices[m.cursor]
			m.done = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m selectModel) View() tea.View {
	if m.done {
		return tea.NewView("\r\033[2K")
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("\n %s\n\n", menuTitleStyle.Render(m.title)))
	for i, choice := range m.choices {
		if m.cursor == i {
			b.WriteString(fmt.Sprintf(" %s %s\n", menuCursorStyle.Render("▸"), menuSelectedStyle.Render(choice)))
			continue
		}
		b.WriteString(fmt.Sprintf("   %s\n", menuChoiceStyle.Render(choice)))
	}
	b.WriteString(fmt.Sprintf("\n %s %s %s %s %s\n",
		menuKeyStyle.Render("↑/↓"), menuHelpStyle.Render("move ·"),
		menuKeyStyle.Render("enter"), menuHelpStyle.Render("select ·"),
		menuKeyStyle.Render("q/esc")+menuHelpStyle.Render(" cancel")))
	return tea.NewView(b.String())
}
