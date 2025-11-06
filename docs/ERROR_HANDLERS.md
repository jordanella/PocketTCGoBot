# Default Error Handlers

## Overview

The bot includes default error handlers for common, diagnosable error conditions. These handlers provide smart defaults for handling errors without requiring custom implementation for every error type.

## Error Types and Handlers

### Communication Errors (`ErrorCommunication`)

**What it is**: ADB connection lost, emulator crashed, or device disconnected

**Handler Response**:
- **Handled**: No (cannot recover automatically)
- **Action**: `ActionStop` - Stop the bot entirely
- **Rationale**: Cannot continue without ADB connection

**Example Scenarios**:
- ADB server crashed
- Emulator window closed
- USB debugging disabled
- Device disconnected

**Recommended Recovery**:
- Check ADB connection status
- Restart ADB server
- Restart emulator if needed
- Verify device is connected

---

### Maintenance Mode (`ErrorMaintenance`)

**What it is**: Game server is in maintenance mode

**Handler Response**:
- **Handled**: Yes (gracefully handled)
- **Action**: `ActionAbort` - Abort current routine
- **Rationale**: Cannot proceed during maintenance, but can retry later

**Example Scenarios**:
- Scheduled game maintenance
- Emergency server maintenance
- Server updates in progress

**Recommended Recovery**:
- Wait for maintenance to complete
- Implement exponential backoff retry logic
- Check game status API if available

---

### Update Required (`ErrorUpdate`)

**What it is**: Game requires an update to continue

**Handler Response**:
- **Handled**: No (requires manual intervention)
- **Action**: `ActionStop` - Stop the bot entirely
- **Rationale**: Bot cannot update the game automatically

**Example Scenarios**:
- Forced client update
- Version mismatch with server
- New game version available

**Recommended Recovery**:
- Update game through Play Store
- Verify game version matches requirements
- Update bot templates/logic if game changed significantly

---

### Account Banned (`ErrorBanned`)

**What it is**: Account has been banned or suspended

**Handler Response**:
- **Handled**: No (account is unusable)
- **Action**: `ActionStop` - Stop the bot
- **Rationale**: Cannot use banned account

**Example Scenarios**:
- ToS violation detected
- Suspicious activity flagged
- Account temporarily suspended

**Recommended Recovery**:
- Mark account as banned in database
- Remove from active account pool
- Review ban reason if available
- DO NOT retry with same account

---

### Title Screen (`ErrorTitleScreen`)

**What it is**: Unexpectedly returned to title screen

**Handler Response**:
- **Handled**: Yes (can recover)
- **Action**: `ActionAbort` - Abort routine, caller can restart
- **Rationale**: Session expired or lost, need to re-login

**Example Scenarios**:
- Session timeout
- Server kicked player
- Connection lost mid-game
- Game crashed and restarted

**Recommended Recovery**:
- Re-login from title screen
- Restart the aborted routine
- Verify account credentials
- Check for connection issues

---

### No Response (`ErrorNoResponse`)

**What it is**: Game appears frozen or unresponsive

**Handler Response**:
- **Handled**: No (requires external intervention)
- **Action**: `ActionAbort` - Abort routine
- **Rationale**: Game may need restart

**Example Scenarios**:
- Game frozen on loading screen
- Infinite loading loop
- Memory leak causing hang
- Graphics driver issue

**Recommended Recovery**:
- Force close and restart game
- Restart emulator if problem persists
- Check system resources (RAM, CPU)
- Consider reducing graphics settings

---

### Timeout (`ErrorTimeout`)

**What it is**: Action sequence exceeded maximum runtime

**Handler Response**:
- **Handled**: Yes (safety mechanism working)
- **Action**: `ActionAbort` - Abort for safety
- **Rationale**: Prevent infinite loops and stuck routines

**Example Scenarios**:
- Routine stuck in loop
- Unexpected game state
- Network request taking too long
- Animation longer than expected

**Recommended Recovery**:
- Review why timeout occurred
- Increase timeout if legitimate
- Fix logic causing infinite loop
- Add better state detection

---

### Popup (`ErrorPopup`)

**What it is**: Unexpected popup detected (level up, rewards, etc.)

**Handler Response**:
- **Handled**: No (requires custom logic)
- **Action**: `ActionContinue` - Continue execution
- **Rationale**: Popup handling is complex and context-specific

**Example Scenarios**:
- Level up notification
- Daily reward popup
- Event notification
- Friend request

**Recommended Recovery**:
- Implement custom popup handler
- Use template matching to detect popup type
- Click appropriate dismiss button
- Update routine to expect popup

---

### Stuck (`ErrorStuck`)

**What it is**: Bot stuck on same screen for extended period

**Handler Response**:
- **Handled**: No (requires analysis)
- **Action**: `ActionAbort` - Abort routine
- **Rationale**: Stuck detection is complex, needs custom recovery

**Example Scenarios**:
- Navigation failed
- Button not responding
- Wrong screen reached
- State machine confusion

**Recommended Recovery**:
- Implement custom recovery logic
- Use screen history to diagnose
- Navigate to known good state (home)
- Review navigation flow logic

## Using Default Handlers

### Basic Usage

```go
import "jordanella.com/pocket-tcg-go/internal/monitor"

// Use the default comprehensive handler
handler := monitor.GetDefaultHandler()

// Use with ActionBuilder
err := l.Action().
    WithErrorHandler(handler).
    Click(100, 200).
    Delay(2).
    Click(300, 400).
    Execute()
```

### Per-Error-Type Handlers

```go
// Get handler for specific error type
commHandler := monitor.GetHandlerForType(monitor.ErrorCommunication)
maintHandler := monitor.GetHandlerForType(monitor.ErrorMaintenance)

// Use type-specific handler
err := l.Action().
    WithErrorHandler(commHandler).
    // ... actions ...
    Execute()
```

### Custom Handler with Default Fallback

```go
func myCustomHandler(event *monitor.ErrorEvent) monitor.ErrorResponse {
    // Handle custom errors
    if event.Type == monitor.ErrorPopup {
        // Custom popup handling logic
        return handleMyPopup(event)
    }

    // Fall back to default for other errors
    return monitor.DefaultErrorHandler(event)
}

// Use custom handler
err := l.Action().
    WithErrorHandler(myCustomHandler).
    // ... actions ...
    Execute()
```

## Handler Decision Matrix

| Error Type | Handled | Action | Stops Bot | Requires Manual |
|-----------|---------|---------|-----------|-----------------|
| Communication | No | Stop | Yes | Yes |
| Maintenance | Yes | Abort | No | No (wait) |
| Update | No | Stop | Yes | Yes |
| Banned | No | Stop | Yes | Yes |
| TitleScreen | Yes | Abort | No | No (re-login) |
| NoResponse | No | Abort | No | Maybe |
| Timeout | Yes | Abort | No | No (review) |
| Popup | No | Continue | No | No (custom) |
| Stuck | No | Abort | No | No (custom) |

## Integration with Database

When errors occur, you should log them to the database:

```go
handler := func(event *monitor.ErrorEvent) monitor.ErrorResponse {
    // Get default response
    response := monitor.DefaultErrorHandler(event)

    // Log to database
    errorID, _ := db.LogError(
        &accountID,
        &activityID,
        event.Type.String(),
        event.Severity.String(),
        event.Message,
        nil, nil, nil, nil,
    )

    // Mark as recovered if handled
    if response.Handled {
        db.MarkErrorRecovered(errorID, response.Message, int(response.RecoveryTime.Milliseconds()))
    }

    return response
}
```

## Best Practices

1. **Always log errors**: Use database logging even with default handlers
2. **Monitor recovery rates**: Track which errors are successfully recovered
3. **Custom handlers for common errors**: Implement custom logic for popups and stuck states
4. **Graceful degradation**: Let handlers abort routines rather than crash
5. **Exponential backoff**: Wait progressively longer after repeated errors
6. **Alert on critical errors**: Notify when communication or ban errors occur
7. **Review timeout errors**: Investigate why timeouts occurred
8. **Test error handlers**: Simulate errors to verify handler behavior

## Testing Error Handlers

```go
func TestMyErrorHandling(t *testing.T) {
    event := &monitor.ErrorEvent{
        Type:       monitor.ErrorMaintenance,
        Severity:   monitor.SeverityHigh,
        Message:    "Test maintenance",
        DetectedAt: time.Now(),
    }

    response := monitor.HandleMaintenanceError(event)

    if !response.Handled {
        t.Error("Expected maintenance to be handled")
    }

    if response.Action != monitor.ActionAbort {
        t.Error("Expected abort action")
    }
}
```

## Future Enhancements

Potential improvements to default handlers:

- **Smart retry logic**: Automatic retry with exponential backoff
- **Popup detection**: Template-based popup detection and dismissal
- **Stuck recovery**: Automatic navigation to home screen
- **Health monitoring**: Track error frequency to predict issues
- **Auto-rotation**: Switch to different account on ban
- **Notification**: Discord/webhook alerts for critical errors
