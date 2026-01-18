# Capturing Sidecar Screenshots

This guide explains how to capture screenshots of Sidecar for documentation purposes.

## Prerequisites

- `tmux` - for running sidecar in a detached session
- `termshot` - for rendering terminal output as PNG (`brew install homeport/tap/termshot`)
- `aha` (optional) - for HTML output (`brew install aha`)

## Terminal Size

For documentation screenshots, resize your terminal to approximately **120x45** characters before capturing. This produces screenshots that fit well in documentation without being too large.

## Agent-Controlled Screenshots (Recommended)

Use the helper script `scripts/tmux-screenshot.sh` with simple subcommands:

### Step 1: Start sidecar

```bash
./scripts/tmux-screenshot.sh start
```

This starts sidecar in a detached tmux session sized to your current terminal.

### Step 2: Attach and navigate

```bash
./scripts/tmux-screenshot.sh attach
```

Or directly: `tmux attach -t sidecar-screenshot`

Once attached:
1. Navigate to screens using number keys: **1=TD, 2=Git, 3=Files, 4=Conversations, 5=Worktrees**
2. Within a screen, use **j/k** (or arrow keys) to navigate up/down
3. Press **Enter** or **Space** to interact with items
4. Detach from tmux with **Ctrl+A D** (the tmux prefix in this session is Ctrl+A)

### Step 3: Capture the screenshot

```bash
./scripts/tmux-screenshot.sh capture sidecar-td
```

This captures the current view and:
1. Renders terminal output as PNG with proper fonts and colors (requires `termshot`)
2. Optionally creates HTML backup (if `aha` is installed)
3. Saves files to `docs/screenshots/`

### Step 4: Repeat or cleanup

Repeat steps 2-3 for additional screenshots, then:

```bash
./scripts/tmux-screenshot.sh stop
```

## Script Commands
|| Command | Description |
|---------|-------------|
| `start` | Start sidecar in a tmux session |
| `attach` | Attach to navigate (detach with Ctrl+A/B D) |
| `capture NAME` | Capture current view to `docs/screenshots/NAME.html` and `NAME.png` |
| `list` | List existing screenshots |
| `stop` | Quit sidecar and kill session |

## LLM Workflow

For AI agents, run `tmux attach -t sidecar-screenshot` in **interact mode** to navigate. The workflow:

1. `./scripts/tmux-screenshot.sh start`
2. `tmux attach -t sidecar-screenshot` (in interact mode) → navigate to screen using number keys → interact with content → `Ctrl+A D` to detach
3. `./scripts/tmux-screenshot.sh capture sidecar-{plugin}`
4. Repeat 2-3 for each plugin
5. `./scripts/tmux-screenshot.sh stop`

### Important: Tmux Setup for Agent Interaction

Before you can interact with sidecar via tmux, you must configure tmux to allow direct key input:

1. **Create or update `~/.tmux.conf`** with these settings:
   ```
   set -g mouse on
   set -g mode-keys vi
   unbind C-b
   set -g prefix C-a
   bind C-a send-prefix
   ```
   The critical settings are:
   - `set -g prefix C-a` - Sets the tmux prefix to Ctrl+A (used to detach with Ctrl+A D)
   - `set -g mode-keys vi` - Enables vi key bindings (j/k for navigation, etc.)

2. **Reload the config or restart tmux:**
   ```bash
   tmux source-file ~/.tmux.conf
   # or
   tmux kill-server  # kills all sessions
   ```
   After restarting tmux, the session will be ready for agent interaction.

3. **In interact mode:**
   - Press number keys to navigate screens (1-5)
   - Use `j/k` or arrow keys to scroll through content
   - Press `Enter` or `Space` to select/preview items
   - Press `Ctrl+A D` to detach (not `Ctrl+B D`)

**Screen navigation keys:**
- **1** = TD (task management)
- **2** = Git
- **3** = Files (file browser)
- **4** = Conversations
- **5** = Worktrees

**Within a screen:**
- **j/k** or arrow keys = navigate items
- **Enter/Space** = interact with selected item (select commit, preview file, etc.)
- **Ctrl+A D** = detach from tmux session

**Important for agents:** Tmux must be configured with `set -g prefix C-a` (see "Tmux Setup for Agent Interaction" above). Always use **Ctrl+A D** to detach from the tmux session.

## Fully Automated Screenshots with tui-send-keys.sh

For CI/CD or fully automated screenshot generation, use `scripts/tui-send-keys.sh` with keys files:

### Quick Start

```bash
# Capture td plugin screenshot
./scripts/tui-send-keys.sh "sidecar" -w Sidecar -o ./screenshots \
  --cols 200 --rows 50 \
  -f scripts/sequences/capture-sidecar-td-screenshot.keys
```

### Keys File Format

Keys files contain one key per line with optional comments:

```bash
# scripts/sequences/capture-sidecar-td-screenshot.keys
@sleep:500                # Wait for sidecar to be ready
Escape                    # Clear any focus/input mode
Escape                    # Ensure we're at top level
@sleep:100                # Brief pause
1                         # Switch to td pane (plugin index 0)
@sleep:300                # Wait for pane to render
@capture:sidecar-td.png   # Take screenshot
q                         # Quit
y                         # Confirm quit
```

### Available Actions

| Action | Description |
|--------|-------------|
| `@sleep:MS` | Wait MS milliseconds |
| `@wait:PATTERN` | Wait for text pattern to appear |
| `@capture` | Capture to stdout |
| `@capture:NAME.png` | Save as PNG (requires termshot) |
| `@capture:NAME.html` | Save as HTML (requires aha) |
| `@capture:NAME.txt` | Save as text with ANSI codes |
| `@capture:NAME` | Save all available formats |
| `@pause` | Wait for Enter (interactive debugging) |

### Key Learnings

**Plugin switching:** Number keys (1-5) switch plugins, but only work when not in a text input context. Always send `Escape` twice before plugin switching to clear any input focus.

**Terminal size:** Use `--cols` and `--rows` to control screenshot dimensions. Without these, the current terminal size is used.

**Timing:** Add `@sleep` commands between actions. The file browser and other plugins need time to render after navigation.

### Creating New Sequences

1. Create a new `.keys` file in `scripts/sequences/`
2. Start with `@sleep:500` to let the app initialize
3. Use `Escape Escape` to clear input state before plugin switching
4. Add `@sleep` between navigation and capture
5. End with `q` and `y` to quit cleanly

## Why Interactive? (Legacy Approach)

The interactive `tmux-screenshot.sh` approach is still useful when you need to:
- Navigate to specific screens
- Select commits, files, or other items to display interesting content
- Capture the full interactive state of sidecar

The interact mode provides a live PTY interface where you can press keys in real-time, making it ideal for getting the UI into the exact state you want before capturing.

## Viewing Captures

```bash
./scripts/tmux-screenshot.sh list       # List screenshots
open docs/screenshots/sidecar-td.html   # View HTML in browser
open docs/screenshots/sidecar-td.png    # View PNG image
```

Both HTML and PNG files are retained as artifacts. The PNG provides pixel-perfect rendering for documentation, while the HTML preserves the original ANSI-to-HTML conversion for reference.
