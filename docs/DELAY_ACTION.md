# Delay Action

The `Delay` action provides configurable timing between actions based on the bot's configuration settings.

## Overview

Unlike `Sleep` which takes an absolute duration, `Delay` multiplies the configured delay value by a multiplier. This makes it easy to scale timing throughout your bot without hardcoding durations.

## Usage

```go
// Basic usage - delay for 1x the configured delay
l.Action().
    Click(100, 200).
    Delay(1).  // Wait for config.Delay milliseconds
    Click(300, 400).
    Execute()

// Longer delay - delay for 3x the configured delay
l.Action().
    Click(100, 200).
    Delay(3).  // Wait for config.Delay * 3 milliseconds
    Click(300, 400).
    Execute()
```

## How It Works

The `Delay` action:
1. Reads the `Delay` value from the bot configuration (in milliseconds)
2. Multiplies it by the provided multiplier
3. Sleeps for that duration

### Example

If `config.Delay = 100` (milliseconds):

- `Delay(1)` = 100ms delay
- `Delay(2)` = 200ms delay
- `Delay(3)` = 300ms delay
- `Delay(10)` = 1000ms (1 second) delay

## When to Use Delay vs Sleep

### Use `Delay(multiplier)`

- When you want timing to scale with the bot's speed settings
- For consistent pacing throughout your bot
- When different users might have different performance requirements
- For delays that should adapt to emulator responsiveness

```go
// Good: Scales with configuration
l.Action().
    Click(100, 200).
    Delay(2).        // 2x configured delay
    Screenshot().
    Delay(1).        // 1x configured delay
    Execute()
```

### Use `Sleep(duration)`

- When you need a specific, exact duration
- For waiting on external processes (network, animations, etc.)
- When the delay should NOT scale with configuration

```go
// Good: Specific durations for external constraints
l.Action().
    Click(100, 200).
    Sleep(500 * time.Millisecond).  // Always wait 500ms for animation
    Screenshot().
    Sleep(2 * time.Second).          // Always wait 2s for network
    Execute()
```

## Common Patterns

### Quick Successive Actions

```go
l.Action().
    Click(x1, y1).
    Delay(1).    // Brief pause
    Click(x2, y2).
    Delay(1).
    Click(x3, y3).
    Execute()
```

### Waiting for UI Updates

```go
l.Action().
    Click(openMenuX, openMenuY).
    Delay(3).    // Wait for menu animation (3x base delay)
    Click(menuItemX, menuItemY).
    Execute()
```

### Combining with Other Actions

```go
l.Action().
    Screenshot().
    Delay(1).
    Click(buttonX, buttonY).
    Delay(2).                // Wait for action to complete
    Screenshot().            // Capture result
    Execute()
```

## Configuration

The delay multiplier is based on the `Delay` setting in the bot configuration:

```go
config := &bot.Config{
    Delay: 100,  // Base delay in milliseconds
    // ... other settings
}
```

Users can adjust this value to make the bot faster or slower:
- **Fast**: `Delay: 50` (50ms base)
- **Normal**: `Delay: 100` (100ms base)
- **Slow**: `Delay: 200` (200ms base)
- **Very Slow**: `Delay: 500` (500ms base)

## Performance Considerations

- `Delay(0)` is valid and will not sleep at all
- Multipliers can be any positive integer
- Negative multipliers are not validated (will result in no delay)
- Very large multipliers (>100) should be avoided - use `Sleep` instead for long waits

## Examples from the Codebase

### Opening a Pack

```go
l.Action().
    Click(openPackX, openPackY).
    Delay(5).    // Wait for pack opening animation (5x base)
    Screenshot().
    Execute()
```

### Navigating Menus

```go
l.Action().
    Click(menuButtonX, menuButtonY).
    Delay(2).    // Wait for menu to appear
    Click(submenuX, submenuY).
    Delay(2).    // Wait for submenu
    Click(optionX, optionY).
    Execute()
```

### Form Input

```go
l.Action().
    Click(inputFieldX, inputFieldY).
    Delay(1).
    Input("username").
    Delay(1).
    Click(nextFieldX, nextFieldY).
    Delay(1).
    Input("password").
    Execute()
```
