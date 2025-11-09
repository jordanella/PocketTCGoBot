# Orchestrator and PoolManager Integration

This document explains how the Orchestrator integrates with the PoolManager to provide seamless account pool management for bot groups.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                        Orchestrator                          │
│                                                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │  Bot Group   │  │  Bot Group   │  │  Bot Group   │      │
│  │  "Premium"   │  │  "Fresh"     │  │  "Retry"     │      │
│  │              │  │              │  │              │      │
│  │  Pool: ──────┼──┼─> Pool: ─────┼──┼─> Pool: ─────┼──┐   │
│  │  "Premium    │  │  "Fresh      │  │  "Retry      │  │   │
│  │   Farmers"   │  │   Accounts"  │  │   Pool"      │  │   │
│  └──────────────┘  └──────────────┘  └──────────────┘  │   │
│                                                          │   │
└──────────────────────────────────────────────────────────┼──┘
                                                           │
                                                           ▼
                                           ┌───────────────────────┐
                                           │    PoolManager        │
                                           │                       │
                                           │  ┌─────────────────┐  │
                                           │  │ Premium Farmers │  │
                                           │  │ Pool (SQL)      │  │
                                           │  └─────────────────┘  │
                                           │                       │
                                           │  ┌─────────────────┐  │
                                           │  │ Fresh Accounts  │  │
                                           │  │ Pool (SQL)      │  │
                                           │  └─────────────────┘  │
                                           │                       │
                                           │  ┌─────────────────┐  │
                                           │  │ Retry Pool      │  │
                                           │  │ (SQL)           │  │
                                           │  └─────────────────┘  │
                                           │                       │
                                           │  ┌─────────────────┐  │
                                           │  │ Manual List     │  │
                                           │  │ (File)          │  │
                                           │  └─────────────────┘  │
                                           └───────────────────────┘
                                                           │
                                                           ▼
                                           ┌───────────────────────┐
                                           │  pools/ directory     │
                                           │                       │
                                           │  premium_accounts.yaml│
                                           │  fresh_accounts.yaml  │
                                           │  retry_pool.yaml      │
                                           │  manual_list.yaml     │
                                           └───────────────────────┘
```

## Key Components

### 1. Orchestrator
- Manages multiple bot groups
- Contains reference to PoolManager
- Resolves pool names to pool instances
- Provides pool refresh functionality

### 2. PoolManager
- Discovers pools from `pools/` directory
- Creates and caches pool instances (File or SQL)
- Manages pool lifecycle
- Provides pool testing functionality

### 3. BotGroup
- References pool by name: `AccountPoolName`
- Holds resolved pool instance: `AccountPool`
- Pool is resolved at launch time or on demand

## Usage Flow

### 1. Initialization

```go
// Create PoolManager
poolManager := accountpool.NewPoolManager("./pools", db)

// Discover pools from YAML files
poolManager.DiscoverPools()

// Create Orchestrator with PoolManager
orchestrator := bot.NewOrchestrator(
    config,
    templateRegistry,
    routineRegistry,
    emulatorManager,
    poolManager,  // Pass PoolManager
)
```

### 2. Creating a Group with Pool

```go
// Create group with pool reference (by name)
group, err := orchestrator.CreateGroup(
    "Premium Farmers",           // Group name
    "OpenPacks",                  // Routine
    []int{0, 1, 2},               // Instances
    2,                            // Bot count
    map[string]string{},          // Config
    "Premium Farmers Pool",       // Pool name
)
```

The pool name is stored but **not resolved** until launch.

### 3. Testing Pool Before Launch (Optional)

```go
// Test pool to preview results
testResult, err := poolManager.TestPool("Premium Farmers Pool")
if testResult.Success {
    fmt.Printf("Found %d accounts\n", testResult.AccountsFound)
}
```

### 4. Refreshing Pool Before Launch

This is where the **user prompt** happens in GUI:

```go
// Prompt user: "Pool 'Premium Farmers Pool' was last refreshed 5m ago. Refresh now?"
// If user says yes:
err := orchestrator.RefreshGroupAccountPool("Premium Farmers")
```

### 5. Launching Group

```go
// Launch with options
launchOptions := bot.LaunchOptions{
    ValidateRoutine:   true,
    ValidateTemplates: true,
    ValidateEmulators: true,
    OnConflict:        bot.ConflictResolutionSkip,
    StaggerDelay:      3 * time.Second,
    EmulatorTimeout:   60 * time.Second,
    RestartPolicy:     bot.RestartPolicyNever,
}

result, err := orchestrator.LaunchGroup("Premium Farmers", launchOptions)
```

**Launch sequence:**
1. **Phase 0: Resolve Pool** - If `AccountPoolName` is set but `AccountPool` is nil, resolve via PoolManager
2. Phase 1: Validate Routine
3. Phase 2: Acquire Instances
4. Phase 3: Launch Bots

### 6. Monitoring Pool Usage

```go
// Get pool stats
stats := group.AccountPool.GetStats()
fmt.Printf("Available: %d, In Use: %d, Completed: %d\n",
    stats.Available, stats.InUse, stats.Completed)
```

## Pool Resolution Flow

```
User Creates Group
       │
       ▼
Group.AccountPoolName = "Premium Farmers Pool"
Group.AccountPool = nil  (not resolved yet)
       │
       ▼
User Launches Group
       │
       ▼
LaunchGroup() checks:
  - If AccountPoolName != "" && AccountPool == nil
       │
       ▼
  resolveAccountPool(poolName)
       │
       ▼
  PoolManager.GetPool(poolName)
       │
       ├─> Check cache
       │
       ├─> If not cached, create instance:
       │   - Load pool definition from YAML
       │   - Create FileAccountPool or SQLAccountPool
       │   - Cache instance
       │
       ▼
  Return pool instance
       │
       ▼
Group.AccountPool = pool
Manager.SetAccountPool(pool)
       │
       ▼
Continue with launch...
```

## API Reference

### Orchestrator Methods

#### `GetPoolManager() *accountpool.PoolManager`
Returns the pool manager for direct access to pool operations.

#### `SetGroupAccountPool(groupName, poolName string) error`
Sets a group's account pool by name. Resolves the pool via PoolManager.

**Example:**
```go
err := orchestrator.SetGroupAccountPool("Premium Farmers", "Fresh Accounts Pool")
```

#### `RefreshGroupAccountPool(groupName string) error`
Manually refreshes a group's account pool (re-executes query for SQL pools, re-scans directory for file pools).

**Example:**
```go
err := orchestrator.RefreshGroupAccountPool("Premium Farmers")
```

### PoolManager Methods (via `orchestrator.GetPoolManager()`)

#### `DiscoverPools() error`
Scans the `pools/` directory and loads all `.yaml` pool definitions.

#### `ListPools() []string`
Returns names of all discovered pools.

#### `GetPoolDefinition(name string) (*PoolDefinition, error)`
Retrieves pool definition (metadata) without creating instance.

#### `GetPool(name string) (AccountPool, error)`
Returns a pool instance (creates and caches if needed).

#### `TestPool(name string) (*TestResult, error)`
Tests a pool by executing query/scan without creating persistent instance.

#### `CreatePool(poolDef *PoolDefinition) error`
Saves a new pool definition to disk.

#### `UpdatePool(name string, poolDef *PoolDefinition) error`
Modifies an existing pool definition.

#### `DeletePool(name string) error`
Removes a pool definition from disk.

#### `RefreshPool(name string) error`
Manually refreshes a pool instance.

#### `ClosePool(name string) error`
Closes a pool instance (removes from cache).

#### `CloseAll() error`
Closes all active pool instances.

## Pool Types

### SQL Pool
Queries database for accounts matching criteria.

**Definition:** `pools/premium_accounts.yaml`
```yaml
name: "Premium Farmers Pool"
type: "sql"
query:
  select: |
    SELECT id, xml_path, pack_count, last_modified, status, failure_count, last_error
    FROM accounts
    WHERE status IN (?, ?) AND pack_count >= ? AND failure_count < ?
    ORDER BY pack_count DESC
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
  refresh_interval: 30s
```

### File Pool
Reads accounts from XML files in a directory.

**Definition:** `pools/manual_list.yaml`
```yaml
name: "Manual Account List"
type: "file"
directory: "./accounts/manual"
pool_config:
  retry_failed: false
  max_failures: 1
```

## GUI Integration Points

### Account Pools Tab

**List View:**
```
┌─────────────────────────────────────────────────────────┐
│ Account Pools                              [+ New Pool] │
├─────────────────────────────────────────────────────────┤
│ Name                     Type   Accounts   Last Refresh │
│ Premium Farmers Pool     SQL    247        2m ago       │
│ Fresh Accounts Pool      SQL    1,043      5m ago       │
│ Retry Pool               SQL    18         1m ago       │
│ Manual List              File   5          N/A          │
└─────────────────────────────────────────────────────────┘
```

**Operations:**
- **New Pool** - Opens pool creation wizard
- **Edit** - Opens visual query builder (SQL) or directory selector (File)
- **Test** - Executes pool query and shows preview
- **Refresh** - Manually refreshes pool
- **Delete** - Removes pool definition

### Manager Groups Tab

**Group Configuration:**
```
┌─────────────────────────────────────────┐
│ Group: Premium Farmers                  │
├─────────────────────────────────────────┤
│ Routine: [OpenPacks           ▼]        │
│ Bot Count: [2]                          │
│ Instances: [0, 1, 2]                    │
│ Account Pool: [Premium Farmers Pool ▼]  │  ← Dropdown from discovered pools
│                                         │
│ [Launch] [Stop] [Delete]                │
└─────────────────────────────────────────┘
```

### Pre-Launch Pool Refresh Prompt

When user clicks **Launch** on a group with a pool:

```
┌─────────────────────────────────────────────────────────┐
│                    Refresh Account Pool?                 │
├─────────────────────────────────────────────────────────┤
│ Pool "Premium Farmers Pool" was last refreshed 5m ago.  │
│                                                          │
│ Would you like to refresh it now to get the latest      │
│ account data before launching?                           │
│                                                          │
│                  [Skip]  [Refresh & Launch]              │
└─────────────────────────────────────────────────────────┘
```

## Benefits of This Architecture

### 1. Separation of Concerns
- Pools are managed independently of groups
- Groups reference pools by name (loose coupling)
- Pool definitions stored in version-controllable YAML

### 2. Flexibility
- Multiple groups can use the same pool
- Pools can be created/modified without affecting running groups
- SQL pools auto-refresh based on configuration

### 3. Safety
- Pools resolved at launch time (latest version)
- User prompted to refresh before launch
- Query validation prevents SQL injection

### 4. Scalability
- Pool instances cached for performance
- Lazy loading (only create when needed)
- Background auto-refresh for SQL pools

### 5. User-Friendly
- Visual query builder (no SQL exposure)
- Pool testing before use
- Clear pool status in GUI

## Example Scenarios

### Scenario 1: Premium Account Rotation
```go
// Week 1: Use premium pool (10+ packs)
orchestrator.CreateGroup("Farmers", "OpenPacks", instances, 5, config, "Premium Farmers Pool")
orchestrator.LaunchGroup("Farmers", options)

// Week 2: Switch to fresh accounts
orchestrator.SetGroupAccountPool("Farmers", "Fresh Accounts Pool")
orchestrator.LaunchGroup("Farmers", options)
```

### Scenario 2: Retry Failed Accounts
```go
// Create retry pool for accounts that failed once
// pools/retry_pool.yaml filters for failure_count BETWEEN 1 AND 2

// Create group using retry pool
orchestrator.CreateGroup("Retry Batch", "OpenPacks", instances, 3, config, "Retry Pool")

// Refresh to get latest failed accounts
orchestrator.RefreshGroupAccountPool("Retry Batch")

// Launch
orchestrator.LaunchGroup("Retry Batch", options)
```

### Scenario 3: Multiple Groups, Same Pool
```go
// Group 1: Premium farmers on instances 0-2
orchestrator.CreateGroup("Group A", "OpenPacks", []int{0,1,2}, 3, config, "Premium Farmers Pool")

// Group 2: Also uses premium pool on instances 3-5
orchestrator.CreateGroup("Group B", "OpenPacks", []int{3,4,5}, 3, config, "Premium Farmers Pool")

// Both groups share the same pool instance (cached by PoolManager)
// Each bot gets a different account from the pool
```

## Best Practices

1. **Always discover pools at startup:**
   ```go
   poolManager.DiscoverPools()
   ```

2. **Test pools before heavy use:**
   ```go
   testResult, _ := poolManager.TestPool(poolName)
   if !testResult.Success {
       log.Printf("Pool test failed: %s", testResult.Error)
   }
   ```

3. **Prompt users to refresh before launch:**
   ```go
   // In GUI: Show last refresh time
   // Offer "Refresh Now" button
   if userClickedRefresh {
       orchestrator.RefreshGroupAccountPool(groupName)
   }
   ```

4. **Close pools when done:**
   ```go
   defer poolManager.CloseAll()
   ```

5. **Use descriptive pool names:**
   - ✓ "Premium Farmers Pool (10+ packs)"
   - ✗ "Pool1"

6. **Set appropriate refresh intervals for SQL pools:**
   - Fast-changing queries: 30s - 1m
   - Stable queries: 5m - 10m
   - Manual-only: 0 (disable auto-refresh)

## Next Steps

- Implement GUI Account Pools tab
- Add visual query builder
- Add pre-launch refresh prompt
- Add pool status monitoring
- Add pool import/export functionality
