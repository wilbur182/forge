# Project Switcher Developer Guide

Implementation guide for the project switcher modal (`@` hotkey).

## Architecture Overview

The project switcher is an app-level modal in `internal/app/`. It consists of:

| Component | File | Purpose |
|-----------|------|---------|
| Model state | `model.go:47-53` | Modal visibility, cursor, scroll, filter |
| Init/reset | `model.go:346-404` | State initialization and cleanup |
| Keyboard | `update.go:385-488` | Key event handling |
| Mouse | `update.go:697-811` | Click, scroll, hover |
| View | `view.go:108-258` | Modal rendering |
| Project switch | `model.go:406-440` | Plugin context reinit |

## Model State

```go
// internal/app/model.go

// Project switcher modal
showProjectSwitcher      bool                    // Modal visibility
projectSwitcherCursor    int                     // Selected index in filtered list
projectSwitcherScroll    int                     // Scroll offset for long lists
projectSwitcherHover     int                     // Mouse hover index (-1 = none)
projectSwitcherInput     textinput.Model         // Filter text input
projectSwitcherFiltered  []config.ProjectConfig  // Filtered project list
```

## Initialization

### Opening the Modal

When `@` is pressed (`update.go:554`):

```go
case "@":
    m.showProjectSwitcher = !m.showProjectSwitcher
    if m.showProjectSwitcher {
        m.activeContext = "project-switcher"
        m.initProjectSwitcher()
    } else {
        m.resetProjectSwitcher()
        m.updateContext()
    }
```

### initProjectSwitcher()

`model.go:356-376` - Sets up the modal state:

```go
func (m *Model) initProjectSwitcher() {
    ti := textinput.New()
    ti.Placeholder = "Filter projects..."
    ti.Focus()
    ti.CharLimit = 50
    ti.Width = 40
    m.projectSwitcherInput = ti
    m.projectSwitcherFiltered = m.cfg.Projects.List
    m.projectSwitcherCursor = 0
    m.projectSwitcherScroll = 0
    m.projectSwitcherHover = -1

    // Pre-select current project
    for i, proj := range m.projectSwitcherFiltered {
        if proj.Path == m.ui.WorkDir {
            m.projectSwitcherCursor = i
            break
        }
    }
}
```

### resetProjectSwitcher()

`model.go:346-354` - Cleans up when modal closes:

```go
func (m *Model) resetProjectSwitcher() {
    m.showProjectSwitcher = false
    m.activeContext = ""
    m.projectSwitcherInput = textinput.Model{}
    m.projectSwitcherCursor = 0
    m.projectSwitcherScroll = 0
    m.projectSwitcherHover = -1
    m.projectSwitcherFiltered = nil
}
```

## Keyboard Handling

All keyboard logic is in `update.go:385-488`.

### Key Priority

1. **KeyType switch** (`msg.Type`) - Handles special keys:
   - `KeyEsc` - Clear filter or close modal
   - `KeyEnter` - Select project
   - `KeyUp/KeyDown` - Arrow navigation

2. **String switch** (`msg.String()`) - Handles named keys:
   - `ctrl+n/ctrl+p` - Emacs-style navigation
   - `@` - Close modal

3. **Fallthrough** - All other keys forwarded to textinput (including printable characters)

### Esc Behavior

Esc has two behaviors (`update.go:400-411`):

```go
case tea.KeyEsc:
    // Clear filter if set
    if m.projectSwitcherInput.Value() != "" {
        m.projectSwitcherInput.SetValue("")
        m.projectSwitcherFiltered = allProjects
        m.projectSwitcherCursor = 0
        m.projectSwitcherScroll = 0
        return m, nil
    }
    // Otherwise close modal
    m.resetProjectSwitcher()
    m.updateContext()
    return m, nil
```

### Navigation with Scroll

Navigation updates cursor and ensures visibility (`update.go:423-440`):

```go
case tea.KeyDown:
    m.projectSwitcherCursor++
    if m.projectSwitcherCursor >= len(projects) {
        m.projectSwitcherCursor = len(projects) - 1
    }
    if m.projectSwitcherCursor < 0 {
        m.projectSwitcherCursor = 0
    }
    m.projectSwitcherScroll = projectSwitcherEnsureCursorVisible(
        m.projectSwitcherCursor, m.projectSwitcherScroll, 8)
    return m, nil
```

### Filter Input

Keys not matching special cases go to textinput (`update.go:471-486`):

```go
// Forward to text input
var cmd tea.Cmd
m.projectSwitcherInput, cmd = m.projectSwitcherInput.Update(msg)

// Re-filter on change
m.projectSwitcherFiltered = filterProjects(allProjects, m.projectSwitcherInput.Value())
m.projectSwitcherHover = -1  // Clear hover on filter change

// Clamp cursor to valid range
if m.projectSwitcherCursor >= len(m.projectSwitcherFiltered) {
    m.projectSwitcherCursor = len(m.projectSwitcherFiltered) - 1
}
if m.projectSwitcherCursor < 0 {
    m.projectSwitcherCursor = 0
}
```

## Filtering

### filterProjects()

`model.go:378-392` - Case-insensitive substring match:

```go
func filterProjects(all []config.ProjectConfig, query string) []config.ProjectConfig {
    if query == "" {
        return all
    }
    q := strings.ToLower(query)
    var matches []config.ProjectConfig
    for _, p := range all {
        if strings.Contains(strings.ToLower(p.Name), q) ||
           strings.Contains(strings.ToLower(p.Path), q) {
            matches = append(matches, p)
        }
    }
    return matches
}
```

Searches both `Name` and `Path` fields.

### Scroll Helper

`model.go:394-404` - Keeps cursor in visible window:

```go
func projectSwitcherEnsureCursorVisible(cursor, scroll, maxVisible int) int {
    if cursor < scroll {
        return cursor
    }
    if cursor >= scroll+maxVisible {
        return cursor - maxVisible + 1
    }
    return scroll
}
```

## Mouse Handling

Mouse logic is in `update.go:697-811`.

### Layout Calculation

The modal layout for hit detection (`update.go:718-740`):

```go
// Modal content lines: title + input + count + projects + help
modalContentLines := 2 + 1 + 1 + visibleCount*2 + 2
if m.projectSwitcherScroll > 0 {
    modalContentLines++ // scroll indicator above
}
if len(projects) > m.projectSwitcherScroll+visibleCount {
    modalContentLines++ // scroll indicator below
}

// ModalBox adds padding and border (~2 on each side)
modalHeight := modalContentLines + 4
modalWidth := 50
modalX := (m.width - modalWidth) / 2
modalY := (m.height - modalHeight) / 2
```

### Click Detection

Project click detection (`update.go:749-776`):

```go
// Content starts at modalY + 2 (border + padding)
// Title: 2 lines, Input: 1 line, Count: 1 line
contentStartY := modalY + 2 + 2 + 1 + 1
if m.projectSwitcherScroll > 0 {
    contentStartY++ // scroll indicator
}

// Each project takes 2 lines (name + path)
relY := msg.Y - contentStartY
if relY >= 0 && relY < visibleCount*2 {
    projectIdx := m.projectSwitcherScroll + relY/2
    // Handle click...
}
```

### Hover State

Mouse motion updates hover index (`update.go:772-781`):

```go
case tea.MouseActionMotion:
    m.projectSwitcherHover = projectIdx
```

Hover is cleared when:
- Mouse moves outside project list area
- Filter changes (ensures no invalid index)
- Modal closes

### Scroll Wheel

Wheel events move cursor and scroll (`update.go:784-804`):

```go
case tea.MouseButtonWheelUp:
    m.projectSwitcherCursor--
    // clamp and update scroll
case tea.MouseButtonWheelDown:
    m.projectSwitcherCursor++
    // clamp and update scroll
```

## View Rendering

View logic is in `view.go:108-258`.

### Modal Structure

```
┌─────────────────────────────────────────┐
│ Switch Project  @                       │  <- Title (2 lines)
│                                         │
│ [Filter projects...                  ]  │  <- Input (1 line)
│ 3 of 10 projects                        │  <- Count (1 line, only when filtering)
│   ↑ 2 more above                        │  <- Scroll indicator (optional)
│ → sidecar                               │  <- Project name (cursor/hover)
│   ~/code/sidecar                        │  <- Project path
│   td (current)                          │  <- Current project (green)
│   ~/code/td                             │
│   ↓ 5 more below                        │  <- Scroll indicator (optional)
│                                         │
│ esc clear  @ close                      │  <- Help hints
└─────────────────────────────────────────┘
```

### Empty States

Two empty states exist:

1. **No projects configured** (`view.go:120-139`) - Shows config example
2. **No filter matches** (`view.go:152-164`) - Shows "No matches" with hints

### Project Item Styling

Each project has conditional styling (`view.go:209-227`):

| State | Name Style | Path Style |
|-------|------------|------------|
| Normal | Secondary (blue) | Subtle |
| Cursor/Hover | Primary + Bold | Muted |
| Current | Success (green) + Bold | Subtle |
| Current + Selected | Success + Bold | Muted |

Current project shows "(current)" label.

### Scroll Indicators

Show when list overflows (`view.go:176-179`, `247-249`):

```go
if scrollOffset > 0 {
    b.WriteString(styles.Muted.Render(fmt.Sprintf("  ↑ %d more above\n", scrollOffset)))
}
// ... render projects ...
if remaining > 0 {
    b.WriteString(styles.Muted.Render(fmt.Sprintf("  ↓ %d more below\n", remaining)))
}
```

## Project Switching

### switchProject()

`model.go:406-440` - Handles the actual switch:

```go
func (m *Model) switchProject(projectPath string) tea.Cmd {
    // Skip if same project
    if projectPath == m.ui.WorkDir {
        return m.toast("Already on this project")
    }

    return func() tea.Msg {
        // 1. Stop all plugins
        m.registry.Stop()

        // 2. Update working directory
        m.ui.WorkDir = projectPath

        // 3. Reinitialize plugins
        m.registry.Reinit(plugin.Context{
            WorkDir: projectPath,
            // ...
        })

        // 4. Restore active plugin for this project
        // 5. Show toast notification

        return ProjectSwitchedMsg{Path: projectPath}
    }
}
```

### Plugin Reinitialization

When switching projects, plugins receive a new `Init()` call. Plugins must reset their state - see `internal/plugins/workspace/plugin.go:259-265` for an example of proper state reset.

## Adding New Features

### Adding a Keyboard Shortcut

1. Add case in `update.go` string switch (after KeyType switch)
2. Return early to prevent textinput forwarding

```go
case "ctrl+d":
    // Custom action
    return m, nil
```

### Adding Project Metadata

1. Extend `config.ProjectConfig` in `internal/config/types.go`
2. Update `filterProjects()` to search new fields
3. Update view rendering to display new fields

### Changing Filter Algorithm

Replace `filterProjects()` body. Current: substring match. Options:
- Fuzzy matching (like command palette)
- Regex support
- Field-specific search (`name:foo`)

## Testing

Currently no dedicated tests exist for the project switcher. Recommended test coverage:

1. **filterProjects()** - Various query inputs
2. **projectSwitcherEnsureCursorVisible()** - Scroll boundary cases
3. **Keyboard navigation** - Cursor bounds, scroll sync
4. **Mouse hit detection** - Click accuracy with scroll

See `td-1a735359` for the test task.

## Common Pitfalls

1. **Forgetting updateContext()** - Call after closing modal to restore app context
2. **Stale hover index** - Clear `projectSwitcherHover` when filter changes
3. **Cursor out of bounds** - Always clamp after filtering
4. **Printable keys vs navigation** - Keep printable characters routed to textinput so filtering always works
5. **Mouse Y calculation** - Account for scroll indicators and filter count line

## File Locations

| File | Contents |
|------|----------|
| `internal/app/model.go` | State, init, reset, filter, switch |
| `internal/app/update.go` | Keyboard and mouse handlers |
| `internal/app/view.go` | Rendering |
| `internal/config/types.go` | ProjectConfig struct |
| `internal/config/loader.go` | Config loading with path validation |
