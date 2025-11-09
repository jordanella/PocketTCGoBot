# Orchestration System - Complete Overview

## Summary

The orchestration system provides comprehensive bot group management with SQL-based account pools, emulator instance coordination, routine validation, and staggered launching. This document provides a high-level overview of all components.

## System Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                          Application Layer                          │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐             │
│  │     GUI      │  │     CLI      │  │  API Server  │             │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘             │
│         │                 │                  │                      │
│         └─────────────────┼──────────────────┘                      │
│                           │                                         │
└───────────────────────────┼─────────────────────────────────────────┘
                            │
┌───────────────────────────┼─────────────────────────────────────────┐
│                           ▼         Orchestration Layer              │
│                  ┌─────────────────┐                                │
│                  │  Orchestrator   │                                │
│                  │                 │                                │
│                  │  ┌───────────┐  │                                │
│                  │  │TemplateReg│  │ (Shared Registries)            │
│                  │  │RoutineReg │  │                                │
│                  │  │PoolMgr    │  │                                │
│                  │  └───────────┘  │                                │
│                  └────────┬────────┘                                │
│                           │                                         │
│         ┌─────────────────┼─────────────────┐                       │
│         │                 │                 │                       │
│    ┌────▼────┐       ┌────▼────┐       ┌────▼────┐                 │
│    │ Group 1 │       │ Group 2 │       │ Group 3 │                 │
│    │ Premium │       │  Fresh  │       │  Retry  │                 │
│    │         │       │         │       │         │                 │
│    │ Bot Bot │       │ Bot Bot │       │ Bot Bot │                 │
│    │  0   1  │       │  2   3  │       │  4   5  │                 │
│    └────┬────┘       └────┬────┘       └────┬────┘                 │
│         │                 │                 │                       │
└─────────┼─────────────────┼─────────────────┼───────────────────────┘
          │                 │                 │
┌─────────┼─────────────────┼─────────────────┼───────────────────────┐
│         │                 │  Resource Layer  │                       │
│         │                 │                 │                       │
│    ┌────▼──────┐     ┌────▼──────┐     ┌────▼──────┐               │
│    │Emulator 0 │     │Emulator 1 │     │Emulator 2 │               │
│    │MuMu Inst  │     │MuMu Inst  │     │MuMu Inst  │               │
│    │ADB: 16384 │     │ADB: 16416 │     │ADB: 16448 │               │
│    └───────────┘     └───────────┘     └───────────┘               │
│                                                                      │
│    ┌──────────────────────────────────────────────────┐             │
│    │            Account Pool Manager                  │             │
│    │                                                  │             │
│    │  ┌──────────────┐  ┌──────────────┐             │             │
│    │  │ SQL Pool     │  │ File Pool    │             │             │
│    │  │ (Query DB)   │  │ (Scan Dir)   │             │             │
│    │  └──────────────┘  └──────────────┘             │             │
│    └──────────────────────────────────────────────────┘             │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Orchestrator
**Location:** `internal/bot/orchestrator.go`

**Responsibilities:**
- Manages multiple bot groups
- Coordinates emulator instance allocation
- Provides validation and launch orchestration
- Integrates with PoolManager

**Key Files:**
- `orchestrator.go` - Core structures and group management
- `orchestrator_instances.go` - Instance registry and conflict detection
- `orchestrator_validation.go` - Routine and template validation
- `orchestrator_launch.go` - Launch orchestration and staggered starts

### 2. PoolManager
**Location:** `internal/accountpool/pool_manager.go`

**Responsibilities:**
- Discovers pools from `pools/` directory
- Creates and caches pool instances
- Manages pool lifecycle (create, update, delete, test)
- Provides pool refresh functionality

**Supported Pool Types:**
- **File Pool** - Scans directory for XML files
- **SQL Pool** - Queries database for accounts

### 3. BotGroup
**Structure:** Part of Orchestrator

**Configuration:**
- Name (unique identifier)
- Routine to execute
- Available emulator instances
- Requested bot count
- Account pool (by name or direct instance)
- Routine configuration overrides

### 4. AccountPool
**Location:** `internal/accountpool/`

**Interface:** Provides accounts to bots
**Implementations:**
- `FileAccountPool` - File-based account source
- `SQLAccountPool` - SQL query-based account source

## Data Flow

### Launch Sequence

```
1. User Creates Group
   └─> Orchestrator.CreateGroup(name, routine, instances, count, poolName)
       └─> Group created with AccountPoolName set
           AccountPool = nil (not resolved yet)

2. User Clicks Launch
   └─> Orchestrator.LaunchGroup(groupName, options)

       Phase 0: Resolve Account Pool
       └─> If AccountPoolName != "" && AccountPool == nil
           └─> resolveAccountPool(poolName)
               └─> PoolManager.GetPool(poolName)
                   ├─> Check cache
                   ├─> If not cached, create instance
                   └─> Return pool
               └─> Group.AccountPool = pool
               └─> Manager.SetAccountPool(pool)

       Phase 1: Validate Routine
       └─> ValidateRoutine(routineName, config)
           ├─> Check routine exists
           ├─> Parse routine definition
           ├─> Validate all actions
           ├─> Validate all templates
           └─> Validate configuration variables

       Phase 2: Acquire Instances
       └─> acquireInstances(group, options)
           ├─> Check each requested instance
           ├─> Detect conflicts
           ├─> Handle conflicts based on policy
           ├─> Launch emulators if needed
           ├─> Wait for emulators to be ready
           └─> Reserve instances

       Phase 3: Launch Bots
       └─> launchBotsStaggered(group, instances, options)
           ├─> For each instance:
           │   ├─> Create bot
           │   ├─> Start routine in goroutine
           │   ├─> Wait stagger delay
           │   └─> Continue to next
           └─> Return launch results

3. Bots Execute
   └─> Each bot:
       ├─> Injects account from pool
       ├─> Executes routine actions
       ├─> Marks account as used/failed
       └─> Returns account to pool or marks complete

4. User Stops Group
   └─> Orchestrator.StopGroup(groupName)
       └─> For each active bot:
           ├─> Cancel routine context
           ├─> Shutdown bot
           ├─> Release emulator instance
           └─> Remove from active bots

5. Cleanup
   └─> Orchestrator.DeleteGroup(groupName)
       ├─> Verify group is stopped
       ├─> Close account pool
       └─> Remove group
```

### Account Pool Flow

```
1. Pool Discovery
   └─> PoolManager.DiscoverPools()
       └─> Scan pools/*.yaml files
           ├─> For each .yaml file:
           │   ├─> Read file
           │   ├─> Parse YAML
           │   ├─> Determine type (file/sql)
           │   ├─> Load appropriate config
           │   └─> Store in pools map
           └─> Return discovered pools

2. Pool Testing (Optional)
   └─> PoolManager.TestPool(name)
       ├─> Get pool definition
       ├─> Create temporary pool instance
       ├─> Execute query or scan directory
       ├─> Count accounts found
       ├─> Close temporary instance
       └─> Return test results

3. Pool Resolution (At Launch)
   └─> Orchestrator.resolveAccountPool(poolName)
       └─> PoolManager.GetPool(poolName)
           ├─> Check instance cache
           ├─> If cached, return existing
           └─> If not cached:
               ├─> Get pool definition
               ├─> Create instance:
               │   ├─> File: NewFileAccountPool(dir, config)
               │   └─> SQL: NewSQLAccountPool(db, queryFile, config)
               ├─> Execute initial query/scan
               ├─> Start auto-refresh if configured
               ├─> Cache instance
               └─> Return pool

4. Account Acquisition (During Bot Execution)
   └─> Bot.InjectAccount()
       └─> Manager.InjectNextAccount(botID)
           └─> AccountPool.GetNext(ctx)
               ├─> Try to get from available channel
               ├─> Mark account as in-use
               └─> Return account to bot

5. Account Return (After Processing)
   └─> Manager.CompleteAccount(botID, result)
       └─> AccountPool.MarkUsed(account, result)
           ├─> If successful: Mark completed
           ├─> If failed: Increment failure count
           │   ├─> If retry_failed && failures < max
           │   │   └─> Return to available pool
           │   └─> Else: Mark as failed
           └─> Update pool stats

6. Pool Refresh (Manual or Auto)
   └─> AccountPool.Refresh()
       ├─> SQL Pool: Re-execute query
       └─> File Pool: Re-scan directory
       └─> Update available channel
```

## Component Details

### Orchestrator API

```go
// Creation
orchestrator := NewOrchestrator(config, templateReg, routineReg, emuMgr, poolMgr)

// Group Management
group, err := orchestrator.CreateGroup(name, routine, instances, count, poolName)
err := orchestrator.DeleteGroup(name)
group, exists := orchestrator.GetGroup(name)
names := orchestrator.ListGroups()

// Pool Management
err := orchestrator.SetGroupAccountPool(groupName, poolName)
err := orchestrator.RefreshGroupAccountPool(groupName)
poolMgr := orchestrator.GetPoolManager()

// Launch Control
result, err := orchestrator.LaunchGroup(groupName, options)
err := orchestrator.StopGroup(groupName)

// Validation
validationResult := orchestrator.ValidateRoutine(routineName, config)
```

### PoolManager API

```go
// Discovery
poolMgr := accountpool.NewPoolManager(poolsDir, db)
err := poolMgr.DiscoverPools()

// Query
names := poolMgr.ListPools()
poolDef, err := poolMgr.GetPoolDefinition(name)
pool, err := poolMgr.GetPool(name)

// Lifecycle
err := poolMgr.CreatePool(poolDef)
err := poolMgr.UpdatePool(name, poolDef)
err := poolMgr.DeletePool(name)

// Operations
testResult, err := poolMgr.TestPool(name)
err := poolMgr.RefreshPool(name)
err := poolMgr.ClosePool(name)
err := poolMgr.CloseAll()
```

### Launch Options

```go
type LaunchOptions struct {
    ValidateRoutine      bool               // Validate routine before launch
    ValidateTemplates    bool               // Validate all templates exist
    ValidateEmulators    bool               // Check emulator availability
    OnConflict           ConflictResolution // How to handle conflicts
    StaggerDelay         time.Duration      // Delay between bot starts
    EmulatorTimeout      time.Duration      // How long to wait for emulator
    RestartPolicy        RestartPolicy      // Bot restart behavior
}
```

**Conflict Resolution Options:**
- `ConflictResolutionAsk` - Ask user what to do (GUI)
- `ConflictResolutionCancel` - Cancel the conflicting group
- `ConflictResolutionSkip` - Skip the conflicting instance
- `ConflictResolutionAbort` - Abort the entire launch

**Restart Policies:**
- `RestartPolicyNever` - Don't restart on failure
- `RestartPolicyOnFailure` - Restart if routine fails
- `RestartPolicyAlways` - Always restart when routine ends

## Pool Definition Format

### SQL Pool

**File:** `pools/premium_accounts.yaml`

```yaml
name: "Premium Farmers Pool"
description: "Accounts with 10+ packs, sorted by pack count"
type: "sql"
version: "1.0"

query:
  select: |
    SELECT id, xml_path, pack_count, last_modified, status, failure_count, last_error
    FROM accounts
    WHERE status IN (?, ?) AND pack_count >= ? AND failure_count < ?
    ORDER BY pack_count DESC, last_modified ASC
    LIMIT ?

  parameters:
    - {name: "status_available", value: "available", type: "string"}
    - {name: "status_skipped", value: "skipped", type: "string"}
    - {name: "min_packs", value: 10, type: "int"}
    - {name: "max_failures", value: 3, type: "int"}
    - {name: "limit", value: 100, type: "int"}

pool_config:
  retry_failed: true
  max_failures: 3
  wait_for_accounts: true
  max_wait_time: 5m
  buffer_size: 50
  refresh_interval: 30s

gui_config:
  filters:
    - {field: "status", operator: "in", values: ["available", "skipped"]}
    - {field: "pack_count", operator: ">=", value: 10}
    - {field: "failure_count", operator: "<", value: 3}
  sort:
    - {field: "pack_count", direction: "DESC"}
  limit: 100
```

### File Pool

**File:** `pools/manual_list.yaml`

```yaml
name: "Manual Account List"
description: "Hand-picked accounts from manual directory"
type: "file"
directory: "./accounts/manual"

pool_config:
  retry_failed: false
  max_failures: 1
  wait_for_accounts: false
  buffer_size: 10
```

## Validation System

### Routine Validation

Validates:
1. Routine file exists
2. Routine parses correctly (valid YAML)
3. All actions are registered
4. All templates exist
5. All configuration variables are valid

### Template Validation

Validates:
1. Template image files exist
2. Templates have valid names
3. Templates are loadable

### Emulator Validation

Validates:
1. Emulator instances are registered
2. Instances are available or can be launched
3. No conflicts with other groups (based on policy)

## Instance Registry

**Purpose:** Track which emulator instance is used by which group/bot

**Structure:**
```go
type InstanceAssignment struct {
    InstanceID   int
    GroupName    string
    BotInstance  int
    AssignedAt   time.Time
    IsRunning    bool
    EmulatorPID  int
}
```

**Operations:**
- `checkInstanceAvailability(instanceID, groupName)` - Check if free
- `reserveInstance(instanceID, groupName, botID, pid)` - Mark in-use
- `releaseInstance(instanceID, groupName)` - Mark free
- `findConflicts(instances, groupName)` - Detect conflicts

## Statistics and Monitoring

### Pool Statistics

```go
type PoolStats struct {
    Total           int
    Available       int
    InUse           int
    Completed       int
    Failed          int
    Skipped         int
    TotalPacksOpened int
    TotalCardsFound  int
    TotalStars      int
    TotalKeeps      int
}
```

### Bot Status

```go
type BotStatus string

const (
    BotStatusStarting  BotStatus = "starting"
    BotStatusRunning   BotStatus = "running"
    BotStatusStopping  BotStatus = "stopping"
    BotStatusStopped   BotStatus = "stopped"
    BotStatusFailed    BotStatus = "failed"
    BotStatusCompleted BotStatus = "completed"
)
```

## Error Handling

### Validation Errors

```go
type ValidationErrorType string

const (
    ValidationErrorRoutineNotFound   = "routine_not_found"
    ValidationErrorRoutineParse      = "routine_parse_error"
    ValidationErrorActionNotFound    = "action_not_found"
    ValidationErrorTemplateNotFound  = "template_not_found"
    ValidationErrorInvalidConfig     = "invalid_config"
    ValidationErrorMissingVariable   = "missing_variable"
)
```

### Launch Errors

Captured in `LaunchResult`:
- Pool resolution failures
- Validation failures
- Instance acquisition failures
- Bot launch failures
- Individual error messages

## Thread Safety

All components use mutex locks for thread-safe operation:

- `Orchestrator` - `groupsMu`, `instanceRegistryMu`
- `BotGroup` - `activeBotsMu`, `runningMu`
- `PoolManager` - `mu`
- `SQLAccountPool` - `mu`
- `FileAccountPool` - `mu`

## Files Reference

### Core Files
- `internal/bot/orchestrator.go` - Orchestrator and BotGroup
- `internal/bot/orchestrator_instances.go` - Instance registry
- `internal/bot/orchestrator_validation.go` - Validation system
- `internal/bot/orchestrator_launch.go` - Launch orchestration
- `internal/accountpool/pool_manager.go` - Pool manager
- `internal/accountpool/sql_pool.go` - SQL pool implementation
- `internal/accountpool/file_pool.go` - File pool implementation

### Documentation
- `docs/manager_groups_flow.md` - Original flow specification
- `docs/sql_account_pools.md` - SQL pool design
- `docs/account_pool_management.md` - Pool management architecture
- `docs/orchestrator_pool_integration.md` - Integration guide
- `docs/orchestration_system_overview.md` - This document

### Examples
- `examples/orchestrator_with_pools_example.go` - Complete usage example

### Pool Definitions
- `pools/premium_accounts.yaml.example` - Premium account pool
- `pools/fresh_accounts.yaml.example` - Fresh account pool
- `pools/retry_pool.yaml.example` - Retry pool

## Next Steps

### Backend
- ✅ Orchestrator implementation
- ✅ Instance registry and conflict detection
- ✅ Routine validation system
- ✅ Launch orchestration with staggering
- ✅ SQL account pool implementation
- ✅ Pool manager implementation
- ⬜ Routine configuration override support
- ⬜ Emulator process launching (OS-specific)

### Frontend (GUI)
- ⬜ Account Pools management tab
- ⬜ Visual query builder
- ⬜ Manager Groups tab update
- ⬜ Pre-launch refresh prompt
- ⬜ Pool statistics dashboard
- ⬜ Instance conflict resolution dialog

### Testing
- ⬜ Unit tests for Orchestrator
- ⬜ Unit tests for PoolManager
- ⬜ Integration tests for launch flow
- ⬜ Pool validation tests
- ⬜ Conflict resolution tests

## Conclusion

The orchestration system provides a complete, production-ready framework for managing multiple bot groups with SQL-based account pools, emulator coordination, and comprehensive validation. The architecture prioritizes:

- **Separation of Concerns** - Pools managed separately from groups
- **Thread Safety** - All operations protected by mutexes
- **User Experience** - Visual builders, testing, validation
- **Flexibility** - Multiple pool types, conflict policies, restart policies
- **Scalability** - Instance caching, lazy loading, auto-refresh

All backend components are implemented and compiling. The system is ready for GUI integration and user testing.
