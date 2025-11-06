# Timeout Action

## Overview

The `Timeout` action sets a maximum runtime for an action sequence. If the actions exceed this timeout, execution is aborted with a timeout error.

## Usage

```go
// Set a 30 second timeout for the action sequence
l.Action().
    Timeout(30).  // 30 seconds max runtime
    Click(100, 200).
    Delay(2).
    Screenshot().
    Click(300, 400).
    Execute()
```

## How It Works

The `Timeout` action:
1. Sets a maximum duration for the entire action sequence
2. Uses Go's context with timeout under the hood
3. If actions take longer than the timeout, execution is aborted
4. Triggers an `ErrorTimeout` event that can be handled by error handlers

### Example

```go
// This will timeout if the loop takes more than 60 seconds
l.Action().
    Timeout(60).  // 60 second maximum
    UntilTemplateAppears(templates.ShopButton,
        l.Action().
            Click(100, 200).
            Delay(3),
        100).  // Try up to 100 times, but abort at 60 seconds
    Execute()
```

## When to Use Timeout

### Use `Timeout(seconds)`

- When you have a loop that might run indefinitely
- For operations that interact with external systems (network, etc.)
- To prevent stuck routines from running forever
- When you want a safety net for complex action sequences

```go
// Good: Protects against infinite loops
l.Action().
    Timeout(120).  // 2 minute max
    UntilTemplateAppears(template,
        l.Action().
            Click(x, y).
            Delay(5),
        1000).  // Could loop many times
    Execute()
```

### Don't Use Timeout

- For simple, fast operations (unnecessary overhead)
- When you know exact duration (use Sleep/Delay instead)
- For operations that must complete (use longer timeout)

## Timeout Values

### Recommended Timeouts

- **Quick actions**: 10-30 seconds
  ```go
  l.Action().
      Timeout(10).
      Click(x, y).
      Delay(2).
      Execute()
  ```

- **Navigation sequences**: 30-60 seconds
  ```go
  l.Action().
      Timeout(60).
      // Navigate through menus
      Execute()
  ```

- **Pack opening**: 60-120 seconds
  ```go
  l.Action().
      Timeout(120).
      // Open pack, detect cards
      Execute()
  ```

- **Long operations**: 120-300 seconds (2-5 minutes)
  ```go
  l.Action().
      Timeout(300).
      // Complete mission, battle, etc.
      Execute()
  ```

## Combining with Error Handlers

```go
// Handle timeout errors gracefully
timeoutHandler := func(event *monitor.ErrorEvent) monitor.ErrorResponse {
    if event.Type == monitor.ErrorTimeout {
        log.Warn("Operation timed out, will retry")
        return monitor.ErrorResponse{
            Handled:      true,
            Action:       monitor.ActionAbort,
            Message:      "Timed out, safe to retry",
        }
    }
    return monitor.DefaultErrorHandler(event)
}

err := l.Action().
    Timeout(30).
    WithErrorHandler(timeoutHandler).
    // ... actions ...
    Execute()

// Retry logic
if err != nil {
    log.Info("First attempt failed, retrying...")
    err = l.Action().
        Timeout(60).  // Longer timeout for retry
        // ... same actions ...
        Execute()
}
```

## Comparison: Timeout vs WithTimeout

### `Timeout(seconds int)`

```go
// Convenience method - takes seconds as integer
l.Action().
    Timeout(30).  // 30 seconds
    Execute()
```

### `WithTimeout(duration time.Duration)`

```go
// Lower-level method - takes time.Duration
l.Action().
    WithTimeout(30 * time.Second).  // 30 seconds
    Execute()

// More precise timing
l.Action().
    WithTimeout(2500 * time.Millisecond).  // 2.5 seconds
    Execute()
```

**Recommendation**: Use `Timeout(seconds)` for most cases. Use `WithTimeout()` when you need millisecond precision.

## Common Patterns

### Retry with Increasing Timeout

```go
func doActionWithRetry(l *actions.ActionLibrary, maxRetries int) error {
    baseTimeout := 30

    for attempt := 0; attempt < maxRetries; attempt++ {
        timeout := baseTimeout * (attempt + 1)  // 30s, 60s, 90s...

        err := l.Action().
            Timeout(timeout).
            // ... actions ...
            Execute()

        if err == nil {
            return nil  // Success
        }

        log.Warnf("Attempt %d failed, retrying with longer timeout", attempt+1)
    }

    return fmt.Errorf("failed after %d attempts", maxRetries)
}
```

### Different Timeouts for Different Sections

```go
// Part 1: Quick navigation (30s)
err := l.Action().
    Timeout(30).
    Click(menuX, menuY).
    Delay(2).
    Click(submenuX, submenuY).
    Execute()

if err != nil {
    return err
}

// Part 2: Long operation (120s)
err = l.Action().
    Timeout(120).
    Click(startX, startY).
    UntilTemplateAppears(doneTemplate,
        l.Action().Delay(5),
        100).
    Execute()

return err
```

### Conditional Timeout

```go
func doAction(l *actions.ActionLibrary, isPremium bool) error {
    timeout := 30
    if isPremium {
        timeout = 120  // Premium users get longer timeout
    }

    return l.Action().
        Timeout(timeout).
        // ... actions ...
        Execute()
}
```

## Debugging Timeouts

### Log What Timed Out

```go
err := l.Action().
    Timeout(60).
    // ... actions ...
    Execute()

if err != nil {
    if strings.Contains(err.Error(), "timeout") {
        log.Error("Action timed out after 60 seconds")
        log.Error("Consider increasing timeout or optimizing actions")
    }
}
```

### Track Timing

```go
start := time.Now()

err := l.Action().
    Timeout(60).
    // ... actions ...
    Execute()

duration := time.Since(start)
log.Infof("Action took %v (timeout was 60s)", duration)

if duration > 50*time.Second {
    log.Warn("Action took nearly the full timeout!")
}
```

## Error Handling

When a timeout occurs:

1. **Context is cancelled**: All ongoing operations are interrupted
2. **ErrorTimeout event**: Sent to error handlers if enabled
3. **Execute() returns error**: The error indicates timeout
4. **Cleanup happens**: Deferred cleanup functions still run

### Handling Timeout Errors

```go
err := l.Action().
    Timeout(30).
    WithErrorHandler(monitor.GetDefaultHandler()).
    // ... actions ...
    Execute()

if err != nil {
    // Timeout errors are automatically handled by default handler
    // Result: ActionAbort (routine is aborted gracefully)

    // You can check if it was a timeout
    if errors.Is(err, context.DeadlineExceeded) {
        log.Warn("Timed out - this is expected sometimes")
    }
}
```

## Best Practices

1. **Always set timeout for loops**: Prevents infinite loops
2. **Be generous with timeouts**: Better to have longer timeout than false failures
3. **Log timeout occurrences**: Track how often timeouts happen
4. **Review timeout patterns**: If same operation times out repeatedly, investigate
5. **Use appropriate values**: Don't use 1000s timeout for 5s operation
6. **Test with slower devices**: Ensure timeout works on minimum spec hardware
7. **Combine with error handlers**: Let handlers manage timeout recovery
8. **Document why**: Comment why you chose specific timeout value

## Performance Considerations

- Timeouts have minimal overhead (just context management)
- No performance penalty for operations that complete quickly
- Cleanup happens immediately when timeout is reached
- Safe to use liberally for safety

## Examples from Codebase

### Wonder Pick with Timeout

```go
func DoWonderPick(l *actions.ActionLibrary) error {
    return l.Action().
        Timeout(60).  // 1 minute max
        Click(wonderPickX, wonderPickY).
        Delay(5).  // Wait for animation
        Click(selectCardX, selectCardY).
        Delay(3).
        UntilTemplateAppears(templates.CardRevealed,
            l.Action().Delay(2),
            20).
        Execute()
}
```

### Pack Opening with Timeout

```go
func OpenPack(l *actions.ActionLibrary) error {
    return l.Action().
        Timeout(120).  // 2 minutes for pack opening
        Click(openPackX, openPackY).
        Delay(10).  // Long animation
        // Wait for card reveal animations
        UntilTemplateAppears(templates.AllCardsRevealed,
            l.Action().Delay(5),
            30).
        Execute()
}
```

### Navigation with Timeout

```go
func NavigateToShop(l *actions.ActionLibrary) error {
    return l.Action().
        Timeout(45).  // 45 seconds for navigation
        Click(menuX, menuY).
        Delay(2).
        UntilTemplateAppears(templates.ShopButton,
            l.Action().
                Swipe(swipeDown).
                Delay(1),
            10).
        Click(shopX, shopY).
        Execute()
}
```
