package ui

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

type modelSelectModel struct {
	choices  []string
	cursor   int
	selected string
	done     bool
}

func SelectModelMenu(choices []string) (string, error) {
	if len(choices) == 0 {
		return "", errors.New("no models available for selection")
	}
	m := modelSelectModel{choices: choices}
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return "", err
	}
	selected := final.(modelSelectModel).selected
	if strings.TrimSpace(selected) == "" {
		return "", errors.New("no model selected")
	}
	return selected, nil
}

func (m modelSelectModel) Init() tea.Cmd {
	return nil
}

func (m modelSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

func (m modelSelectModel) View() tea.View {
	if m.done {
		return tea.NewView("\r\033[2K")
	}
	var b strings.Builder
	b.WriteString("\nSelect a model:\n\n")
	for i, choice := range m.choices {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}
		b.WriteString(fmt.Sprintf(" %s %s\n", cursor, choice))
	}
	b.WriteString("\nEnter to select, q/esc to cancel.\n")
	return tea.NewView(b.String())
}
