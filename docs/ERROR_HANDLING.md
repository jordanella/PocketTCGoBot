# Error Detection and Handling System

## Overview

The PocketTCGoBot includes a comprehensive error detection and handling system that allows routines to respond to game errors (popups, disconnections, unexpected screens) without interrupting the bot's flow. The system uses channel-based communication between a monitoring service and executing routines.

## Architecture

### Components

1. **ErrorMonitor** ([internal/monitor/error_monitor.go](internal/monitor/error_monitor.go))
   - Runs background goroutines that poll for errors at different intervals
   - Sends error events to routines via a buffered channel
   - Can be enabled/disabled during critical operations

2. **Error Types** ([internal/monitor/error_types.go](internal/monitor/error_types.go))
   - Defines error classifications, severities, and actions
   - Provides request/response types for two-way communication

3. **Helper Functions** ([internal/monitor/helpers.go](internal/monitor/helpers.go))
   - Utility functions for checking errors and sending responses
   - Convenience functions for common error handling patterns

4. **Action Integration** ([internal/actions/error_handling.go](internal/actions/error_handling.go))
   - Extends ActionBuilder with error-aware execution methods
   - Provides automatic and manual error checking capabilities

## Communication Pattern

The system uses a **two-way channel communication** pattern:

```
┌─────────────────┐                    ┌──────────────────┐
│  ErrorMonitor   │                    │  Executing       │
│  (Polling       │                    │  Routine         │
│   Goroutines)   │                    │                  │
└────────┬────────┘                    └────────▲─────────┘
         │                                      │
         │  1. Detect Error                     │
         │                                      │
         │  2. Create ErrorEvent                │
         │     with ResponseChan                │
         │                                      │
         ├──────── ErrorEvent ──────────────────┤
         │                                      │
         │                            3. Handle Error
         │                                      │
         │                            4. Create Response
         │                                      │
         ├────── ErrorResponse ─────────────────┤
         │                                      │
         │  5. Process Response                 │
         │                                      │
         ▼                                      │
    Continue Monitoring              Continue/Abort/Retry
```

### Error Flow

1. **Detection**: ErrorMonitor polls for errors at configurable intervals
2. **Event Creation**: When error detected, creates `ErrorEvent` with response channel
3. **Event Transmission**: Sends event to routine via main error channel
4. **Handling**: Routine receives event, executes recovery logic
5. **Response**: Routine sends `ErrorResponse` back via event's response channel
6. **Action**: Monitor and routine both act based on the response action

## Error Types

### ErrorType
```go
ErrorCommunication  // ADB disconnected, emulator crashed
ErrorStuck          // Bot stuck on same screen
ErrorNoResponse     // Game not responding
ErrorPopup          // Unexpected popup (level up, rewards, etc.)
ErrorMaintenance    // Maintenance mode
ErrorUpdate         // Update required
ErrorBanned         // Account banned/suspended
ErrorTitleScreen    // Returned to title screen unexpectedly
ErrorCustom         // Custom error type
```

### ErrorSeverity
```go
SeverityCritical    // Stop bot immediately
SeverityHigh        // Interrupt routine, handle, then decide
SeverityMedium      // Handle when convenient
SeverityLow         // Log only
```

### ErrorAction
```go
ActionContinue      // Continue routine execution
ActionAbort         // Abort current routine
ActionRetry         // Retry the current step
ActionStop          // Stop the bot entirely
ActionRestart       // Restart the routine from beginning
```

## Usage Examples

### 1. Basic Error Handling in a Routine

```go
func (l *Library) MyRoutine() error {
    // Define error handler
    errorHandler := func(event *monitor.ErrorEvent) monitor.ErrorResponse {
        startTime := time.Now()

        switch event.Type {
        case monitor.ErrorPopup:
            // Handle popup - click to dismiss
            l.bot.ADB().Click(139, 424)
            time.Sleep(1 * time.Second)

            return monitor.ErrorResponse{
                Handled:      true,
                Action:       monitor.ActionContinue,
                Message:      "Dismissed popup",
                RecoveryTime: time.Since(startTime),
            }

        case monitor.ErrorTitleScreen:
            return monitor.ErrorResponse{
                Handled:      true,
                Action:       monitor.ActionAbort,
                Message:      "Returned to title screen",
                RecoveryTime: time.Since(startTime),
            }

        default:
            return monitor.DefaultErrorHandler(event)
        }
    }

    // Execute with automatic error monitoring
    return l.Action().
        Click(100, 100).
        Sleep(1 * time.Second).
        Click(200, 200).
        ExecuteWithErrorMonitoring(2*time.Second, errorHandler)
}
```

### 2. Manual Error Checking Between Steps

```go
func (l *Library) ComplexRoutine() error {
    // Step 1
    if err := l.Action().Click(100, 100).Execute(); err != nil {
        return err
    }

    // Check for errors manually
    if err := l.Action().CheckForErrors(myErrorHandler); err != nil {
        return err
    }

    // Step 2
    return l.Action().Click(200, 200).Execute()
}
```

### 3. Using Default Error Handler

```go
func (l *Library) SimpleRoutine() error {
    // Use the built-in default handler
    return l.Action().
        Click(150, 150).
        Sleep(500 * time.Millisecond).
        ExecuteWithErrorMonitoring(1*time.Second, DefaultErrorHandler)
}
```

### 4. Loop with Error Checking

```go
func (l *Library) RepeatingTask() error {
    return l.Action().
        Click(100, 100).
        Sleep(500 * time.Millisecond).
        LoopWithErrorChecking(10, 1*time.Second, DefaultErrorHandler)
}
```

## Registering Error Handlers

Error detection logic is added by registering handlers with the ErrorMonitor:

```go
// In bot initialization or routine setup
bot.ErrorMonitor().RegisterHandler(monitor.ErrorHandler{
    ErrorType:     monitor.ErrorPopup,
    Priority:      monitor.PriorityHigh,
    Template:      templates.LevelUp,
    CheckInterval: 2 * time.Second,
    // Recovery function is called by YOUR code in the monitoring loop
    // The framework provides the structure, you implement the detection
})
```

## Implementing Error Detection

The ErrorMonitor provides the polling infrastructure, but **YOU implement the actual detection logic**. Here's the pattern:

```go
// In your error detection code (you'll add this to the monitor loops)
func (em *ErrorMonitor) monitorHighPriorityErrors() {
    // ... existing code ...

    for _, handler := range handlers {
        if handler.Priority == PriorityHigh {
            // YOUR DETECTION LOGIC HERE
            // Example:
            bot := em.bot.(*Bot) // Type assertion
            result, _ := bot.CV().FindTemplate(handler.Template, nil)

            if result != nil && result.Confidence > 0.8 {
                // Error detected! Send event
                em.TriggerError(
                    handler.ErrorType,
                    SeverityHigh,
                    "Popup detected",
                    handler.Template,
                )
            }
        }
    }
}
```

## Polling Intervals

The ErrorMonitor uses different polling frequencies based on priority:

- **Critical** (1 second): ADB connection, emulator crashes
- **High** (3 seconds): Popups, level ups, unexpected screens
- **Medium** (5 seconds): Warnings, minor issues
- **Low** (10 seconds): Health checks, statistics

## Enabling/Disabling Detection

You can temporarily disable error detection during critical operations:

```go
// Disable during account injection
bot.ErrorMonitor().DisableDetection()
// ... perform critical operation ...
bot.ErrorMonitor().EnableDetection()
```

## Helper Functions

### CheckForErrors
```go
// Non-blocking check
event := monitor.CheckForErrors(errorChan)
if event != nil {
    // Handle error
}
```

### CheckForErrorsWithContext
```go
// Check with context cancellation support
event, err := monitor.CheckForErrorsWithContext(ctx, errorChan)
```

### HandleError
```go
// Send response back to monitor
monitor.HandleError(event, true, monitor.ActionContinue, "Success", nil)
```

### HandleWithCallback
```go
// Handle with automatic response timing
monitor.HandleWithCallback(event, func(e *monitor.ErrorEvent) monitor.ErrorResponse {
    // Your handling logic
    return monitor.CreateSimpleResponse(monitor.ActionContinue, "Handled")
})
```

## Best Practices

1. **Always handle errors appropriately**
   - Critical errors (ADB disconnect) should stop the bot
   - Recoverable errors (popups) should continue after handling
   - Unexpected screens may require aborting the routine

2. **Use timeouts for error detection**
   - Don't poll too frequently (wastes CPU)
   - Don't poll too slowly (miss errors)
   - Adjust intervals based on error priority

3. **Provide clear error messages**
   - Include what was detected
   - Include what action was taken
   - Include recovery time for debugging

4. **Test error scenarios**
   - Manually trigger errors to test handling
   - Verify routines recover correctly
   - Ensure bot doesn't get stuck

5. **Log error events**
   - Track error frequency
   - Identify problematic screens
   - Improve detection over time

## Error Handling Patterns

### Pattern 1: Continue After Handling
```go
// For recoverable errors like popups
return monitor.ErrorResponse{
    Handled:      true,
    Action:       monitor.ActionContinue,
    Message:      "Popup dismissed",
    RecoveryTime: duration,
}
```

### Pattern 2: Abort Routine
```go
// For errors that invalidate the current routine
return monitor.ErrorResponse{
    Handled:      true,
    Action:       monitor.ActionAbort,
    Message:      "Unexpected screen, aborting",
    RecoveryTime: duration,
}
```

### Pattern 3: Stop Bot
```go
// For critical errors
return monitor.ErrorResponse{
    Handled:      false,
    Action:       monitor.ActionStop,
    Message:      "Critical error",
    Error:        err,
    RecoveryTime: duration,
}
```

## Integration with Existing Code

The error handling system is **already integrated** into the Bot struct:

- ErrorMonitor is created in `bot.Initialize()`
- ErrorMonitor is started automatically
- ErrorMonitor is stopped in `bot.Shutdown()`
- Accessible via `bot.ErrorMonitor()`

No changes to existing routines are required - error handling is **opt-in**. Legacy routines continue to work as-is.

## Future Enhancements

Potential improvements to consider:

1. **Error Statistics**: Track error frequency and types
2. **Adaptive Polling**: Adjust polling frequency based on error rate
3. **Error Priorities**: Queue errors by severity
4. **Recovery Strategies**: Pre-defined recovery action sequences
5. **Error Logging**: Persistent error logs for analysis
6. **Discord Notifications**: Alert on critical errors

## Files Reference

- [internal/monitor/error_monitor.go](internal/monitor/error_monitor.go) - Core monitoring service
- [internal/monitor/error_types.go](internal/monitor/error_types.go) - Type definitions
- [internal/monitor/helpers.go](internal/monitor/helpers.go) - Helper functions
- [internal/actions/error_handling.go](internal/actions/error_handling.go) - ActionBuilder integration
- [internal/actions/error_aware_example.go](internal/actions/error_aware_example.go) - Usage examples
- [internal/bot/bot.go](internal/bot/bot.go) - Bot integration

## Summary

The error handling system provides:

- ✅ **Non-blocking error detection** via polling goroutines
- ✅ **Two-way communication** between monitor and routines
- ✅ **Flexible error handling** with custom handlers
- ✅ **Automatic and manual** error checking modes
- ✅ **Graceful degradation** - routines decide how to respond
- ✅ **Opt-in design** - no changes to existing code required

You now have the **architecture and communication infrastructure** in place. The next step is implementing the actual error detection logic based on your game's specific error conditions.
