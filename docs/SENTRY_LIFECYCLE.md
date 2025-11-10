# Sentry Lifecycle Management

This document describes the global sentry lifecycle management system that handles registration, deduplication, and reference counting for sentries across nested routine executions.

## Problem Statement

The original sentry implementation had several issues with nested routine execution:

1. **Duplicate Sentries**: When a subroutine with sentries was called, new sentry instances were created, leading to multiple sentries monitoring the same condition
2. **Wasted Resources**: Each routine execution started its own sentry engine, even if the same sentries were already running
3. **Inconsistent Poll Rates**: Multiple instances of the same sentry could have different frequencies, causing unpredictable behavior
4. **Lifecycle Mismatch**: Sentries stopped when a subroutine ended, even if the parent routine still needed them

## Solution: Global Sentry Manager

The `SentryManager` provides centralized sentry lifecycle management at the bot level, ensuring:
- **Single Instance**: Only one instance of each unique sentry runs at a time
- **Reference Counting**: Sentries persist as long as any routine needs them
- **Optimal Poll Rate**: Uses the lowest frequency (fastest polling) when multiple routines request the same sentry
- **Automatic Cleanup**: Sentries stop when the last routine using them completes

---

## Architecture

### SentryManager

**Location**: `internal/actions/sentry_manager.go`

```go
type SentryManager struct {
    bot    BotInterface
    active map[string]*ManagedSentry  // Key: sentry routine name
}

type ManagedSentry struct {
    Sentry       Sentry
    RefCount     int            // Number of active routines using this sentry
    Engine       *SentryEngine  // The actual running sentry engine
    MinFrequency int            // Lowest frequency (highest poll rate)
}
```

**Key Operations**:
- `Register(sentries)` - Register sentries for a routine execution
- `Unregister(sentries)` - Unregister when routine completes
- `StopAll()` - Stop all sentries during bot shutdown

### Integration with Bot

**Location**: `internal/bot/bot.go`

```go
type Bot struct {
    // ...
    sentryManager *actions.SentryManager  // Global sentry lifecycle manager
}

// Initialized during bot initialization
func (b *Bot) initializeInternal(sharedRegistries bool) error {
    // ... other initialization ...

    b.sentryManager = actions.NewSentryManager(b)
    return nil
}

// Accessor for BotInterface
func (b *Bot) SentryManager() *actions.SentryManager {
    return b.sentryManager
}
```

### Integration with RoutineExecutor

**Location**: `internal/actions/routine_executor.go`

```go
func (re *RoutineExecutor) Execute(bot BotInterface) error {
    // Register sentries with global manager
    if len(re.sentries) > 0 {
        sentryMgr := bot.SentryManager()
        if err := sentryMgr.Register(re.sentries); err != nil {
            return err
        }
        defer sentryMgr.Unregister(re.sentries)
    }

    // Execute routine
    return re.routine.Execute(bot)
}
```

---

## Lifecycle Flow

### Scenario 1: Simple Routine with Sentries

```yaml
# main_routine.yaml
routine_name: "Main Routine"
sentries:
  - routine: error_dialog_check
    frequency: 5
    on_failure: force_stop

steps:
  - action: Click
    x: 540
    y: 800
```

**Flow**:
1. `RoutineExecutor.Execute()` called for `main_routine`
2. `SentryManager.Register()` called with `[error_dialog_check]`
3. Manager checks: `error_dialog_check` not in active map
4. New `ManagedSentry` created with `RefCount=1`
5. `SentryEngine` started for `error_dialog_check`
6. Main routine executes
7. Routine completes → `defer SentryManager.Unregister()`
8. Manager decrements `RefCount` to 0
9. `SentryEngine` stopped and sentry removed from active map

**Result**: Sentry runs only while routine is active

---

### Scenario 2: Nested Routines with Same Sentry

```yaml
# parent_routine.yaml
routine_name: "Parent Routine"
sentries:
  - routine: error_dialog_check
    frequency: 10  # Poll every 10 seconds

steps:
  - action: RunRoutine
    routine: child_routine.yaml

# child_routine.yaml
routine_name: "Child Routine"
sentries:
  - routine: error_dialog_check
    frequency: 5  # Poll every 5 seconds (faster!)

steps:
  - action: Click
    x: 540
    y: 800
```

**Flow**:

1. **Parent Starts**
   - `Register([error_dialog_check freq=10])`
   - Creates `ManagedSentry`: `RefCount=1, MinFrequency=10`
   - Starts `SentryEngine` with 10s interval

2. **Child Starts** (via `RunRoutine`)
   - `Register([error_dialog_check freq=5])`
   - Manager finds existing `error_dialog_check`
   - Increments `RefCount` to 2
   - **Detects faster frequency**: 5 < 10
   - **Stops old engine** and **restarts with 5s interval**
   - Logs: "Sentry 'error_dialog_check' frequency updated from 10s to 5s"

3. **Child Completes**
   - `Unregister([error_dialog_check])`
   - Decrements `RefCount` to 1
   - Sentry **continues running** (parent still needs it)
   - **Keeps 5s frequency** (lowest requested remains)

4. **Parent Completes**
   - `Unregister([error_dialog_check])`
   - Decrements `RefCount` to 0
   - Stops `SentryEngine`
   - Removes from active map

**Result**:
- No duplicate sentries created
- Uses fastest poll rate (5s) while both routines active
- Sentry persists until last routine completes

---

### Scenario 3: Multiple Different Sentries

```yaml
# routine.yaml
routine_name: "Multiple Sentries Example"
sentries:
  - routine: error_dialog_check
    frequency: 5
  - routine: optional_popup_dismiss
    frequency: 3
  - routine: connection_check
    frequency: 30

steps:
  - action: Click
    x: 540
    y: 800
```

**Flow**:
1. `Register([error_dialog_check, optional_popup_dismiss, connection_check])`
2. Manager creates 3 separate `ManagedSentry` entries
3. Each gets its own `SentryEngine` with respective frequency
4. All run in parallel during routine execution
5. `Unregister([...])` stops all 3 when routine completes

**Result**: Each unique sentry managed independently

---

### Scenario 4: Overlapping Routines with Shared Sentries

```yaml
# farm_routine_1.yaml
sentries:
  - routine: error_dialog_check
    frequency: 5

# farm_routine_2.yaml
sentries:
  - routine: error_dialog_check
    frequency: 8
  - routine: connection_check
    frequency: 30
```

**Timeline**:

```
Time 0s:  Routine 1 starts
          Register: error_dialog_check (5s) → RefCount=1, MinFreq=5

Time 2s:  Routine 2 starts
          Register: error_dialog_check (8s) → RefCount=2, MinFreq=5 (no change, already faster)
          Register: connection_check (30s) → RefCount=1, MinFreq=30

Time 10s: Routine 1 completes
          Unregister: error_dialog_check → RefCount=1 (still running for Routine 2)

Time 15s: Routine 2 completes
          Unregister: error_dialog_check → RefCount=0 (now stopped)
          Unregister: connection_check → RefCount=0 (stopped)
```

**Result**: Sentries intelligently shared across concurrent routines

---

## Reference Counting Algorithm

### Registration

```go
func (sm *SentryManager) Register(sentries []Sentry) error {
    for each sentry:
        key = sentry.Routine

        if key exists in active:
            // Existing sentry
            existing.RefCount++

            if sentry.Frequency < existing.MinFrequency:
                // Need faster polling
                existing.Engine.Stop()
                existing.MinFrequency = sentry.Frequency
                existing.Sentry.Frequency = sentry.Frequency
                existing.Engine = NewSentryEngine(...)
                existing.Engine.Start()
        else:
            // New sentry
            active[key] = &ManagedSentry{
                Sentry:       sentry,
                RefCount:     1,
                Engine:       NewSentryEngine(...),
                MinFrequency: sentry.Frequency,
            }
}
```

### Unregistration

```go
func (sm *SentryManager) Unregister(sentries []Sentry) {
    for each sentry:
        key = sentry.Routine
        existing = active[key]

        existing.RefCount--

        if existing.RefCount <= 0:
            // Last routine using this sentry
            existing.Engine.Stop()
            delete(active, key)
}
```

---

## Frequency Selection Logic

When multiple routines request the same sentry with different frequencies:

**Rule**: Use the **lowest** frequency value (fastest polling rate)

**Rationale**:
- Lower frequency number = more frequent checks
- Ensures the most critical routine's requirements are met
- Over-polling is preferable to under-polling for error detection

**Example**:
- Routine A requests `error_dialog_check` at 10s
- Routine B requests `error_dialog_check` at 5s
- Result: Sentry polls every 5s (faster)

**When Frequency Changes**:
1. Stop existing `SentryEngine`
2. Update `MinFrequency` to new value
3. Create and start new `SentryEngine` with updated frequency
4. Log the change for debugging

---

## Bot Shutdown

When a bot shuts down:

```go
func (b *Bot) shutdownInternal(sharedRegistries bool) {
    // Stop all sentries FIRST
    if b.sentryManager != nil {
        b.sentryManager.StopAll()
    }

    // Then stop other services...
}
```

**`SentryManager.StopAll()`**:
1. Iterates through all active sentries
2. Calls `Engine.Stop()` on each
3. Clears the active map
4. Logs: "Stopping all sentries (N active)"

**Result**: Clean shutdown with no orphaned goroutines

---

## Debugging and Monitoring

### Get Active Sentry Count

```go
count := bot.SentryManager().GetActiveCount()
fmt.Printf("Bot has %d active sentries\n", count)
```

### Get Sentry Info

```go
info := bot.SentryManager().GetSentryInfo()
for name, details := range info {
    fmt.Printf("Sentry: %s, RefCount: %d, Frequency: %ds\n",
        name, details.RefCount, details.Frequency)
}
```

**Output Example**:
```
Sentry: error_dialog_check, RefCount: 2, Frequency: 5s
Sentry: connection_check, RefCount: 1, Frequency: 30s
```

### Log Messages

The SentryManager emits detailed logs:

```
Bot 1: Sentry manager initialized
Bot 1: Starting new sentry 'error_dialog_check' (frequency: 5s)
Bot 1: Sentry 'error_dialog_check' already active (refcount: 2)
Bot 1: Sentry 'error_dialog_check' frequency updated from 10s to 5s (faster polling)
Bot 1: Sentry 'error_dialog_check' unregistered (refcount: 1)
Bot 1: Stopping sentry 'error_dialog_check' (no more active routines)
Bot 1: Stopping all sentries (2 active)
```

---

## Benefits

### 1. No Duplicate Sentries
- Only one instance per unique sentry routine
- Eliminates wasted CPU/memory from duplicates

### 2. Proper Lifecycle
- Sentries persist across nested routine boundaries
- Stop only when truly no longer needed

### 3. Optimal Performance
- Uses fastest poll rate when multiple routines need same sentry
- Automatically adjusts frequency on-the-fly

### 4. Clean Shutdown
- All sentries stopped during bot shutdown
- No orphaned goroutines

### 5. Thread-Safe
- All operations protected by mutex
- Safe for concurrent routine execution

---

## Migration Notes

### Old Behavior (Before SentryManager)

```go
// RoutineExecutor.Execute() - OLD
if len(re.sentries) > 0 {
    re.sentryEngine = NewSentryEngine(bot, re.sentries)
    re.sentryEngine.Start()
    defer re.sentryEngine.Stop()
}
```

**Issues**:
- Each routine creates its own `SentryEngine`
- Duplicate sentries when routines nest
- All sentries stop when subroutine ends

### New Behavior (With SentryManager)

```go
// RoutineExecutor.Execute() - NEW
if len(re.sentries) > 0 {
    bot.SentryManager().Register(re.sentries)
    defer bot.SentryManager().Unregister(re.sentries)
}
```

**Benefits**:
- Global sentry management
- Automatic deduplication
- Reference counting
- Frequency optimization

### No YAML Changes Required

The sentry YAML syntax remains the same:

```yaml
sentries:
  - routine: error_dialog_check
    frequency: 5
    severity: high
    on_success: resume
    on_failure: force_stop
```

All improvements are transparent to routine authors!

---

## Technical Details

### Key Data Structures

```go
// SentryManager - Bot-level singleton
type SentryManager struct {
    bot    BotInterface
    mu     sync.RWMutex
    active map[string]*ManagedSentry
}

// ManagedSentry - Tracks a single sentry instance
type ManagedSentry struct {
    Sentry       Sentry          // Sentry definition
    RefCount     int             // Active routine count
    Engine       *SentryEngine   // Running engine
    MinFrequency int             // Fastest poll rate
}

// SentryInfo - Display information
type SentryInfo struct {
    Routine   string
    RefCount  int
    Frequency int
    Severity  string
    OnSuccess string
    OnFailure string
}
```

### Thread Safety

- All public methods acquire `mu` lock
- Use `RLock` for read-only operations
- Use `Lock` for mutations
- Engine start/stop called within lock to prevent races

### Memory Management

- Active map only grows with unique sentries
- Entries removed when `RefCount` reaches 0
- No memory leaks from orphaned sentries

---

## See Also

- [Sentry Control Actions](SENTRY_CONTROL.md) - SentryHalt/SentryResume
- [Bot Lifecycle](BOT_LIFECYCLE.md) - Complete initialization flow
- [Sentry System](SENTRIES.md) - Sentry configuration guide
