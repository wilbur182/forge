# Forge

One window. All workflows.

**Status: v0.1.0 Released.** Forge is a unified TUI development environment that combines the best of companion tools with native AI agent integration.

[Architecture](./FORGE.md) · [Development Setup](#development)

## Overview

Forge puts your entire development workflow in one window: plan tasks with [td](https://github.com/marcus/td), monitor AI agent conversations, review diffs, stage commits, browse code, and manage workspaces—all without context switching.

**Forge is a rebranded fork of [Sidecar](https://github.com/marcus/sidecar) with native OpenCode session integration.**

## Installation

### From Source

Ensure you have Go 1.21+ installed.

```bash
# Build the binary
go build -o forge ./cmd/forge

# Install to your $GOPATH/bin
go install ./cmd/forge
```

## Requirements

- macOS, Linux, or WSL
- Go 1.21+ (for building from source)

## Quick Start

After installation, run from any project directory:

```bash
forge
```

## Suggested Use

Split your terminal horizontally: run your coding agent (Claude Code, OpenCode, etc.) on the left and Forge on the right.

```
┌─────────────────────────────┬─────────────────────┐
│                             │                     │
│   AI Agent (OpenCode/etc.)  │       Forge         │
│                             │                     │
│   $ opencode                │   [Git] [Files]     │
│   > fix the auth bug...     │   [Tasks] [Conversations]│
│                             │                     │
└─────────────────────────────┴─────────────────────┘
```

As the agent works, you can:

- Watch tasks move through the workflow in TD Monitor
- See files change in real-time in the Git plugin
- Browse and preview code in the File Browser
- **View and resume conversations** across all supported agent adapters
- Switch between built-in and community themes with live previews

## Usage

```bash
# Run from any project directory
forge

# Specify project root
forge --project /path/to/project

# Enable debug logging
forge --debug

# Check version
forge --version
```

## Plugins

### Git Status

View staged, modified, and untracked files with a split-pane interface. The sidebar shows files and recent commits; the main pane shows syntax-highlighted diffs.

**Features:**
- Stage/unstage files with `s`/`u`
- View diffs inline or full-screen with `d`
- Toggle side-by-side diff view with `v`
- Browse commit history and view commit diffs
- Auto-refresh on file system changes

### Conversations (OpenCode Integration)

Browse session history from multiple AI coding agents with message content, token usage, and search. Forge features deep integration with OpenCode, automatically discovering sessions in your local storage.

**Supported Agents:**
- OpenCode
- Claude Code
- Codex
- Gemini CLI
- Cursor
- Amp
- Kiro
- Warp

**Features:**
- Unified view across all supported agents
- View all sessions grouped by date
- Search sessions with `/`
- Expand messages to see full content

### TD Monitor

Integration with [TD](https://github.com/marcus/td), a task management system designed for AI agents working across context windows.

**Features:**
- Current focused task display
- Scrollable task list with status indicators
- Activity log with session context

### File Browser

Navigate project files with a tree view and syntax-highlighted preview.

**Features:**
- Collapsible directory tree
- Code preview with syntax highlighting
- Auto-refresh on file changes

### Workspaces

Manage workspaces for parallel development with integrated agent support. Create isolated branches as sibling directories, link tasks from TD, and launch coding agents directly from Forge.

**Features:**
- Create and delete workspaces with `n`/`D`
- Link TD tasks to workspaces for context tracking
- Launch coding agents (Claude, OpenCode, Codex, Gemini, Cursor) with `a`
- Merge workflow: commit, push, create PR, and cleanup with `m`

## Keyboard Shortcuts

| Key                 | Action                           |
| ------------------- | -------------------------------- |
| `q`, `ctrl+c`       | Quit                             |
| `@`                 | Open project switcher            |
| `W`                 | Open worktree switcher           |
| `#`                 | Open theme switcher              |
| `tab` / `shift+tab` | Navigate plugins                 |
| `1-9`               | Focus plugin by number           |
| `j/k`, `↓/↑`        | Navigate items                   |
| `ctrl+d/u`          | Page down/up in scrollable views |
| `g/G`               | Jump to top/bottom               |
| `enter`             | Select                           |
| `esc`               | Back/close                       |
| `r`                 | Refresh                          |
| `?`                 | Toggle help                      |

## Configuration

Config file: `~/.config/forge/config.json`

## Development

```bash
go build -o forge ./cmd/forge
go test ./...
```

## Attribution

**Forge** is a fork of [Sidecar](https://github.com/marcus/sidecar) by [Marcus](https://github.com/marcus).

Integrates [OpenCode](https://github.com/code-yeongyu/oh-my-opencode) session support.

## License

MIT
