package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/flobilosaurus/agent-env/internal/config"
)

var (
	accentStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	borderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

type ProfilePrompter interface {
	ChooseProfile(agent string, profiles []config.Profile) (profile string, create bool, err error)
}

type ProfileRemovePrompter interface {
	ChooseProfileToRemove(profiles []config.Profile) (profile string, err error)
}

type BubblePrompter struct{}

func (BubblePrompter) ChooseProfile(agent string, profiles []config.Profile) (string, bool, error) {
	m := newModel(agent, profiles)
	p := tea.NewProgram(m, tea.WithAltScreen())
	res, err := p.Run()
	if err != nil {
		return "", false, err
	}
	fm := res.(model)
	if fm.cancelled {
		return "", false, fmt.Errorf("profile selection cancelled")
	}
	return fm.selected, fm.created, nil
}

func (BubblePrompter) ChooseProfileToRemove(profiles []config.Profile) (string, error) {
	m := newRemoveModel(profiles)
	p := tea.NewProgram(m, tea.WithAltScreen())
	res, err := p.Run()
	if err != nil {
		return "", err
	}
	fm := res.(removeModel)
	if fm.cancelled {
		return "", fmt.Errorf("profile removal cancelled")
	}
	return fm.selected, nil
}

type mode int

const (
	modeSelect mode = iota
	modeCreate
)

type model struct {
	agent     string
	profiles  []config.Profile
	cursor    int
	mode      mode
	input     textinput.Model
	error     string
	selected  string
	created   bool
	cancelled bool
}

func newModel(agent string, profiles []config.Profile) model {
	ti := textinput.New()
	ti.Placeholder = "profile-name"
	ti.PromptStyle = selectedStyle
	ti.PlaceholderStyle = mutedStyle
	ti.TextStyle = selectedStyle
	ti.Focus()
	return model{agent: agent, profiles: profiles, input: ti}
}
func (m model) Init() tea.Cmd { return textinput.Blink }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "up", "k":
			if m.mode == modeSelect && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.mode == modeSelect && m.cursor < len(m.profiles) {
				m.cursor++
			}
		case "enter":
			if m.mode == modeSelect {
				if m.cursor == len(m.profiles) {
					m.mode = modeCreate
					return m, nil
				}
				m.selected = m.profiles[m.cursor].Name
				return m, tea.Quit
			}
			name := strings.TrimSpace(m.input.Value())
			if err := config.ValidateProfileName(name); err != nil {
				m.error = err.Error()
				return m, nil
			}
			m.selected, m.created = name, true
			return m, tea.Quit
		}
	}
	if m.mode == modeCreate {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	if m.mode == modeCreate {
		lines := []string{"", accentStyle.Render("  Create a Profile"), mutedStyle.Render("  Allowed: lowercase, numbers, dot, dash, underscore"), "", "  " + m.input.View(), "", mutedStyle.Render("  enter create • esc/ctrl+c cancel")}
		if m.error != "" {
			lines = append(lines, errorStyle.Render("  "+m.error))
		}
		return renderProfileBox(strings.TrimSpace(m.input.Value()), m.agent, lines)
	}
	items := []string{"", accentStyle.Render("  Select a Profile"), mutedStyle.Render("  Choose an isolated HOME for this project"), ""}
	for i, p := range m.profiles {
		prefix := "    "
		line := prefix + p.Name
		if i == m.cursor {
			line = selectedStyle.Render("  ▸ " + p.Name)
		}
		items = append(items, line)
	}
	items = append(items, "")
	createLine := "    ＋ Create new profile"
	if m.cursor == len(m.profiles) {
		createLine = selectedStyle.Render("  ▸ ＋ Create new profile")
	}
	items = append(items, createLine)
	items = append(items, "", mutedStyle.Render("  ↑/↓/j/k move • enter select • esc/ctrl+c cancel"))
	return renderProfileBox(m.currentProfileLabel(), m.agent, items)
}

func (m model) currentProfileLabel() string {
	if m.cursor >= 0 && m.cursor < len(m.profiles) {
		return m.profiles[m.cursor].Name
	}
	return ""
}

type removeModel struct {
	profiles  []config.Profile
	cursor    int
	selected  string
	cancelled bool
}

func newRemoveModel(profiles []config.Profile) removeModel {
	return removeModel{profiles: profiles}
}

func (m removeModel) Init() tea.Cmd { return nil }
func (m removeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.profiles)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.profiles) > 0 {
				m.selected = m.profiles[m.cursor].Name
			}
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m removeModel) View() string {
	items := []string{"", accentStyle.Render("  Remove a Profile"), mutedStyle.Render("  Select a profile to delete with its folder"), ""}
	for i, p := range m.profiles {
		line := "    " + p.Name
		if i == m.cursor {
			line = selectedStyle.Render("  ▸ " + p.Name)
		}
		items = append(items, line)
	}
	items = append(items, "", mutedStyle.Render("  ↑/↓/j/k move • enter remove • esc/ctrl+c cancel"))
	return renderActionBox(items)
}

func renderActionBox(lines []string) string {
	const width = 58
	var b strings.Builder
	b.WriteString(borderStyle.Render("╭─ ") + accentStyle.Render("agentenv") + borderStyle.Render(" ───────────────────────────────────────────────╮") + "\n")
	for _, line := range lines {
		b.WriteString(profileLine(width, line))
	}
	b.WriteString(borderStyle.Render("╰──────────────────────────────────────────────────────────╯"))
	return b.String()
}

func renderProfileBox(profile, agent string, lines []string) string {
	const width = 58
	var b strings.Builder
	b.WriteString(borderStyle.Render("╭─ ") + accentStyle.Render("agentenv") + borderStyle.Render(" ───────────────────────────────────────────────╮") + "\n")
	b.WriteString(profileLine(width, fmt.Sprintf(" %s • %s", profile, agent)))
	b.WriteString(borderStyle.Render("├──────────────────────────────────────────────────────────┤") + "\n")
	for _, line := range lines {
		b.WriteString(profileLine(width, line))
	}
	b.WriteString(borderStyle.Render("╰──────────────────────────────────────────────────────────╯"))
	return b.String()
}

func profileLine(width int, line string) string {
	if pad := width - lipgloss.Width(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	return borderStyle.Render("│") + line + borderStyle.Render("│") + "\n"
}

func Banner(profile, agent string) string {
	const width = 46
	text := fmt.Sprintf("%s • %s", profile, agent)
	line := " " + text
	if pad := width - lipgloss.Width(line); pad > 0 {
		line += strings.Repeat(" ", pad)
	}
	return borderStyle.Render("┌─ ") + accentStyle.Render("agentenv") + borderStyle.Render(" ───────────────────────────────────┐") + "\n" +
		borderStyle.Render("│") + line + borderStyle.Render("│") + "\n" +
		borderStyle.Render("└──────────────────────────────────────────────┘")
}
