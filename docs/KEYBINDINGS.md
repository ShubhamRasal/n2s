# N2S Keybindings

## Global Keybindings

| Key | Action |
|-----|--------|
| `Ctrl+C` | Quit application |
| `?` | Show help |
| `c` | Switch to context selection |
| `Esc` | Go back / Cancel |

## Context Selection View

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate contexts |
| `j/k` | Navigate contexts (Vim-style) |
| `Enter` | Connect to selected context |
| `q` | Quit |

## Stream List View

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate streams |
| `j/k` | Navigate streams (Vim-style) |
| `Enter` | View stream details |
| `/` | **Filter streams (opens search box)** |
| `d` | **Describe Stream** |
| `x` | Delete stream (with confirmation) |
| `p` | Purge stream messages (with confirmation) |
| `m` | View messages in stream |
| `r` | Refresh |
| `Esc` | Clear filter (if active) or back to context selection |

### Search Mode
| Key | Action |
|-----|--------|
| Type | Filter streams in real-time |
| `Enter` | Close search (keep filter active) |
| `Esc` | Clear filter and close search |

## Stream Detail View

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate consumers |
| `j/k` | Navigate consumers (Vim-style) |
| `Enter` | View consumer details |
| `d` | **Describe Stream** |
| `m` | View messages in stream |
| `x` | Delete selected consumer |
| `r` | Refresh |
| `Esc` | Back to stream list |

## Describe View

| Key | Action |
|-----|--------|
| `r` | Refresh description |
| `Esc` | Back to stream detail |

## Consumer Detail View

| Key | Action |
|-----|--------|
| `d` | Delete consumer (with confirmation) |
| `r` | Refresh |
| `Esc` | Back to stream detail |

## Message Browser View

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate messages |
| `j/k` | Navigate messages (Vim-style) |
| `Enter` | View message detail |
| `r` | Refresh |
| `Esc` | Back |

## Help View

| Key | Action |
|-----|--------|
| `Esc` | Close help |

## Metrics Graphs View

| Key | Action |
|-----|--------|
| `r` | Refresh metrics |
| `Esc` | Back to stream list |

## Tips

- **Read-only mode**: Use `--read-only` flag to prevent deletions in production
- **Auto-refresh**: Views automatically refresh every 2 seconds
- **Manual refresh**: Press `r` in any view to force an immediate refresh
- **Quick context switch**: Press `c` from anywhere to switch contexts
- **Confirmation dialogs**: Destructive actions (delete, purge) require confirmation

