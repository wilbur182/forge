# Forge Architecture (v0.1.0)

## Overview

Forge is a unified TUI development environment that puts your entire workflow—agent chat, git, files, tasks, and workspaces—into a single terminal window.

It is a fully rebranded fork of [marcus/sidecar](https://github.com/marcus/sidecar), developed as `github.com/wilbur182/forge`. It integrates the best of Sidecar's companion tools with native support for viewing and resuming AI agent conversations, specifically featuring deep integration with OpenCode.

## Core Philosophy

**One window. All workflows.** No more split terminals. No more context switching. Everything you need to monitor and manage your development loop is accessible via a single interface.

## Architecture

Forge is built using the **Bubble Tea** TUI framework and **Lipgloss** for styling.

```
┌─────────────────────────────────────────────────────────────────┐
│                        FORGE TUI                                 │
├─────────────────────────────────────────────────────────────────┤
│  UI Layer (Bubble Tea + Lipgloss)                               │
│  ├─ Tabbed interface (Git | Files | Tasks | Conversations | ...) │
│  ├─ Split-pane support                                          │
│  └─ Unified keybinding system                                   │
├─────────────────────────────────────────────────────────────────┤
│  Plugin System                                                  │
│  ├─ Git Status: Interactive diffing and staging                 │
│  ├─ File Browser: Code preview and navigation                   │
│  ├─ TD Monitor: Task tracking integration                       │
│  ├─ Workspaces: Multi-agent project management                  │
│  └─ Conversations: Unified agent session history                │
├─────────────────────────────────────────────────────────────────┤
│  Adapter Layer (internal/adapter/)                              │
│  ├─ OpenCode: Reads from ~/.local/share/opencode/storage/       │
│  ├─ Claude Code, Codex, Gemini CLI, Cursor, Amp, Kiro, Warp     │
│  └─ Platform-native session discovery                           │
├─────────────────────────────────────────────────────────────────┤
│  Platform Layer                                                 │
│  ├─ Git operations                                              │
│  ├─ File system operations                                      │
│  └─ Terminal/process management                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Monorepo Structure

```
forge/
├── cmd/forge/                    # Main TUI entry point
│   └── main.go                   # Builds the 'forge' binary
├── cmd/sidecar/                  # Backward compatibility wrapper
│   └── main.go
├── internal/
│   ├── core/                     # TUI framework and application loop
│   ├── plugins/                  # Plugin implementations
│   │   ├── gitstatus/            # Git staging and diffing
│   │   ├── filebrowser/          # Tree view and previews
│   │   ├── tdmonitor/            # Integration with 'td' task manager
│   │   ├── workspaces/           # Workspace and agent launcher
│   │   └── conversations/        # Unified agent session viewer
│   ├── adapter/                  # AI agent session discovery
│   │   ├── opencode/             # OpenCode session integration
│   │   ├── claudecode/           # Claude Code session integration
│   │   └── ...                   # Other agent adapters
│   └── config/                   # Configuration management
├── internal/agent-wasm-runtime/  # Future: WASM stubs (not implemented)
├── packages/agent-wasm/          # Future: WASM stubs (not implemented)
└── configs/
    └── forge.json                # Default configuration
```

## Component Integration

### 1. OpenCode Integration

Forge features deep integration with OpenCode. The OpenCode adapter (`internal/adapter/opencode/`) automatically discovers and reads agent sessions from:
- `~/.local/share/opencode/storage/` (Linux/General)
- `~/Library/Application Support/opencode/storage/` (macOS)

These sessions are displayed in the Conversations plugin, allowing you to monitor agent progress in real-time.

### 2. Supported Agents

The Conversations and Workspace plugins support a wide array of AI agents:
- Claude Code
- OpenCode
- Codex
- Gemini CLI
- Cursor
- Amp
- Kiro
- Warp

### 3. Backward Compatibility

To ensure a smooth transition for users coming from Sidecar, Forge maintains a `cmd/sidecar/` entry point. While the primary binary is `forge`, `sidecar` can still be built and used as a compatibility alias.

## Configuration

Forge uses a JSON configuration file located at `~/.config/forge/config.json`. This file controls plugin enablement, refresh intervals, and UI preferences.

## Development Workflow

Forge is written in Go (v1.21+).

```bash
# Build the forge binary
go build -o forge ./cmd/forge

# Install to $GOPATH/bin
go install ./cmd/forge

# Run tests
go test ./...
```

## Future Phase: WASM Integration

While stubs exist in the codebase for a WASM-based agent runtime, this is currently considered a future phase of development. The current version of Forge focuses on native integration and session discovery of external agents.

## License

MIT (Forked from marcus/sidecar)
