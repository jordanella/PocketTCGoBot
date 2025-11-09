# Account Pool Management - Revised Architecture

## Overview
Account pools are standalone resources that are created, configured, and managed independently. Bot groups simply select which pool to use from a dropdown.

---

## Account Pool as a Resource

### Pool Types

1. **File-Based Pool**
   - Scans a directory for XML files
   - Filters by pack count, modification date
   - Simple and fast

2. **SQL-Based Pool**
   - Queries database for accounts
   - Complex filtering via visual query builder
   - Dynamic - reflects latest database state

---

## GUI Structure

```
Main GUI Tabs:
├── Dashboard
├── Bot Launcher
├── Manager Groups          ← Groups reference pools
├── Configuration
├── Account Pools          ← NEW: Manage pools here
├── Database
└── ...
```

### Account Pools Tab

```
┌─────────────────────────────────────────────────────┐
│ Account Pools                                       │
├─────────────────────────────────────────────────────┤
│                                                     │
│  [+ Create Pool]  [Refresh All]                    │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │ Premium Farmers Pool                  [SQL] │   │
│  │ ─────────────────────────────────────────── │   │
│  │ Pack Count >= 10                            │   │
│  │ Failures < 3                                │   │
│  │ Status: Available, Skipped                  │   │
│  │                                              │   │
│  │ Accounts: 47 available                      │   │
│  │ Last Refresh: 2 minutes ago                 │   │
│  │                                              │   │
│  │ [Edit] [Test] [Refresh] [Delete]            │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
│  ┌─────────────────────────────────────────────┐   │
│  │ Fresh Starters Pool              [File Dir] │   │
│  │ ─────────────────────────────────────────── │   │
│  │ Directory: ./accounts/fresh                 │   │
│  │ Min Packs: 5                                │   │
│  │ Sort: Pack Count Descending                 │   │
│  │                                              │   │
│  │ Accounts: 23 available                      │   │
│  │ Last Scan: 5 minutes ago                    │   │
│  │                                              │   │
│  │ [Edit] [Refresh] [Delete]                   │   │
│  └─────────────────────────────────────────────┘   │
│                                                     │
└─────────────────────────────────────────────────────┘
```

---

## Pool Creation Flow

### Step 1: Choose Pool Type

```
┌─────────────────────────────────────────┐
│ Create Account Pool                     │
├─────────────────────────────────────────┤
│ Choose pool type:                       │
│                                          │
│  ○ File Directory                       │
│     Scan a folder for account XML files │
│                                          │
│  ● SQL Query                            │
│     Build a query to select accounts    │
│                                          │
│            [Next]  [Cancel]             │
└─────────────────────────────────────────┘
```

### Step 2A: File Directory Configuration

```
┌─────────────────────────────────────────┐
│ File Directory Pool                     │
├─────────────────────────────────────────┤
│ Pool Name:                              │
│ [Fresh Accounts                       ] │
│                                          │
│ Directory:                              │
│ [./accounts/fresh           ] [Browse] │
│                                          │
│ ─── Filters ───                         │
│ Min Packs:     [5        ]              │
│ Max Packs:     [0 (none) ]              │
│                                          │
│ ─── Sort ───                            │
│ [Pack Count Descending ▼]               │
│                                          │
│ ─── Retry ───                           │
│ ☑ Retry failed accounts                 │
│ Max Failures:  [3        ]              │
│                                          │
│     [Create]  [Back]  [Cancel]          │
└─────────────────────────────────────────┘
```

### Step 2B: SQL Query Builder

```
┌───────────────────────────────────────────────────┐
│ SQL Query Pool                                    │
├───────────────────────────────────────────────────┤
│ Pool Name:                                        │
│ [Premium Farmers Pool                          ] │
│                                                   │
│ Description: (optional)                           │
│ [High pack count accounts for premium farming  ] │
│                                                   │
│ ─── Filters ───────────────────────────────────  │
│                                                   │
│  Status                                           │
│  [In ▼] [Available, Skipped                   ▼] │
│  [Remove]                                         │
│                                                   │
│  Pack Count                                       │
│  [>= ▼] [10                                    ] │
│  [Remove]                                         │
│                                                   │
│  Failure Count                                    │
│  [< ▼]  [3                                     ] │
│  [Remove]                                         │
│                                                   │
│  [+ Add Filter]                                   │
│                                                   │
│ ─── Sort ──────────────────────────────────────  │
│                                                   │
│  1. [Pack Count ▼] [Descending ▼] [Remove]       │
│  2. [Last Modified ▼] [Ascending ▼] [Remove]     │
│  [+ Add Sort]                                     │
│                                                   │
│ ─── Limit ─────────────────────────────────────  │
│  Max Results: [100                             ] │
│                                                   │
│ ─── Refresh ───────────────────────────────────  │
│  ☑ Auto-refresh every [30] seconds               │
│                                                   │
│ ─── Preview ───────────────────────────────────  │
│  [Test Query] - Shows: 47 accounts found         │
│                                                   │
│      [Create]  [Back]  [Cancel]                  │
└───────────────────────────────────────────────────┘
```

### Filter Builder Details

When clicking "Add Filter", show available fields:

```
┌─────────────────────────────────┐
│ Add Filter                      │
├─────────────────────────────────┤
│ Field:                          │
│ [Status              ▼]         │
│                                  │
│ Operator:                       │
│ [In                  ▼]         │
│                                  │
│ Values: (multi-select)          │
│ ☑ Available                     │
│ ☑ Skipped                       │
│ ☐ Failed                        │
│ ☐ Completed                     │
│                                  │
│       [Add]  [Cancel]           │
└─────────────────────────────────┘
```

Available filter fields and their operators:

| Field | Type | Operators |
|-------|------|-----------|
| Status | Enum | `=`, `!=`, `In` |
| Pack Count | Integer | `=`, `!=`, `<`, `>`, `<=`, `>=`, `Between` |
| Last Modified | Date | `<`, `>`, `<=`, `>=`, `Between`, `Last N Days` |
| Failure Count | Integer | `=`, `!=`, `<`, `>`, `<=`, `>=` |
| Cards Found | Integer | `=`, `!=`, `<`, `>`, `<=`, `>=` |
| Stars Total | Integer | `=`, `!=`, `<`, `>`, `<=`, `>=` |
| Completed At | Date | `Is Null`, `Is Not Null`, `<`, `>` |
| Last Error | Text | `Contains`, `Not Contains`, `Is Null` |

---

## Manager Group Integration

When creating a bot group, simply select an existing pool:

```
┌─────────────────────────────────────────┐
│ Create Manager Group                    │
├─────────────────────────────────────────┤
│ Group Name:                             │
│ [Premium Team                         ] │
│                                          │
│ Routine:                                │
│ [farm_premium_packs.yaml              ] │
│                                          │
│ Bot Instances:                          │
│ [1-4                                  ] │
│                                          │
│ ─── Account Pool (Optional) ───         │
│ Pool:                                   │
│ [Premium Farmers Pool             ▼]   │
│   ├─ Premium Farmers Pool (SQL)        │
│   ├─ Fresh Starters Pool (File)        │
│   ├─ High Value Retry Pool (SQL)       │
│   └─ (No Pool)                          │
│                                          │
│ ☑ Refresh pool before launching         │
│                                          │
│       [Create]  [Cancel]                │
└─────────────────────────────────────────┘
```

---

## SQL Query Generation

The visual query builder generates SQL behind the scenes. Users never see it.

### Example: Visual Builder Input

```
Filters:
- Status IN (Available, Skipped)
- Pack Count >= 10
- Failure Count < 3

Sort:
- Pack Count DESC
- Last Modified ASC

Limit: 100
```

### Generated SQL (Behind the Scenes)

```sql
SELECT
  id, xml_path, pack_count, last_modified,
  status, failure_count, last_error
FROM accounts
WHERE status IN (?, ?)
  AND pack_count >= ?
  AND failure_count < ?
ORDER BY pack_count DESC, last_modified ASC
LIMIT ?
```

### Generated YAML (Saved to Disk)

```yaml
# pools/premium_farmers_pool.yaml
name: "Premium Farmers Pool"
description: "High pack count accounts for premium farming"
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
    - {name: "status_1", value: "available", type: "string"}
    - {name: "status_2", value: "skipped", type: "string"}
    - {name: "min_packs", value: 10, type: "int"}
    - {name: "max_failures", value: 3, type: "int"}
    - {name: "limit", value: 100, type: "int"}

pool_config:
  retry_failed: true
  max_failures: 3
  buffer_size: 50
  refresh_interval: 30s

# Visual builder config (for editing later)
gui_config:
  filters:
    - {field: "status", operator: "in", values: ["available", "skipped"]}
    - {field: "pack_count", operator: ">=", value: 10}
    - {field: "failure_count", operator: "<", value: 3}

  sort:
    - {field: "pack_count", direction: "DESC"}
    - {field: "last_modified", direction: "ASC"}

  limit: 100
```

---

## Pool Operations

### Refresh Pool
Manually re-execute the query or re-scan the directory.

```
User clicks [Refresh] on a pool
    ↓
If SQL Pool:
  - Re-execute query
  - Update account list
  - Show: "Refreshed - 47 accounts found"

If File Pool:
  - Re-scan directory
  - Update account list
  - Show: "Scanned - 23 accounts found"
```

### Test Pool
Preview what accounts would be selected.

```
User clicks [Test] on a pool
    ↓
┌─────────────────────────────────────────┐
│ Pool Test Results                       │
├─────────────────────────────────────────┤
│ Premium Farmers Pool                    │
│                                          │
│ Query executed successfully!            │
│                                          │
│ Found: 47 accounts                      │
│                                          │
│ Sample accounts:                        │
│ ┌────────────────────────────────────┐  │
│ │ ID: acc_1234  Packs: 15  Status: A│  │
│ │ ID: acc_5678  Packs: 14  Status: A│  │
│ │ ID: acc_9012  Packs: 13  Status: S│  │
│ │ ...                                │  │
│ └────────────────────────────────────┘  │
│                                          │
│            [Close]                      │
└─────────────────────────────────────────┘
```

### Edit Pool
Modify the pool configuration.

```
User clicks [Edit] on a pool
    ↓
Opens the same visual builder used to create it,
pre-filled with current settings.

User can modify filters, sorts, limits, etc.
    ↓
Saves back to the same YAML file
```

---

## Pool Storage

Pools are stored as YAML files in a `pools/` directory:

```
pools/
├── premium_farmers_pool.yaml      (SQL)
├── fresh_starters_pool.yaml       (File)
├── high_value_retry_pool.yaml     (SQL)
└── mission_runners_pool.yaml      (File)
```

Each pool file is standalone and can be:
- Backed up
- Shared with other users
- Version controlled in Git
- Imported/exported

---

## Pool Discovery

On startup, the GUI scans the `pools/` directory and loads all pool definitions:

```go
// GUI initialization
poolManager := NewAccountPoolManager("./pools", db)
poolManager.DiscoverPools()

// List available pools for dropdown
pools := poolManager.ListPools()
// Returns: ["Premium Farmers Pool", "Fresh Starters Pool", ...]

// Get pool by name
pool, _ := poolManager.GetPool("Premium Farmers Pool")
```

---

## AccountPoolManager Structure

```go
package accountpool

type PoolManager struct {
    poolsDir  string
    db        *sql.DB
    pools     map[string]PoolDefinition  // name -> definition
    instances map[string]AccountPool     // name -> active pool instance
}

type PoolDefinition struct {
    Name        string
    Type        string  // "file" or "sql"
    FilePath    string  // Path to YAML definition
    Config      interface{}  // FilePoolConfig or QueryDefinition
}

// DiscoverPools scans the pools directory
func (pm *PoolManager) DiscoverPools() error

// GetPool retrieves or creates a pool instance
func (pm *PoolManager) GetPool(name string) (AccountPool, error)

// CreatePool saves a new pool definition
func (pm *PoolManager) CreatePool(def PoolDefinition) error

// UpdatePool modifies an existing pool
func (pm *PoolManager) UpdatePool(name string, def PoolDefinition) error

// DeletePool removes a pool
func (pm *PoolManager) DeletePool(name string) error

// TestPool executes the pool query/scan without creating instance
func (pm *PoolManager) TestPool(name string) (*TestResult, error)
```

---

## Benefits of This Approach

1. **Separation of Concerns**
   - Pools are resources, groups consume resources
   - Clean, modular design

2. **Reusability**
   - Multiple groups can use the same pool
   - Don't duplicate pool configuration

3. **Manageability**
   - Centralized pool management
   - Easy to test, edit, delete pools

4. **User-Friendly**
   - Visual query builder - no SQL knowledge needed
   - Dropdown selection - just pick a pool

5. **Flexibility**
   - Mix file and SQL pools as needed
   - Easy to switch a group's pool

6. **Discoverability**
   - All pools visible in one place
   - See stats for each pool at a glance

---

## Implementation Priority

1. **PoolManager** - Resource discovery and lifecycle
2. **Pool Selection in Group Dialog** - Dropdown instead of inline config
3. **Visual Query Builder** - GUI for creating SQL pools
4. **Pool Management Tab** - Create, edit, test, delete pools
5. **Pre-Launch Refresh Prompt** - Ask to refresh before group launch

This architecture is cleaner, more maintainable, and much more user-friendly!
