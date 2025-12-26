# Sidecar (Sidebar) - Spec Draft

## Goal
Build a standalone TUI sidebar that provides a live view of agent work across tools (Codex, Claude, OpenCode, etc.). The UI is built in Go with Bubble Tea and Lipgloss and uses a compiled-in plugin system to surface tool-specific capabilities. It should update continuously as the agent works, but degrade gracefully when integrations are unavailable.

## Non-Goals (v1)
- No external plugin loading or dynamic code injection.
- No remote network dependency (all integrations via local files, IPC, or on-disk SQLite).
- No assumption of any single agent vendor or format.

## Architecture Overview

### Core App
- Bubble Tea root model orchestrating layout, global events, and focus management.
- Global store for shared state (selection, filters, focus, time window).
- Central command registry with contexts and a configurable keymap layer.
- Theme system (Lipgloss) with semantic styles (focus, selection, warning, muted).

### Plugin System (Compiled-In)
Plugins are statically linked and registered in main(). Each plugin exposes:
- Init(ctx) tea.Cmd
- Update(msg) (PluginState, tea.Cmd)
- View(state) string
- Commands() []CommandSpec
- FocusContext() string
- Optional PollInterval() time.Duration

The host maps focus to a plugin context and routes key commands through a centralized command registry.

### Keymap / Command Registry
- Command ID -> handler + context
- Global default bindings and per-plugin bindings
- User overrides in ~/.sidecar/keymap.yaml (or json)
- Conflict resolution: focused context wins; global fallback

## Core Plugins (v1)

### 1) Git Status Tree + Diff Viewer
- Tree view of changed files grouped by staged/unstaged.
- Navigation: arrows and vim keys (j/k) via centralized keymap.
- Diff viewer: unified + optional side-by-side; hunk navigation and file-level review.
- Data source: git CLI (status --porcelain=v2 -z, diff, diff --cached).

### 2) TD Monitor (SQLite)
- Mirrors td monitor functionality using the local SQLite database (.todos/issues.db).
- Adapter opens DB (prefer read-only mode, fallback to normal open).
- Reuses the data aggregation logic in td's monitor (focused issue, task list, activity, sessions).
- Poll interval default 2s with debounce to reduce UI churn.

## Optional Plugins (v2+)

### Background Tasks
- Shows running/queued tasks across agents.
- Adapter for tools that expose task state or local logs.
- Actions: cancel/kill task when supported.

### Agent Conversation Browser
- Session threads with filter/search.
- Adapter reads local session logs or tool-specific stores.
- Read-only in early versions.

### Usage Dashboard
- Tokens/time/latency per session, plus live session metrics.
- Adapter reads local telemetry when available, otherwise shows "unsupported".

## Agent-Agnostic Adapter Layer
Define a capability-first interface and implement adapters per tool.

Suggested interface surface:
- Capabilities() CapabilitySet
- SubscribeEvents(ctx) (<-chan Event)
- FetchTasks / FetchConversations / FetchUsage / FetchMonitorData

Adapters must avoid hard dependencies on network access. Prefer:
- Local log files
- SQLite
- IPC sockets
- CLI output parsing

## Data & Persistence
- Cache and user config in ~/.sidecar/
- Minimal session indexing for conversation browser and usage dashboard.
- Avoid writes to agent-owned stores unless explicitly supported.

## UI Layout
- Left: nav + plugin list
- Right: active plugin panel
- Optional bottom bar: global status + key hints (toggleable)

## Feasibility & Risks
- Git + diff viewer are straightforward.
- TD monitor is low risk due to existing SQLite schema and queries.
- Agent-agnostic features depend on standardized local telemetry. Proposed mitigation: simple JSONL event spec for tool adapters.

## Phased Delivery
1. Core app skeleton + keymap registry + plugin host
2. Git status tree + diff viewer
3. TD monitor plugin (SQLite adapter)
4. Optional plugins: tasks, conversation browser, usage dashboard

## Open Questions
- Final name: sidecar vs sidebar
- Preferred config format (yaml/json/toml)
- Behavior when multiple adapters claim the same capability

