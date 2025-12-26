# Sidecar

Terminal UI for monitoring AI coding agent sessions.

**Status: Alpha** - Functional but expect rough edges and breaking changes.

## Overview

Sidecar provides a unified terminal interface for viewing Claude Code conversations, git status, and task progress. Built for developers who want visibility into their AI coding sessions without leaving the terminal.

## Requirements

- Go 1.23+

## Quick Start

```bash
git clone https://github.com/marcus/sidecar
cd sidecar
make install
sidecar
```

## Usage

```bash
# Run from any project directory
sidecar

# Specify project root
sidecar --project /path/to/project

# Enable debug logging
sidecar --debug

# Check version
sidecar --version
```

## Plugins

### Git Status
View staged, modified, and untracked files. Stage/unstage with `s`/`u`, view diffs with `d`.

### TD Monitor
Track tasks from the TD task management system. View in-progress, ready, and reviewable issues.

### Conversations
Browse Claude Code session history with message content and token usage.

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `q`, `ctrl+c` | Quit |
| `tab` / `shift+tab` | Navigate plugins |
| `1-9` | Focus plugin by number |
| `j/k`, `↓/↑` | Navigate items |
| `enter` | Select |
| `esc` | Back/close |
| `r` | Refresh |
| `?` | Toggle help |

## Configuration

Config file: `~/.config/sidecar/config.json`

```json
{
  "plugins": {
    "git-status": { "enabled": true, "refreshInterval": "1s" },
    "td-monitor": { "enabled": true, "refreshInterval": "2s" },
    "conversations": { "enabled": true }
  },
  "ui": {
    "showFooter": true,
    "showClock": true
  }
}
```

## Development

```bash
make build        # Build to ./bin/sidecar
make test         # Run tests
make test-v       # Verbose test output
make install-dev  # Install with git version info
make fmt          # Format code
```

## License

MIT
