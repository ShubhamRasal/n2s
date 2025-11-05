# N2S Architecture

## Overview

N2S is built with a clean architecture separating concerns into distinct layers:

```
┌─────────────────────────────────────────┐
│            cmd/n2s (Entry)              │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│         internal/app (Orchestrator)      │
└──────────────┬──────────────────────────┘
               │
       ┌───────┴────────┐
       │                │
┌──────▼─────┐    ┌────▼──────┐
│   config   │    │   nats    │
└────────────┘    └────┬──────┘
                       │
                  ┌────▼──────┐
                  │  models   │
                  └───────────┘
```

## Layer Responsibilities

### 1. Entry Point (`cmd/n2s`)
- CLI argument parsing using Cobra
- Application initialization
- Error handling and exit codes

### 2. Application Layer (`internal/app`)
- Orchestrates initialization of all components
- Connects configuration, NATS client, and UI
- Lifecycle management

### 3. Configuration (`internal/config`)
- Context management (multiple NATS servers)
- Configuration file I/O
- Default configuration creation
- Refresh interval management

### 4. NATS Client (`internal/nats`)
- NATS server connection management
- JetStream operations:
  - Stream operations (list, get, delete, purge)
  - Consumer operations (list, get, delete)
  - Message operations (list, get details)
- Connection health monitoring
- Automatic reconnection

### 5. Data Models (`internal/models`)
- Stream, Consumer, Message structures
- Clean data transfer objects
- Type safety for API contracts

### 6. UI Layer (`internal/ui`)

#### UI Manager
- Central coordinator for all views
- Page navigation
- Modal management
- Auto-refresh ticker (2s interval)
- Global keybinding handler

#### Views
Each view follows the same pattern:
- **Refresh()** - Updates data from NATS
- **GetPrimitive()** - Returns tview primitive
- **setupKeybindings()** - Configures input handling

Views:
- **ContextView** - Server selection
- **StreamListView** - Stream overview
- **StreamDetailView** - Stream info + consumers
- **ConsumerDetailView** - Consumer metrics
- **MessageView** - Message browser
- **HelpView** - Keybinding reference

#### Components
Reusable UI components:
- **Header** - Status bar
- **Footer** - Keybinding hints
- **Modal** - Confirmation/error dialogs

## Data Flow

### Stream Listing
```
User → StreamListView → UIManager → NATS Client → NATS Server
                                         ↓
                        models.Stream ←─┘
                              ↓
                        StreamListView.updateTable()
                              ↓
                          tview.Table
```

### Navigation Flow
```
ContextView → Select Context → StreamListView
                                     ↓
                               Enter on Stream
                                     ↓
                              StreamDetailView
                                     ↓
                             Enter on Consumer
                                     ↓
                           ConsumerDetailView
```

## Design Decisions

### 1. Why tview?
- Terminal UI framework with rich widgets
- Active maintenance
- Excellent table and modal support
- Mouse support

### 2. Why not embed config in binary?
- Allows users to add contexts without rebuilding
- Standard XDG config location (~/.config/n9s/)
- Easy to share/version control

### 3. Auto-refresh Strategy
- Ticker-based (2s default, configurable)
- Per-view refresh to avoid unnecessary updates
- Only refreshes the active view

### 4. Read-only Mode
- Safety for production environments
- Prevents accidental deletions/purges
- Clearly indicated in header

### 5. Error Handling
- Errors shown as modals (non-blocking)
- Graceful degradation (show what's available)
- Connection errors don't crash the app

## Concurrency Model

- **Main goroutine**: UI event loop (tview.Application)
- **Ticker goroutine**: Auto-refresh every 2s
- All UI updates via `app.QueueUpdateDraw()` (thread-safe)

## Testing Strategy

### Unit Tests (Future)
- NATS client operations
- Config management
- Data model validation

### Integration Tests
- Docker-based NATS environment
- Automated setup with test data
- Makefile targets for easy testing

### Manual Testing
- Interactive TUI testing
- Real NATS server testing
- Context switching validation

## Future Enhancements

1. **Stream Creation**: Add ability to create streams via TUI
2. **Consumer Creation**: Add ability to create consumers
3. **Message Publishing**: Publish messages to streams
4. **Search/Filter**: Filter streams/consumers by name
5. **Metrics Graphs**: Visual metrics over time
6. **Multiple Servers**: Monitor multiple contexts simultaneously
7. **Export**: Export messages to JSON/CSV
8. **Logging**: Debug logging to file

## Performance Considerations

1. **Message Limit**: Only fetch last 100 messages by default
2. **Lazy Loading**: Views only refresh when visible
3. **Connection Pooling**: Reuse NATS connection
4. **Minimal Allocations**: Reuse table cells where possible

## Security Considerations

1. **Read-only Mode**: Safe for production monitoring
2. **No Credentials Storage**: Uses NATS connection URL
3. **Config File Permissions**: 0644 (readable by owner)
4. **No Network Exposure**: Client-only application

