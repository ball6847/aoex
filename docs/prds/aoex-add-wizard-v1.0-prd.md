# aoex-add-wizard-v1.0-prd.md

## Requirements Description

### Background

`aoe add` supports many flags (`--title`, `--group`, `--cmd`, `--launch`, `--worktree`, `--sandbox`, etc.). Most users only need a small subset for day-to-day use. Remembering the exact flag names and order is friction.

### Feature Overview

**Core feature**: `aoex add` — an interactive wizard that prompts for the most common `aoe add` arguments, then invokes the original tool.

**Feature boundaries**:

- IN: interactive prompts for the common flags
- IN: auto-discovery of context (current dir, git branch, installed agents)
- IN: review screen before execution
- IN: pass-through of all collected args to `aoe add .`
- OUT: support for advanced flags (`--parent`, `--cockpit`, `--sandbox-image`, `--extra-args`)
- OUT: direct modification of aoe session storage (always shells out)

### Detailed Requirements

#### Input/Output

**Input**: user responses to interactive prompts
**Output**: shell execution of `aoe add . [computed-flags]`

#### User Interaction Flow

1. **Project path** — defaults to current directory
2. **Session title** — defaults to folder name (or current git branch if worktree mode)
3. **Group path** — defaults to parent folder name
4. **Agent / command** — list of auto-detected installed agents (`claude`, `codex`, `opencode`, etc.); allow free-text override
5. **Launch now?** — yes/no (maps to `--launch`)
6. **Worktree branch** — optional; if in a git repo, suggest current branch as default
7. **Sandbox mode** — yes/no (maps to `--sandbox`)
8. **Review & confirm** — show the computed `aoe add . ...` command before executing

#### Data Requirements

| Wizard Prompt   | `aoe add` Flag           | Default Value                       |
| --------------- | ------------------------ | ----------------------------------- |
| Path            | positional (default `.`) | `os.Getwd()`                        |
| Title           | `--title`                | `filepath.Base(cwd)`                |
| Group           | `--group`                | `filepath.Base(filepath.Dir(cwd))`  |
| Agent / command | `--cmd`                  | auto-detected from `$PATH`          |
| Launch now      | `--launch`               | `false`                             |
| Worktree branch | `--worktree`             | current `git branch --show-current` |
| Sandbox         | `--sandbox`              | `false`                             |

#### Empty Input Validation

If the user clears a default value and presses Enter with no input:
- **Re-prompt until valid**: Display an error message (e.g., "Title cannot be empty") and present the prompt again with the default pre-filled.
- This applies to all required text fields (Title, Group, Agent/Command).

#### Edge Cases

- If `aoe` is not found on `$PATH`: error immediately before any prompts (or warn and allow continuing)
- If not in a git repo: skip or disable the worktree branch prompt
- If no agents detected on `$PATH`: allow manual free-text entry for command
- User cancels at any prompt: clean exit with no side effects
- Duplicate session (same title + path detected by aoe): show aoe's error message

## Design Decisions

### Technical Approach

- **Architecture**: Single command CLI with modular packages: `cmd/`, `internal/prompts/`, `internal/detector/`, `internal/executor/`
- **Prompt Library**: `bubbletea` (charmbracelet/bubbletea) — provides a rich, interactive TUI experience for the wizard.
- **Integration**: `exec.Command("aoe", "add", ".", ...)` — no import of aoe source code
- **Discovery logic**: `os.Getwd()`, `exec.Command("git", ...).Output()`, `exec.LookPath()` for agent binaries

### Constraints

- Must work cross-platform (macOS, Linux)
- Must not write to aoe session storage directly
- Only supports aoe `add` subcommand for now

### Risk Assessment

- **Binary not found**: `aoe` may not be on `$PATH` → detect early and print install instructions
- **Git not installed**: worktree detection fails gracefully → disable worktree prompt
- **Prompt library TTY issues**: bubbletea may behave differently in CI/non-TTY → consider `--non-interactive` flag in future

## Acceptance Criteria

### Functional Acceptance

- [ ] Running `aoex add` in any directory starts the interactive wizard
- [ ] Each prompt has a sensible default pre-filled ( press Enter to accept)
- [ ] The "Review & confirm" step prints the exact `aoe add . ...` command that will run
- [ ] Confirming the review executes `aoe add` with collected arguments
- [ ] Cancelling at any step exits cleanly with code 0 and no shell command executed

### Quality Standards

- [ ] `go build` produces a single static binary
- [ ] `go vet` and `gofmt` pass with no issues
- [ ] Errors include context (e.g., `fmt.Errorf("failed to detect git branch: %w", err)`)

## Execution Phases

### Phase 1: Bootstrap

**Goal**: Initialize Go module and basic CLI skeleton

- [ ] `go mod init github.com/ball6847/aoex`
- [ ] Add `cobra` and `bubbletea` dependencies
- [ ] Create `main.go` with root command and `add` subcommand placeholder
- **Time**: ~30 min

### Phase 2: Prompts & Discovery

**Goal**: Implement all prompts with auto-discovery

- [ ] Create `internal/detector/` package (cwd, git branch, agents on PATH)
- [ ] Create `internal/prompts/` package with bubbletea prompts
- [ ] Wire prompts into `cmd/add.go`
- **Time**: ~1-2 hrs

### Phase 3: Execution & Polish

**Goal**: Shell out to `aoe add` and handle errors

- [ ] Create `internal/executor/` package that builds and runs the `aoe` command
- [ ] Add "Review & confirm" step
- [ ] Handle edge cases (aoe not found, git not found, empty inputs)
- **Time**: ~1 hr

### Phase 4: Build & Verify

**Goal**: Ensure binary builds and runs correctly

- [ ] `go build -o aoex .`
- [ ] Test `aoex add` end-to-end
- [ ] Update AGENTS.md if needed
- **Time**: ~30 min

---

**Document Version**: 1.0
**Created**: 2025-05-17
**Clarification Rounds**: 1
**Quality Score**: 97/100
