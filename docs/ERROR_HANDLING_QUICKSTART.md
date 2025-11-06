# Error Handling Quick Start Guide

## TL;DR

Error checking is **intrinsic to Execute()**. Just add `.WithErrorHandler()` or `.WithErrorChecking()` before `.Execute()`!

## Quick Examples

### Use Error Handling in a Routine

```go
func (l *Library) MyRoutine() error {
    // Define handler
    handler := func(event *monitor.ErrorEvent) monitor.ErrorResponse {
        switch event.Type {
        case monitor.ErrorPopup:
            l.bot.ADB().Click(139, 424) // Dismiss
            return monitor.CreateSimpleResponse(monitor.ActionContinue, "Dismissed popup")
        case monitor.ErrorTitleScreen:
            return monitor.CreateSimpleResponse(monitor.ActionAbort, "Title screen")
        default:
            return defaultErrorHandler(event)
        }
    }

    // Just add .WithErrorHandler() before .Execute()!
    return l.Action().
        Click(100, 100).
        Sleep(1 * time.Second).
        Click(200, 200).
        WithErrorHandler(handler). // Error checking enabled
        Execute()                   // Checks for errors automatically
}
```

### Use Default Error Handler

```go
func (l *Library) SimpleRoutine() error {
    // Use built-in default handler with custom interval
    return l.Action().
        Click(100, 100).
        Sleep(1 * time.Second).
        WithErrorChecking(1 * time.Second). // Check every second
        Execute()
}
```

### Manually Check Between Steps

```go
func (l *Library) MyRoutine() error {
    // Step 1 - no error checking
    l.Action().Click(100, 100).Execute()

    // Manually check for errors
    if err := l.Action().CheckForErrors(myHandler); err != nil {
        return err
    }

    // Step 2 - with error checking
    l.Action().Click(200, 200).WithErrorChecking(500*time.Millisecond).Execute()
}
```

### Disable Error Checking

```go
func (l *Library) CriticalOperation() error {
    // Explicitly disable for critical operations
    return l.Action().
        Click(100, 100).
        DisableErrorChecking().
        Execute()
}
```

## Implementing Error Detection

Add your detection logic to the monitor loops in [error_monitor.go](internal/monitor/error_monitor.go):

```go
func (em *ErrorMonitor) monitorHighPriorityErrors() {
    // ... existing ticker code ...

    for _, handler := range handlers {
        if handler.Priority == PriorityHigh {
            // YOUR CODE HERE
            // Example: Check for level up popup
            bot := em.bot.(BotInterface)
            if bot.CV().TemplateExists(templates.LevelUp) {
                em.TriggerError(
                    ErrorPopup,
                    SeverityHigh,
                    "Level up popup detected",
                    templates.LevelUp,
                )
            }
        }
    }
}
```

## Register Error Handlers

```go
// In bot setup or routine start
bot.ErrorMonitor().RegisterHandler(monitor.ErrorHandler{
    ErrorType:     monitor.ErrorPopup,
    Priority:      monitor.PriorityHigh,
    Template:      templates.LevelUp,
    CheckInterval: 2 * time.Second,
})
```

## Error Response Patterns

### Continue (Level Up, Rewards)
```go
return monitor.ErrorResponse{
    Handled: true,
    Action:  monitor.ActionContinue,
    Message: "Handled and continuing",
}
```

### Abort (Title Screen, Unexpected State)
```go
return monitor.ErrorResponse{
    Handled: true,
    Action:  monitor.ActionAbort,
    Message: "Must restart routine",
}
```

### Stop (Critical Errors)
```go
return monitor.ErrorResponse{
    Handled: false,
    Action:  monitor.ActionStop,
    Error:   err,
    Message: "Critical error, stopping bot",
}
```

## Common Scenarios

### Scenario 1: Level Up Popup
**Detection**: Monitor checks for level up template every 3s
**Handling**: Click to dismiss, send `ActionContinue`
**Result**: Routine continues seamlessly

### Scenario 2: Returned to Title Screen
**Detection**: Monitor detects title screen template
**Handling**: Send `ActionAbort`
**Result**: Routine exits, main loop restarts it

### Scenario 3: ADB Disconnected
**Detection**: Monitor fails to communicate with ADB
**Handling**: Send `ActionStop`
**Result**: Bot stops entirely

## Architecture Summary

```
ErrorMonitor (3 goroutines)
    ├─> Critical (1s) - ADB, crashes
    ├─> High (3s)     - Popups, level ups
    └─> Medium (5s)   - Warnings, health

    Detects error → Creates ErrorEvent → Sends to routine
                                              ↓
    Waits for response ← ErrorResponse ← Routine handles
```

## Fluent API Methods

| Method | Description | Default Interval |
|--------|-------------|------------------|
| `.WithErrorHandler(handler)` | Enable with custom handler | 1 second |
| `.WithErrorChecking(interval)` | Enable with default handler | User-specified |
| `.DisableErrorChecking()` | Explicitly disable | N/A |
| `.CheckForErrors(handler)` | Manual check (not fluent) | N/A |

## Files to Know

- [ERROR_HANDLING.md](ERROR_HANDLING.md) - Full documentation
- [internal/monitor/error_monitor.go](internal/monitor/error_monitor.go) - Add detection here
- [internal/monitor/error_types.go](internal/monitor/error_types.go) - Error types
- [internal/monitor/helpers.go](internal/monitor/helpers.go) - Utility functions
- [internal/actions/builder.go](internal/actions/builder.go) - Intrinsic error checking
- [internal/actions/error_handling.go](internal/actions/error_handling.go) - Manual checking
- [internal/actions/error_aware_example.go](internal/actions/error_aware_example.go) - Examples

## Next Steps

1. **Identify error conditions** you want to detect (popups, screens, states)
2. **Add templates** for those conditions if needed
3. **Implement detection** in monitor loops (TODO sections)
4. **Use `.WithErrorHandler()`** in routines that need error checking
5. **Test** error scenarios manually

## Key Points

- ✅ Error checking is **opt-in via fluent API**
- ✅ Default handler included
- ✅ Custom handlers per action
- ✅ Configurable check intervals
- ✅ Can be disabled for critical operations
- ✅ No breaking changes to existing code
- ⏳ You implement the actual detection logic
- ⏳ You decide which routines use error handling

## Questions?

See [ERROR_HANDLING.md](ERROR_HANDLING.md) for comprehensive documentation.
