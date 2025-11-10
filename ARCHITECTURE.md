# PocketTCGoBot Architecture Documentation

This document details the architecture of the PocketTCGoBot automation system, covering the core systems, design patterns, and execution flow.

---

## Table of Contents

1. [CV (Computer Vision) System](#1-cv-computer-vision-system)
2. [Four Core Registries](#2-four-core-registries)
3. [YAML Loading & Configuration](#3-yaml-loading--configuration)
4. [Bot Groups & Orchestration](#4-bot-groups--orchestration)
5. [Account Pools](#5-account-pools)
6. [Accounts System](#6-accounts-system)
7. [Routines](#7-routines)
8. [Actions System](#8-actions-system)
9. [Sentry System](#9-sentry-system)
10. [Execution Flow](#10-execution-flow)
11. [Design Patterns](#design-patterns)
12. [Architecture Strengths](#architecture-strengths)

---

## 1. CV (Computer Vision) System

**Location**: [internal/cv/service.go](internal/cv/service.go)

The CV service provides intelligent template matching with frame caching and multiple matching algorithms.

### Key Features

- **Frame Caching**: 100ms TTL prevents redundant screenshots
- **Three Algorithms**:
  - SAD (Sum of Absolute Differences) - Fast
  - SSD (Sum of Squared Differences) - Balanced
  - NCC (Normalized Cross-Correlation) - Accurate
- **Title Bar Exclusion**: Automatically ignores window chrome
- **Template Integration**: Templates loaded from Template Registry and cached for performance

### Key Operations

```go
FindTemplate(templateName, config)     // Find single match
FindMultipleTemplates(paths, config)   // Find all at once
WaitForTemplate(name, config, timeout) // Wait until found
CheckColor(x, y, color, tolerance)     // Pixel color verification
```

Templates are loaded from the Template Registry and cached for performance. The service supports search regions and scaling for flexible matching.

---

## 2. Four Core Registries

### A. Template Registry
**Location**: [pkg/templates/registry.go](pkg/templates/registry.go)

Manages image templates loaded from YAML definitions.

- Thread-safe with RWMutex
- Optional image caching layer
- Supports preloading and lazy loading
- Stores template metadata (path, scale, regions)

### B. Routine Registry
**Location**: [internal/actions/routine_registry.go](internal/actions/routine_registry.go)

Discovers and manages routine definitions from the `routines/` folder.

- Recursive discovery from routines folder
- Eager loading & validation at startup (fail-fast)
- Stores compiled `ActionBuilder`, sentries, configs, metadata
- Supports namespacing (e.g., `combat/battle_loop`)
- Separates valid from invalid routines

### C. Action Registry
**Location**: [internal/actions/registry.go](internal/actions/registry.go)

Maps action names to concrete types via reflection.

- Enables polymorphic YAML unmarshaling
- 50+ action types registered (click, sleep, if, while, etc.)
- Used during YAML parsing to create correct action instances
- Centralized registration point for all action types

### D. Account Pool Manager
**Location**: [internal/accountpool/pool_manager.go](internal/accountpool/pool_manager.go)

Registry for named account pools.

- Creates `UnifiedAccountPool` instances from definitions
- Manages multiple independent pools
- Provides pool lookup by name
- Handles pool lifecycle

---

## 3. YAML Loading & Configuration

### INI Configuration
**Location**: [internal/config/loader.go](internal/config/loader.go)

Legacy `Settings.ini` file loaded into `bot.Config` struct with 100+ fields:
- Instance configuration
- Pack preferences
- Delays and timing
- ADB paths
- S4T settings
- OCR configuration

### Routine YAML Structure
**Location**: [internal/actions/routine.go](internal/actions/routine.go)

```yaml
routine_name: "Display Name"
description: "Purpose of this routine"
tags: ["tag1", "tag2"]       # Metadata for filtering

config:                       # User-configurable parameters
  - name: param_name
    type: int
    default: 10
    persist: true             # Save between runs

sentries:                      # Error handlers
  - routine: error_handling/handler
    frequency: 15              # Check every 15 seconds
    severity: high
    on_success: resume         # Resume main routine if sentry succeeds
    on_failure: stop           # Stop bot if sentry fails

steps:                         # Action sequence
  - action: click
    template: button_name
  - action: sleep
    duration: 2000
  - action: ifimagefound
    template: popup
    actions:
      - action: click
        template: close_button
```

#### Loading Process

1. Read YAML file
2. Parse into `Routine` struct
3. Custom `UnmarshalYAML` handles polymorphic steps
4. Each step unmarshals via action registry
5. Validate all steps
6. Build into executable `ActionBuilder`

### Pool Definition YAML
**Location**: [internal/accountpool/unified_pool.go](internal/accountpool/unified_pool.go)

Combines multiple account sources:
- SQL queries for account sources
- Include/exclude lists
- Watched folders for imports
- Sort methods and retry policies

---

## 4. Bot Groups & Orchestration

**Location**: [internal/bot/orchestrator.go](internal/bot/orchestrator.go)

### Architecture

```
Orchestrator (global coordinator)
├─ Shared Registries
│  ├─ TemplateRegistry
│  ├─ RoutineRegistry
│  ├─ PoolManager (pool definitions)
│  └─ EmulatorManager
└─ BotGroups (multiple independent groups)
   └─ BotGroup "farm-group"
      ├─ OrchestrationID (UUID: "550e8400-...")
      ├─ Manager (lifecycle control)
      ├─ RoutineName (which routine to execute)
      ├─ AvailableInstances ([1, 2, 3, 4, 5])
      ├─ AccountPool (execution-specific pool instance)
      ├─ InitialAccountCount (for progress monitoring)
      └─ ActiveBots (currently running)
         ├─ Bot instance 1 → Emulator Instance 1
         ├─ Bot instance 2 → Emulator Instance 2
         └─ Bot instance N → Emulator Instance N
```

### Key Concepts

- **Orchestrator**: Manages multiple independent bot groups
- **BotGroup**: Set of bots running same routine with same account pool
  - Each group gets a unique **OrchestrationID** (UUID) for execution isolation
  - Tracks **InitialAccountCount** for progress monitoring
- **Bot Instance**: Single bot running on specific emulator
- **Emulator Instance**: MuMu emulator (mapped 1:1 to bot instance)

### Orchestration ID System

Each bot group receives a unique UUID on creation:
- **Isolates execution contexts** - prevents stale records from old runs affecting new groups
- **Tracks routine executions** - `routine_executions` table includes `orchestration_id`
- **Manages account checkouts** - database tracks which orchestration has which account
- **Enables multi-tenancy** - multiple groups can run same routine without conflicts

### Workflow

1. **Create Group**: Generate unique orchestration ID, create manager
2. **Set Account Pool**: Create execution-specific pool instance from definition, log initial count
3. **Validate**: Check routine exists and templates are available
4. **Allocate**: Reserve emulator instances
5. **Launch**: For each bot - create, initialize, launch (staggered)
6. **Monitor**: Track bot status and account progress
7. **Shutdown**: Release all account checkouts for this orchestration
8. **Handle Failures**: Per restart policy

---

## 5. Account Pools

**Location**: [internal/accountpool/unified_pool.go](internal/accountpool/unified_pool.go)

### Pool Architecture

Account pools operate at two distinct levels:

1. **Pool Definitions** (YAML files): Shared templates defining how to query accounts
   - Multiple orchestrations can use the same pool definition
   - Stored in `pools/` directory
   - Managed by `PoolManager`

2. **Execution-Specific Pools**: Per-orchestration queue instances
   - Each bot group creates its own pool instance from a definition
   - Independent queues prevent cross-orchestration interference
   - Tracked via `InitialAccountCount` for progress monitoring

### Pool Definition

A pool combines 4 account sources:

1. **SQL Queries**: `SELECT device_account, device_password, ...`
2. **Manual Include**: Specific account IDs to add
3. **Watched Paths**: Folders to scan for XML imports
4. **Exclude List**: Accounts to remove

### Resolution Workflow

```
Query results → Include manual → Watched paths → Exclude
```

### Account Lifecycle with Checkout System

```
Pool Definition (YAML)
    ↓
Orchestration creates execution-specific pool
    ↓
Available → GetNext() → [Database Checkout Check] → In Use → [Finalize] → Terminal State
                              ↓                                    │
                        Already checked out?                       ├─ CompleteAccount → Completed (REMOVED + released)
                        → Defer & retry                            ├─ MarkAccountFailed → Failed (REMOVED + released)
                              ↓                                    └─ ReturnAccount → Available (BACK + released)
                        Checkout successful
                        → Inject into emulator
```

**Important**:
- Accounts do NOT automatically return to the pool. Routines must explicitly call one of the finalization actions.
- Each bot group execution has a unique `orchestration_id` (UUID) to isolate execution contexts.
- Database checkout system prevents duplicate injections across orchestrations.

### Database Checkout System

**Purpose**: Global mutex preventing simultaneous account injection across multiple orchestrations.

**Checkout Columns** (accounts table):
- `checked_out_to_orchestration`: UUID of bot group using this account
- `checked_out_to_instance`: Emulator instance number
- `checked_out_at`: Timestamp of checkout

**Workflow**:
1. `GetNext()` retrieves account from execution-specific pool queue
2. `CheckoutAccount()` atomically checks database:
   - If already checked out to different orchestration → defer & retry next account
   - If available or checked out to same orchestration → proceed
3. Account injected into emulator
4. On completion/failure/return → `ReleaseAccount()` clears orchestration ID
5. On group shutdown → `ReleaseAllAccountsForOrchestration()` cleanup

**Stale Checkout Detection**:
- Checkouts older than 10 minutes are considered stale and can be reclaimed
- Handles ungraceful shutdowns and crashed routines

### Operations

- `GetNext()`: Get next available account from channel, mark as InUse
- `MarkUsed()`: Mark complete/failed with stats, REMOVE from circulation
- `MarkFailed()`: Permanently fail account, REMOVE from circulation (unless retry enabled and under max failures)
- `Return()`: Put back into available channel for retry (only if explicitly called)
- `ListAccounts()`: Get all accounts
- `GetStats()`: Summary statistics (includes `InitialAccountCount` for progress)

### Auto-Refresh

Optional periodic refresh (e.g., every 60 seconds) to reload accounts from sources.

---

## 6. Accounts System

### Account Structure
**Location**: [internal/database/models.go](internal/database/models.go)

- **Device login**: email/password
- **Resources**: shinedust, hourglasses, pack points
- **Statistics**: packs opened, level, wonder picks
- **Timestamps**: created, last used, stamina recovery
- **Metadata**: file path, active status, banned status

### Account Injection
**Location**: [internal/accounts/injector.go](internal/accounts/injector.go)

Process for injecting account data into the game:

1. Force-stop Pokemon TCG app via ADB
2. Push account XML to `/sdcard/deviceAccount.xml`
3. Copy to game data: `/data/data/jp.pokemon.pokemontcgp/shared_prefs/deviceAccount:.xml`
4. Clean up temporary file

### Account Actions
**Location**: [internal/actions/account.go](internal/actions/account.go)

Actions for managing accounts within routines:

- `InjectNextAccount`: Get from pool, inject, track assignment
- `CompleteAccount`: Mark complete with stats (packs opened, cards found, etc.)
- `ReturnAccount`: Put back for retry
- `MarkAccountFailed`: Permanently fail with reason

---

## 7. Routines

**Location**: [internal/actions/routine.go](internal/actions/routine.go)

### Definition

A routine is a YAML file defining a sequence of actions with optional error handlers (sentries).

### Example

```yaml
routine_name: "Pack Opener"
description: "Opens packs and collects cards"
tags: ["farming", "packs"]

config:
  - name: max_packs
    type: int
    default: 10

steps:
  - action: click
    template: open_pack_button
  - action: sleep
    duration: 2000
  - action: repeat
    count: ${max_packs}
    actions:
      - action: ifimagefound
        template: pack_available
        actions:
          - action: click
            template: open

sentries:
  - routine: error_handling/popup_handler
    frequency: 15
    severity: medium
    on_success: resume
```

### Compilation Flow

1. YAML parsed into `Routine` struct
2. Steps unmarshaled via action registry (polymorphic)
3. Each step validates
4. Each step builds into `ActionBuilder`
5. Result: Executable `ActionBuilder` with all steps

### Execution

- `RoutineExecutor` loads sentries
- Registers sentries with global `SentryManager`
- Executes main routine steps sequentially
- Unregisters sentries on completion

---

## 8. Actions System

**Location**: [internal/actions/](internal/actions/)

### Core Pattern

```go
type ActionStep interface {
    Validate(ab *ActionBuilder) error        // Build-time validation
    Build(ab *ActionBuilder) *ActionBuilder  // Compile to Step
}
```

### Example: Click Action

```go
type Click struct {
    X        int    `yaml:"x"`
    Y        int    `yaml:"y"`
    Template string `yaml:"template"`  // Optional
}

func (c *Click) Validate(ab *ActionBuilder) error {
    // Validate template exists at build-time
    return nil
}

func (c *Click) Build(ab *ActionBuilder) *ActionBuilder {
    step := Step{
        name: "Click",
        execute: func(bot BotInterface) error {
            // Resolve coordinates
            x, y := c.X, c.Y
            if c.Template != "" {
                result, _ := bot.CV().FindTemplate(c.Template, nil)
                x, y = result.Location.X, result.Location.Y
            }
            // Execute click via ADB
            return bot.ADB().Tap(x, y)
        },
    }
    ab.steps = append(ab.steps, step)
    return ab
}
```

### 50+ Action Types

#### Navigation
- Click, Swipe, Input, SendKey

#### Vision
- IfImageFound, WhileImageFound, UntilImageFound, WaitForImage

#### Control Flow
- If, While, Until, Repeat, Break

#### Variables
- SetVariable, Increment, Decrement

#### Account Management
- InjectNextAccount, CompleteAccount, ReturnAccount

#### Database Operations
- UpdateAccountField, GetAccountField

#### Sentry Control
- SentryHalt, SentryResume

#### Utility
- Sleep, Comment

### Variable Interpolation

Variables can be substituted at runtime:

```yaml
- action: repeat
  count: ${max_packs}  # Substituted from config
```

---

## 9. Sentry System

**Location**:
- [internal/actions/sentry_manager.go](internal/actions/sentry_manager.go)
- [internal/actions/sentry_engine.go](internal/actions/sentry_engine.go)

### Purpose

Background routines that monitor for errors while the main routine runs.

### Sentry Definition

```yaml
sentries:
  - routine: error_handling/popup_handler
    frequency: 15        # Poll every 15 seconds
    severity: low
    on_success: resume   # If handler succeeds, resume main
    on_failure: stop     # If handler fails, stop bot
```

### Reference Counting Lifecycle

```
Routine A starts → Register sentry X (refcount=1, engine starts)
Routine B starts → Register sentry X (refcount=2, engine continues)
Routine B ends   → Unregister sentry X (refcount=1, still running)
Routine A ends   → Unregister sentry X (refcount=0, engine stops)
```

### Key Features

- **Deduplication**: Same sentry not run multiple times
- **Reference Counting**: Cleanup when last user unregisters
- **Frequency Optimization**: Uses fastest frequency if multiple requesters
- **Routine Control**: Can pause/stop main routine via `RoutineController`

### SentryEngine

Each sentry runs in a separate goroutine:
- Polls at configured frequency
- Executes sentry routine
- Takes configured action if condition matches
- Stops when reference count reaches zero

---

## 10. Execution Flow

### End-to-End Workflow

```
1. App Startup
   ├─ Load Settings.ini
   ├─ Create TemplateRegistry (load all YAML templates)
   ├─ Create RoutineRegistry (load all YAML routines)
   ├─ Create PoolManager (load all pool definitions)
   └─ Create Orchestrator (pass shared registries)

2. User Launches Bot Group
   ├─ Orchestrator validates routine exists
   ├─ Allocate emulator instances
   └─ For each bot (staggered launch):
      ├─ Create Bot instance
      ├─ Bot.Initialize() (ADB, CV, inject registries)
      └─ Launch bot.ExecuteRoutine() in goroutine

3. Bot Executes Routine
   ├─ Get routine from RoutineRegistry
   ├─ Initialize config variables
   ├─ Create RoutineExecutor with sentries
   ├─ SentryManager.Register(sentries)
   │  └─ SentryEngine starts in background
   │     └─ Each sentry polls on its frequency
   ├─ ActionBuilder.Execute(bot)
   │  └─ For each step in sequence:
   │     ├─ Execute step function
   │     ├─ Handle errors via recovery
   │     ├─ Check pause/stop from sentries
   │     └─ Apply step timeout if configured
   ├─ SentryManager.Unregister(sentries)
   │  └─ SentryEngine stops (if refcount=0)
   └─ Account marked complete/failed

4. Routine Completion
   ├─ All steps executed
   ├─ Account status finalized:
   │  ├─ CompleteAccount: Marks complete/failed, REMOVES from pool
   │  ├─ ReturnAccount: Returns to pool for reuse
   │  └─ MarkAccountFailed: Permanently fails account
   ├─ Bot status updated
   └─ Orchestrator may restart per policy
```

---

## Design Patterns

The architecture employs several well-known design patterns:

1. **Builder Pattern**: ActionBuilder separates validation from execution
2. **Registry Pattern**: Centralized lookup (Templates, Routines, Actions)
3. **Polymorphic Unmarshaling**: Use reflection + registry for YAML deserialization
4. **Dependency Injection**: BotInterface breaks circular dependencies
5. **Reference Counting**: Sentry lifecycle management
6. **Factory Pattern**: Orchestrator creates bot instances
7. **Observer Pattern**: Error monitoring via sentries

---

## Architecture Strengths

- **Composable**: Actions combine for complex behaviors
- **Testable**: Clear interfaces, dependency injection
- **Scalable**: Multiple bots can run in parallel
- **Flexible**: YAML-based routines (no recompilation needed)
- **Robust**: Sentries, error monitoring, health checks
- **Maintainable**: Clear separation of concerns

---

## Key Files Reference

| Component | File | Purpose |
|-----------|------|---------|
| CV Service | [internal/cv/service.go](internal/cv/service.go) | Screenshot + template matching |
| Template Registry | [pkg/templates/registry.go](pkg/templates/registry.go) | Template YAML loading |
| Routine Registry | [internal/actions/routine_registry.go](internal/actions/routine_registry.go) | Routine discovery & loading |
| Action Registry | [internal/actions/registry.go](internal/actions/registry.go) | Action type mapping |
| ActionBuilder | [internal/actions/builder.go](internal/actions/builder.go) | Action compilation |
| Routine YAML | [internal/actions/routine.go](internal/actions/routine.go) | YAML parsing + polymorphism |
| Unified Pool | [internal/accountpool/unified_pool.go](internal/accountpool/unified_pool.go) | Account pool implementation |
| Orchestrator | [internal/bot/orchestrator.go](internal/bot/orchestrator.go) | Multi-bot coordination |
| Sentry Manager | [internal/actions/sentry_manager.go](internal/actions/sentry_manager.go) | Lifecycle management |
| Sentry Engine | [internal/actions/sentry_engine.go](internal/actions/sentry_engine.go) | Background execution |
| Account Actions | [internal/actions/account.go](internal/actions/account.go) | Pool interaction actions |
| Injector | [internal/accounts/injector.go](internal/accounts/injector.go) | ADB-based XML injection |

---

## Action Reference

This section provides comprehensive documentation for all available actions in the system.

### Navigation Actions

#### Click
Perform a tap at specific coordinates.

```yaml
- action: click
  x: 500
  y: 300
```

**Parameters:**
- `x` (int, required): X coordinate
- `y` (int, required): Y coordinate

#### Swipe
Perform a swipe gesture between two points.

```yaml
- action: swipe
  x1: 100
  y1: 500
  x2: 400
  y2: 500
  duration: 300
```

**Parameters:**
- `x1`, `y1` (int, required): Starting coordinates
- `x2`, `y2` (int, required): Ending coordinates
- `duration` (int, optional): Swipe duration in milliseconds (default: 300)

#### Input
Input text into a field.

```yaml
- action: input
  text: "Hello World"
```

**Parameters:**
- `text` (string, required): Text to input

#### SendKey
Send a key press event.

```yaml
- action: send_key
  key: "KEYCODE_BACK"
```

**Parameters:**
- `key` (string, required): Android keycode (e.g., KEYCODE_BACK, KEYCODE_HOME)

---

### Vision Actions

#### FindImage
Find a template on screen and save coordinates.

```yaml
- action: findimage
  template: button_name
  save_x: x_coord
  save_y: y_coord
  threshold: 0.85
```

**Parameters:**
- `template` (string, required): Template name from registry
- `save_x` (string, optional): Variable name to store X coordinate
- `save_y` (string, optional): Variable name to store Y coordinate
- `threshold` (float, optional): Override template's threshold
- `region` (object, optional): Search region `{x1, y1, x2, y2}`

#### ClickIfImageFound
Click template if found (single check).

```yaml
- action: clickifimagefound
  template: button_name
  offset_x: 10
  offset_y: 20
```

**Parameters:**
- `template` (string, required): Template name
- `offset_x`, `offset_y` (int, optional): Click offset from template center
- `threshold` (float, optional): Override threshold
- `region` (object, optional): Search region

#### ClickIfImageNotFound
Click coordinates if template NOT found.

```yaml
- action: clickifimagenotfound
  template: popup
  x: 500
  y: 300
```

**Parameters:**
- `template` (string, required): Template to check for absence
- `x`, `y` (int, required): Coordinates to click if not found
- `threshold` (float, optional): Override threshold

#### WaitForImage
Wait until template appears (with timeout).

```yaml
- action: waitforimage
  template: loading_complete
  timeout: 30000
  check_interval: 500
```

**Parameters:**
- `template` (string, required): Template to wait for
- `timeout` (int, optional): Timeout in milliseconds (default: 30000)
- `check_interval` (int, optional): Check interval in ms (default: 500)
- `threshold` (float, optional): Override threshold

---

### Control Flow Actions

#### If
Execute actions based on structured conditions (supports elseif and else).

```yaml
- action: if
  condition:
    type: variable_greater_than
    variable: level
    value: "10"
  then:
    - action: click
      x: 100
      y: 200
  elseif:
    - condition:
        type: variable_equals
        variable: level
        value: "5"
      then:
        - action: click
          x: 300
          y: 400
  else:
    - action: sleep
      duration: 1000
```

**Parameters:**
- `condition` (object, required): Structured condition (see Boolean Conditions section)
- `then` (list, required): Actions to execute if condition is true
- `elseif` (list, optional): Additional condition branches
- `else` (list, optional): Actions if no conditions match

#### IfImageFound
Execute actions if template is found.

```yaml
- action: ifimagefound
  template: popup_close
  actions:
    - action: click
      template: popup_close
```

**Parameters:**
- `template` (string, required): Template to check
- `actions` (list, required): Actions to execute if found
- `threshold` (float, optional): Override threshold
- `region` (object, optional): Search region

#### IfImageNotFound
Execute actions if template is NOT found.

```yaml
- action: ifimagenotfound
  template: loading_screen
  actions:
    - action: click
      x: 500
      y: 300
```

**Parameters:**
- `template` (string, required): Template to check for absence
- `actions` (list, required): Actions to execute if not found

#### IfAnyImagesFound
Execute actions if ANY of the templates are found.

```yaml
- action: ifanyimagesfound
  templates:
    - popup1
    - popup2
    - popup3
  actions:
    - action: click
      x: 500
      y: 300
```

**Parameters:**
- `templates` (list, required): List of template names
- `actions` (list, required): Actions to execute if any found

#### IfAllImagesFound
Execute actions if ALL templates are found.

```yaml
- action: ifallimagesfound
  templates:
    - icon1
    - icon2
  actions:
    - action: click
      x: 500
      y: 300
```

**Parameters:**
- `templates` (list, required): List of template names
- `actions` (list, required): Actions to execute if all found

#### IfNoImagesFound
Execute actions if NONE of the templates are found.

```yaml
- action: ifnoimagesfound
  templates:
    - error1
    - error2
  actions:
    - action: click
      x: 500
      y: 300
```

**Parameters:**
- `templates` (list, required): List of template names
- `actions` (list, required): Actions to execute if none found

---

### Loop Actions

#### Repeat
Repeat actions a fixed number of times.

```yaml
- action: repeat
  iterations: 5
  actions:
    - action: click
      x: 500
      y: 300
    - action: sleep
      duration: 1000
```

**Parameters:**
- `iterations` (int, required): Number of times to repeat (supports `${variable}`)
- `actions` (list, required): Actions to repeat

#### While
Repeat actions while structured condition is true.

```yaml
- action: while
  condition:
    type: variable_less_than
    variable: counter
    value: "10"
  max_attempts: 100
  actions:
    - action: click
      x: 500
      y: 300
    - action: increment
      variable: counter
```

**Parameters:**
- `condition` (object, required): Structured condition (see Boolean Conditions section)
- `max_attempts` (int, optional): Safety limit (0 = infinite)
- `actions` (list, required): Actions to repeat

#### WhileImageFound
Repeat actions while template is visible.

```yaml
- action: whileimagefound
  template: continue_button
  max_iterations: 50
  actions:
    - action: click
      template: continue_button
    - action: sleep
      duration: 2000
```

**Parameters:**
- `template` (string, required): Template to check
- `max_iterations` (int, optional): Safety limit (default: 1000)
- `actions` (list, required): Actions to repeat
- `threshold` (float, optional): Override threshold

#### WhileAnyImagesFound
Repeat while ANY template is visible.

```yaml
- action: whileanyimagesfound
  templates:
    - loading1
    - loading2
  max_iterations: 100
  actions:
    - action: sleep
      duration: 500
```

**Parameters:**
- `templates` (list, required): List of template names
- `max_iterations` (int, optional): Safety limit
- `actions` (list, required): Actions to repeat

#### Until
Repeat actions until structured condition becomes true.

```yaml
- action: until
  condition:
    type: variable_equals
    variable: success
    value: "true"
  max_attempts: 50
  actions:
    - action: click
      x: 500
      y: 300
```

**Parameters:**
- `condition` (object, required): Structured condition (see Boolean Conditions section)
- `max_attempts` (int, optional): Safety limit (0 = infinite)
- `actions` (list, required): Actions to repeat

#### UntilImageFound
Repeat actions until template appears.

```yaml
- action: untilimagefound
  template: success_screen
  max_iterations: 30
  actions:
    - action: click
      x: 500
      y: 300
    - action: sleep
      duration: 1000
```

**Parameters:**
- `template` (string, required): Template to wait for
- `max_iterations` (int, optional): Safety limit (default: 1000)
- `actions` (list, required): Actions to repeat
- `threshold` (float, optional): Override threshold

#### UntilAnyImagesFound
Repeat until ANY template appears.

```yaml
- action: untilanyimagesfound
  templates:
    - result1
    - result2
  max_iterations: 50
  actions:
    - action: click
      x: 500
      y: 300
    - action: sleep
      duration: 500
```

**Parameters:**
- `templates` (list, required): List of template names
- `max_iterations` (int, optional): Safety limit
- `actions` (list, required): Actions to repeat

#### Break
Exit current loop early.

```yaml
- action: repeat
  iterations: 100
  actions:
    - action: ifimagefound
      template: complete
      actions:
        - action: break
```

**Parameters:** None

---

### Variable Actions

#### SetVariable
Set a variable to a value.

```yaml
- action: setvariable
  variable: counter
  value: "10"
```

**Parameters:**
- `variable` (string, required): Variable name
- `value` (string, required): Value to set (supports interpolation)

#### GetVariable
Get a variable value (primarily for debugging).

```yaml
- action: getvariable
  variable: counter
```

**Parameters:**
- `variable` (string, required): Variable name

#### Increment
Increment a numeric variable.

```yaml
- action: increment
  variable: counter
  amount: 1
```

**Parameters:**
- `variable` (string, required): Variable name
- `amount` (int, optional): Increment amount (default: 1)

#### Decrement
Decrement a numeric variable.

```yaml
- action: decrement
  variable: counter
  amount: 1
```

**Parameters:**
- `variable` (string, required): Variable name
- `amount` (int, optional): Decrement amount (default: 1)

---

### Account Pool Actions

#### InjectNextAccount
Get and inject the next account from the pool.

```yaml
- action: injectnextaccount
  timeout: 30000
  save_result: account_id
  on_no_accounts: stop
```

**Parameters:**
- `timeout` (int, optional): Wait timeout in ms (default: 30000)
- `save_result` (string, optional): Variable to store account ID
- `on_no_accounts` (string, optional): Action if pool empty: "wait", "stop", "continue" (default: "stop")

#### CompleteAccount
Mark current account as completed with stats.

```yaml
- action: completeaccount
  packs_opened: ${packs_count}
  cards_found: ${cards_count}
  shinedust_gained: ${dust_amount}
```

**Parameters:**
- `packs_opened` (int, optional): Number of packs opened
- `cards_found` (int, optional): Number of cards found
- `shinedust_gained` (int, optional): Shinedust gained
- `hourglasses_used` (int, optional): Hourglasses used
- `pack_points_used` (int, optional): Pack points used
- All parameters support variable interpolation

#### ReturnAccount
Return current account to pool without completing.

```yaml
- action: returnaccount
  reason: "timeout"
```

**Parameters:**
- `reason` (string, optional): Reason for return

#### MarkAccountFailed
Mark current account as failed.

```yaml
- action: markaccountfailed
  reason: "banned"
```

**Parameters:**
- `reason` (string, required): Failure reason

---

### Database Actions

#### UpdateAccountField
Update a specific field in the account database.

```yaml
- action: updateaccountfield
  field: level
  value: ${current_level}
```

**Parameters:**
- `field` (string, required): Field name (e.g., "level", "shinedust")
- `value` (string, required): New value (supports interpolation)

#### IncrementAccountField
Increment a numeric field in the account database.

```yaml
- action: incrementaccountfield
  field: packs_opened
  amount: 1
```

**Parameters:**
- `field` (string, required): Field name
- `amount` (int, optional): Increment amount (default: 1)

#### GetAccountField
Get a field value from account database.

```yaml
- action: getaccountfield
  field: level
  save_to: current_level
```

**Parameters:**
- `field` (string, required): Field name
- `save_to` (string, required): Variable to store result

#### UpdateRoutineMetrics
Update metrics for routine execution.

```yaml
- action: updateroutinemetrics
  routine_name: ${routine_name}
  success: true
  duration: ${execution_time}
```

**Parameters:**
- `routine_name` (string, required): Routine identifier
- `success` (bool, required): Whether execution succeeded
- `duration` (int, optional): Execution time in milliseconds

---

### Routine Control Actions

#### RunRoutine
Execute another routine by name.

```yaml
- action: runroutine
  routine: error_handling/popup_closer
```

**Parameters:**
- `routine` (string, required): Routine name/path

---

### Sentry Control Actions

#### SentryHalt
Temporarily halt all active sentries.

```yaml
- action: sentryhalt
```

**Parameters:** None

**Use Case:** During critical operations where sentry interruption would cause issues.

#### SentryResume
Resume all halted sentries.

```yaml
- action: sentryresume
```

**Parameters:** None

---

### Timing Actions

#### Sleep
Wait for a fixed duration.

```yaml
- action: sleep
  duration: 2000
```

**Parameters:**
- `duration` (int, required): Duration in milliseconds (supports `${variable}`)

#### Delay
Alias for sleep.

```yaml
- action: delay
  duration: 1000
```

**Parameters:**
- `duration` (int, required): Duration in milliseconds

---

## Boolean Conditions

The `If`, `While`, and `Until` actions support sophisticated structured conditions (not string expressions).

### Condition Types

#### ImageExists
Check if a template is visible.

```yaml
condition:
  type: image_exists
  template: button_name
  threshold: 0.85
  region:
    x1: 100
    y1: 100
    x2: 500
    y2: 500
```

#### ImageNotExists
Check if a template is NOT visible.

```yaml
condition:
  type: image_not_exists
  template: loading_screen
```

#### VariableEquals
Check if a variable equals a value.

```yaml
condition:
  type: variable_equals
  variable: counter
  value: "10"
```

#### VariableNotEquals
Check if a variable does not equal a value.

```yaml
condition:
  type: variable_not_equals
  variable: status
  value: "complete"
```

#### VariableGreaterThan
Numeric comparison: variable > value.

```yaml
condition:
  type: variable_greater_than
  variable: level
  value: "5"
```

#### VariableLessThan
Numeric comparison: variable < value.

```yaml
condition:
  type: variable_less_than
  variable: attempts
  value: "10"
```

#### VariableGreaterThanOrEqual
Numeric comparison: variable >= value.

```yaml
condition:
  type: variable_greater_than_or_equal
  variable: score
  value: "100"
```

#### VariableLessThanOrEqual
Numeric comparison: variable <= value.

```yaml
condition:
  type: variable_less_than_or_equal
  variable: health
  value: "50"
```

#### VariableContains
Check if variable contains substring.

```yaml
condition:
  type: variable_contains
  variable: message
  substring: "error"
```

#### VariableStartsWith
Check if variable starts with prefix.

```yaml
condition:
  type: variable_starts_with
  variable: filename
  prefix: "output_"
```

#### VariableEndsWith
Check if variable ends with suffix.

```yaml
condition:
  type: variable_ends_with
  variable: filename
  suffix: ".txt"
```

### Logical Operators

#### All (AND)
All conditions must be true.

```yaml
condition:
  type: all
  conditions:
    - type: variable_greater_than
      variable: level
      value: "5"
    - type: image_exists
      template: ready_button
```

#### Any (OR)
At least one condition must be true.

```yaml
condition:
  type: any
  conditions:
    - type: image_exists
      template: error1
    - type: image_exists
      template: error2
```

#### Not
Negate a condition.

```yaml
condition:
  type: not
  condition:
    type: image_exists
    template: loading
```

#### None (NOR)
None of the conditions can be true.

```yaml
condition:
  type: none
  conditions:
    - type: image_exists
      template: error1
    - type: image_exists
      template: error2
```

### Complex Example

```yaml
- action: if
  condition:
    type: all
    conditions:
      - type: variable_greater_than
        variable: level
        value: "10"
      - type: any
        conditions:
          - type: image_exists
            template: bonus_available
          - type: variable_equals
            variable: bonus_mode
            value: "enabled"
  then:
    - action: click
      template: bonus_button
  elseif:
    - condition:
        type: variable_equals
        variable: level
        value: "5"
      then:
        - action: click
          x: 500
          y: 300
  else:
    - action: sleep
      duration: 1000
```

---

## Variable Interpolation

All string and numeric parameters support variable interpolation using `${variable_name}` syntax:

```yaml
- action: repeat
  iterations: ${max_loops}
  actions:
    - action: click
      x: ${button_x}
      y: ${button_y}
    - action: sleep
      duration: ${delay_ms}
```

Variables can come from:
1. **Routine config**: Defined in routine's `config` section
2. **Runtime variables**: Set via `setvariable`, `increment`, `decrement`
3. **Action results**: Saved via `save_result`, `save_to`, `save_x`, `save_y` parameters

---

This is a sophisticated, well-architected system for game automation with enterprise-grade patterns and clear separation of concerns.
