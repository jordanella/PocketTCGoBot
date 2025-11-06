# Template Image Caching

The template registry includes an **intelligent image caching system** that significantly improves performance by keeping frequently-used template images in memory.

## Overview

### The Problem
Loading template images from disk is expensive:
- Typical PNG load: **5-20ms**
- For a template checked 1000x/hour: **5-20 seconds** of disk I/O
- Multiple bots loading same templates: **Wasted memory and I/O**

### The Solution
**Intelligent image caching** with three modes:
1. **Preload** - Load at startup, keep in memory (for frequent templates)
2. **On-Demand** - Load when first used, cache until unload (default)
3. **Unload After Use** - Load, use once, free memory (for rare templates)

## Quick Start

### 1. Configure Templates in YAML

```yaml
templates:
  # Frequently used - preload at startup
  - name: OK
    path: ui/OK.png
    preload: true  # Loads at startup, stays in memory

  - name: Main
    path: ui/Main.png
    preload: true  # Hot path template

  # Moderate use - load on-demand (default)
  - name: Settings
    path: ui/Settings.png
    # No preload or unload_after = cache when first used

  # Rarely used - unload after use
  - name: ErrorDialog
    path: ui/ErrorDialog.png
    unload_after: true  # Free memory after detection
```

### 2. Load Templates with Caching

```go
// Initialize registry with image caching enabled (default)
registry := templates.InitializeGlobalRegistry("templates")
registry.LoadFromDirectory("templates")

// Preload all templates marked with preload: true
if err := registry.PreloadAll(); err != nil {
    log.Printf("Warning: some templates failed to preload: %v", err)
}

log.Printf("Loaded %d templates", registry.Count())
stats := registry.CacheStats()
log.Printf("Preloaded %d images", stats.Loads)
```

### 3. Use Templates (Automatic Caching)

The caching happens automatically when your CV service uses templates:

```go
// First use: loads from disk (5-20ms)
result, _ := bot.CV().FindTemplate("OK", &cv.MatchConfig{...})

// Subsequent uses: from cache (<1ms)
result, _ := bot.CV().FindTemplate("OK", &cv.MatchConfig{...})
result, _ := bot.CV().FindTemplate("OK", &cv.MatchConfig{...})

// If unload_after=true, automatically freed after action completes
```

## Caching Modes

### Mode 1: Preload (Recommended for Frequent Templates)

**Use for**: Templates checked multiple times per second
- UI navigation elements (Main, Menu, Home)
- Common buttons (OK, Confirm, Cancel)
- Frequently detected elements

**Benefits**:
- Zero latency after startup
- Predictable performance
- No disk I/O during operation

**Configuration**:
```yaml
- name: Main
  path: ui/Main.png
  preload: true
  threshold: 0.8
```

**Memory impact**: ~100KB per template (typical)

### Mode 2: On-Demand (Default for Moderate Use)

**Use for**: Templates checked occasionally
- Less common UI elements
- Specific game states
- Conditional checks

**Benefits**:
- Automatic caching after first use
- No startup cost
- Memory efficient

**Configuration**:
```yaml
- name: Settings
  path: ui/Settings.png
  # No flags = on-demand caching
```

**Behavior**:
- First use: Load from disk (5-20ms)
- Cached until: Manual unload or registry clear
- Memory: Only loaded templates

### Mode 3: Unload After Use (For Rare Templates)

**Use for**: Templates checked very infrequently
- Error dialogs
- One-time setup screens
- Account creation flows

**Benefits**:
- Minimal memory footprint
- Still faster than no caching (cached during action)
- Automatic cleanup

**Configuration**:
```yaml
- name: ErrorDialog
  path: ui/ErrorDialog.png
  unload_after: true  # Free after use
```

**Behavior**:
- Loaded when action starts
- Cached during action execution
- Auto-unloaded when action completes

## Performance Comparison

### Without Caching
```
Check template "OK" 100 times:
  Disk reads: 100 × 10ms = 1000ms
  Memory:     Loaded/freed 100 times
```

### With Preload
```
Check template "OK" 100 times:
  Disk reads: 1 × 10ms = 10ms (at startup)
  Subsequent: 100 × <1ms = ~100ms
  Total:      110ms (9x faster!)
  Memory:     Persistent (~100KB)
```

### With On-Demand
```
Check template "OK" 100 times:
  First:      1 × 10ms = 10ms
  Subsequent: 99 × <1ms = ~99ms
  Total:      109ms (9x faster!)
  Memory:     Persistent (~100KB)
```

### With Unload After
```
Check template "OK" once per action:
  Load:       10ms
  Use:        <1ms (from cache)
  Unload:     <1ms
  Per action: ~11ms
  Memory:     Temporary
```

## Memory Management

### Estimating Memory Usage

**Typical template sizes**:
- Small UI element (32×32): ~10KB
- Button (64×64): ~30KB
- Large UI element (128×128): ~100KB
- Screen region (256×256): ~400KB

**Example configuration**:
```yaml
# 20 preloaded templates × 50KB average = 1MB
# + 10 on-demand templates loaded = +500KB
# Total: ~1.5MB for 30 templates
```

**Rule of thumb**: 50-100KB per template

### Memory-Conscious Strategy

```yaml
templates:
  # Hot path (10 templates × 50KB = 500KB)
  - name: Main
    preload: true
  # ... 9 more frequent templates

  # Moderate use (no preload)
  - name: Settings
    # Loaded when first used
  # ... 15 more occasional templates

  # Rare use (5 templates, temporary load)
  - name: ErrorDialog
    unload_after: true
  # ... 4 more rare templates
```

**Total memory**: 500KB persistent + occasional 50-100KB

## Cache Statistics

Monitor cache performance:

```go
stats := registry.CacheStats()
fmt.Printf("Cache Stats:\n")
fmt.Printf("  Hits:         %d\n", stats.Hits)
fmt.Printf("  Misses:       %d\n", stats.Misses)
fmt.Printf("  Hit Rate:     %.1f%%\n", float64(stats.Hits)/float64(stats.Hits+stats.Misses)*100)
fmt.Printf("  Loads:        %d\n", stats.Loads)
fmt.Printf("  Unloads:      %d\n", stats.Unloads)
fmt.Printf("  Preload Fail: %d\n", stats.PreloadFail)
```

**Example output**:
```
Cache Stats:
  Hits:         45823    # Used cached image
  Misses:       142      # Had to load from disk
  Hit Rate:     99.7%    # Excellent!
  Loads:        142      # Disk loads
  Unloads:      87       # Memory freed
  Preload Fail: 2        # Missing files
```

## Advanced Usage

### Disabling Cache for Specific Registry

```go
// Create registry without caching
registry := templates.NewTemplateRegistry("templates").
    WithoutImageCache()

// Templates loaded from disk every time (legacy behavior)
```

### Manual Cache Control

```go
// Get image cache
cache := registry.ImageCache()

// Preload specific template
cache.Get("SpecificTemplate")

// Release specific template
cache.Release("SpecificTemplate")

// Unload all
registry.UnloadAll()
```

### Programmatic Registration

```go
// Register template with caching options
template := cv.Template{
    Name:      "CustomTemplate",
    Path:      "path/to/image.png",
    Threshold: 0.85,
}

// Add to registry
registry.Register(template)

// Manually add to cache with options
cache := registry.ImageCache()
cache.Register(template, true, false)  // preload=true, unloadAfter=false
```

## Integration with Actions

### WhileTemplateExists with Caching

```yaml
- action: WhileTemplateExists
  template_name: "ClaimButton"  # Uses cached image automatically
  max_attempts: 10
  actions:
    - action: Click
      x: 140
      y: 400
```

**Behavior**:
1. First loop iteration: Load "ClaimButton" if not cached
2. Subsequent iterations: Use cached image (<1ms)
3. If `unload_after: true`: Unload after action completes

### Custom Action with Manual Release

```go
func (a *MyAction) Build(ab *ActionBuilder) *ActionBuilder {
    step := Step{
        name: "MyAction",
        execute: func(bot BotInterface) error {
            // Get template (auto-cached)
            template, _ := bot.Templates().Get(a.TemplateName)

            // Use template for CV operations
            result, _ := bot.CV().FindTemplate(template.Path, &cv.MatchConfig{
                Threshold: template.Threshold,
            })

            // If template has unload_after=true, release it
            if cache := bot.Templates().ImageCache(); cache != nil {
                defer cache.Release(a.TemplateName)
            }

            // ... process result
            return nil
        },
    }
    return ab
}
```

## Best Practices

### 1. Profile Your Templates

Identify frequency of use:
```
High frequency (>10x/min):  preload: true
Medium frequency (1-10x/min): default (on-demand)
Low frequency (<1x/min):     unload_after: true
```

### 2. Start with Preload for Core Templates

```yaml
# Always preload navigation
- name: Main
  preload: true
- name: Menu
  preload: true
- name: Home
  preload: true

# Preload common actions
- name: OK
  preload: true
- name: Confirm
  preload: true
- name: Cancel
  preload: true
```

### 3. Use unload_after for One-Time Events

```yaml
# Account creation (once per bot lifetime)
- name: Welcome
  unload_after: true
- name: TOS
  unload_after: true

# Error screens (hopefully rare)
- name: ErrorDialog
  unload_after: true
- name: Maintenance
  unload_after: true
```

### 4. Monitor Cache Performance

```go
// Log stats periodically
ticker := time.NewTicker(5 * time.Minute)
go func() {
    for range ticker.C {
        stats := registry.CacheStats()
        hitRate := float64(stats.Hits) / float64(stats.Hits+stats.Misses) * 100
        log.Printf("Template cache hit rate: %.1f%% (%d hits, %d misses)",
            hitRate, stats.Hits, stats.Misses)
    }
}()
```

### 5. Handle Preload Failures Gracefully

```go
if err := registry.PreloadAll(); err != nil {
    // Don't fail startup - templates can still load on-demand
    log.Printf("Warning: some templates failed to preload: %v", err)

    // Continue with degraded performance (on-demand loading)
}
```

## Troubleshooting

### High Memory Usage

**Symptom**: Bot using too much memory

**Solution**: Reduce preloaded templates
```yaml
# Before: 50 preloaded templates = 5MB
# After:  20 preloaded templates = 2MB
# Use on-demand for less frequent templates
```

### Slow First Detection

**Symptom**: First template check is slow (5-20ms)

**Solution**: Add `preload: true` for that template
```yaml
- name: SlowTemplate
  path: ui/SlowTemplate.png
  preload: true  # Load at startup instead of first use
```

### Templates Not Unloading

**Symptom**: Memory grows over time

**Solution**: Ensure `unload_after: true` is working
```go
// Check if cache is enabled
if registry.ImageCache() == nil {
    log.Println("Warning: Image cache disabled")
}

// Verify template configuration
template, _ := registry.Get("TemplateName")
// Check if unloadAfter is set in YAML
```

### Preload Failures

**Symptom**: Warnings about failed preloads

**Causes**:
1. Image file doesn't exist
2. Corrupted image file
3. Wrong path in YAML

**Solution**:
```bash
# Verify files exist
ls -la templates/ui/TemplateName.png

# Check YAML path matches file location
```

## Migration from Non-Cached System

### Step 1: Enable Caching (Already Default)

```go
// Old: No change needed
registry := templates.NewTemplateRegistry("templates")

// Caching is enabled by default
```

### Step 2: Identify Frequent Templates

```go
// Add logging to track template usage
templateCounts := make(map[string]int)
// ... track calls to FindTemplate
// ... analyze after 1 hour of operation
```

### Step 3: Update YAML Configuration

```yaml
# Add preload: true to top 10-20 templates
# Add unload_after: true to rarely used templates
```

### Step 4: Add Preload Call

```go
registry.LoadFromDirectory("templates")
registry.PreloadAll()  // Add this line
```

### Step 5: Monitor Performance

```go
stats := registry.CacheStats()
// Expect >95% hit rate for good performance
```

## Summary

Template image caching provides:
- **9x faster** template detection for cached images
- **Configurable** memory vs performance trade-offs
- **Automatic** management with preload and unload options
- **Statistics** for monitoring and optimization
- **Zero code changes** - works with existing actions

Choose caching strategy based on template frequency:
- High: `preload: true` (persistent memory)
- Medium: Default (on-demand caching)
- Low: `unload_after: true` (temporary memory)
