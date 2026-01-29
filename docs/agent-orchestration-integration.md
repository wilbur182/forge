# Native Agent Orchestration in Sidecar

Spec for agent orchestration in sidecar, backed by td as the task engine.

## Problem Statement

Single-agent coding sessions degrade over long contexts. The model accumulates thousands of tokens of history and optimizes for "done" over "correct." There's no independent check on the work, no structured audit trail, and no way to recover if the agent goes off-track.

Current multi-agent orchestrators share several problems:

1. **Opaque execution** - Agent work happens in subprocesses with limited visibility. The user watches terminal output scroll by without structured insight into progress, decisions, or blockers.

2. **Over-prescriptive prompts** - Agent instructions run 500+ lines of rigid rules that constrain model reasoning. Models know how to write code. They don't need to be told "NEVER use git diff" or "Maximum informativeness, minimum verbosity. NO EXCEPTIONS."

3. **Model selection as core complexity** - Multi-tier complexity classifiers mapped to model levels across multiple providers add configuration surface area that most users don't need.

4. **Separate tooling** - Orchestration runs outside the developer's primary workflow, with no integration into git status, file browsing, or task management.

5. **Duplicate state systems** - Orchestrators maintain their own SQLite ledgers for state, duplicating what a task engine like td already provides.

## Design Principles

### Plan-Build-Review Cycle
The core loop: plan the work, implement it, independently validate it. Isolated agents checking each other's work produces better code than single-agent runs where context degradation is real.

### Blind Validation
Validators never see the implementer's context or justifications. They evaluate work on merit alone. This prevents the "rubber stamp" problem where reviewers unconsciously accept work because the reasoning sounds plausible.

### Task-Driven Execution
Every run starts from a task with acceptance criteria. The system knows what "done" means before work begins. This maps directly to td's issue model.

### Rejection Loop
When validators reject work, findings route back to the implementer with specific, actionable feedback. The loop continues until consensus or explicit failure. This is fundamentally different from "run once and hope."

### Workspace Isolation
Git worktrees prevent agent work from contaminating the main branch. Sidecar's workspace plugin already manages worktrees - this is a natural integration point.

### Task-ID-Driven Prompts
The orchestrator passes a task ID and brief td commands to agents. It does **not** inject task content into prompts. Agents read their own context from td. This keeps prompts small, avoids stale context, and trains agents to treat the task engine as their source of truth.

### Minimal Prompt Philosophy
Use minimal role descriptions and let the model's native capabilities drive behavior. The prompt describes *what* the agent needs to accomplish, not *how* to think.

Example agent prompt:
```
You are implementing task td-a1b2.

View the task and its full context:
  td show td-a1b2
  td context td-a1b2

Start a work session and log your progress:
  td usage --new-session
  td log "your progress here"

Work in the worktree at /path/to/worktree. Commit when done.
```

The orchestrator does **not** pass task titles, descriptions, acceptance criteria, or any other task content in the prompt. It passes a task ID and brief td commands. The agent reads the task itself using td. This is a deliberate design choice with several benefits:

1. **Agents learn td** - Every agent interaction reinforces td as the source of truth. Agents that use td commands to read context, log progress, and record decisions produce better audit trails than agents that receive pre-digested context.

2. **No stale context** - If a prior agent updated the task (added logs, comments, or a handoff), the next agent reads the live state, not a snapshot the orchestrator captured earlier.

3. **Task engine is pluggable** - The orchestrator doesn't parse td output or understand td's data model. It just knows "pass the task ID and these commands." Swapping td for Jira, Linear, or a custom backend means changing the command templates, not the orchestration logic.

4. **Smaller prompts** - Task context can be large (descriptions, logs, handoffs, comments, file lists). Letting the agent fetch what it needs keeps the initial prompt minimal. The agent's tool use fetches exactly what's relevant.

The model knows how to write code. Tell it where to find its assignment, not what it says.

### Provider-Agnostic by Default
Start with a single configured provider (the user's preferred CLI agent) and add multi-provider support later if needed. Most users use one provider.

### Transparent Execution
Sidecar's TUI shows agent work in real-time: which files are being modified, what the plan is, validation progress, rejection reasons. Orchestration is not a black box.

### td as the Native Task Engine
td is the state store. Tasks, logs, handoffs, sessions, and reviews all use td's existing infrastructure. No parallel state system.

### Agents Use td Directly
Agents are told a task ID and given td commands to run. They read their own context, log their own progress, and record their own decisions. The orchestrator coordinates the lifecycle (which agent runs when, in what worktree, with what role) but does not mediate the task content.

## Architecture

### Component Overview

```
                    ┌─────────────────────────────────┐
                    │          Sidecar TUI             │
                    │                                  │
                    │  ┌───────┐ ┌──────┐ ┌────────┐  │
                    │  │Git    │ │Files │ │TD      │  │
                    │  │Status │ │      │ │Monitor │  │
                    │  └───────┘ └──────┘ └────────┘  │
                    │  ┌──────────────────────────┐   │
                    │  │   Agent Orchestrator      │   │
                    │  │   Plugin (new)            │   │
                    │  └────────────┬─────────────┘   │
                    └───────────────┼──────────────────┘
                                    │
                    ┌───────────────┼──────────────────┐
                    │               ▼                   │
                    │     Orchestration Engine          │
                    │     (internal/orchestrator/)      │
                    │                                   │
                    │  ┌─────────┐  ┌──────────────┐   │
                    │  │Planner  │  │Implementer   │   │
                    │  │Agent    │  │Agent         │   │
                    │  └────┬────┘  └──────┬───────┘   │
                    │       │              │            │
                    │  ┌────▼──────────────▼───────┐   │
                    │  │   Validator Agents         │   │
                    │  │   (1-N, blind, parallel)   │   │
                    │  └───────────────────────────┘   │
                    │                                   │
                    │  ┌───────────────────────────┐   │
                    │  │   Agent Runner             │   │
                    │  │   (shells out to CLI)      │   │
                    │  └───────────────────────────┘   │
                    └───────────────────────────────────┘
                                    │
                    ┌───────────────┼──────────────────┐
                    │               ▼                   │
                    │      td (task engine)             │
                    │  tasks, logs, handoffs, sessions  │
                    └──────────────────────────────────┘
```

### Core Packages

#### `internal/orchestrator/`

The orchestration engine, independent of the TUI. Can be tested and run standalone.

```
internal/orchestrator/
  engine.go          # Core lifecycle: plan → build → validate → iterate
  agent.go           # Agent abstraction (role, prompt builder, runner)
  runner.go          # Shells out to CLI agents (claude, codex, gemini)
  planner.go         # Planning phase logic
  validator.go       # Validation phase logic (blind, parallel)
  workspace.go       # Worktree/isolation management
  events.go          # Event types emitted during orchestration
  config.go          # Orchestration settings
  taskengine.go      # TaskEngineAdapter interface
  taskengine_td.go   # td implementation of TaskEngineAdapter
```

#### `internal/plugins/orchestrator/`

The sidecar plugin that provides the TUI for orchestration.

```
internal/plugins/orchestrator/
  plugin.go          # Plugin interface implementation
  view.go            # Rendering (plan view, progress, validation results)
  handlers.go        # Key/mouse input handling
  commands.go        # Plugin commands for footer hints
```

### Orchestration Engine Design

#### Engine Lifecycle

```go
type Engine struct {
    taskID     string            // task ID (e.g. "td-a1b2")
    workspace  *Workspace        // git worktree or direct
    runner     AgentRunner       // CLI agent executor
    taskEngine TaskEngineAdapter // td (or other task backend)
    events     chan Event        // progress events for TUI
    config     *Config           // orchestration settings
}

type Config struct {
    Provider       string         // "claude", "codex", "gemini"
    MaxIterations  int            // rejection loop limit (default: 3)
    ValidatorCount int            // number of validators (default: 2)
    Workspace      string         // "worktree" (default), "direct", "docker"
    AutoMerge      bool           // auto-merge worktree on success
    AgentTimeout   time.Duration  // kill agent if no output (default: 10m)
}
```

#### Phases

**Core principle: task ID in, td commands in, everything else the agent reads itself.**

The orchestrator never reads task content from td and injects it into prompts. It passes a task ID and short instructions for how the agent should orient itself using the task engine. This keeps prompts small, avoids stale context, and trains agents to treat td as their source of truth.

**Phase 1: Plan**

The planner agent receives a minimal prompt:

```
You are planning the implementation for task td-a1b2.

Read the task and its full context:
  td show td-a1b2
  td context td-a1b2

Start a work session and log your progress:
  td usage --new-session
  td log --decision "your plan here"

Produce a plan: which files to modify, the approach, and any risks.
Log your plan as a decision in td when done.
```

The orchestrator does **not** read the task and inject its content. The planner agent runs `td show` and `td context` itself, reads acceptance criteria, prior logs, handoffs, and comments, then produces a plan.

The plan is captured from the agent's output (or read from td logs) and displayed in the TUI. The user can accept, modify, or reject before implementation begins.

```go
type Plan struct {
    Summary     string
    Steps       []string
    FilesTouch  []string
    Risks       []string
    Accepted    bool      // user can review before proceeding
}
```

**Phase 2: Implement**

The implementer agent works in an isolated worktree. It receives:

```
You are implementing task td-a1b2.

Read the task and plan:
  td show td-a1b2
  td context td-a1b2

Log progress as you work:
  td usage --new-session
  td log "what you did"

Work in the worktree at /path/to/worktree. Commit when done.
```

The implementer reads the task and the planner's logged decisions from td itself. It does not receive: validator instructions, previous rejection details from other tasks, or prescriptive coding rules.

Progress events stream to the TUI:
- Files being modified
- Commits made
- Agent thinking/reasoning (if available from provider)

**Phase 3: Validate**

N validator agents run in parallel. Each receives:

```
You are reviewing the implementation of task td-a1b2.

Read the task requirements:
  td show td-a1b2

The implementation diff is in the worktree at /path/to/worktree.
Review the diff against the task's acceptance criteria.

Log your review:
  td usage --new-session
  td log "your findings"

Approve or reject with specific findings.
```

Validators are **blind**: they read the task requirements from td and review the diff, but do not see the implementer's logs, reasoning, or session history. The orchestrator ensures validators get a td session that only exposes the task definition, not implementation logs. (This can be achieved by passing a filtered `td show` rather than `td context`, or by using td's query capabilities to scope what's visible.)

Each validator independently assesses:
- Does the implementation satisfy acceptance criteria?
- Are there bugs, security issues, or missing edge cases?
- Do tests pass? (if the validator can run them)

Validators return structured results:

```go
type ValidationResult struct {
    Approved bool
    Findings []Finding
}

type Finding struct {
    Severity string  // "error", "warning", "info"
    File     string
    Line     int
    Message  string
}
```

**Phase 4: Iterate or Complete**

If all validators approve: mark complete, optionally merge worktree, update td task status.

If any validator rejects: the orchestrator logs findings to td as comments or blocker logs, then launches a fresh implementer with:

```
You are fixing issues found during review of task td-a1b2.

Read the task and review feedback:
  td show td-a1b2
  td context td-a1b2

The review comments describe what needs to be fixed.
Work in the worktree at /path/to/worktree. Commit when done.

Log progress:
  td usage --new-session
  td log "what you fixed"
```

The implementer reads the rejection findings from td (where they were logged as comments/blockers), not from an inline prompt blob. Loop back to Phase 3.

After `MaxIterations` rejections: stop, report failure, log to td with details.

#### Agent Runner

The runner shells out to CLI agents:

```go
type AgentRunner interface {
    Run(ctx context.Context, prompt string, workDir string, env []string) (*AgentResult, error)
    Stream(ctx context.Context, prompt string, workDir string, env []string) (<-chan AgentEvent, error)
}

// ClaudeRunner implements AgentRunner using claude CLI
type ClaudeRunner struct {
    binary string  // path to claude binary
}

// CodexRunner implements AgentRunner using codex CLI
type CodexRunner struct {
    binary string
}
```

Each runner:
- Spawns the CLI process with a minimal prompt (task ID + td commands)
- Sets `TD_SESSION` env var so the agent gets its own td session
- Sets working directory to the worktree
- Captures stdout/stderr
- Optionally streams events (for real-time TUI updates)
- Returns structured output or raw text

The prompt is intentionally small. The agent's first actions will be running td commands to read its assignment. This means the agent needs tool access to run shell commands (which CLI agents like Claude Code and Codex already have).

No model level abstraction. The user configures their CLI agent with whatever model they want. The orchestrator doesn't care.

#### Event System

The engine emits events consumed by the TUI plugin:

```go
type Event struct {
    Type      EventType
    Timestamp time.Time
    Data      interface{}
}

type EventType int
const (
    EventPlanStarted EventType = iota
    EventPlanReady
    EventImplementationStarted
    EventFileModified
    EventImplementationDone
    EventValidationStarted
    EventValidatorResult
    EventIterationStarted
    EventComplete
    EventFailed
)
```

#### Heartbeat & Stuck Agent Detection

CLI agents can hang, crash, or loop indefinitely. The orchestrator monitors agent liveness by watching for output activity:

- The agent runner tracks the timestamp of the last stdout/stderr output from the agent process.
- Every `AgentTimeout` (default 10 minutes), the engine checks if the agent has produced any output.
- If no output for `AgentTimeout`, the engine kills the process and logs the timeout to td.

This is simpler than a database heartbeat column. The runner already captures stdout/stderr - it just needs a timer alongside it.

```go
// In the agent runner's Stream() implementation
select {
case event := <-agentOutput:
    lastActivity = time.Now()
    events <- event
case <-time.After(config.AgentTimeout):
    process.Kill()
    taskEngine.LogBlocker(taskID, fmt.Sprintf(
        "%s agent timed out after %v with no output", role, config.AgentTimeout))
    events <- Event{Type: EventFailed, Data: TimeoutError{Role: role}}
}
```

The TUI shows the time since last agent output in the progress view, so users can see if an agent appears stuck before the timeout fires.

#### Run State & Crash Recovery

Orchestration runs need to survive sidecar restarts. Instead of maintaining a separate state file, the orchestrator stores run state in td itself using structured logs. td is already the single source of truth for task lifecycle - run state is part of that lifecycle.

**How it works:**

The orchestrator logs phase transitions to td with a structured prefix that can be parsed on recovery:

```
td log --type orchestration "phase:plan provider:claude worktree:/path/to/wt validators:2 max_iterations:3"
td log --type orchestration "phase:implement iteration:1"
td log --type orchestration "phase:validate iteration:1 validators_started:2"
td log --type orchestration "phase:validate iteration:1 validator:1 approved:true"
td log --type orchestration "phase:validate iteration:1 validator:2 approved:false"
td log --type orchestration "phase:iterate iteration:2"
td log --type orchestration "phase:complete"
```

On sidecar startup, the orchestrator plugin checks td for any task that is `in_progress` and has orchestration logs but no `phase:complete` or `phase:failed` entry. This means the run was interrupted.

**Recovery logic:**

```go
func (e *Engine) RecoverIfNeeded(taskID string) (*RecoveryState, error) {
    // Read orchestration logs from td
    logs := e.taskEngine.GetOrchestrationLogs(taskID)
    if logs == nil {
        return nil, nil // no active run
    }

    lastPhase := logs.LastPhase()
    switch lastPhase {
    case "plan":
        // Planner was running or plan wasn't accepted yet
        // Offer: resume planning or start over
        return &RecoveryState{Phase: "plan", Action: AskUser}, nil

    case "implement":
        // Implementer was running - worktree may have partial changes
        // Offer: resume implementation (agent reads td context for where it left off)
        // or abandon and start fresh
        return &RecoveryState{
            Phase:      "implement",
            Iteration:  logs.LastIteration(),
            Worktree:   logs.WorktreePath(),
            Action:     AskUser,
        }, nil

    case "validate":
        // Some validators may have finished
        completed := logs.CompletedValidators()
        if len(completed) == logs.ValidatorCount() {
            // All validators finished but orchestrator died before processing results
            // Can resume directly: check results and proceed
            return &RecoveryState{Phase: "validate", Action: AutoResume}, nil
        }
        // Some validators still pending - restart just those
        return &RecoveryState{
            Phase:     "validate",
            Remaining: logs.ValidatorCount() - len(completed),
            Action:    AutoResume,
        }, nil

    case "iterate":
        // Same as implement - worktree has partial changes
        return &RecoveryState{
            Phase:     "iterate",
            Iteration: logs.LastIteration(),
            Worktree:  logs.WorktreePath(),
            Action:    AskUser,
        }, nil
    }
    return nil, nil
}
```

**Why td instead of a local state file:**

1. **Single source of truth.** The task's td logs already capture everything that happened. Adding orchestration phase logs to the same stream means there's one place to look, not two.

2. **Survives worktree deletion.** If the user deletes a worktree or cleans up `.sidecar-*` files, the run state is still in td.

3. **Portable.** If someone resumes a task on a different machine (or a different sidecar instance), the run state travels with the task. The user does need to remember which task to restart, but `td list --status in_progress` shows them.

4. **Auditable.** The orchestration log entries are visible in `td context`, so humans and future agents can see exactly what phases ran and what happened.

**What td needs for this:**

A new log type: `orchestration`. td already supports typed logs (`progress`, `blocker`, `decision`, `hypothesis`, `tried`, `result`, `security`). Adding `orchestration` is a one-line change to the enum. The TaskEngineAdapter encapsulates this:

```go
type TaskEngineAdapter interface {
    // ... existing methods ...

    // Run state (stored as structured logs in the task engine)
    LogOrchestrationEvent(taskID string, event string) error
    GetOrchestrationLogs(taskID string) ([]OrchestrationLog, error)
}
```

For non-td backends (Jira, Linear), orchestration logs could map to:
- Jira: Custom field or structured comment with a `[orchestration]` prefix
- Linear: Comment with metadata
- GitHub Issues: Comment with a parseable format

The adapter parses whatever format the backend uses. The orchestrator only sees typed `OrchestrationLog` structs.

**The one thing a local file does better:** Reading run state from td requires a subprocess call (`td` CLI), which takes ~50ms. A file read takes microseconds. On sidecar startup, checking "are there any interrupted runs?" is slower via td. This is acceptable - it happens once at startup, and `td list --status in_progress` is fast enough. If it becomes a problem, we can add a lightweight cache file that's purely a performance optimization, not a source of truth.

### TUI Plugin Design

The orchestrator plugin integrates with sidecar's existing plugin system.

#### View Modes

1. **Task Selection** - Pick a td task to work on (or create one)
2. **Plan Review** - See the agent's plan, accept/modify/reject
3. **Implementation Progress** - Watch files being modified, see agent output
4. **Validation Results** - See each validator's findings, approval/rejection
5. **Iteration View** - Show rejection feedback being sent back to implementer
6. **Complete/Failed** - Final status with summary

#### Cross-Plugin Integration

The orchestrator plugin leverages sidecar's existing plugins:

- **Git Status**: Shows real-time diff as agent modifies files in the worktree
- **File Browser**: Navigate to files the agent is changing
- **TD Monitor**: Task status updates automatically as orchestration progresses
- **Workspace**: Worktree creation/management for isolated agent work

Messages between plugins:

```go
// Orchestrator → Git Status
gitstatus.RefreshMsg{}

// Orchestrator → File Browser
filebrowser.NavigateToFileMsg{Path: "src/auth/oauth.go"}

// Orchestrator → TD Monitor (via td CLI)
// td log "Plan accepted: implement OAuth with JWT"
// td start td-123

// Orchestrator → Workspace
workspace.CreateWorktreeMsg{Branch: "agent/td-123-oauth"}
```

#### Launch Modal

The primary entry point for orchestration. Designed for one-keypress launch on the happy path while exposing configuration for users who want it.

**Design philosophy**: The workspace create modal is a multi-step form wizard with 6+ focus steps because worktree creation has many independent parameters (name, branch, prompt, task, agent, permissions). Orchestration is different - the task already exists, and most configuration has sensible defaults. The modal should feel more like a confirmation dialog than a form.

**Trigger**: Press `Enter` on a task in the task list, or `r` (run) from anywhere in the orchestrator plugin. Can also be invoked cross-plugin from TD Monitor (e.g., "Run orchestration" action on a task).

**Layout**:

```
┌─ Run Task ────────────────────────────────────────────┐
│                                                        │
│  td-a1b2: Add rate limiting to API endpoints           │
│  P1 · feature · 5pts                                   │
│                                                        │
│  Provider                                              │
│  ▸ Claude Code                                         │
│    Codex                                               │
│    Gemini                                              │
│    OpenCode                                            │
│                                                        │
│  ─────────────────────────────────────────────────     │
│  Iterations: 3    Validators: 2    Workspace: worktree │
│  ─────────────────────────────────────────────────     │
│                                                        │
│             [ Run ]          [ Cancel ]                 │
│                                                        │
│  Enter to run · Tab for options · Esc to cancel        │
└────────────────────────────────────────────────────────┘
```

**Focus steps** (4 total, compared to workspace modal's 8):

| Step | ID | Component | Notes |
|------|----|-----------|-------|
| 0 | `provider-list` | `modal.List` (single-focus) | Provider selection, j/k to navigate |
| 1 | `options-row` | Custom section | Iterations / validators / workspace (inline editable) |
| 2 | `run-btn` | Button | Primary action |
| 3 | `cancel-btn` | Button | Cancel |

**Fast path**: Modal opens with focus on the provider list. If the user's last-used provider is pre-selected (persisted in state.json), pressing `Enter` immediately hits the Run button. One keypress to open, one to confirm. Two keypresses total from task list to running orchestration.

**Task header** (non-interactive): The top of the modal shows the task summary, read from td at modal open time. This is the one place the orchestrator reads task content - purely for display in the modal, never passed to agent prompts. Shows:
- Task ID and title
- Priority, type, and points (single line, muted style)

**Provider list**: Same agent types as the workspace plugin (`AgentTypeOrder`), minus `AgentNone`. Reuses the existing `AgentType` constants and display names. Pre-selects the last provider used (from state.json) or the first available.

```go
// Reuse from workspace plugin
AgentTypeOrder = []AgentType{
    AgentClaude,
    AgentCodex,
    AgentGemini,
    AgentCursor,
    AgentOpenCode,
}
```

Detection: Gray out unavailable providers (binary not found) but still show them. Workspace plugin already has `detectAvailableAgents()` - reuse this.

**Options row** (collapsed by default): Shows current values inline as a read-only summary. When focused (via Tab), each value becomes editable:

- **Iterations**: Number input, default 3, range 1-10. Controls `MaxIterations`.
- **Validators**: Number input, default 2, range 0-5. Zero means no validation (single-agent mode, like TRIVIAL complexity).
- **Workspace**: Cycle through "worktree" / "direct" / "docker". Default "worktree".

Most users never touch these. The collapsed display keeps the modal compact while making configuration accessible.

**Keyboard shortcuts**:

| Key | Action |
|-----|--------|
| `Enter` | Run (from any focus) or confirm selection |
| `Esc` | Cancel |
| `Tab` / `Shift+Tab` | Cycle focus |
| `j` / `k` or `↑` / `↓` | Navigate provider list (focus 0) |
| `←` / `→` | Adjust numeric values in options row (focus 1) |

**Quick-run shortcut**: From the task list view, `Shift+Enter` skips the modal entirely and launches with the last-used provider and default settings. True one-keypress launch for repeat users.

**State on submit**:

```go
type LaunchConfig struct {
    TaskID         string
    Provider       AgentType
    MaxIterations  int
    ValidatorCount int
    Workspace      string   // "worktree", "direct", "docker"
}
```

On submit, the orchestrator:
1. Creates the worktree (if workspace mode is "worktree")
2. Creates td sessions for each agent role
3. Calls `td start <taskID>`
4. Emits `EventPlanStarted`
5. Transitions to Plan Review view mode
6. Persists selected provider to state.json

**Modal construction** (using existing modal library):

```go
p.launchModal = modal.New("Run Task",
    modal.WithWidth(60),
    modal.WithPrimaryAction(launchRunID),
    modal.WithHints(true),
).
    AddSection(p.taskHeaderSection(taskID)).     // Custom: reads td show
    AddSection(modal.Spacer()).
    AddSection(modal.Text("Provider")).
    AddSection(modal.List(launchProviderID,       // Single-focus list
        providerItems,
        modal.WithSelected(lastUsedIdx),
        modal.WithSingleFocus(true),
    )).
    AddSection(modal.Spacer()).
    AddSection(p.optionsRowSection()).            // Custom: inline config
    AddSection(modal.Spacer()).
    AddSection(modal.Buttons(
        modal.Btn(" Run ", launchRunID),
        modal.Btn(" Cancel ", launchCancelID),
    ))
```

**Comparison with workspace create modal**:

| Aspect | Workspace Create | Orchestration Launch |
|--------|-----------------|---------------------|
| Focus steps | 8 (name, branch, prompt, task, agent, perms, create, cancel) | 4 (provider, options, run, cancel) |
| Required input | Name (text input with validation) | None (task already selected) |
| Dropdowns | Branch (filtered), Task (filtered) | None |
| Text inputs | 3 (name, branch, task search) | 0 |
| Primary purpose | Configure from scratch | Confirm and go |
| Fast path | Cannot skip - name is required | Shift+Enter skips modal entirely |
| Conditional sections | Prompt ticket mode hides/shows task | Validators=0 could hide related options |

The orchestration modal is deliberately simpler. The workspace modal is a creation form; the orchestration modal is a launch confirmation.

#### Keyboard Commands

```
Context: orchestrator-select
  Enter    Open launch modal for selected task
  Shift+Enter  Quick-run with last provider + defaults
  n        Create new task (opens td create, then launch modal)
  /        Search tasks

Context: orchestrator-plan
  Enter    Accept plan
  e        Edit plan (opens in editor)
  r        Regenerate plan
  Esc      Cancel

Context: orchestrator-running
  v        Toggle validator detail view
  d        View diff so far
  f        View modified files
  c        Cancel run
  Tab      Switch to git status plugin (shows live diff)

Context: orchestrator-results
  m        Merge worktree to main
  d        View final diff
  Enter    Accept and close task
  r        Retry with modifications
```

### td Integration Points

There are two levels of td integration: what the **orchestrator** does (lifecycle management) and what **agents** do (self-directed via td commands in their prompt).

#### Orchestrator-side (lifecycle coordination)

The orchestrator calls td directly for lifecycle transitions that require coordination:

| Event | td Command | Why orchestrator, not agent |
|-------|------------|----------------------------|
| Run starts | `td start <id>` | Must happen before any agent spawns |
| Session creation | `TD_SESSION` env var per agent | Agents need isolated sessions |
| Phase transition | `td log --type orchestration "phase:..."` | Run state for crash recovery |
| Agent timeout | `td log --blocker "agent timed out"` | Orchestrator monitors liveness |
| Validation pass | `td review <id>` | Orchestrator knows all validators passed |
| Iteration start | `td log --blocker "findings..."` | Routes validator output to td before next implementer |
| Run complete | `td log --type orchestration "phase:complete"` | Marks run state as finished |
| Task complete | `td approve <id>` (or user approves) | Orchestrator knows the lifecycle is done |
| Run failed | `td log --type orchestration "phase:failed"` | Marks run state for recovery |
| Handoff | `td handoff <id> --done "..." --remaining "..."` | Captures final state for future sessions |

#### Agent-side (self-directed)

Agents are told to use td commands in their prompt. The orchestrator does not enforce or verify these - it trusts the agent to follow the instructions:

| Action | td Command in Prompt | Purpose |
|--------|---------------------|---------|
| Read assignment | `td show <id>` | Agent reads task title, description, criteria |
| Read full context | `td context <id>` | Agent reads logs, handoffs, comments, deps |
| Orient session | `td usage --new-session` | Agent sees its session state and open work |
| Log progress | `td log "what I did"` | Creates audit trail |
| Log decisions | `td log --decision "why I chose X"` | Captures reasoning for future sessions |
| Log blockers | `td log --blocker "stuck on Y"` | Signals issues |

This split means the orchestrator handles **when** things happen (lifecycle), while agents handle **what** they learn and record (content). The orchestrator never needs to parse task content.

#### Session Management

The orchestrator creates a td session for each agent role (planner, implementer, validator-1, validator-2) by setting `TD_SESSION` as an environment variable. This preserves td's session isolation - the implementer session cannot approve its own work.

### Task Engine Adapter Pattern

The orchestrator's only coupling to td is a set of **prompt templates** and **lifecycle commands**. These are defined in a task engine adapter:

```go
type TaskEngineAdapter interface {
    // Lifecycle commands (called by orchestrator)
    StartTask(taskID string) error
    SubmitForReview(taskID string) error
    ApproveTask(taskID string) error
    LogBlocker(taskID string, message string) error
    RecordHandoff(taskID string, done, remaining string) error

    // Prompt fragments (included in agent prompts)
    ViewTaskCmd(taskID string) string      // e.g. "td show td-a1b2"
    FullContextCmd(taskID string) string   // e.g. "td context td-a1b2"
    OrientSessionCmd() string              // e.g. "td usage --new-session"
    LogProgressCmd() string                // e.g. "td log \"your progress\""
    LogDecisionCmd() string                // e.g. "td log --decision \"your decision\""

    // Session management
    CreateSession(role string) (sessionID string, err error)
}
```

The default implementation uses td. Alternative implementations could wrap Jira, Linear, GitHub Issues, or any other task system. The adapter provides both the CLI commands the orchestrator calls and the command strings injected into agent prompts.

This means swapping task engines is a matter of implementing a new adapter - the orchestration logic, agent runner, and TUI plugin don't change.

### Configuration

Added to sidecar's `config.json`:

```json
{
  "plugins": {
    "orchestrator": {
      "enabled": true,
      "provider": "claude",
      "maxIterations": 3,
      "validatorCount": 2,
      "workspace": "worktree",
      "autoMerge": false,
      "providerBinary": ""
    }
  }
}
```

Minimal configuration. The provider binary is auto-detected if not specified. Model selection is left to the CLI agent's own configuration.

## Implementation Phases

### Phase 1: Engine Core

Build the orchestration engine as a standalone package (`internal/orchestrator/`). No TUI dependency. Testable with mock runners.

- Engine lifecycle (plan → build → validate → iterate)
- Agent runner interface + Claude runner implementation
- Task engine adapter interface + td implementation (including orchestration log type)
- Workspace management (worktree creation, cleanup)
- Event emission
- Heartbeat timeout (agent liveness monitoring)
- Crash recovery (read orchestration logs from td, reconstruct run state)
- Unit tests with mock agent runner

### Phase 2: TUI Plugin

Build the sidecar plugin that wraps the engine.

- Plugin boilerplate (ID, Init, Start, Stop, Update, View, Commands)
- Task selection view (reads from td)
- Plan review view
- Implementation progress view (event stream rendering)
- Validation results view
- Cross-plugin messaging (git status refresh, file browser navigation)
- Keyboard commands and footer hints

### Phase 3: Multi-Provider

Add runners for additional CLI agents.

- Codex runner
- Gemini runner
- Provider auto-detection
- Per-task provider override

### Phase 4: Advanced Features

- Docker workspace isolation
- Parallel task orchestration (multiple tasks running simultaneously)
- Custom validator configurations (security-focused, test-focused)
- Orchestration templates (configurable agent topologies)
- Resume interrupted orchestrations

## Design Decisions

### Why shell out to CLI agents instead of API calls?

CLI agents handle authentication, model selection, tool use, and context management. The orchestrator doesn't need to reimplement any of that. It just needs to give the agent a prompt and a working directory.

### Why not a separate process communicating with sidecar?

Adding IPC complexity for something that benefits from tight TUI integration is wrong. The orchestrator needs to emit events that update the view in real-time, trigger cross-plugin navigation, and read td state directly. In-process is simpler and more responsive.

### Why pass task IDs instead of task content?

Three reasons:

1. **Agents that use tools learn faster than agents that receive pre-digested context.** When an agent runs `td show td-a1b2` and reads the task itself, it exercises the same tool-use patterns it needs for the rest of the work. When it runs `td log "implemented rate limiter"`, it builds an audit trail that persists beyond its session. Spoon-feeding context in the prompt teaches the agent nothing about the workflow.

2. **The orchestrator stays simple and pluggable.** It doesn't parse td output, doesn't understand td's data model, doesn't manage token budgets for task content. It knows: "pass this task ID and these command templates." Swapping td for Jira means changing the adapter's command strings, not the orchestration logic.

3. **Context is always live.** If the planner logs a decision, the implementer reads it from td in real-time. If a validator logs a rejection, the next implementer reads it fresh. No risk of caching stale snapshots or assembling context from an outdated read.

The tradeoff: agents need tool access to run shell commands. CLI agents (Claude Code, Codex, Gemini CLI) already have this. API-only agents without tool use would need a different integration path.

### Why store run state in td instead of a local file?

A local `.sidecar-orchestration/run.json` file would be faster to read (~microseconds vs ~50ms for a td CLI call) and wouldn't require parsing log messages. But it creates a second source of truth. If the file says "phase: validate" but td's logs show the last event was implementation, which do you trust? The answer is always "td" - so just use td.

Storing orchestration state as td logs means:
- One place to look for everything about a task's lifecycle
- State survives worktree deletion and machine switches
- `td context` shows orchestration events alongside progress logs and decisions
- The TaskEngineAdapter pattern keeps this pluggable - other backends store run state however they need to

td needs one small addition: an `orchestration` log type. This is a one-line enum change. The structured content (`phase:validate iteration:2 validator:1 approved:true`) is just the log message string, parsed by the adapter on recovery.

### Why keep blind validation?

When validators see the implementer's reasoning, they unconsciously defer to it. Blind validation catches real bugs that sighted review misses. Worth the extra agent invocations.

### Why not complexity-based model selection?

It adds substantial configuration surface area for marginal benefit. Most users want to use their preferred model for everything. If cost matters, they can configure their CLI agent to use a cheaper model. The orchestrator doesn't need to second-guess the user's model choice.

If demand emerges, this can be added later as an optional feature without changing the core architecture.

## Open Questions

1. **Plan editing UX** - Should the plan be editable in sidecar's inline editor, or should it open an external editor? Inline is more integrated; external is more capable for large edits.

2. **Validator prompt customization** - Should users be able to configure what validators look for (e.g., "focus on security" or "focus on test coverage")? Or is the default "assess against acceptance criteria" sufficient?

3. **Agent output streaming** - Different CLI agents have different streaming capabilities. Claude Code streams; others may not. How much of the real-time progress view depends on streaming?

4. **Worktree naming convention** - `agent/<task-id>-<slug>`? `orchestrator/<task-id>`? Should this match the workspace plugin's conventions?

5. **Failure escalation** - After max iterations, the orchestrator logs a handoff to td and marks the run as failed. Should it also offer to open the worktree in the user's editor for manual intervention? The worktree still exists with the agent's partial work.

6. **Orchestration log format** - The structured `phase:X key:value` format is simple but ad-hoc. Should we use JSON log messages for machine parseability, or keep the human-readable format and accept some parsing fragility?
