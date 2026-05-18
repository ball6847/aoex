---
createdAt: "2026-05-18T22:14:00Z"
implementedAt: "2026-05-18T22:18:00Z"
reviewedAt: null
---

# Plan: Fix Branch/Worktree Selection in aoex add Wizard

## Overview

Fixes the `aoex add` step 2 (branch/worktree selection) to properly handle three scenarios: (1) attaching to existing worktrees by passing the resolved path directly without `-w` flag, (2) creating new worktrees from existing branches using `--worktree` flag only, and (3) creating new branches with new worktrees using `--worktree` + `-b` flags. Currently, selecting a branch with an existing worktree incorrectly passes both the path and `--worktree` flag, causing `aoe` to fail.

## Target Structure

No new files or directory structure changes. Modifications limited to existing files.

## Files to Modify

### 1. internal/prompts/wizard.go

**Location**: Lines 336-346 in the `stepBranch` case of `Update()` method

**Required Changes**:
- When a worktree exists for the selected branch: set `answers.Worktree = ""` to prevent `--worktree` flag from being added
- When a worktree does NOT exist but branch exists: keep `answers.Worktree = q` and `answers.NewBranch = false` (creates worktree from existing branch)
- When branch does NOT exist: keep `answers.Worktree = q` and `answers.NewBranch = true` (creates branch + worktree)
- Ensure `answers.Path` is set correctly in all scenarios

**Behavior Matrix**:

| Branch Exists | Worktree Exists | Worktree Field | NewBranch | Path Field | aoe Command Pattern |
|--------------|-----------------|----------------|-----------|------------|---------------------|
| Yes | Yes | `""` | `false` | `wkPath` | `aoe add /path/to/wt --title X --cmd Y` |
| Yes | No | `q` | `false` | `.` | `aoe add . --title X --cmd Y --worktree X` |
| No | N/A | `q` | `true` | `.` | `aoe add . --title X --cmd Y --worktree X -b` |

**Reference**: Follow existing pattern in `wizard.go` lines 315-346 for the `stepBranch` case.

### 2. internal/executor/executor.go

**Location**: Lines 27-28 (comment) and lines 42-44 (flag construction)

**Required Changes**:
- Update comment on lines 27-28 to accurately reflect the new behavior
- No code changes needed - the existing logic already handles empty `Worktree` correctly by skipping the `--worktree` flag

**Reference**: Existing logic at lines 42-44 already checks `if args.Worktree != ""` before appending flag.

## Files to Create

None. All changes are modifications to existing files.

## Files to Delete

None.

## Diagrams

### State Transition Diagram

```
[User Selects Branch]
    |
    v
[Branch Exists?] --no--> [New Branch] --create--> [New Branch + New Worktree]
    |                                    (Worktree="q", NewBranch=true)
    yes
    |
    v
[Worktree Exists?] --yes--> [Attach to Existing Worktree]
                     |             (Worktree="", NewBranch=false, Path=wkPath)
                     no
                     |
                     v
             [Create Worktree from Existing Branch]
                     (Worktree="q", NewBranch=false, Path=".")
```

### Sequence Diagram

```
User    wizard.go          executor.go          aoe
  |         |                 |                 |
  |--select branch -->|         |                 |
  |         |-- check worktrees |                 |
  |         |                 |                 |
  |         |-- set Args -->|                 |
  |         |                 |-- build cmd -->|
  |         |                 |                 |-- aoe add [correct flags]
```

## Test Cases

### TC-001: Attach to Existing Worktree

**Priority:** P0
**Type:** Functional

#### Objective
Verify that selecting a branch with an existing worktree passes the resolved path without `--worktree` flag.

#### Preconditions
- Git repository with at least one worktree created via `git worktree add`
- Worktree mapped to a branch (e.g., `feature/existing` at `/path/to/worktree`)
- `aoex` built with changes

#### Test Steps
1. Run `aoex add` in a terminal
2. Complete step 1 (select agent)
3. In step 2, select the branch that has an existing worktree (e.g., `feature/existing`)
4. Complete remaining steps (sandbox, review)
5. Confirm the command

**Expected:** 
- Executed command is `aoe add /path/to/worktree --title "feature/existing" --cmd "<agent>"`
- NO `--worktree` flag present
- NO `-b` flag present

#### Post-conditions
- `aoe` receives the worktree path as positional argument
- No worktree-related flags passed

---

### TC-002: Create Worktree from Existing Branch

**Priority:** P0
**Type:** Functional

#### Objective
Verify that selecting an existing branch without a worktree creates a new worktree.

#### Preconditions
- Git repository with a branch that has NO worktree (e.g., `feature/no-wt`)
- `aoex` built with changes

#### Test Steps
1. Run `aoex add`
2. Complete step 1 (select agent)
3. In step 2, select the existing branch without worktree (e.g., `feature/no-wt`)
4. Complete remaining steps
5. Confirm the command

**Expected:** 
- Executed command is `aoe add . --title "feature/no-wt" --cmd "<agent>" --worktree "feature/no-wt"`
- `--worktree` flag present with branch name
- NO `-b` flag present

#### Post-conditions
- `aoe` receives `--worktree` flag to create worktree from existing branch

---

### TC-003: Create New Branch with Worktree

**Priority:** P0
**Type:** Functional

#### Objective
Verify that typing a new branch name creates both branch and worktree.

#### Preconditions
- Git repository
- Branch name that does NOT exist (e.g., `feature/new-branch`)
- `aoex` built with changes

#### Test Steps
1. Run `aoex add`
2. Complete step 1 (select agent)
3. In step 2, type a new branch name (e.g., `feature/new-branch`)
4. Complete remaining steps
5. Confirm the command

**Expected:** 
- Executed command is `aoe add . --title "feature/new-branch" --cmd "<agent>" --worktree "feature/new-branch" -b`
- Both `--worktree` and `-b` flags present

#### Post-conditions
- `aoe` receives flags to create new branch and new worktree

---

### TC-004: Current Worktree Detection in UI

**Priority:** P1
**Type:** UI

#### Objective
Verify UI correctly displays worktree status indicators.

#### Preconditions
- Git repository with mixed branches (some with worktrees, some without)

#### Test Steps
1. Run `aoex add`
2. Navigate to step 2 (branch selection)
3. Observe the branch list

**Expected:** 
- Branches with existing worktrees show `"worktree exists [will attach]"` description
- Branches without worktrees show no special description
- Current branch (if any) appears first in list

#### Post-conditions
- UI provides clear visual feedback about worktree status

---

### TC-005: Existing Tests Still Pass

**Priority:** P0
**Type:** Regression

#### Objective
Verify no regressions in existing test suite.

#### Preconditions
- All existing test files in `internal/executor/` and `internal/prompts/`

#### Test Steps
1. Run `go test ./...`

**Expected:** 
- All existing tests pass
- No test failures or errors

#### Post-conditions
- Existing functionality remains intact

## Verification Commands

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/prompts/... -v
go test ./internal/executor/... -v
go test ./internal/detector/... -v

# Build and test manually
go build -o aoex .
./aoex add
```

## Expected Outcome

- `aoex add` correctly handles all three branch/worktree scenarios
- No `--worktree` flag passed when attaching to existing worktrees
- `--worktree` flag passed (without `-b`) for existing branches without worktrees
- `--worktree` + `-b` flags passed for new branches
- All existing tests pass
- UI correctly indicates worktree status

## Rollback Plan

If issues arise:
1. Revert changes to `internal/prompts/wizard.go`
2. Changes to `internal/executor/executor.go` are comment-only, can be reverted or left as-is
3. Git commands:
   ```bash
   git checkout HEAD -- internal/prompts/wizard.go
   git checkout HEAD -- internal/executor/executor.go
   ```

## Implementation

- **Status**: Completed
- **Changes Made**:
  - Modified `internal/prompts/wizard.go` (lines 337-347): Added logic to clear `Worktree` field when attaching to existing worktree, and properly set `Worktree` and `NewBranch` for all three scenarios
  - Updated comments in `internal/executor/executor.go` (lines 17, 27-29): Clarified behavior for worktree/branch handling
  - Added test cases in `internal/executor/executor_test.go`: Three new test cases covering all scenarios
- **Tests**: All tests pass (`go test ./...`)
- **Build**: Successfully builds (`go build -o aoex .`)
