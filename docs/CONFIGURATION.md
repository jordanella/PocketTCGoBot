# Configuration System

## Overview

The PocketTCGoBot configuration system provides comprehensive control over bot behavior, coordinate translation, multi-instance management, and global action timing. Configuration is centralized in the `Config` struct and automatically applies sensible defaults when values are not explicitly set.

## Table of Contents

- [Configuration Fields](#configuration-fields)
- [Coordinate Translation](#coordinate-translation)
- [Multi-Instance Settings](#multi-instance-settings)
- [Global Action Timing](#global-action-timing)
- [Monitor and Display Settings](#monitor-and-display-settings)
- [Configuration Loading](#configuration-loading)
- [Default Values](#default-values)
- [Examples](#examples)

## Configuration Fields

### Instance Configuration

```go
type Config struct {
    Instance         int    // Bot instance number
    Columns          int    // Number of columns for window arrangement
    RowGap           int    // Gap between rows in pixels
    SelectedMonitor  int    // Monitor index for bot windows
    DefaultLanguage  string // "Scale100" or "Scale125"
    FolderPath       string // Path to MuMu emulator folder
}
```

### Coordinate Translation Settings

Controls how template coordinates are translated to device screen coordinates.

```go
// Coordinate Translation Settings
SourceScreenWidth  int // Source coordinate system width (default: 277)
SourceScreenHeight int // Source coordinate system height (default: 489)
GameBoardHeight    int // Actual game board height (default: 489)
WindowBorderHeight int // Border/padding height (default: 4)
```

**Purpose:**
- `SourceScreenWidth/Height`: The coordinate system used in template definitions
- `GameBoardHeight`: The actual game area height in pixels
- `WindowBorderHeight`: Additional window chrome (borders, padding)

**Defaults:**
- SourceScreenWidth: 277 (template coordinate width)
- SourceScreenHeight: 489 (game board height)
- GameBoardHeight: 489
- WindowBorderHeight: 4

### Multi-Instance Settings

Controls timing for launching and managing multiple bot instances.

```go
// Multi-Instance Settings
InstanceStartDelay  int // Delay between instance starts in seconds (default: 10)
InstanceLaunchDelay int // Delay when launching emulators in seconds (default: 2)
```

**Purpose:**
- `InstanceStartDelay`: Prevents resource contention when starting multiple bots
- `InstanceLaunchDelay`: Allows emulator to fully initialize before connecting

**Defaults:**
- InstanceStartDelay: 10 seconds
- InstanceLaunchDelay: 2 seconds

### Global Action Timing

Default timing values for all bot actions. Individual actions can override these.

```go
// Global Action Timing
GlobalClickDelay      int // Delay after clicks in ms (default: uses Delay)
GlobalSwipeDelay      int // Delay after swipes in ms (default: uses SwipeSpeed)
GlobalTemplateTimeout int // Template matching timeout in ms (default: 5000)
GlobalRetryAttempts   int // Retry attempts for actions (default: 3)
GlobalRetryDelay      int // Delay between retries in ms (default: 1000)
```

**Purpose:**
- `GlobalClickDelay`: Standard delay after click actions
- `GlobalSwipeDelay`: Standard delay after swipe gestures
- `GlobalTemplateTimeout`: How long to wait for template matching
- `GlobalRetryAttempts`: Number of times to retry failed actions
- `GlobalRetryDelay`: Time between retry attempts

**Defaults:**
- GlobalClickDelay: 250ms (or value from `Delay` field)
- GlobalSwipeDelay: 500ms (or value from `SwipeSpeed` field)
- GlobalTemplateTimeout: 5000ms (5 seconds)
- GlobalRetryAttempts: 3
- GlobalRetryDelay: 1000ms (1 second)

### Monitor and Display Settings

```go
// Monitor and Display Settings
MonitorScaleFactor float64 // DPI scaling (default: 1.0 for Scale100, 1.25 for Scale125)
MonitorOffsetX     int     // X offset for selected monitor
MonitorOffsetY     int     // Y offset for selected monitor
MuMuWindowWidth    int     // MuMu window width (auto-set based on scale)
MuMuWindowHeight   int     // MuMu window height (auto-set based on scale)
TitleBarHeight     int     // Title bar height (default: 45, auto-detected)
```

**Auto-Detection:**
- `MonitorScaleFactor`: Set automatically based on `DefaultLanguage` ("Scale100" = 1.0, "Scale125" = 1.25)
- `MuMuWindowWidth/Height`: Set automatically based on scale (Scale100: 540x960, Scale125: 675x1200)
- `TitleBarHeight`: Detected based on MuMu version (V5: 45px, V12: 50px)

### Performance Tuning

```go
// Performance tuning
Delay      int  // Base delay in milliseconds (default: 250)
SwipeSpeed int  // Swipe duration in milliseconds (default: 500)
SlowMotion bool // Enable slow motion mode
WaitTime   int  // Screenshot wait time in seconds (default: 5)
```

## Coordinate Translation

### How It Works

The coordinate translation system converts template coordinates (defined at a standard resolution) to device screen coordinates (which vary based on window scale).

```
Template Coordinate (277x489)  →  Translation  →  Device Coordinate (540x960 or 675x1200)

Example:
  Click at template (100, 200)

  Scale100 (540x960):
    X: 100 * (540 / 277) = 195
    Y: (200 - 44) * (960 / 489) = 306

  Scale125 (675x1200):
    X: 100 * (675 / 277) = 243
    Y: (200 - 44) * (1200 / 489) = 383
```

### Y-Coordinate Offset

Y coordinates account for the title bar by subtracting `TitleBarHeight` before scaling:

```go
translatedY = (sourceY - TitleBarHeight) * (TargetHeight / SourceHeight)
```

This ensures template coordinates are relative to the game board, not the window.

### CoordinateTranslator

The `CoordinateTranslator` struct handles all coordinate math:

```go
translator := NewCoordinateTranslator(config.GetCoordinateTranslationConfig())

// Translate individual coordinates
translatedX := translator.TranslateX(100)
translatedY := translator.TranslateY(200)

// Translate a point
x, y := translator.TranslatePoint(100, 200)

// Translate a region
x1, y1, x2, y2 := translator.TranslateRegion(100, 200, 150, 250)
```

### ADB Integration

The ADB controller automatically uses the coordinator translator:

```go
// Coordinator is set up automatically during bot initialization
b.adb.Click(100, 200) // Automatically translates before sending to device
b.adb.Swipe(100, 200, 150, 250, 500) // All coordinates translated
```

### Fallback Behavior

If no translator is configured, ADB commands fall back to hardcoded defaults:
- Target: 540x960 (Scale100)
- Source: 277x489
- Title bar: 44px

## Multi-Instance Settings

### Instance Start Delay

When launching multiple bot instances, `InstanceStartDelay` staggers the start times:

```
Bot 1: Start immediately
Bot 2: Start after 10 seconds
Bot 3: Start after 20 seconds
Bot 4: Start after 30 seconds
...
```

**Purpose:** Prevents overwhelming system resources and allows each instance to initialize properly.

**Configuration:**
```go
config.InstanceStartDelay = 15 // 15 seconds between starts
```

### Instance Launch Delay

When launching emulator instances, `InstanceLaunchDelay` waits for the emulator to start:

```
1. Launch emulator process
2. Wait InstanceLaunchDelay seconds
3. Connect ADB
4. Begin bot operations
```

**Purpose:** Ensures emulator is fully initialized before ADB connection attempts.

**Configuration:**
```go
config.InstanceLaunchDelay = 3 // 3 second wait after launch
```

## Global Action Timing

### Click Delay

Delay after every click action (unless overridden):

```yaml
- action: Click
  template: button
  # Uses GlobalClickDelay (default: 250ms)

- action: Click
  template: button
  delay: 500  # Override: use 500ms for this specific click
```

### Swipe Delay

Delay after every swipe action (unless overridden):

```yaml
- action: Swipe
  x1: 100
  y1: 200
  x2: 150
  y2: 250
  duration: 500  # Swipe takes 500ms
  # Then waits GlobalSwipeDelay (default: 500ms)
```

### Template Timeout

How long to wait for template matching before timing out:

```yaml
- action: WaitForTemplate
  template: loading_screen
  # Waits up to GlobalTemplateTimeout (default: 5000ms)

- action: WaitForTemplate
  template: rare_popup
  timeout: 10000  # Override: wait up to 10 seconds
```

### Retry Configuration

Controls automatic retry behavior for failed actions:

```yaml
- action: Click
  template: unstable_button
  # Will retry GlobalRetryAttempts times (default: 3)
  # With GlobalRetryDelay between attempts (default: 1000ms)

- action: Click
  template: critical_button
  max_retries: 5      # Override: try 5 times
  retry_delay: 2000   # Override: wait 2 seconds between attempts
```

## Monitor and Display Settings

### Monitor Scale Factor

Automatically set based on `DefaultLanguage`:

- **Scale100**: MonitorScaleFactor = 1.0
- **Scale125**: MonitorScaleFactor = 1.25

Used for DPI-aware coordinate calculations.

### Window Dimensions

Automatically set based on scale:

| Scale | Width | Height |
|-------|-------|--------|
| Scale100 | 540 | 960 |
| Scale125 | 675 | 1200 |

Can be overridden manually:

```go
config.MuMuWindowWidth = 600
config.MuMuWindowHeight = 1000
```

### Monitor Offsets

For multi-monitor setups, specify the selected monitor's position:

```go
config.MonitorOffsetX = 1920  // Second monitor starts at X=1920
config.MonitorOffsetY = 0
```

Used when positioning bot windows.

## Configuration Loading

### Apply Defaults

Call `ApplyDefaults()` to fill in missing configuration values:

```go
config := &bot.Config{
    DefaultLanguage: "Scale125",
    Delay: 300,
}

config.ApplyDefaults()  // Fills in all unset fields with defaults

// Now config has:
// - SourceScreenWidth: 277
// - GlobalClickDelay: 300 (from Delay)
// - GlobalRetryAttempts: 3
// - etc...
```

### Get Coordinate Config

Extract coordinate translation parameters:

```go
coordConfig := config.GetCoordinateTranslationConfig()
// Returns CoordinateConfig with:
// - SourceWidth, SourceHeight
// - TargetWidth, TargetHeight
// - TitleBarHeight, GameBoardHeight
// - ScaleFactor
```

### Automatic Initialization

Bot initialization automatically applies defaults and sets up translation:

```go
bot, err := bot.New(instance, config)
if err != nil {
    return err
}

if err := bot.Initialize(); err != nil {
    return err
}
// ApplyDefaults() called automatically
// Coordinate translator configured automatically
// ADB controller set up with translator
```

## Default Values

### Complete Default Configuration

When `ApplyDefaults()` is called, the following defaults are applied:

| Field | Default Value | Notes |
|-------|---------------|-------|
| SourceScreenWidth | 277 | Template coordinate width |
| SourceScreenHeight | 489 | Game board height |
| GameBoardHeight | 489 | Actual game area |
| WindowBorderHeight | 4 | Window chrome |
| InstanceStartDelay | 10 | Seconds between instance starts |
| InstanceLaunchDelay | 2 | Seconds after emulator launch |
| GlobalClickDelay | 250 or Delay | Milliseconds |
| GlobalSwipeDelay | 500 or SwipeSpeed | Milliseconds |
| GlobalTemplateTimeout | 5000 | 5 seconds |
| GlobalRetryAttempts | 3 | Number of retries |
| GlobalRetryDelay | 1000 | 1 second |
| MonitorScaleFactor | 1.0 or 1.25 | Based on DefaultLanguage |
| TitleBarHeight | 45 | Detected by emulator manager |
| Delay | 250 | Milliseconds |
| SwipeSpeed | 500 | Milliseconds |
| WaitTime | 5 | Seconds |

## Examples

### Example 1: Basic Configuration with Defaults

```go
config := &bot.Config{
    Instance: 1,
    DefaultLanguage: "Scale100",
    FolderPath: "C:\\Program Files\\Netease\\MuMuPlayer-12.0",
}

// All defaults applied automatically during bot.Initialize()
bot, _ := bot.New(1, config)
bot.Initialize()
// Now using:
// - 540x960 window
// - 277x489 template coordinates
// - 250ms click delay
// - 3 retry attempts
// - etc.
```

### Example 2: Custom Timing Configuration

```go
config := &bot.Config{
    DefaultLanguage: "Scale125",
    Delay: 500,                   // 500ms base delay
    SwipeSpeed: 750,              // 750ms swipes
    GlobalRetryAttempts: 5,       // Retry 5 times
    GlobalRetryDelay: 2000,       // 2 seconds between retries
    InstanceStartDelay: 15,       // 15 seconds between instance starts
}

config.ApplyDefaults()
// Click delay: 500ms (from Delay)
// Swipe delay: 750ms (from SwipeSpeed)
// Retry attempts: 5
// Retry delay: 2000ms
```

### Example 3: Custom Coordinate Translation

```go
config := &bot.Config{
    DefaultLanguage: "Scale100",
    MuMuWindowWidth: 600,         // Custom window width
    MuMuWindowHeight: 1000,       // Custom window height
    SourceScreenWidth: 300,       // Custom source width
    SourceScreenHeight: 500,      // Custom source height
    TitleBarHeight: 50,           // Custom title bar
}

config.ApplyDefaults()

// Coordinate translation will use custom values:
// Template (100, 200) -> Device (200, 300) for this specific configuration
```

### Example 4: Multi-Monitor Setup

```go
config := &bot.Config{
    SelectedMonitor: 2,
    MonitorOffsetX: 1920,         // Second monitor at X=1920
    MonitorOffsetY: 0,
    Columns: 3,                   // 3 columns of bot windows
    RowGap: 100,                  // 100px gap between rows
}

// Bot windows will be positioned on second monitor
// in a 3-column grid
```

### Example 5: Fast Mode Configuration

```go
config := &bot.Config{
    Delay: 100,                   // Fast clicks
    SwipeSpeed: 200,              // Fast swipes
    GlobalRetryAttempts: 1,       // Don't retry much
    GlobalTemplateTimeout: 2000,  // Quick timeouts
    InstanceStartDelay: 5,        // Quick stagger
}

config.ApplyDefaults()
// Bot runs in "fast mode" with minimal delays
```

### Example 6: Stable Mode Configuration

```go
config := &bot.Config{
    Delay: 500,                   // Generous delays
    SwipeSpeed: 1000,             // Slow swipes
    GlobalRetryAttempts: 5,       // Many retries
    GlobalRetryDelay: 3000,       // Long retry waits
    GlobalTemplateTimeout: 10000, // Long template waits
    InstanceStartDelay: 20,       // Large instance stagger
}

config.ApplyDefaults()
// Bot runs in "stable mode" prioritizing reliability over speed
```

## Best Practices

### 1. Always Use ApplyDefaults()

```go
config := LoadConfigFromFile("config.json")
config.ApplyDefaults()  // Fill in missing values
```

### 2. Let Coordinate Translation Happen Automatically

```go
// DON'T manually translate coordinates
x := int(float64(100) * scaleFactorX)
bot.Click(x, y)

// DO let ADB controller handle it
bot.Click(100, 200)  // Automatically translated
```

### 3. Override Defaults Only When Needed

```go
// Good: Use defaults for most fields
config := &bot.Config{
    DefaultLanguage: "Scale125",
    InstanceStartDelay: 15,  // Only override what's needed
}

// Avoid: Setting every field manually
config := &bot.Config{
    GlobalClickDelay: 250,
    GlobalSwipeDelay: 500,
    GlobalRetryAttempts: 3,
    // ... lots of redundant settings
}
```

### 4. Tune Timing Based on System Performance

```go
// High-end system: faster timing
config.Delay = 150
config.InstanceStartDelay = 5

// Low-end system: slower timing
config.Delay = 500
config.InstanceStartDelay = 20
```

### 5. Use Consistent Scale Settings

```go
// Ensure DefaultLanguage matches actual window configuration
config.DefaultLanguage = "Scale125"
config.MuMuWindowWidth = 675   // Matches Scale125
config.MuMuWindowHeight = 1200 // Matches Scale125
```

## Troubleshooting

### Clicks Missing Targets

**Problem:** Bot clicks near but not on target buttons

**Solution:** Check coordinate translation settings

```go
// Verify scale matches window
fmt.Printf("Window: %dx%d\n", config.MuMuWindowWidth, config.MuMuWindowHeight)
fmt.Printf("Scale: %s\n", config.DefaultLanguage)

// Check translator output
translator := bot.NewCoordinateTranslator(config.GetCoordinateTranslationConfig())
fmt.Println(translator.String())
```

### Instances Starting Too Fast

**Problem:** System overloaded when launching multiple bots

**Solution:** Increase stagger delay

```go
config.InstanceStartDelay = 20  // 20 seconds between starts
config.InstanceLaunchDelay = 5  // 5 seconds after emulator launch
```

### Actions Timing Out

**Problem:** Template matching fails frequently

**Solution:** Increase timeout or reduce retry delay

```go
config.GlobalTemplateTimeout = 10000  // Wait 10 seconds
config.GlobalRetryDelay = 2000        // 2 seconds between retries
```

### Swipes Not Completing

**Problem:** Swipe gestures too fast or slow

**Solution:** Adjust swipe speed

```go
config.SwipeSpeed = 750  // Slower swipes (750ms duration)
```

## Related Documentation

- [Health Monitoring](HEALTH_MONITORING.md) - Bot health and recovery
- [Actions Documentation](ACTIONS.md) - Available bot actions
- [Routine System](ROUTINES.md) - Routine execution and composition

---

**Last Updated:** 2025-11-08
**Version:** v0.2.0
**Status:** Production Ready
