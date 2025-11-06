# Window Capture Coordinate System

## Overview

The CV system uses **window-relative coordinates** for all capture and template matching operations. This means all coordinates are relative to the window's client area top-left corner (0, 0), not screen coordinates.

## How It Works

### 1. Window Capture ([capture_windows.go](capture_windows.go#L98))

When capturing from a window:
```go
capture, err := cv.NewWindowCapture(hwnd)
frame, err := capture.CaptureFrame()
```

The `GetClientRect` Win32 API call returns:
- **Left, Top**: Always (0, 0) - the origin of the client area
- **Right, Bottom**: The width and height of the window

The resulting `image.RGBA` has bounds: `image.Rect(0, 0, width, height)`

### 2. Template Matching ([matching.go](matching.go#L46))

When searching for a template:
```go
result := cv.FindTemplate(frame, template, config)
// result.Location is in window-relative coordinates
```

The `MatchResult.Location` point is relative to the haystack image, which starts at (0, 0).

### 3. Search Regions ([matching.go](matching.go#L33))

When using `SearchRegion`, specify coordinates relative to the window:
```go
config := cv.DefaultMatchConfig()
// Search only in the top-left quadrant of the window
config.SearchRegion = &image.Rectangle{
    Min: image.Point{X: 0, Y: 0},
    Max: image.Point{X: 400, Y: 300},
}
```

## Coordinate Translation

### ✅ Already Window-Relative

**Good news**: You don't need to translate coordinates! The system is already window-relative:

1. **Capture**: Returns image with bounds starting at (0, 0)
2. **Template Search**: Returns locations relative to capture origin
3. **Pixel Access**: Uses window-relative coordinates

### 🔴 When You Need Screen Coordinates

If you need screen coordinates (e.g., for mouse clicks), you must translate:

```go
// Get window position on screen
var rect RECT
procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))

// Convert window-relative to screen coordinates
screenX := rect.Left + windowRelativeX
screenY := rect.Top + windowRelativeY
```

**Note**: For ADB input, you typically want window-relative coordinates, which the system already provides.

## Testing

Run the test suite to verify:

```bash
# Run all CV tests
go test -v ./internal/cv

# Test window capture (requires an open window)
go test -v ./internal/cv -run TestWindowCapture

# Test coordinate system
go test -v ./internal/cv -run TestCoordinateTranslation

# Benchmark capture performance
go test -v ./internal/cv -bench BenchmarkWindowCapture
```

### Test Requirements

The window capture tests require a visible window with one of these titles:
- `MuMuPlayer` (recommended for your use case)
- `Untitled - Notepad`
- `Calculator`

The test will:
1. ✅ Verify window capture works
2. ✅ Confirm coordinates start at (0, 0)
3. ✅ Save a test capture to `test_capture.png` for visual inspection
4. ✅ Verify template matching returns window-relative coordinates
5. ✅ Test search regions work with window-relative coordinates

## Title Bar Exclusion

To prevent searching in the window title bar area (which can contain system UI elements), the CV service supports automatic title bar exclusion:

```go
// Initialize with title bar exclusion (20 pixels from top)
capture, _ := cv.NewWindowCapture(hwnd)
service := cv.NewServiceWithTitleBar(capture, 20)

// All searches automatically exclude top 20 pixels
result, _ := service.FindTemplate("templates/button.png", nil)
```

The title bar exclusion is applied automatically when:
- Title bar height is > 0
- No custom `SearchRegion` is specified in the `MatchConfig`

You can also set/update it dynamically:
```go
service.SetTitleBarHeight(25) // Change exclusion to 25 pixels
height := service.GetTitleBarHeight() // Get current setting
```

### Configuration

In the bot config, set `TitleBarHeight` to automatically exclude the title bar:

```go
config := &bot.Config{
    TitleBarHeight: 20, // Exclude top 20 pixels (MuMu title bar)
    // ... other config
}
```

If not set, the bot defaults to 20 pixels for MuMu emulator windows.

## Example Usage

### Basic Template Search
```go
// Initialize
capture, _ := cv.NewWindowCapture(hwnd)
service := cv.NewService(capture)

// Find template
result, err := service.FindTemplate("templates/button.png", nil)
if result.Found {
    // result.Location.X and result.Location.Y are window-relative
    fmt.Printf("Found at window coords: (%d, %d)\n",
        result.Location.X, result.Location.Y)
}
```

### With Title Bar Exclusion
```go
// Initialize with automatic title bar exclusion
capture, _ := cv.NewWindowCapture(hwnd)
service := cv.NewServiceWithTitleBar(capture, 20)

// All searches automatically skip top 20 pixels
result, _ := service.FindTemplate("templates/button.png", nil)
```

### Search in Specific Region
```go
config := cv.DefaultMatchConfig()
// Search only in bottom half of window
width, height := service.GetDimensions()
config.SearchRegion = &image.Rectangle{
    Min: image.Point{X: 0, Y: height / 2},
    Max: image.Point{X: width, Y: height},
}

result, err := service.FindTemplate("templates/button.png", config)
```

### Pixel Color Check
```go
// Check color at window-relative coordinates
color, err := service.GetPixelColor(100, 50)
if err == nil {
    r, g, b, _ := color.RGBA()
    fmt.Printf("Color at (100, 50): RGB(%d, %d, %d)\n", r>>8, g>>8, b>>8)
}
```

## Architecture Benefits

### ✅ Advantages of Window-Relative Coordinates

1. **Window Movement**: If the window moves on screen, your coordinates remain valid
2. **Multi-Monitor**: Works correctly regardless of which monitor the window is on
3. **Simplicity**: No need to track window position or translate coordinates
4. **ADB Compatible**: ADB input uses window-relative coordinates

### 🎯 Integration with ADB

When sending input via ADB (e.g., tap at template location):
```go
result, _ := service.FindTemplate("templates/button.png", nil)
if result.Found {
    // Can use directly with ADB - no translation needed
    adb.Tap(result.Location.X, result.Location.Y)
}
```

## Common Pitfalls

### ❌ Don't Mix Coordinate Systems

```go
// WRONG - mixing screen and window coordinates
screenX, screenY := getMousePosition() // Screen coordinates
result, _ := service.FindTemplate(template, nil) // Window coordinates
if screenX == result.Location.X { // This comparison is invalid!
    // ...
}
```

### ❌ Don't Forget SearchRegion is Window-Relative

```go
// WRONG - using screen coordinates for search region
config.SearchRegion = &image.Rectangle{
    Min: image.Point{X: 1920, Y: 0}, // Screen coordinates!
    Max: image.Point{X: 3840, Y: 1080},
}

// CORRECT - using window-relative coordinates
width, height := service.GetDimensions()
config.SearchRegion = &image.Rectangle{
    Min: image.Point{X: width/2, Y: 0}, // Window-relative
    Max: image.Point{X: width, Y: height},
}
```

## Performance Notes

- **Frame Caching**: The service caches frames for 100ms by default
- **Search Regions**: Using search regions can significantly improve performance
- **Template Caching**: Templates are automatically cached by path
- **Capture Speed**: Window capture is fast (~1-2ms on modern hardware)

## See Also

- [capture_windows.go](capture_windows.go) - Window capture implementation
- [service.go](service.go) - CV service with caching
- [matching.go](matching.go) - Template matching algorithms
- [capture_test.go](capture_test.go) - Comprehensive test suite
