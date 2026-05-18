package prompts

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/ball6847/aoex/internal/detector"
	"github.com/ball6847/aoex/internal/executor"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	stepAgent int = iota
	stepCustomAgent
	stepBranch
	stepSandbox
	stepReview
)

const otherOption = "Other (type manually)"

// agentItem implements the list Item interfaces.
type agentItem struct {
	title string
	desc  string
}

func (i agentItem) FilterValue() string { return i.title }
func (i agentItem) Title() string       { return i.title }
func (i agentItem) Description() string { return i.desc }

type wizardModel struct {
	step    int
	ctx     *detector.Context
	answers executor.Args

	// Navigation history for going back.
	stepHistory []int

	// Step: Agent
	agentList   list.Model
	customAgent textinput.Model

	// Step: Branch
	branchFilter textinput.Model
	branchList   list.Model
	allBranches  []string // all branches for filtering

	// Step: Sandbox
	sandboxYes bool

	// Step: Review
	launchYes bool

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
	// Build agent list: detected agents + custom agents from config + "Other"
	defaultTool := ctx.Config.DefaultTool
	if defaultTool == "" {
		defaultTool = "opencode"
	}

	// Collect unique available agents.
	agentSet := make(map[string]bool, len(ctx.Agents))
	for _, a := range ctx.Agents {
		agentSet[a] = true
	}

	if !agentSet[defaultTool] {
		agentSet[defaultTool] = true
	}

	// Build ordered items; default tool first.
	items := make([]list.Item, 0)
	items = append(items, agentItem{title: defaultTool})
	delete(agentSet, defaultTool)

	// Then remaining agents in deterministic order.
	remaining := make([]string, 0, len(agentSet))
	for name := range agentSet {
		remaining = append(remaining, name)
	}

	for _, name := range remaining {
		items = append(items, agentItem{title: name})
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

	// Branch filter + list.
	branchTi := textinput.New()
	branchTi.Placeholder = "Filter or type a new branch name"
	branchTi.SetValue(ctx.GitBranch)
	branchTi.Focus()
	branchTi.CharLimit = 156
	branchTi.Width = 50

	// Build branch list items.
	branchItems := makeBranchItems(ctx.GitBranch, ctx.Branches, ctx.Worktrees)
	branchLi := list.New(branchItems, list.NewDefaultDelegate(), 50, 15)
	branchLi.Title = "Select a branch"
	branchLi.SetShowStatusBar(false)
	branchLi.SetFilteringEnabled(false)
	branchLi.SetShowHelp(false)

	return wizardModel{
		step:         stepAgent,
		ctx:          ctx,
		answers:      executor.Args{Path: ctx.CWD},
		agentList:    agentLi,
		customAgent:  customTi,
		branchFilter: branchTi,
		branchList:   branchLi,
		allBranches:  ctx.Branches,
		sandboxYes:   ctx.Config.Sandbox,
		launchYes:    false,
		stepHistory:  make([]int, 0, 4),
	}
}

func makeBranchItems(current string, branches []string, worktrees map[string]string) []list.Item {
	items := make([]list.Item, 0, len(branches))
	seen := make(map[string]bool)
	// Always put current branch first.
	if current != "" {
		desc := ""
		if _, ok := worktrees[current]; ok {
			desc = "worktree exists [will attach]"
		}
		items = append(items, agentItem{title: current, desc: desc})
		seen[current] = true
	}

	for _, b := range branches {
		if seen[b] {
			continue
		}

		seen[b] = true
		desc := ""
		if _, ok := worktrees[b]; ok {
			desc = "worktree exists [will attach]"
		}
		items = append(items, agentItem{title: b, desc: desc})
	}

	return items
}

func filterBranchItems(current string, branches []string, worktrees map[string]string, query string) []list.Item {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return makeBranchItems(current, branches, worktrees)
	}

	items := make([]list.Item, 0)
	seen := make(map[string]bool)

	if current != "" && strings.Contains(strings.ToLower(current), q) {
		desc := ""
		if _, ok := worktrees[current]; ok {
			desc = "worktree exists [will attach]"
		}
		items = append(items, agentItem{title: current, desc: desc})
		seen[current] = true
	}

	for _, b := range branches {
		if seen[b] {
			continue
		}

		if strings.Contains(strings.ToLower(b), q) {
			desc := ""
			if _, ok := worktrees[b]; ok {
				desc = "worktree exists [will attach]"
			}
			items = append(items, agentItem{title: b, desc: desc})
			seen[b] = true
		}
	}

	return items
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
		case tea.KeyShiftTab:
			if len(m.stepHistory) > 0 {
				lastIdx := len(m.stepHistory) - 1
				m.step = m.stepHistory[lastIdx]
				m.stepHistory = m.stepHistory[:lastIdx]
				m.errorMsg = ""
			}

			return m, nil
		}

		switch m.step {
		case stepAgent:
			switch msg.Type {
			case tea.KeyEnter, tea.KeyTab:
				sel := m.agentList.SelectedItem()
				if sel == nil {
					return m, nil
				}

				item := sel.(agentItem)
				if item.title == otherOption {
					m.stepHistory = append(m.stepHistory, m.step)
					m.step = stepCustomAgent

					return m, nil
				}

				m.answers.Cmd = item.title
				m.stepHistory = append(m.stepHistory, m.step)
				m.step = stepBranch

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
				m.stepHistory = append(m.stepHistory, m.step)
				m.step = stepBranch

				return m, nil
			}

			var cmd tea.Cmd

			m.customAgent, cmd = m.customAgent.Update(msg)

			return m, cmd

		case stepBranch:
			switch msg.Type {
			case tea.KeyEnter, tea.KeyTab:
				q := strings.TrimSpace(m.branchFilter.Value())
				if reason, ok := isValidBranchName(q); !ok {
					m.errorMsg = reason

					return m, nil
				}

				m.errorMsg = ""
				matched := false

				for _, b := range m.allBranches {
					if b == q {
						matched = true

						break
					}
				}

				// Check if a worktree already exists for this branch.
				if wkPath, ok := m.ctx.Worktrees[q]; ok {
					// Attach to existing worktree: use its path directly, no --worktree flag.
					m.answers.Path = wkPath
					m.answers.Worktree = "" // Clear worktree to omit --worktree flag
					m.answers.NewBranch = false
				} else if !matched {
					// New branch → create worktree with -b.
					m.answers.NewBranch = true
					m.answers.Worktree = q
				} else {
					// Existing branch without worktree → create worktree without -b.
					m.answers.NewBranch = false
					m.answers.Worktree = q
				}

				m.answers.Title = q
				m.stepHistory = append(m.stepHistory, m.step)
				m.step = stepSandbox

				return m, nil
			case tea.KeyUp, tea.KeyDown:
				var cmd tea.Cmd

				m.branchList, cmd = m.branchList.Update(msg)
				if sel := m.branchList.SelectedItem(); sel != nil {
					m.branchFilter.SetValue(sel.(agentItem).title)
				}

				return m, cmd
			}

			var cmd tea.Cmd

			oldVal := m.branchFilter.Value()

			m.branchFilter, cmd = m.branchFilter.Update(msg)

			if m.branchFilter.Value() != oldVal {
				items := filterBranchItems(m.ctx.GitBranch, m.allBranches, m.ctx.Worktrees, m.branchFilter.Value())
				m.branchList.SetItems(items)

				if len(items) > 0 {
					m.branchList.Select(0)
				}
			}

			return m, cmd

		case stepSandbox:
			switch msg.Type {
			case tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown:
				m.sandboxYes = !m.sandboxYes

				return m, nil
			case tea.KeyEnter, tea.KeyTab:
				m.answers.Sandbox = m.sandboxYes
				m.stepHistory = append(m.stepHistory, m.step)
				m.step = stepReview

				return m, nil
			}

			return m, nil

		case stepReview:
			switch msg.Type {
			case tea.KeyLeft, tea.KeyRight, tea.KeyUp, tea.KeyDown:
				m.launchYes = !m.launchYes

				return m, nil
			case tea.KeyEnter, tea.KeyTab:
				if m.launchYes {
					m.answers.Launch = true
				}

				m.done = true

				return m, tea.Quit
			case tea.KeyRunes:
				if len(msg.Runes) == 1 {
					switch msg.Runes[0] {
					case 'y', 'Y':
						m.answers.Launch = true
						m.done = true

						return m, tea.Quit
					case 'n', 'N':
						m.answers.Launch = false
						m.done = true

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
		m.branchList.SetWidth(msg.Width)

		bh := msg.Height - 12
		if bh < 5 {
			bh = 5
		}

		m.branchList.SetHeight(bh)

		return m, nil
	}

	return m, nil
}

func (m wizardModel) View() string {
	var b strings.Builder

	switch m.step {
	case stepAgent:
		b.WriteString(titleStyle.Render("Step 1/4: Agent") + "\n\n")
		b.WriteString(m.agentList.View() + "\n")
		b.WriteString(hintStyle.Render("↑/↓ to navigate, Enter to select, Shift+Tab to go back") + "\n")

	case stepCustomAgent:
		b.WriteString(titleStyle.Render("Step 1/4: Custom Agent") + "\n\n")
		b.WriteString(m.customAgent.View() + "\n")

		if m.errorMsg != "" {
			b.WriteString(errorStyle.Render(m.errorMsg) + "\n")
		}

		b.WriteString(hintStyle.Render("Press Enter to confirm, Shift+Tab to go back") + "\n")

	case stepBranch:
		b.WriteString(titleStyle.Render("Step 2/4: Branch / Worktree") + "\n\n")
		b.WriteString(m.branchFilter.View() + "\n\n")
		b.WriteString(m.branchList.View() + "\n")

		if m.errorMsg != "" {
			b.WriteString(errorStyle.Render(m.errorMsg) + "\n")
		}

		b.WriteString(hintStyle.Render("Type to filter, ↑/↓ to navigate, Enter to select, Shift+Tab to go back") + "\n")
		b.WriteString(hintStyle.Render("If branch doesn't exist, it will be created with -b") + "\n")

	case stepSandbox:
		b.WriteString(titleStyle.Render("Step 3/4: Run in sandbox?") + "\n\n")
		b.WriteString(renderToggle(m.sandboxYes) + "\n\n")
		b.WriteString(hintStyle.Render("←/→ to toggle, Enter to confirm, Shift+Tab to go back") + "\n")

	case stepReview:
		b.WriteString(titleStyle.Render("Step 4/4: Review") + "\n\n")
		b.WriteString(m.answers.String() + "\n\n")
		b.WriteString("Launch immediately?\n")
		b.WriteString(renderToggle(m.launchYes) + "\n\n")
		b.WriteString(hintStyle.Render("y/Enter to confirm and launch, n to save without launching, ←/→ to toggle, Shift+Tab to go back") + "\n")
	}

	return b.String()
}

// isValidBranchName validates a git branch name against core git rules.
// It rejects empty names, names starting with '.', names containing '@{',
// names with spaces or control characters, names ending with '.lock',
// names with double-dots, and names with consecutive slashes.
func isValidBranchName(name string) (string, bool) {
	if name == "" {
		return "Branch name cannot be empty", false
	}

	if strings.HasPrefix(name, ".") {
		return "Branch name cannot start with '.'", false
	}

	if strings.Contains(name, "@{") {
		return "Branch name cannot contain '@{'", false
	}

	if strings.HasSuffix(name, ".lock") {
		return "Branch name cannot end with '.lock'", false
	}

	// Check for double-dots (revision range syntax).
	if strings.Contains(name, "..") {
		return "Branch name cannot contain '..'", false
	}

	// Check for consecutive slashes and trailing slash.
	if strings.Contains(name, "//") || strings.HasSuffix(name, "/") {
		return "Branch name cannot have '//' or end with '/'", false
	}

	// Check each rune for invalid characters.
	for _, r := range name {
		if r == '~' || r == '^' || r == ':' || r == '\\' || r == ' ' || r == '\t' || unicode.IsControl(r) {
			return fmt.Sprintf("Branch name contains invalid character: %q", r), false
		}
	}

	return "", true
}

func renderToggle(yes bool) string {
	yesStr := "Yes"
	noStr := "No"

	if yes {
		return selectedStyle.Render("> "+yesStr) + "    " + unselectedStyle.Render(noStr)
	}

	return unselectedStyle.Render(yesStr) + "    " + selectedStyle.Render("> "+noStr)
}
