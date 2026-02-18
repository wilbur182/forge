# Forge Architecture

## Overview

Forge is a unified TUI system that combines the best of Sidecar (companion tools) and OpenCode (AI agent) into a single, cohesive development environment.

## Core Philosophy

**One window. All workflows.** No more split terminals. No more context switching. Everything—agent chat, git, files, tasks, conversations—in a single TUI.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        FORGE TUI                                 │
├─────────────────────────────────────────────────────────────────┤
│  UI Layer (Bubble Tea + Lipgloss)                               │
│  ├─ Tabbed interface (Agent | Git | Files | Tasks | ...)        │
│  ├─ Split-pane support                                          │
│  └─ Unified keybinding system                                   │
├─────────────────────────────────────────────────────────────────┤
│  Plugin System                                                  │
│  ├─ Core Plugins (Go native)                                    │
│  │  ├─ Git Status                                               │
│  │  ├─ File Browser                                             │
│  │  ├─ TD Task Monitor                                          │
│  │  └─ Workspaces                                               │
│  ├─ Agent Plugin (WASM runtime)                                 │
│  │  └─ Conversations (migrated to agent context)                │
│  └─ External Plugins (WASM or Go plugins)                       │
│     └─ Community extensions                                     │
├─────────────────────────────────────────────────────────────────┤
│  Shared Context Layer                                           │
│  ├─ Project/workspace state manager                             │
│  ├─ Event bus for inter-plugin communication                    │
│  └─ Unified file system watcher                                 │
├─────────────────────────────────────────────────────────────────┤
│  Agent Runtime (WASM)                                           │
│  ├─ OpenCode agent compiled to WASM                             │
│  ├─ LLM provider abstraction (Claude, GPT, Gemini, Copilot)     │
│  ├─ Tool execution bridge (Go ↔ WASM)                           │
│  └─ Conversation history & context management                   │
├─────────────────────────────────────────────────────────────────┤
│  Platform Layer                                                 │
│  ├─ Git operations                                              │
│  ├─ File system operations                                      │
│  ├─ Terminal/process management                                 │
│  └─ LLM API clients                                             │
└─────────────────────────────────────────────────────────────────┘
```

## Monorepo Structure

```
forge/
├── cmd/forge/                    # Main TUI entry point
│   └── main.go
├── internal/
│   ├── core/                     # Core app framework (from Sidecar)
│   ├── plugins/                  # Plugin implementations
│   │   ├── gitstatus/
│   │   ├── filebrowser/
│   │   ├── tdmonitor/
│   │   ├── workspaces/
│   │   └── agent/                # NEW: Agent plugin with WASM runtime
│   ├── agent-wasm-runtime/       # NEW: WASM execution environment
│   │   ├── runtime.go            # WASM runtime manager
│   │   ├── bridge.go             # Go ↔ WASM bridge
│   │   └── callbacks.go          # Host function callbacks
│   ├── bridge/                   # NEW: Integration layer
│   │   ├── context.go            # Shared context manager
│   │   └── events.go             # Event bus
│   └── config/                   # Configuration management
├── packages/
│   └── agent-wasm/               # NEW: OpenCode agent WASM build
│       ├── src/                  # TypeScript source (from forge-agent)
│       ├── build/                # Build scripts
│       └── dist/                 # Compiled WASM output
├── configs/
│   └── forge.json                # Default configuration
├── docs/
│   └── architecture/             # Architecture documentation
└── scripts/
    └── build-wasm.sh             # WASM build automation
```

## Component Integration

### 1. Agent Plugin ←→ Core System

The agent plugin replaces Sidecar's read-only conversation viewer with a full interactive agent:

**Before (Sidecar)**:
- Conversations plugin: Read-only viewer of agent history
- External agent runs in separate terminal

**After (Forge)**:
- Agent plugin: Interactive chat + history + tools
- Agent runs inside Forge via WASM

**Integration Points**:
- Agent UI uses Bubble Tea components (like other plugins)
- Agent tools call Go bridge functions (file ops, bash, etc.)
- Agent context shared with other plugins via event bus

### 2. Shared Context

All plugins share unified context:

```go
type ProjectContext struct {
    RootPath      string
    CurrentBranch string
    ActiveTask    *td.Task
    OpenFiles     []string
    AgentSession  *AgentSession
}
```

Events broadcast to all plugins:
- `FileChangedEvent` → Git plugin refreshes, Agent sees file changes
- `TaskUpdatedEvent` → TD monitor updates, Agent gets context
- `AgentMessageEvent` → Conversations view updates

### 3. WASM Bridge

The Go runtime exposes host functions to WASM:

```go
// Host functions callable from WASM agent
func hostReadFile(path string) ([]byte, error)
func hostWriteFile(path string, content []byte) error
func hostExecuteCommand(cmd string, args []string) (string, error)
func hostListDirectory(path string) ([]FileInfo, error)
func hostSearchCode(query string) ([]SearchResult, error)
func hostSendLLMRequest(provider string, messages []Message) (string, error)
```

## Migration Strategy

### Phase 1: Foundation (Current)
- [x] Fork repositories
- [ ] Create monorepo structure
- [ ] Set up WASM build pipeline
- [ ] Document architecture

### Phase 2: Agent Integration
- [ ] Port OpenCode agent to WASM-compatible subset
- [ ] Implement WASM runtime in Go
- [ ] Create bridge layer for tool execution
- [ ] Build agent plugin UI

### Phase 3: Unification
- [ ] Merge conversation viewer into agent plugin
- [ ] Implement shared context layer
- [ ] Add event bus for inter-plugin communication
- [ ] Unify keybindings and UI components

### Phase 4: Polish
- [ ] Single-window UX refinements
- [ ] Performance optimization
- [ ] Plugin API documentation
- [ ] Community plugin examples

## Key Design Decisions

### Why WASM for the Agent?

1. **Language flexibility**: Keep OpenCode's TypeScript agent logic
2. **Security**: Sandboxed execution environment
3. **Portability**: Agent can run in other WASM hosts
4. **Performance**: Near-native speed for compute-heavy tasks

### Why Go for the Core?

1. **Sidecar's foundation**: Already proven, battle-tested
2. **Binary size**: Single static binary, easy distribution
3. **TUI ecosystem**: Bubble Tea, Lipgloss mature libraries
4. **Plugin system**: Go plugins or WASM plugins both supported

### Plugin Architecture

Plugins are self-contained units with:
- **Model**: Plugin state and business logic
- **View**: Bubble Tea view function
- **Update**: Message handler
- **Commands**: Keybindings for footer

Agent plugin follows same pattern but delegates LLM interaction to WASM runtime.

## Configuration

```json
{
  "forge": {
    "version": "1.0.0",
    "agent": {
      "enabled": true,
      "defaultProvider": "claude",
      "providers": {
        "claude": { "model": "claude-opus-4.5" },
        "openai": { "model": "gpt-5.2" },
        "gemini": { "model": "gemini-3-pro" }
      }
    },
    "plugins": {
      "git-status": { "enabled": true },
      "file-browser": { "enabled": true },
      "td-monitor": { "enabled": true },
      "workspaces": { "enabled": true },
      "agent": { "enabled": true }
    },
    "ui": {
      "layout": "tabs",
      "theme": "default"
    }
  }
}
```

## Development Workflow

```bash
# Build everything
make build

# Build just the Go core
make build-core

# Build WASM agent
make build-wasm

# Run tests
make test

# Run Forge
./bin/forge
```

## Future Considerations

- **Remote development**: WASM agent could run on remote server
- **Multi-agent**: Multiple agents in different tabs/workspaces
- **Collaboration**: Shared sessions between developers
- **Custom tools**: Plugin API for user-defined tools

## License

MIT (same as Sidecar and OpenCode)
