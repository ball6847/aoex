package prompts

import (
	"fmt"
	"strings"

	"github.com/ball6847/aoex/internal/detector"
	"github.com/ball6847/aoex/internal/executor"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	stepPath int = iota
	stepTitle
	stepGroup
	stepAgent
	stepCustomAgent
	stepLaunch
	stepWorktree
	stepSandbox
	stepReview
)

const otherOption = "Other (type manually)"

// agentItem implements the list Item interfaces.
type agentItem struct {
	title string
}

func (i agentItem) FilterValue() string { return i.title }
func (i agentItem) Title() string       { return i.title }
func (i agentItem) Description() string { return "" }

type wizardModel struct {
	step    int
	ctx     *detector.Context
	answers executor.Args

	pathInput     textinput.Model
	titleInput    textinput.Model
	groupInput    textinput.Model
	agentList     list.Model
	customAgent   textinput.Model
	launchYes     bool
	worktreeInput textinput.Model
	sandboxYes    bool
	reviewYes     bool

	errorMsg  string
	done      bool
	cancelled bool
}

var (
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	errorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	selectedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	unselectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	hintStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
)

func initialModel(ctx *detector.Context) wizardModel {
	// Path input.
	pathTi := textinput.New()
	pathTi.Placeholder = "Path"
	pathTi.SetValue(ctx.CWD)
	pathTi.Focus()
	pathTi.CharLimit = 256
	pathTi.Width = 50

	// Title input.
	titleTi := textinput.New()
	titleTi.Placeholder = "Title"
	titleTi.SetValue(ctx.FolderName)
	titleTi.Focus()
	titleTi.CharLimit = 156
	titleTi.Width = 50

	// Group input.
	groupTi := textinput.New()
	groupTi.Placeholder = "Group"
	groupTi.SetValue(ctx.ParentFolder)
	groupTi.Focus()
	groupTi.CharLimit = 156
	groupTi.Width = 50

	// Agent list.
	items := make([]list.Item, 0, len(ctx.Agents)+1)
	for _, a := range ctx.Agents {
		items = append(items, agentItem{title: a})
	}
	items = append(items, agentItem{title: otherOption})

	listHeight := len(items) + 4
	if listHeight < 6 {
		listHeight = 6
	}

	agentLi := list.New(items, list.NewDefaultDelegate(), 40, listHeight)
	agentLi.Title = "Select an agent"
	agentLi.SetShowStatusBar(false)
	agentLi.SetFilteringEnabled(false)
	agentLi.SetShowHelp(false)

	// Custom agent input.
	customTi := textinput.New()
	customTi.Placeholder = "Custom agent command"
	customTi.Focus()
	customTi.CharLimit = 156
	customTi.Width = 50

	// Worktree input.
	worktreeTi := textinput.New()
	worktreeTi.Placeholder = "Worktree (git branch)"
	worktreeTi.SetValue(ctx.GitBranch)
	worktreeTi.Focus()
	worktreeTi.CharLimit = 156
	worktreeTi.Width = 50

	return wizardModel{
		step:          stepPath,
		ctx:           ctx,
		pathInput:     pathTi,
		titleInput:    titleTi,
		groupInput:    groupTi,
		agentList:     agentLi,
		customAgent:   customTi,
		launchYes:     false,
		worktreeInput: worktreeTi,
		sandboxYes:    false,
		reviewYes:     true,
	}
}

// RunWizard starts the interactive wizard and returns the collected arguments.
// If the user cancels, it returns (nil, nil).
func RunWizard(ctx *detector.Context) (*executor.Args, error) {
	m := initialModel(ctx)
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return nil, fmt.Errorf("wizard failed: %w", err)
	}
	wm := finalModel.(wizardModel)
	if wm.cancelled || !wm.done {
		return nil, nil
	}
	return &wm.answers, nil
}

func (m wizardModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.cancelled = true
			return m, tea.Quit
		}

		switch m.step {
		case stepPath:
			switch msg.Type {
			case tea.KeyEnter, tea.KeyTab:
				m.answers.Path = m.pathInput.Value()
				m.step = stepTitle
				return m, nil
			}
			var cmd tea.Cmd
			m.pathInput, cmd = m.pathInput.Update(msg)
			return m, cmd

		case stepTitle:
			switch msg.Type {
			case tea.KeyEnter, tea.KeyTab:
				m.answers.Title = m.titleInput.Value()
				if m.answers.Title == "" {
					m.errorMsg = "Title cannot be empty"
					return m, nil
				}
				m.errorMsg = ""
				m.step = stepGroup
				return m, nil
			}
			var cmd tea.Cmd
			m.titleInput, cmd = m.titleInput.Update(msg)
			return m, cmd

		case stepGroup:
			switch msg.Type {
			case tea.KeyEnter, tea.KeyTab:
				m.answers.Group = m.groupInput.Value()
				if m.answers.Group == "" {
					m.errorMsg = "Group cannot be empty"
					return m, nil
				}
				m.errorMsg = ""
				m.step = stepAgent
				return m, nil
			}
			var cmd tea.Cmd
			m.groupInput, cmd = m.groupInput.Update(msg)
			return m, cmd

		case stepAgent:
			switch msg.Type {
			case tea.KeyEnter, tea.KeyTab:
				sel := m.agentList.SelectedItem()
				if sel == nil {
					return m, nil
				}
				item := sel.(agentItem)
				if item.title == otherOption {
					m.step = stepCustomAgent
					return m, nil
				}
				m.answers.Cmd = item.title
				m.step = stepLaunch
				return m, nil
			}
			var cmd tea.Cmd
			m.agentList, cmd = m.agentList.Update(msg)
			return m, cmd

		case stepCustomAgent:
			switch msg.Type {
			case tea.KeyEnter, tea.KeyTab:
				m.answers.Cmd = m.customAgent.Value()
				if m.answers.Cmd == "" {
					m.errorMsg = "Agent command cannot be empty"
					return m, nil
				}
				m.errorMsg = ""
				m.step = stepLaunch
				return m, nil
			}
			var cmd tea.Cmd
			m.customAgent, cmd = m.customAgent.Update(msg)
			return m, cmd

		case stepLaunch:
			switch msg.Type {
			case tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown:
				m.launchYes = !m.launchYes
				return m, nil
			case tea.KeyEnter, tea.KeyTab:
				m.answers.Launch = m.launchYes
				if m.ctx.IsGitRepo {
					m.step = stepWorktree
				} else {
					m.step = stepSandbox
				}
				return m, nil
			}
			return m, nil

		case stepWorktree:
			switch msg.Type {
			case tea.KeyEnter, tea.KeyTab:
				m.answers.Worktree = m.worktreeInput.Value()
				m.step = stepSandbox
				return m, nil
			}
			var cmd tea.Cmd
			m.worktreeInput, cmd = m.worktreeInput.Update(msg)
			return m, cmd

		case stepSandbox:
			switch msg.Type {
			case tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown:
				m.sandboxYes = !m.sandboxYes
				return m, nil
			case tea.KeyEnter, tea.KeyTab:
				m.answers.Sandbox = m.sandboxYes
				m.step = stepReview
				return m, nil
			}
			return m, nil

		case stepReview:
			switch msg.Type {
			case tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown:
				m.reviewYes = !m.reviewYes
				return m, nil
			case tea.KeyEnter, tea.KeyTab:
				if m.reviewYes {
					m.done = true
				} else {
					m.cancelled = true
				}
				return m, tea.Quit
			case tea.KeyRunes:
				if len(msg.Runes) == 1 {
					switch msg.Runes[0] {
					case 'y', 'Y':
						m.reviewYes = true
						m.done = true
						return m, tea.Quit
					case 'n', 'N':
						m.reviewYes = false
						m.cancelled = true
						return m, tea.Quit
					}
				}
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.agentList.SetWidth(msg.Width)
		h := msg.Height - 10
		if h < 5 {
			h = 5
		}
		m.agentList.SetHeight(h)
		return m, nil
	}

	return m, nil
}

func (m wizardModel) View() string {
	var b strings.Builder

	switch m.step {
	case stepPath:
		b.WriteString(titleStyle.Render("Step 1/8: Path") + "\n\n")
		b.WriteString(m.pathInput.View() + "\n")
		b.WriteString(hintStyle.Render("Press Enter to confirm") + "\n")

	case stepTitle:
		b.WriteString(titleStyle.Render("Step 2/8: Title") + "\n\n")
		b.WriteString(m.titleInput.View() + "\n")
		if m.errorMsg != "" {
			b.WriteString(errorStyle.Render(m.errorMsg) + "\n")
		}
		b.WriteString(hintStyle.Render("Press Enter to confirm") + "\n")

	case stepGroup:
		b.WriteString(titleStyle.Render("Step 3/8: Group") + "\n\n")
		b.WriteString(m.groupInput.View() + "\n")
		if m.errorMsg != "" {
			b.WriteString(errorStyle.Render(m.errorMsg) + "\n")
		}
		b.WriteString(hintStyle.Render("Press Enter to confirm") + "\n")

	case stepAgent:
		b.WriteString(titleStyle.Render("Step 4/8: Agent") + "\n\n")
		b.WriteString(m.agentList.View() + "\n")
		b.WriteString(hintStyle.Render("↑/↓ to navigate, Enter to select") + "\n")

	case stepCustomAgent:
		b.WriteString(titleStyle.Render("Step 4/8: Custom Agent") + "\n\n")
		b.WriteString(m.customAgent.View() + "\n")
		if m.errorMsg != "" {
			b.WriteString(errorStyle.Render(m.errorMsg) + "\n")
		}
		b.WriteString(hintStyle.Render("Press Enter to confirm") + "\n")

	case stepLaunch:
		b.WriteString(titleStyle.Render("Step 5/8: Launch immediately?") + "\n\n")
		b.WriteString(renderToggle(m.launchYes) + "\n\n")
		b.WriteString(hintStyle.Render("←/→ to toggle, Enter to confirm") + "\n")

	case stepWorktree:
		b.WriteString(titleStyle.Render("Step 6/8: Worktree") + "\n\n")
		b.WriteString(m.worktreeInput.View() + "\n")
		b.WriteString(hintStyle.Render("Press Enter to confirm") + "\n")

	case stepSandbox:
		b.WriteString(titleStyle.Render("Step 7/8: Run in sandbox?") + "\n\n")
		b.WriteString(renderToggle(m.sandboxYes) + "\n\n")
		b.WriteString(hintStyle.Render("←/→ to toggle, Enter to confirm") + "\n")

	case stepReview:
		b.WriteString(titleStyle.Render("Step 8/8: Review") + "\n\n")
		b.WriteString(m.answers.String() + "\n\n")
		b.WriteString("Confirm execution?\n")
		b.WriteString(renderToggle(m.reviewYes) + "\n\n")
		b.WriteString(hintStyle.Render("y/Enter to confirm, n to cancel, ←/→ to toggle") + "\n")
	}

	return b.String()
}

func renderToggle(yes bool) string {
	yesStr := "Yes"
	noStr := "No"
	if yes {
		return selectedStyle.Render("> "+yesStr) + "    " + unselectedStyle.Render(noStr)
	}
	return unselectedStyle.Render(yesStr) + "    " + selectedStyle.Render("> "+noStr)
}
