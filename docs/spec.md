# Sidecar: Agent-Agnostic TUI Sidebar

## Overview

A standalone Go TUI (Bubbletea/Lipgloss) that runs alongside AI coding agents, displaying real-time status via file/artifact watching. Plugin-based architecture enables git status, td integration, conversation browsing, usage metrics, and more.

**MVP**: Git status plugin only, with full plugin infrastructure for future expansion.

---

## Architecture

### Core Components

```
sidecar/
├── cmd/sidecar/main.go          # Entry point
├── internal/
│   ├── plugin/                  # Plugin system (core)
│   │   ├── interface.go         # Plugin, PanelPlugin, DataSourcePlugin interfaces
│   │   ├── registry.go          # Plugin registration/lifecycle
│   │   └── context.go           # PluginContext (shared resources)
│   ├── keybindings/             # Centralized key handling
│   │   ├── manager.go           # Key dispatch, conflict resolution
│   │   └── config.go            # YAML config loading
│   ├── tui/
│   │   ├── app/                 # Main TUI model
│   │   │   ├── model.go         # Bubbletea Model
│   │   │   ├── view.go          # Panel layout rendering
│   │   │   └── update.go        # Message handling
│   │   └── styles/              # Shared lipgloss styles
│   │       └── styles.go
│   ├── plugins/                 # Built-in plugins
│   │   └── gitstatus/           # MVP plugin
│   │       ├── plugin.go
│   │       ├── view.go
│   │       └── tree.go
│   └── adapter/                 # Agent adapters (future)
│       ├── interface.go
│       └── claude/              # Claude Code adapter stub
└── configs/
    └── keybindings.yaml         # Default keybindings
```

### Plugin Interface

```go
type Plugin interface {
    ID() string
    Name() string
    Init(ctx PluginContext) error
    Start() error
    Stop() error
    View(width, height int) string
    HandleKey(key tea.KeyMsg) (handled bool, cmd tea.Cmd)
    Refresh() tea.Cmd
    IsFocused() bool
    SetFocused(bool)
}

type PanelPlugin interface {
    Plugin
    Cursor() int
    SetCursor(int)
    ScrollOffset() int
    RowCount() int
}

type KeybindingProvider interface {
    Keybindings() []KeyBinding
}
```

### Keybinding System (VSCode-style)

```yaml
# ~/.config/sidecar/keybindings.yaml
keybindings:
  - key: "tab"
    action: "next-panel"
    context: "global"
  - key: "j"
    action: "cursor-down"
    context: "global"
  - key: "d"
    action: "show-diff"
    context: "git-status"
```

Central manager dispatches keys, plugins register their bindings, context determines active bindings.

---

## MVP: Git Status Plugin

### Features
- Tree view of changed files (staged/unstaged/untracked)
- Vim navigation (j/k/g/G) + arrow keys
- File diff preview (modal or inline)
- Stage/unstage individual files
- Expand/collapse directories
- Auto-refresh via fsnotify

### Data Source
- `git status --porcelain` for file list
- `git diff` / `git diff --cached` for diffs
- File watching on `.git/index` for real-time updates

### View Layout
```
┌─ Git Status ─────────────────┐
│ M  src/main.go          +12-3│
│ A  internal/foo.go       +45 │
│ ?  README.md                 │
│ ▸ staged (2)                 │
│   M config.yaml              │
│   A newfile.go               │
└──────────────────────────────┘
```

---

## Future Plugins (Architecture Ready)

### TD Monitor
- Wrap existing `internal/tui/monitor` from td
- Show focused issue, in-progress, activity feed

### Agent Conversations (Claude Code)
- Parse `~/.claude/projects/{project}/*.jsonl`
- List sessions, view messages, token counts
- Requires `adapter/claude/` implementation

### Usage Dashboard
- Aggregate token usage from session logs
- Cost estimation by model
- Cache hit/miss metrics

### Background Tasks
- Detect agent-spawned processes
- Show output, allow kill
- Ring buffer for logs

---

## Implementation Steps

### Phase 1: Core Framework
1. Create new repo `sidecar`
2. Implement `internal/plugin/interface.go` - Plugin, PanelPlugin interfaces
3. Implement `internal/plugin/registry.go` - Register, Init, lifecycle
4. Implement `internal/plugin/context.go` - PluginContext struct
5. Implement `internal/keybindings/manager.go` - Key dispatch
6. Implement `internal/keybindings/config.go` - YAML loading
7. Implement `internal/tui/styles/styles.go` - Color palette, panel styles

### Phase 2: Main TUI Shell
1. Implement `internal/tui/app/model.go` - Main Bubbletea model
2. Implement `internal/tui/app/view.go` - Panel layout, help overlay
3. Implement `internal/tui/app/update.go` - Message routing
4. Implement `cmd/sidecar/main.go` - Entry point, plugin loading

### Phase 3: Git Status Plugin
1. Implement `internal/plugins/gitstatus/plugin.go` - Plugin interface
2. Implement `internal/plugins/gitstatus/tree.go` - Tree data structure
3. Implement `internal/plugins/gitstatus/view.go` - Rendering
4. Add git command execution helpers
5. Add fsnotify watcher for `.git/index`
6. Implement diff modal

### Phase 4: Polish
1. Add `configs/keybindings.yaml` defaults
2. Add config file support (`~/.config/sidecar/`)
3. Add help screen
4. Testing

---

## Key Patterns from td to Reuse

| Pattern | Source File | Usage |
|---------|-------------|-------|
| Bubbletea model structure | `td/internal/tui/monitor/model.go` | Model, Init, Update, View |
| Panel/cursor management | `td/internal/tui/monitor/model.go:64-131` | Per-panel cursors, scroll offsets |
| Lipgloss styling | `td/internal/tui/monitor/styles.go` | Color palette, panel borders |
| Modal overlay | `td/internal/tui/monitor/view.go:924-939` | wrapModal pattern |
| Async commands | `td/internal/tui/monitor/model.go:586-619` | Background refresh |
| Vim keybindings | `td/internal/tui/monitor/model.go:271-450` | j/k/h/l handling |

---

## Agent Adapter Interface (Future)

```go
type Adapter interface {
    ID() string
    Detect() (AgentInfo, error)
    IsAvailable() bool
    WatchSessions(project string) (<-chan SessionEvent, error)
}

type AgentInfo struct {
    Type        AgentType  // claude-code, cursor, opencode
    Version     string
    SessionID   string
    DataPath    string
    Capabilities []string
}
```

Claude Code data lives at `~/.claude/projects/{project}/*.jsonl` with full conversation history, token usage, and tool calls. Other agents will need reverse engineering.

---

## Feasibility Notes

| Feature | Difficulty | Notes |
|---------|------------|-------|
| Git status plugin | Easy | Pure git commands, well-documented |
| Plugin system | Medium | Interface design critical, get right first |
| Keybinding config | Easy | YAML parsing straightforward |
| Claude Code adapter | Medium | JSONL parsing, well-structured data |
| Cursor adapter | Hard | Less accessible data, needs investigation |
| Background tasks | Medium | Process detection + output capture |

---

## Dependencies

```go
require (
    github.com/charmbracelet/bubbletea v1.x
    github.com/charmbracelet/lipgloss v1.x
    github.com/fsnotify/fsnotify v1.x
    gopkg.in/yaml.v3 v3.x
)
```
