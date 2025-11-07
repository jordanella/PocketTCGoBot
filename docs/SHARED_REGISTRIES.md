# Shared Registries for Multi-Bot Systems

When running multiple bot instances (6-8 bots), sharing template and routine registries across all instances provides significant memory savings and consistency benefits.

## Architecture

### Without Shared Registries (❌ Inefficient)
```
Bot 1 → TemplateRegistry (Copy 1) + RoutineRegistry (Copy 1)
Bot 2 → TemplateRegistry (Copy 2) + RoutineRegistry (Copy 2)
Bot 3 → TemplateRegistry (Copy 3) + RoutineRegistry (Copy 3)
...
Bot 8 → TemplateRegistry (Copy 8) + RoutineRegistry (Copy 8)

Memory: 8x templates + 8x routines = 16x overhead
```

### With Shared Registries (✅ Efficient)
```
Manager
  ├── TemplateRegistry (shared)     ← Single instance
  ├── RoutineRegistry (shared)      ← Single instance
  └── Bots
      ├── Bot 1 → references shared registries
      ├── Bot 2 → references shared registries
      ├── Bot 3 → references shared registries
      ...
      └── Bot 8 → references shared registries

Memory: 1x templates + 1x routines = Saves ~87.5% registry memory!
```

## Benefits

1. **Memory Efficiency**: 5-7x less memory for templates and routines
2. **Consistency**: All bots use the same template/routine definitions
3. **Hot Reload**: Update templates/routines once, affects all bots
4. **Faster Startup**: Templates/routines loaded once, not per-bot
5. **Simpler Management**: Single source of truth for configurations

## Usage

### Basic Setup

```go
package main

import (
    "jordanella.com/pocket-tcg-go/internal/bot"
)

func main() {
    // 1. Create shared configuration
    config := &bot.Config{
        FolderPath: "path/to/bot/data",
        // ... other config
    }

    // 2. Create manager with shared registries
    manager, err := bot.NewManager(config)
    if err != nil {
        panic(err)
    }
    defer manager.ShutdownAll()

    // 3. Create multiple bots (they automatically share registries)
    for i := 1; i <= 8; i++ {
        bot, err := manager.CreateBot(i)
        if err != nil {
            log.Printf("Failed to create bot %d: %v", i, err)
            continue
        }

        // Start bot in goroutine
        go func(b *bot.Bot, instance int) {
            if err := b.Run(); err != nil {
                log.Printf("Bot %d error: %v", instance, err)
            }
        }(bot, i)
    }

    // 4. Wait for shutdown signal
    // ... (implement your shutdown logic)
}
```

### Manager API

#### Creating Bots

```go
// Create a bot with shared registries
bot, err := manager.CreateBot(instance)
if err != nil {
    return err
}

// The bot is automatically initialized with shared registries
// No need to manually inject them
```

#### Accessing Shared Registries

```go
// Get shared template registry
templateRegistry := manager.TemplateRegistry()

// Get shared routine registry
routineRegistry := manager.RoutineRegistry()

// Check if routine exists (before creating bots)
if routineRegistry.Has("startup_routine") {
    // All bots will have access to this routine
}
```

#### Managing Bot Lifecycle

```go
// Get active bot count
count := manager.GetActiveCount()
fmt.Printf("Running %d bots\n", count)

// Get specific bot instance
bot, exists := manager.GetBot(3)
if exists {
    // Work with bot 3
}

// Shutdown specific bot
if err := manager.ShutdownBot(3); err != nil {
    log.Printf("Failed to shutdown bot 3: %v", err)
}

// Shutdown all bots and clean up shared registries
manager.ShutdownAll()
```

### Hot Reload During Development

```go
// Reload all routines from disk
// Useful when developing and testing new routines
if err := manager.ReloadRoutines(); err != nil {
    log.Printf("Failed to reload routines: %v", err)
}

// Reload all templates from YAML
if err := manager.ReloadTemplates(); err != nil {
    log.Printf("Failed to reload templates: %v", err)
}

// All running bots will use the reloaded configurations
// on their next template/routine access
```

## Migration from Per-Bot Registries

### Before (Old Approach)
```go
// Each bot creates its own registries
func createBot(instance int, config *Config) (*Bot, error) {
    bot, err := New(instance, config)
    if err != nil {
        return nil, err
    }

    // Initialize (creates dedicated registries)
    if err := bot.Initialize(); err != nil {
        return nil, err
    }

    return bot, nil
}

// Creates 8 separate registry instances
for i := 1; i <= 8; i++ {
    bot, _ := createBot(i, config)
    bots = append(bots, bot)
}
```

### After (New Approach)
```go
// Manager creates shared registries once
manager, err := NewManager(config)
if err != nil {
    panic(err)
}

// All bots share the same registry instances
for i := 1; i <= 8; i++ {
    bot, _ := manager.CreateBot(i)
    bots = append(bots, bot)
}
```

## Thread Safety

All shared registries are **thread-safe** and can be accessed concurrently:

- **TemplateRegistry**: Uses `sync.RWMutex` for concurrent reads
- **RoutineRegistry**: Uses `sync.RWMutex` for concurrent reads
- **Manager**: Uses `sync.RWMutex` for bot map access

Multiple bots can safely:
- Load templates simultaneously
- Load routines simultaneously
- Execute routines simultaneously (each gets its own ActionBuilder copy)

## Memory Comparison

### Example: 100 Templates, 50 Routines, 8 Bots

**Without Sharing:**
```
Templates: 100 templates × 8 bots = 800 template objects
Routines:  50 routines × 8 bots = 400 routine objects
Total: 1,200 objects in memory
```

**With Sharing:**
```
Templates: 100 templates × 1 registry = 100 template objects
Routines:  50 routines × 1 registry = 50 routine objects
Total: 150 objects in memory

Savings: 87.5% reduction in template/routine memory!
```

## Best Practices

### 1. Use Manager for Multi-Bot Scenarios
```go
// ✅ Good: Use manager when running multiple bots
manager, _ := bot.NewManager(config)
bot1, _ := manager.CreateBot(1)
bot2, _ := manager.CreateBot(2)
```

### 2. Use Direct Bot for Single Instance
```go
// ✅ Also Good: Direct bot creation for single instance
bot, _ := bot.New(1, config)
bot.Initialize()
```

### 3. Preload Critical Resources
```go
manager, _ := bot.NewManager(config)

// Preload critical routines (verifies they exist)
criticalRoutines := []string{"startup", "error_recovery", "shutdown"}
for _, name := range criticalRoutines {
    if !manager.RoutineRegistry().Has(name) {
        log.Fatalf("Critical routine '%s' missing!", name)
    }
}

// Now safe to create bots
```

### 4. Centralize Configuration Updates
```go
// During maintenance window
manager.ShutdownAll()  // Stop all bots

// Update routine files on disk
// ... copy new routines ...

manager, _ = bot.NewManager(config)  // Reload with new routines

// Restart bots with updated configurations
```

## Error Handling

### Bot Creation Failures

```go
bot, err := manager.CreateBot(instance)
if err != nil {
    if errors.Is(err, ErrBotAlreadyExists) {
        // Bot instance already running
        log.Printf("Bot %d already exists", instance)
    } else {
        // Other error (ADB connection, etc.)
        log.Printf("Failed to create bot %d: %v", instance, err)
    }
}
```

### Registry Failures

```go
manager, err := NewManager(config)
if err != nil {
    // Failed to initialize registries
    // This is typically due to:
    // - Missing directories
    // - Invalid YAML files
    // - Permission issues
    log.Fatalf("Failed to create manager: %v", err)
}
```

## Advanced: Custom Registry Configuration

```go
// Create manager
manager, _ := NewManager(config)

// Access and configure shared registries before creating bots
routineRegistry := manager.RoutineRegistry()

// Example: Validate all routines upfront
routineFiles := []string{"startup", "main_loop", "shutdown"}
for _, name := range routineFiles {
    if err := routineRegistry.Validate(name); err != nil {
        log.Printf("Warning: Routine '%s' invalid: %v", name, err)
    }
}

// Now create bots knowing all routines are valid
for i := 1; i <= 8; i++ {
    manager.CreateBot(i)
}
```

## Performance Considerations

### Startup Time
- **First bot**: ~normal (loads all registries)
- **Subsequent bots**: ~faster (registries already loaded)
- **Overall**: Faster multi-bot startup vs per-bot registries

### Runtime Performance
- **Template lookups**: Lock-free reads (RWMutex)
- **Routine execution**: Each bot gets independent ActionBuilder
- **Memory access**: Shared read-only data (CPU cache friendly)

### Shutdown
- **Per-bot shutdown**: Fast (doesn't unload shared registries)
- **Manager shutdown**: Cleans up shared resources once

## Monitoring

```go
// Check how many bots are running
count := manager.GetActiveCount()
fmt.Printf("Active bots: %d\n", count)

// Check registry usage
templateRegistry := manager.TemplateRegistry()
if tr, ok := templateRegistry.(*templates.TemplateRegistry); ok {
    stats := tr.CacheStats()
    fmt.Printf("Templates: %d loaded, %d cache hits\n",
        stats.LoadedCount, stats.CacheHits)
}
```
