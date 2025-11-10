# Bot Lifecycle Documentation

This document describes the complete lifecycle of bot initialization, execution, and shutdown in the PocketTCG bot system.

## Table of Contents

1. [Program Startup](#program-startup)
2. [Registry Initialization](#registry-initialization)
3. [Bot Group Creation](#bot-group-creation)
4. [Bot Instance Launch](#bot-instance-launch)
5. [Routine Execution](#routine-execution)
6. [Routine Restart Loop](#routine-restart-loop)
7. [Shutdown](#shutdown)

---

## 1. Program Startup

**Entry Point**: `cmd/bot-gui/main.go`

```go
func main() {
    // Create Fyne application
    myApp := app.NewWithID("com.jordanella.pocket-tcg-go")

    // Load configuration from Settings.ini
    cfg, err := config.LoadFromINI("Settings.ini", 1)

    // Create GUI controller
    controller := gui.NewController(cfg, myApp, mainWindow)

    // Build UI and show window
    mainWindow.ShowAndRun()
}
```

**Actions**:
1. Load configuration from `Settings.ini`
2. Create GUI controller
3. Initialize log tab (must be first for logging)
4. Initialize database (`bot.db`)
5. Run database migrations
6. Detect MuMu emulator instances

**Key Files**:
- `cmd/bot-gui/main.go` - Entry point
- `internal/gui/controller.go` - Main GUI controller
- `internal/database/migrations.go` - Database schema

---

## 2. Registry Initialization

**Location**: `internal/gui/manager_groups.go` - `initializeGlobalRegistries()`

When the Manager Groups tab is opened, global registries are initialized **once** and shared across all bot groups:

```go
func (t *ManagerGroupsTab) initializeGlobalRegistries() {
    // Load templates
    templatesPath := "./templates"
    t.templateRegistry = templates.NewTemplateRegistry(templatesPath)
    t.templateRegistry.LoadFromDirectory("./templates/registry")

    // Load routines
    routinesPath := "./routines"
    t.routineRegistry = actions.NewRoutineRegistry(routinesPath)
        .WithTemplateRegistry(t.templateRegistry)

    // Initialize pool manager (if database available)
    t.poolManager = accountpool.NewPoolManager(db)
    t.poolManager.LoadAllPools()
}
```

**What Gets Loaded**:

### Templates
- **Source**: `templates/registry/*.yaml`
- **Purpose**: Image templates for computer vision matching
- **Loading**: Lazy-loaded when first referenced
- **Storage**: Shared `TemplateRegistry` for all bots

### Routines
- **Source**: `routines/**/*.yaml`
- **Purpose**: Bot automation scripts with steps and sentries
- **Loading**: Parsed and validated on first access
- **Storage**: Shared `RoutineRegistry` for all bots

### Account Pools
- **Source**: Database `pools` and `pool_accounts` tables
- **Purpose**: Account pool definitions and assignments
- **Loading**: All pools loaded into memory
- **Storage**: `PoolManager` with SQL-backed pools

**Key Files**:
- `pkg/templates/template_registry.go` - Template management
- `internal/actions/routine_registry.go` - Routine management
- `internal/accountpool/pool_manager.go` - Account pool management

---

## 3. Bot Group Creation

**Location**: `internal/gui/manager_groups.go` - `showCreateGroupDialog()`

User creates a bot group through the GUI:

```yaml
Group Configuration:
  Name: "Premium Farmers"
  Routine: "farm_premium_packs.yaml"
  Instances: "1-4"  # Creates bots 1, 2, 3, 4
  Pool: "premium_accounts" # Database pool
```

**What Happens**:

```go
func (t *ManagerGroupsTab) createGroup(...) {
    // Create manager with shared registries
    manager := bot.NewManagerWithRegistries(
        config,
        t.templateRegistry,  // Shared
        t.routineRegistry,   // Shared
    )

    // Set account pool if specified
    if poolName != "" {
        pool := t.poolManager.GetPool(poolName)
        manager.SetAccountPool(pool)
    }

    // Set database connection for routine tracking
    manager.SetDatabase(db)

    // Create group object
    group := &ManagerGroup{
        Name:        groupName,
        Manager:     manager,
        RoutineName: routineName,
        InstanceIDs: []int{1, 2, 3, 4},
        AccountPool: pool,
    }
}
```

**Manager State**:
- Manager created but NO bots spawned yet
- Shared registries injected (templates, routines)
- Account pool assigned (optional)
- Database connection set (for tracking)

**Key Files**:
- `internal/bot/manager.go` - Manager creation
- `internal/gui/manager_groups.go` - Group management

---

## 4. Bot Instance Launch

**Location**: `internal/gui/manager_groups.go` - `startGroup()`

User clicks "Start" button on a group:

```go
func (t *ManagerGroupsTab) startGroup(group *ManagerGroup) {
    // Create bot instances
    for _, instanceID := range group.InstanceIDs {
        // CreateBot spawns new bot with shared registries
        botInstance, err := group.Manager.CreateBot(instanceID)

        // Start routine execution in background goroutine
        go func(b *bot.Bot, id int) {
            policy := bot.RestartPolicy{
                Enabled:        true,
                MaxRetries:     5,
                InitialDelay:   10 * time.Second,
                MaxDelay:       5 * time.Minute,
                BackoffFactor:  2.0,
                ResetOnSuccess: true,
            }

            // Execute with automatic restart
            group.Manager.ExecuteWithRestart(id, group.RoutineName, policy)
        }(botInstance, instanceID)
    }
}
```

**Bot Creation Process** (`Manager.CreateBot`):

```go
func (m *Manager) CreateBot(instance int) (*Bot, error) {
    // Create bot with config
    bot, err := New(instance, m.config)

    // Inject shared registries BEFORE initialization
    bot.templateRegistry = m.templateRegistry
    bot.routineRegistry = m.routineRegistry
    bot.manager = m

    // Initialize bot (skips registry loading since already set)
    bot.InitializeWithSharedRegistries()

    // Store in manager's bot map
    m.bots[instance] = bot

    return bot, nil
}
```

**Bot Initialization**:
1. Create ADB controller for emulator communication
2. Create CV service for image recognition
3. Create routine controller for pause/stop signaling
4. Create variable store for runtime variables
5. Create error monitor for error detection
6. Shared registries already injected (NO loading needed)

**Key Files**:
- `internal/bot/bot.go` - Bot creation and initialization
- `internal/bot/manager.go` - Bot lifecycle management

---

## 5. Routine Execution

**Location**: `internal/bot/manager.go` - `ExecuteWithRestart()`

The execution flow for a single routine iteration:

### Phase 1: Load Routine with Sentries

```go
func (m *Manager) ExecuteWithRestart(instance int, routineName string, policy RestartPolicy) {
    // Get routine with sentries from registry
    routineBuilder, sentries, err := bot.Routines().GetWithSentries(routineName)

    // Get routine metadata for config parameters
    routineMetadata := bot.Routines().GetMetadata(routineName + ".yaml")
    configParams := extractConfigParams(routineMetadata)

    // Create routine executor with sentries
    executor := actions.NewRoutineExecutor(routineBuilder, sentries)
}
```

### Phase 2: Start Execution Tracking

```go
// If database available and account injected, track execution
if db != nil && bot.Variables().Get("device_account_id") {
    executionID = database.StartRoutineExecution(db, accountID, routineName, instance)
    bot.Variables().Set("execution_id", fmt.Sprintf("%d", executionID))
}
```

### Phase 3: Execute Iteration

```go
executeIteration := func() error {
    // 1. Clear non-persistent variables
    bot.Variables().ClearNonPersistent()

    // 2. Reinitialize config variables with defaults
    actions.InitializeConfigVariables(bot, configParams, nil)

    // 3. Execute routine with sentries
    return executor.Execute(bot)
}
```

### Phase 4: Routine Executor Flow

**Location**: `internal/actions/routine_executor.go` - `Execute()`

```go
func (re *RoutineExecutor) Execute(bot BotInterface) error {
    // Initialize routine controller
    controller := bot.RoutineController()
    controller.Reset()
    controller.SetRunning()
    defer controller.SetCompleted()

    // Start sentry engine (parallel monitoring)
    if len(re.sentries) > 0 {
        re.sentryEngine = NewSentryEngine(bot, re.sentries)
        re.sentryEngine.Start()  // Spawns goroutines for each sentry
        defer re.sentryEngine.Stop()
    }

    // Execute main routine steps
    err := re.routine.Execute(bot)

    return err
}
```

### Phase 5: Sentry Engine Startup

**Location**: `internal/actions/sentry_engine.go` - `Start()`

```go
func (se *SentryEngine) Start() error {
    for i := range se.sentries {
        sentry := &se.sentries[i]

        // Load sentry routine
        builder, err := bot.Routines().Get(sentry.Routine)
        sentry.SetRoutineBuilder(builder)

        // Start goroutine for this sentry
        go se.monitorSentry(sentry)
    }
}
```

### Phase 6: Sentry Monitoring Loop

```go
func (se *SentryEngine) monitorSentry(sentry *Sentry) {
    ticker := time.NewTicker(sentry.GetFrequency())

    for {
        select {
        case <-ticker.C:
            // Execute sentry check (runs in parallel with main routine)
            se.executeSentry(sentry)
        case <-se.stopChan:
            return
        }
    }
}
```

### Phase 7: Sentry Execution

```go
func (se *SentryEngine) executeSentry(sentry *Sentry) {
    builder := sentry.GetRoutineBuilder()

    // Mark as sentry execution (ignores halt signals)
    builder.AsSentryExecution()

    // Execute sentry routine (runs in parallel)
    err := builder.Execute(se.bot)

    // Handle result
    if err == nil {
        // OnSuccess action (e.g., "resume")
    } else {
        // OnFailure action (e.g., "force_stop")
    }
}
```

**Sentry Behavior**:
- Sentries run **in parallel** with main routine
- Do NOT automatically pause main routine
- Only pause when `SentryHalt` action is called
- Resume with `SentryResume` action
- Immune to halt signals (won't block themselves)

### Phase 8: Main Routine Execution

**Location**: `internal/actions/builder.go` - `executeSteps()`

```go
func (ab *ActionBuilder) executeSteps(ctx context.Context, bot BotInterface) error {
    for _, step := range ab.steps {
        // Check for pause/stop signals (unless sentry execution)
        if !ab.checkExecutionState(bot) {
            return fmt.Errorf("routine stopped by controller")
        }

        // Execute step with timeout and retries
        if err := ab.executeStepWithRetries(ctx, bot, &step); err != nil {
            if !ab.ignoreErrors {
                return err
            }
        }
    }
    return nil
}
```

**Step Execution Features**:
- Individual step timeouts
- Automatic retries with delays
- Pause/stop checking between steps
- Sentry executions ignore pause signals

---

## 6. Routine Restart Loop

**Location**: `internal/bot/manager.go` - `ExecuteWithRestart()` (continued)

After a routine completes (success or failure), the restart logic handles the next iteration:

### Success Path (Infinite Loop)

```go
for {
    err := executeIteration()

    if err == nil {
        // SUCCESS - Routine completed normally

        // 1. Mark execution as completed in database
        database.CompleteRoutineExecution(db, executionID, 0, 0)

        // 2. Reset retry counter
        retryCount = 0
        currentDelay = policy.InitialDelay

        // 3. Start new execution tracking
        executionID = database.StartRoutineExecution(db, accountID, routineName, instance)
        bot.Variables().Set("execution_id", fmt.Sprintf("%d", executionID))

        // 4. Continue to next iteration (INFINITE LOOP)
        continue
    }

    // FAILURE - Handle retry logic below
}
```

### Failure Path (Retry with Backoff)

```go
// Check if exceeded max retries
if policy.MaxRetries > 0 && retryCount >= policy.MaxRetries {
    // Mark as failed and EXIT
    database.FailRoutineExecution(db, executionID, err.Error())
    return fmt.Errorf("routine failed after %d retries: %w", retryCount, err)
}

// Retry with exponential backoff
retryCount++
fmt.Printf("Routine failed (attempt %d/%d): %v. Retrying in %v...\n",
    retryCount, policy.MaxRetries, err, currentDelay)

// Wait before retry
time.Sleep(currentDelay)

// Calculate next backoff delay
nextDelay := InitialDelay * (BackoffFactor ^ retryCount)
if nextDelay > MaxDelay {
    nextDelay = MaxDelay
}
currentDelay = nextDelay

// Reset routine controller for retry
bot.RoutineController().Reset()

// Loop continues to retry
```

### Variable Reinitialization on Each Iteration

```go
executeIteration := func() error {
    // STEP 1: Clear non-persistent variables
    // Persistent variables (marked with persist: true) are kept
    bot.Variables().ClearNonPersistent()

    // STEP 2: Reinitialize config variables to defaults
    // Config parameters from routine YAML are reinitialized
    actions.InitializeConfigVariables(bot, configParams, nil)

    // STEP 3: Execute routine (fresh state)
    return executor.Execute(bot)
}
```

**Variable Lifecycle**:
- **Non-persistent variables**: Cleared each iteration
- **Persistent variables** (`persist: true`): Kept across iterations
- **Config parameters**: Reinitialized to defaults each iteration
- **System variables**: Manually managed (e.g., `execution_id`, `device_account_id`)

### Stopping the Loop

The infinite loop can be stopped in three ways:

1. **User Stops Group**: Calls `Manager.ShutdownAll()` which stops all bots
2. **Sentry Force Stop**: Sentry's `OnFailure: force_stop` triggers `controller.ForceStop()`
3. **Max Retries Exceeded**: After repeated failures, returns error and exits

**Key Files**:
- `internal/bot/manager.go` - Restart loop logic
- `internal/actions/routine_executor.go` - Execution orchestration
- `internal/actions/builder.go` - Step execution

---

## 7. Shutdown

### Graceful Shutdown Process

**Location**: `internal/gui/manager_groups.go` - `stopGroup()`

```go
func (t *ManagerGroupsTab) stopGroup(group *ManagerGroup) {
    // Stop all bots in the group
    group.Manager.ShutdownAll()

    // Close account pool
    if group.AccountPool != nil {
        group.AccountPool.Close()
    }
}
```

**Manager Shutdown** (`Manager.ShutdownAll()`):

```go
func (m *Manager) ShutdownAll() {
    // Shutdown all bot instances
    for instance, bot := range m.bots {
        bot.ShutdownWithSharedRegistries()
        delete(m.bots, instance)
    }

    // Unload all template images (memory cleanup)
    if m.templateRegistry != nil {
        m.templateRegistry.UnloadAll()
    }

    // Note: Routines don't need cleanup (eagerly loaded)
}
```

**Bot Shutdown** (`Bot.ShutdownWithSharedRegistries()`):

```go
func (b *Bot) ShutdownWithSharedRegistries() {
    // Stop error monitor
    if b.errorMonitor != nil {
        b.errorMonitor.Stop()
    }

    // Stop sentries (if running)
    // Handled by deferred Stop() in routine executor

    // Signal shutdown via context cancellation
    if b.cancel != nil {
        b.cancel()
    }

    // Do NOT unload shared registries (still used by other bots)
}
```

**Cleanup Order**:
1. Cancel bot context (signals all goroutines to stop)
2. Stop error monitor
3. Sentry engine stops (deferred from routine executor)
4. ADB controller cleanup
5. CV service cleanup
6. Template registry cleanup (manager level)

**Key Files**:
- `internal/bot/manager.go` - Manager shutdown
- `internal/bot/bot.go` - Bot shutdown
- `internal/gui/manager_groups.go` - GUI shutdown

---

## Complete Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        PROGRAM STARTUP                           │
├─────────────────────────────────────────────────────────────────┤
│ 1. Load Settings.ini                                            │
│ 2. Initialize GUI                                               │
│ 3. Initialize Database                                          │
│ 4. Run Migrations                                               │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│                    REGISTRY INITIALIZATION                       │
├─────────────────────────────────────────────────────────────────┤
│ • Load Template Registry (./templates)                          │
│ • Load Routine Registry (./routines)                            │
│ • Load Account Pool Manager (database)                          │
│ • Parse routine YAML (steps, sentries, config)                  │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│                     BOT GROUP CREATION                           │
├─────────────────────────────────────────────────────────────────┤
│ 1. User creates group via GUI                                   │
│ 2. Manager created with shared registries                       │
│ 3. Account pool assigned (if specified)                         │
│ 4. Database connection set                                      │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│                   BOT INSTANCE LAUNCH                            │
├─────────────────────────────────────────────────────────────────┤
│ For each instance (e.g., 1-4):                                  │
│   1. CreateBot() spawns new bot                                 │
│   2. Inject shared registries                                   │
│   3. Initialize ADB, CV, monitors                               │
│   4. Start goroutine: ExecuteWithRestart()                      │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│                    ROUTINE EXECUTION                             │
├─────────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────┐                     │
│ │ ITERATION START                          │                     │
│ │ 1. Clear non-persistent variables       │                     │
│ │ 2. Reinitialize config variables        │                     │
│ │ 3. Start execution tracking (database)  │                     │
│ └─────────────────────────────────────────┘                     │
│                    ↓                                             │
│ ┌─────────────────────────────────────────┐                     │
│ │ SENTRY ENGINE                           │                     │
│ │ 1. Start sentry goroutines              │  ← Parallel         │
│ │ 2. Monitor at configured intervals      │    Execution        │
│ │ 3. Execute sentry routines              │                     │
│ │ 4. Only halt if SentryHalt called       │                     │
│ └─────────────────────────────────────────┘                     │
│                    ↓                                             │
│ ┌─────────────────────────────────────────┐                     │
│ │ MAIN ROUTINE                             │                     │
│ │ 1. Execute steps sequentially           │                     │
│ │ 2. Check pause/stop between steps       │                     │
│ │ 3. Handle timeouts and retries          │                     │
│ │ 4. Complete or fail                     │                     │
│ └─────────────────────────────────────────┘                     │
│                    ↓                                             │
│ ┌─────────────────────────────────────────┐                     │
│ │ SUCCESS?                                 │                     │
│ │  YES: Mark complete, restart iteration  │  → INFINITE LOOP    │
│ │   NO: Retry with backoff or fail        │  → Max retries exit │
│ └─────────────────────────────────────────┘                     │
└─────────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────────┐
│                         SHUTDOWN                                 │
├─────────────────────────────────────────────────────────────────┤
│ 1. User stops group or max retries exceeded                     │
│ 2. Stop all bot instances                                       │
│ 3. Cancel contexts (signals goroutines)                         │
│ 4. Stop sentry engines                                          │
│ 5. Cleanup ADB, CV, monitors                                    │
│ 6. Unload templates                                             │
│ 7. Close account pools                                          │
└─────────────────────────────────────────────────────────────────┘
```

---

## Summary

### Yes, All Systems Are in Place ✓

1. **Registry Loading**: ✓ Templates, routines, account pools loaded globally
2. **Bot Group Creation**: ✓ Manager with shared registries, account pool, database
3. **Bot Instance Launch**: ✓ CreateBot() spawns bots with shared resources
4. **Sentry Initialization**: ✓ Sentry goroutines started automatically
5. **Routine Execution**: ✓ Executor handles main routine + sentries
6. **Variable Reinitialization**: ✓ Non-persistent cleared, config reinitialized
7. **Infinite Loop**: ✓ Continues until stopped or max retries exceeded
8. **Graceful Shutdown**: ✓ Stops all bots, cleans up resources

### Key Features

- **Shared Registries**: Templates and routines loaded once, shared by all bots
- **Parallel Sentries**: Run independently, only halt when needed
- **Infinite Execution**: Routines restart automatically on success
- **Variable Persistence**: Config params reset, persistent vars kept
- **Database Tracking**: All executions tracked with metrics
- **Retry Logic**: Exponential backoff on failure, restart on success
- **Graceful Shutdown**: Clean resource cleanup on stop

The complete bot lifecycle is fully implemented and operational.
