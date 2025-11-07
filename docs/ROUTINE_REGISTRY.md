# Routine Registry

The Routine Registry provides a lazy-loading, caching system for managing YAML-based routines with proper lifecycle management.

## Overview

The `RoutineRegistry` is responsible for:
- **Lazy loading**: Routines are only loaded when first requested
- **Caching**: Once loaded, routines are cached to avoid redundant file I/O
- **Validation**: Check if routines are valid without fully loading them
- **Lifecycle management**: Reference counting ensures routines are released when no longer needed

## Architecture

### Key Components

1. **RoutineRegistryInterface** - The interface exposed to actions and bots
2. **RoutineRegistry** - The concrete implementation with caching and lifecycle
3. **BotInterface.Routines()** - Accessor method that returns the registry
4. **RunRoutine action** - Uses the registry to execute other routines

### Design Principles

- **Validation at load time**: Routines are fully validated when loaded, ensuring errors are caught early
- **Build on demand**: Routines are only built into executable steps when first accessed
- **Cached after build**: Once built, routines are cached for reuse
- **Released when done**: Reference counting allows cleanup when bots stop

## Usage

### Basic Usage

```go
// In bot initialization
registry := actions.NewRoutineRegistry("routines/")
registry.WithTemplateRegistry(bot.templateRegistry)

// The bot exposes the registry
func (b *Bot) Routines() actions.RoutineRegistryInterface {
    return b.routineRegistry
}

// Actions use it automatically
// In YAML:
steps:
  - action: RunRoutine
    routine_name: my_subroutine
```

### API Methods

#### Has(name string) bool
Checks if a routine file exists without loading it.

```go
if registry.Has("common_navigation") {
    // Routine file exists
}
```

#### Validate(name string) error
Validates a routine's YAML structure and dependencies without caching it.

```go
err := registry.Validate("my_routine")
if err != nil {
    // Routine has errors
}
```

#### Get(name string) (*ActionBuilder, error)
Loads and caches a routine, returning the executable ActionBuilder.

```go
builder, err := registry.Get("my_routine")
if err != nil {
    return err
}
// Execute the routine
err = builder.Execute(bot)
```

#### Acquire(name string) error
Increases the reference count for a routine (optional, for advanced usage).

```go
registry.Acquire("critical_routine")
defer registry.Release("critical_routine")
```

#### Release(name string)
Decreases the reference count. When count reaches zero, the routine is unloaded from cache.

#### ReleaseAll(botID string)
Releases all routines held by a specific bot instance. Called when bot stops.

```go
// On bot shutdown
registry.ReleaseAll(bot.ID())
```

#### Clear()
Removes all cached routines. Useful for testing or forcing reload.

```go
registry.Clear()
```

## Path Resolution

The registry automatically resolves routine paths:

1. **Name only**: `"my_routine"` → `routines/my_routine.yaml`
2. **With extension**: `"my_routine.yaml"` → `routines/my_routine.yaml`
3. **Subdirectory**: `"nav/home"` → `routines/nav/home.yaml`
4. **Absolute path**: `"/custom/path/routine.yaml"` → `/custom/path/routine.yaml`

## Lifecycle

### Loading Flow

```
User calls RunRoutine
    ↓
Registry.Get() called
    ↓
Check cache
    ├─ Found: Return cached builder
    └─ Not found: Load from file
           ↓
       Validate YAML
           ↓
       Build executable steps
           ↓
       Cache the builder
           ↓
       Return builder
```

### Cleanup Flow

```
Bot stops
    ↓
registry.ReleaseAll(botID)
    ↓
Remove bot from all ref counts
    ↓
For each routine with refCount = 0:
    Remove from cache
```

## Thread Safety

The `RoutineRegistry` uses `sync.RWMutex` for thread-safe operations:
- **Read lock** for cache lookups (`Has`, `Get` from cache)
- **Write lock** for cache modifications (loading, releasing, clearing)

## Best Practices

### 1. Initialize Once Per Bot

```go
func (b *Bot) Initialize() error {
    b.routineRegistry = actions.NewRoutineRegistry("routines/")
    b.routineRegistry.WithTemplateRegistry(b.templateRegistry)
    // ...
}
```

### 2. Validate on Startup (Optional)

```go
// Validate critical routines at bot startup
criticalRoutines := []string{"startup", "error_recovery", "shutdown"}
for _, name := range criticalRoutines {
    if err := registry.Validate(name); err != nil {
        return fmt.Errorf("critical routine '%s' invalid: %w", name, err)
    }
}
```

### 3. Use RunRoutine for Composition

Break complex routines into smaller, reusable pieces:

```yaml
# main_routine.yaml
routine_name: "Main Flow"
steps:
  - action: RunRoutine
    routine_name: startup_sequence

  - action: RunRoutine
    routine_name: core_logic

  - action: RunRoutine
    routine_name: cleanup_sequence
```

### 4. Clean Up on Shutdown

```go
func (b *Bot) Shutdown() {
    b.routineRegistry.ReleaseAll(b.id)
    // ...
}
```

## Testing

The registry includes mock support for testing:

```go
func TestMyAction(t *testing.T) {
    // Create temporary routines
    tempDir := t.TempDir()
    registry := actions.NewRoutineRegistry(tempDir)

    // Create mock bot
    bot := NewMockBotWithRoutines(templateRegistry, registry)

    // Test your action
    // ...
}
```

## Error Handling

### Common Errors

1. **Routine Not Found**
   ```
   failed to get routine 'my_routine': routine file not found: routines/my_routine.yaml
   ```
   - Check file exists
   - Verify path is correct
   - Ensure file extension (.yaml or .yml)

2. **Validation Errors**
   ```
   failed to load routine 'my_routine': routine 'MyRoutine' step 3 validation failed: template 'Foo' not found in registry
   ```
   - Check template exists
   - Verify action parameters
   - Review YAML syntax

3. **Circular Dependencies**
   - Current implementation doesn't detect circular dependencies
   - Avoid having routines call each other in a loop
   - Future enhancement: circular dependency detection

## Performance Considerations

- **First access**: File I/O + YAML parsing + validation + building (~1-10ms depending on complexity)
- **Subsequent access**: Memory lookup (~<1μs)
- **Memory usage**: Each cached routine holds the built ActionBuilder in memory
- **Recommended**: For frequently used routines, the cache provides significant performance benefits

## Future Enhancements

Possible future improvements:

1. **Circular dependency detection**: Detect and prevent infinite loops
2. **Hot reload**: Watch for file changes and automatically reload
3. **Metrics**: Track routine usage, execution time, cache hit rate
4. **Preloading**: Bulk load critical routines at startup
5. **Version management**: Track routine versions for debugging
6. **Per-bot isolation**: Separate caches for different bot instances
