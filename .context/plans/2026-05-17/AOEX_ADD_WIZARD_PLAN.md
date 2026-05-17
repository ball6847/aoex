---
createdAt: "2026-05-17T10:00:00Z"
implementedAt: null
reviewedAt: null
---

# Plan: aoex add Interactive Wizard

## Overview

Implement `aoex add`, an interactive CLI wizard built with `charmbracelet/bubbletea` and `cobra`. The wizard prompts the user for the most common `aoe add` flags (title, group, command, launch, worktree, sandbox), auto-discovers defaults from the environment, and shells out to the `aoe` binary after a review & confirm step.

## Target Structure

```
aoex/
├── cmd/
│   ├── root.go          # cobra root command + aoex version
│   └── add.go           # `aoex add` subcommand: orchestrates detector → prompt → executor
├── internal/
│   ├── prompts/         # bubbletea TUI models & styles
│   │   └── wizard.go    # Multi-step wizard Model (path→title→group→agent→launch→worktree→sandbox→review)
│   ├── detector/        # Environment discovery
│   │   ├── detector.go  # Context struct + top-level DetectAll()
│   │   ├── git.go       # Git branch detection (git branch --show-current)
│   │   └── agents.go    # Agent binary detection (lookPath for claude, codex, opencode, etc.)
│   └── executor/        # Shell execution
│       └── executor.go  # Build and run `aoe add . [flags]`
├── main.go              # Cobra entry point
└── go.mod               # cobra, bubbletea deps
```

## Files to Create

### `cmd/root.go`
- **Purpose**: Root command definition using `cobra`
- **Requirements**:
  - Command name: `aoex`
  - Long description: "agent-of-empires CLI extension"
  - Subcommands: `add`
  - No direct logic; delegates to subcommands

### `cmd/add.go`
- **Purpose**: `aoex add` subcommand implementation — the main orchestrator
- **Requirements**:
  - Define `cobra.Command` named `add` with `RunE: runAdd`
  - `runAdd` signature: `func(cmd *cobra.Command, args []string) error`
  - Implementation flow:
    1. **Pre-flight**: verify `aoe` is on `$PATH` via `exec.LookPath("aoe")`; if missing return `fmt.Errorf("aoe binary not found in PATH. Please install agent-of-empires first")` and exit with non-zero status **before any prompts**
    2. Call `detector.DetectAll()` to gather context
    3. Call `prompts.RunWizard(ctx)` to collect user inputs (returns `*executor.Args`)
    4. If `cancelled` or no confirmation in review step → exit `0` with no side effects
    5. Call `executor.Exec(args)` to shell out to `aoe`
    6. Forward stdout/stderr and exit code from `aoe` to the user
  - Errors bubble up with `fmt.Errorf("add: %w", err)`

### `internal/detector/detector.go`
- **Purpose**: Top-level environment detection and the `Context` data model
- **Requirements**:
  - Define exported struct `Context`:
    ```go
    type Context struct {
        CWD          string   // os.Getwd()
        IsGitRepo    bool     // derived from git branch detection
        GitBranch    string   // current branch or ""
        FolderName   string   // filepath.Base(CWD)
        ParentFolder string   // filepath.Base(filepath.Dir(CWD))
        Agents       []string // detected agent binaries (claude, codex, opencode, etc.)
    }
    ```
  - Export `func DetectAll() (*Context, error)` — composites git, cwd, and agent discovery

### `internal/detector/git.go`
- **Purpose**: Git repository and branch detection
- **Requirements**:
  - Signature: `func detectGitBranch(cwd string) (branch string, isRepo bool, err error)`
  - Run `git -C <cwd> branch --show-current`
  - If git exits with 128 (not a repo) → return `("", false, nil)`
  - If git not installed → return `("", false, nil)` (graceful degradation)
  - Any other error → return with `fmt.Errorf("failed to detect git branch: %w", err)`

### `internal/detector/agents.go`
- **Purpose**: Detect installed agent binaries on `$PATH`
- **Requirements**:
  - Signature: `func detectAgents() ([]string, error)`
  - Use `exec.LookPath()` for each known agent name:
    - `claude`, `codex`, `opencode`
  - Return only names found on `$PATH`; preserve order of discovery
  - If `LookupEnv("PATH")` is empty → return `[]string{}, nil`
  - Error cases bubble up with context

### `internal/executor/executor.go`
- **Purpose**: Construct and execute the final `aoe add` shell command
- **Requirements**:
  - Define exported struct `Args`:
    ```go
    type Args struct {
        Path    string // positional arg, always "." for now
        Title   string // --title
        Group   string // --group
        Cmd     string // --cmd
        Launch  bool   // --launch
        Worktree string // --worktree (omitted if empty)
        Sandbox bool   // --sandbox
    }
    ```
  - Export `func Exec(args *Args) error`
    - Build `[]string` of flags using `exec.Command`
    - Pass `-C` to set working directory if Path != "." (future-proofing)
    - Forward stdout, stderr, and exit code
    - If `aoe` exits non-zero → return error containing its stderr

### `internal/prompts/wizard.go`
- **Purpose**: Bubbletea Model implementing the 8-step wizard
- **Requirements**:
  - Export `func RunWizard(ctx *detector.Context) (*executor.Args, error)`
    - Initializes `tea.NewProgram` with the wizard Model
    - Returns `*executor.Args` when user confirms, or `(nil, nil)` if user cancels
  - Define internal `wizardModel` struct implementing `tea.Model`:
    - Fields: `step int`, `ctx *detector.Context`, `answers executor.Args`
    - Steps (use constants or enums):
      0. Path (default: `ctx.CWD`)
      1. Title (default: `ctx.FolderName`; re-prompt if empty)
      2. Group (default: `ctx.ParentFolder`; re-prompt if empty)
      3. Agent (list: `ctx.Agents` + "Other (type manually)"; if "Other" → text input)
      4. Launch (yes/no toggle, default: `false`)
      5. Worktree (pre-filled: `ctx.GitBranch` if `ctx.IsGitRepo`; otherwise skip)
      6. Sandbox (yes/no toggle, default: `false`)
      7. Review (display computed `aoe add . ...` string; y/Enter to confirm, n/Ctrl+C to cancel)
    - `Init() tea.Cmd`
    - `Update(msg tea.Msg) (tea.Model, tea.Cmd)` — handles `tea.KeyMsg` for navigation (Tab, Enter, Ctrl+C, arrow keys)
    - `View() string` — renders current step with instructions
  - Validation rules:
    - Title, Group: required — if empty after user clears, stay on step with error message and redisplay prompt
    - Agent: if "Other (type manually)" selected, show text input; validate non-empty before proceeding
  - `Ctrl+C` or selecting "No" on Review → signal cancellation; return `(nil, nil)` and exit code 0 upstream
  - Use `charmbracelet/bubbles` components: `textinput`, `list`, `confirm`
  - Use `charmbracelet/lipgloss` for minimal styling (optional)

### `main.go`
- **Purpose**: Application entry point
- **Requirements**:
  - `package main`
  - `func main()` calls `cmd.Execute()`
  - Minimal; zero logic beyond delegation

### `go.mod` / dependency management
- **Purpose**: Declare Go module and external dependencies
- **Requirements**:
  - Module path: `github.com/ball6847/aoex`
  - Dependencies: `spf13/cobra`, `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`
  - Go version: latest stable (`1.22+`)

## Files to Modify

- **AGENTS.md** (optional): Update tech stack section to remove undecided alternatives (already done).

## Files to Delete

None.

## Diagrams

### Sequence Diagram

```
User    Terminal   Prompts    Detector   Executor   aoe binary
  |          |          |          |          |          |
   |--- run "aoex add" ------------------------------------>|
   |          |-- lookPath("aoe") |          |          |
   |          |-- fail if missing |          |          |
   |          |----------|          |          |          |
   |          |     DetectAll()    |          |          |
   |          |<---------|          |          |          |
  |          |          |          |-- os.Getwd()        |
  |          |          |          |-- git branch        |
  |          |          |          |-- lookPath agents   |
  |          |          |<---------|          |          |
  |     RunWizard(ctx) |          |          |          |
  |<---------|          |          |          |          |
  | [interactive prompts for title, group, agent, etc.]    |
  |-- input ->|          |          |          |          |
  |          |-- validate          |          |          |
  |<---------| (re-prompt if empty)|          |          |
  |-- input ->|          |          |          |          |
  |          | [Review & Confirm screen]                   |
  |-- Y/Enter->          |          |          |          |
  |          |------ *Args ------->|          |          |
  |          |          |          |     Exec(args)       |
  |          |          |          |          |-- "aoe add . --title ..." -->
  |          |          |          |          |          |
  |          |          |          |          |<-- result |
  |          |          |          |          | [forward stdout/stderr] -->
```

### State Transition Diagram

```
   [Idle]
     |
     v
[Check aoe on PATH] --not found--> [Error: install aoe (exit 1)]
     |found
     v
 [DetectAll()] --error--> [Error: detection failed (exit 1)]
     |success
     v
 [Path Step] --Enter--> [Title Step]
     |                    ^
     | empty              | re-prompt if empty
     |                    |
     v                    v
 [Group Step] <--------- [Validate]
     |
     v
 [Agent Step] --"Other"--> [Custom Input]
     |                          |
     | selected                 | valid
     v                          v
 [Launch Step] <-------------- [Validate]
     |
     v
 [Worktree Step] ----skip if not git repo----> [Sandbox Step]
     |                                                    |
     v                                                    v
 [Sandbox Step] --git repo--> [Review Step]
     |
     v
 [Review Step] --Y/Enter--> [Exec aoe] --> [Done]
     |
     | No / Ctrl+C
     v
 [Cancelled] --> [Exit 0, no side effects]
```

## Test Cases

### TC-01: Full Wizard Execution with Defaults

**Priority:** P0
**Type:** Integration

#### Objective

Verify the complete happy path: running `aoex add` and accepting all defaults successfully invokes `aoe add` with correct computed arguments.

#### Preconditions

- `aoe` binary exists on `$PATH`
- Current directory is a git repository on branch `feature/hello`
- Directory name is `myproject`, parent is `work`
- At least one agent (e.g., `codex`) is on `$PATH`

#### Test Steps

1. Run `aoex add` in the directory.
2. At Path prompt, press Enter to accept default.
3. At Title prompt, press Enter to accept default (`myproject`).
4. At Group prompt, press Enter to accept default (`work`).
5. At Agent prompt, select detected `codex` and press Enter.
6. At Launch prompt, press Enter to accept default (`No`).
7. At Worktree prompt, press Enter to accept default (`feature/hello`).
8. At Sandbox prompt, press Enter to accept default (`No`).
9. At Review prompt, press `Y` then Enter to confirm.

**Expected:** The exact command `aoe add . --title "myproject" --group "work" --cmd "codex" --worktree "feature/hello"` is executed.

#### Post-conditions

- `aoe add` process runs and its stdout/stderr is visible in the terminal.

### TC-02: Cancellation at Review Step

**Priority:** P0
**Type:** Functional

#### Objective

Verify that declining at the Review step exits cleanly with code 0 and does NOT invoke `aoe`.

#### Preconditions

- `aoe` binary exists on `$PATH`

#### Test Steps

1. Run `aoex add`.
2. Proceed through all prompts accepting defaults.
3. At Review prompt, press `n` then Enter to decline.

**Expected:** Application prints a cancellation message (e.g., "Cancelled. No command executed.") and exits with status `0`.

#### Post-conditions

- No `aoe` process is spawned.

### TC-03: Missing `aoe` Binary on PATH

**Priority:** P0
**Type:** Functional

#### Objective

Verify that `aoex add` detects a missing `aoe` binary before starting the wizard and errors out gracefully.

#### Preconditions

- No `aoe` binary on `$PATH`

#### Test Steps

1. Run `aoex add`.

**Expected:** Application immediately prints an error: "aoe binary not found in PATH. Please install agent-of-empires first." and exits with non-zero status.

#### Post-conditions

- No wizard prompts are displayed.

### TC-04: Empty Title Validation (Re-prompt)

**Priority:** P1
**Type:** Functional

#### Objective

Verify that clearing the default title and submitting an empty string causes a re-prompt with an error message.

#### Preconditions

- `aoe` binary exists on `$PATH`

#### Test Steps

1. Run `aoex add`.
2. At Path prompt, press Enter.
3. At Title prompt, clear the default text and press Enter.

**Expected:** Application stays on the Title step and displays an error message (e.g., "Title cannot be empty").

### TC-05: Non-Git Repository (Worktree Skip)

**Priority:** P1
**Type:** Functional

#### Objective

Verify that the Worktree prompt is skipped if the current directory is not a git repository.

#### Preconditions

- `aoe` binary exists on `$PATH`
- Current directory is NOT a git repository

#### Test Steps

1. Run `aoex add` in a non-git directory.
2. Proceed through Path, Title, Group, Agent, Launch prompts accepting defaults.

**Expected:** The wizard skips the Worktree prompt entirely and proceeds directly to the Sandbox prompt.

#### Post-conditions

- The `--worktree` flag is omitted from the final `aoe add` command.

### TC-06: Manual Agent Entry ("Other" Option)

**Priority:** P1
**Type:** Functional

#### Objective

Verify selecting "Other" allows free-text entry of a custom agent command.

#### Preconditions

- `aoe` binary exists on `$PATH`

#### Test Steps

1. Run `aoex add`.
2. At Agent prompt, select "Other (type manually)" and press Enter.
3. Type a custom command (e.g., `my-custom-agent`) and press Enter.
4. Proceed through remaining prompts and confirm.

**Expected:** The final command uses `--cmd "my-custom-agent"`.

### TC-07: User Cancels with Ctrl+C Mid-Wizard

**Priority:** P2
**Type:** Functional

#### Objective

Verify that pressing `Ctrl+C` at any wizard step exits cleanly with code 0.

#### Preconditions

- `aoe` binary exists on `$PATH`

#### Test Steps

1. Run `aoex add`.
2. At any prompt (e.g., Group step), press `Ctrl+C`.

**Expected:** Application exits immediately with status `0` and no side effects.

## Verification Commands

```bash
# Build the binary
go build -o aoex .

# Verify formatting and vetting
go fmt ./...
go vet ./...

# Run tests
go test ./...

# Run the wizard manually (requires aoe to be mocked or available)
./aoex add
```

## Expected Outcome

- `go build -o aoex .` produces a single static binary.
- Running `./aoex add` starts an interactive 8-step wizard (Path, Title, Group, Agent, Launch, Worktree, Sandbox, Review) with bubbletea.
- All defaults are pre-filled and derived from `detector.DetectAll()`.
- Empty required fields trigger re-prompts with error messages.
- The review step shows the exact `aoe add` command string.
- Confirming runs the `aoe add` command.
- Cancelling (n/Ctrl+C) exits with code 0 and executes nothing.
- `go vet` and `gofmt` pass without issues.

## Rollback Plan

- Delete all newly created files under `cmd/` and `internal/`.
- Restore `go.mod` to pre-implementation state (or delete and re-run `go mod init`).
- `AGENTS.md` changes are additive; if needed, revert to the version from the previous commit.
