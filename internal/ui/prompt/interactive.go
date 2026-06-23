package prompt

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	promptTitleStyle = lipgloss.NewStyle().Bold(true)
	promptValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	promptCursor     = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Render("█")
	promptErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	promptHelpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	selectCursor     = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true).Render("> ")
	selectItemStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	selectActiveItem = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229"))
)

type inputModel struct {
	label     string
	value     []rune
	mask      bool
	validate  func(string) error
	errText   string
	final     string
	cancelled bool
}

func newInputModel(label, defaultValue string, mask bool, validate func(string) error) *inputModel {
	return &inputModel{
		label:    strings.TrimSpace(label),
		value:    []rune(defaultValue),
		mask:     mask,
		validate: validate,
	}
}

func (m *inputModel) Init() tea.Cmd {
	return nil
}

func (m *inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			value := string(m.value)
			if m.validate != nil {
				if err := m.validate(value); err != nil {
					m.errText = err.Error()
					return m, nil
				}
			}
			m.final = value
			return m, tea.Quit
		case "backspace", "ctrl+h":
			if len(m.value) > 0 {
				m.value = m.value[:len(m.value)-1]
			}
		case "ctrl+u":
			m.value = nil
		default:
			if len(msg.Runes) > 0 {
				m.value = append(m.value, msg.Runes...)
			}
		}
		m.errText = ""
	}
	return m, nil
}

func (m *inputModel) View() string {
	display := string(m.value)
	if m.mask {
		display = strings.Repeat("*", utf8.RuneCountInString(display))
	}

	out := []string{
		promptTitleStyle.Render(m.label),
		promptValueStyle.Render("> " + display + promptCursor),
	}
	if strings.TrimSpace(m.errText) != "" {
		out = append(out, promptErrorStyle.Render(m.errText))
	}
	out = append(out, promptHelpStyle.Render("Enter: submit | Esc/Ctrl+C: cancel"))
	return strings.Join(out, "\n")
}

type selectModel struct {
	label     string
	items     []string
	cursor    int
	cancelled bool
}

func newSelectModel(label string, items []string) *selectModel {
	return &selectModel{
		label:  strings.TrimSpace(label),
		items:  items,
		cursor: 0,
	}
}

func (m *selectModel) Init() tea.Cmd {
	return nil
}

func (m *selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "ctrl+d", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "up", "k":
			if len(m.items) > 0 {
				m.cursor = (m.cursor - 1 + len(m.items)) % len(m.items)
			}
		case "down", "j":
			if len(m.items) > 0 {
				m.cursor = (m.cursor + 1) % len(m.items)
			}
		case "enter":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *selectModel) View() string {
	lines := make([]string, 0, len(m.items)+2)
	lines = append(lines, promptTitleStyle.Render(m.label))
	for i, item := range m.items {
		if i == m.cursor {
			lines = append(lines, selectCursor+selectActiveItem.Render(item))
			continue
		}
		lines = append(lines, "  "+selectItemStyle.Render(item))
	}
	lines = append(lines, promptHelpStyle.Render("Up/Down or j/k | Enter: select | Esc/Ctrl+C: cancel"))
	return strings.Join(lines, "\n")
}

func runTea(model tea.Model) (tea.Model, error) {
	p := tea.NewProgram(model, tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout))
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}
	return finalModel, nil
}

func InputPrompt(label, defaultValue string, mask bool, validate func(string) error) (string, error) {
	finalModel, err := runTea(newInputModel(label, defaultValue, mask, validate))
	if err != nil {
		return "", err
	}

	m, ok := finalModel.(*inputModel)
	if !ok {
		return "", errors.New("unexpected input model result")
	}
	if m.cancelled {
		return "", ErrCancelled
	}
	return m.final, nil
}

func SelectPrompt(label string, items []string) (int, string, error) {
	if len(items) == 0 {
		return -1, "", fmt.Errorf("no items to select")
	}

	finalModel, err := runTea(newSelectModel(label, items))
	if err != nil {
		return -1, "", err
	}

	m, ok := finalModel.(*selectModel)
	if !ok {
		return -1, "", errors.New("unexpected select model result")
	}
	if m.cancelled {
		return -1, "", ErrCancelled
	}
	return m.cursor, m.items[m.cursor], nil
}
