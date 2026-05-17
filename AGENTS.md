# AGENTS.md

## Overview

`aoex` (agent-of-empires extension) is a Go CLI that wraps the original `aoe` (agent-of-empires) tool with a simpler, interactive interface.

## Tech Stack

- **Language**: Go (latest stable)
- **CLI Framework**: `cobra` or standard `flag` + `bufio` for interactive prompts
- **Prompt Library**: `github.com/AlecAivazis/survey/v2` or `github.com/charmbracelet/bubbletea` for wizard UX
- **Distribution**: Single static binary via `go build`

## Architecture

```
aoex/
├── cmd/
│   └── add.go          # `aoex add` wizard implementation
├── internal/
│   ├── prompts/        # Interactive prompt definitions
│   ├── detector/       # Auto-discovery: git branch, installed agents, current dir
│   └── executor/       # Shell out to `aoe add . <computed-args>`
├── main.go
└── go.mod
```

## Build & Run

```bash
# Development
go run .

# Build binary (development)
go build -o aoex .

# Build release binary (optimized)
make build-release

# Install to $GOPATH/bin
go install .
```

## Development Commands (Makefile)

| Target                | Description                                   |
| --------------------- | --------------------------------------------- |
| `make` or `make help` | Show all available targets                    |
| `make fmt`            | Format code with `gofmt` + `goimports`        |
| `make vet`            | Run `go vet`                                  |
| `make lint`           | Run `golangci-lint` (v2)                      |
| `make lint-fix`       | Run `golangci-lint` with auto-fix             |
| `make test`           | Run all tests                                 |
| `make test-race`      | Run tests with race detector                  |
| `make coverage`       | Generate HTML coverage report                 |
| `make build`          | Build the `aoex` binary                       |
| `make build-release`  | Build optimized release binary                |
| `make install`        | Install to `$GOPATH/bin`                      |
| `make clean`          | Remove build artifacts                        |
| `make ci` | Run full CI pipeline: fmt + vet + lint + test |
| `make deps` | Download and tidy Go module dependencies |

## Integration with `agent-of-empires`

The `agent-of-empires/` directory is a **git submodule** containing the upstream `aoe` source code. It is used for reference only.

`aoex` does **not** import `agent-of-empires` as a Go module. Instead, it:

1. Discovers context by inspecting the filesystem and running `git` commands directly.
2. At the end of the wizard, shells out to the user's installed `aoe` binary:  
   `aoe add . --title "..." --group "..." --cmd "..." [flags]`

## Coding Style

- Standard Go formatting (`gofmt`, `go vet`)
- Use `cobra` for command routing
- Keep prompt logic in `internal/prompts/` separate from execution logic
- Errors: bubble up with `fmt.Errorf("...: %w", err)`

## Development Workflow

### Test-Driven Development (TDD)

Write tests first to define behavior and edge cases, then write implementation to make them pass. To save tokens, **batch test cases into a single table-driven test per package** before switching to implementation — avoid cycling test → impl → test → impl for every micro-feature.

| Step | Action                        | Token Saving Tip               |
| ---- | ----------------------------- | ------------------------------ |
| 1    | Plan & clarify requirements   | Batch concerns into 1–2 rounds |
| 2    | Write all tests for a package | Use table-driven tests         |
| 3    | Run tests → expect failures   |                                |
| 4    | Write implementation to pass  |                                |
| 5    | Run tests → verify green      |                                |
| 6    | Refactor if needed            |                                |

### Skill Activation by Phase

| Phase      | When to Activate                       | Relevant Skills                                                                                                      |
| ---------- | -------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| **Plan**   | New feature or unclear requirements    | `requirements-clarity`, `planner`                                                                                    |
| **Build**  | Writing tests or implementation        | `builder`, `golang-testing`, plus task-specific `golang-*` skills                                                    |
| **Review** | After implementation, before finishing | `reviewer`, `complexity-optimizer`, `golang-modernize`, `golang-code-style`, `golang-performance`, `golang-security` |

- **Plan**: Use `problem-statement` or `epic-hypothesis` when framing larger initiatives. Only use skills that require user interaction (e.g., `question` tool) in this phase.
- **Build**: Load `golang-*` skills as needed (e.g., `golang-error-handling` for error flow, `golang-security` for auth). Avoid loading all at once.
- **Review**: Always verify implementation against the plan.

## Skill Rules

### requirement-clarity

- **ALWAYS** use the `question` tool to ask clarifying questions.
- Provide **2–4 multiple-choice answers** with a **“(Recommended)”** label on the top choice.
- **Always include a final “Type your own answer”** option.

### planner

- After finishing the plan, **re-activate `requirements-clarity`** to validate alignment with original requirements.
- Address gaps before implementation.
