# Complete Unified Account Pool System Guide

## Overview

The **Unified Account Pool System** is a complete redesign of account management that provides maximum flexibility through a single pool type with multiple account sources. This system eliminates the need for separate file-based and SQL-based pools by combining all features into one unified implementation.

## Table of Contents

1. [Features](#features)
2. [Architecture](#architecture)
3. [Quick Start](#quick-start)
4. [GUI Wizard](#gui-wizard)
5. [YAML Configuration](#yaml-configuration)
6. [Account Resolution](#account-resolution)
7. [Global XML Storage](#global-xml-storage)
8. [Import/Export Operations](#importexport-operations)
9. [API Reference](#api-reference)
10. [Best Practices](#best-practices)

---

## Features

### ✅ Single Unified Pool Type
- No more choosing between "file" or "sql" pools
- One pool type supports all use cases
- Simpler configuration and management

### ✅ Multiple Account Sources
- **SQL Queries**: Dynamic account selection from database
- **Manual Inclusions**: Explicitly add specific accounts
- **Watched Paths**: Auto-import from monitored folders
- **Manual Exclusions**: Remove unwanted accounts

### ✅ Flexible Account Resolution
- Accounts aggregated from all sources
- Resolution order: Queries → Include → Watched Paths → Exclude
- Exclusions applied last for maximum control

### ✅ Global XML Storage
- Centralized XML cache at `./account_xmls/`
- On-demand generation from database
- Fast account injection without regeneration
- Easy backup and account trading

### ✅ GUI Wizard
- 5-step wizard for pool creation
- Visual query builder
- Inline query testing
- Pool editing with pre-populated values

### ✅ Complete Import/Export
- Import from any folder
- Export entire pools
- Export individual accounts
- Automatic database synchronization

---

## Architecture

### Components

```
┌─────────────────────────────────────────┐
│         PoolManager                     │
│  - Discovers pool definitions           │
│  - Creates pool instances               │
│  - Manages global XML storage           │
│  - Import/export operations             │
└────────────┬────────────────────────────┘
             │
             │ creates
             ▼
┌─────────────────────────────────────────┐
│      UnifiedAccountPool                 │
│  - Account resolution engine            │
│  - Query execution                      │
│  - Watched path monitoring              │
│  - XML generation & caching             │
└────────────┬────────────────────────────┘
             │
             │ reads/writes
             ▼
┌──────────────────┐    ┌──────────────────┐
│   bot.db         │    │ ./account_xmls/  │
│  (Source of      │◄──►│  (XML Cache)     │
│   Truth)         │    │                  │
└──────────────────┘    └──────────────────┘
```

### Account Flow

```
1. Pool Definition (YAML)
   ↓
2. Account Resolution
   - Execute SQL queries
   - Add manual inclusions
   - Scan watched paths
   - Apply exclusions
   ↓
3. Account Storage
   - Database (credentials, metadata)
   - Global XML cache (for injection)
   ↓
4. Account Injection
   - Bot requests account
   - XML retrieved or generated
   - Account injected to emulator
```

---

## Quick Start

### 1. Create Your First Pool

**Option A: Using GUI Wizard**
1. Open GUI → Account Pools tab
2. Click "+ New Pool"
3. Select "Unified Pool"
4. Follow the 5-step wizard
5. Click "Create Pool"

**Option B: Manual YAML**
```yaml
# pools/my_first_pool.yaml
pool_name: "my_first_pool"
description: "My first unified pool"

queries:
  - name: "active_accounts"
    sql: |
      SELECT device_account, device_password, shinedust, packs_opened, last_used_at
      FROM accounts
      WHERE is_active = 1
      ORDER BY packs_opened DESC
      LIMIT 10

config:
  sort_method: "packs_desc"
  retry_failed: true
  max_failures: 3
  refresh_interval: 300
```

### 2. Test the Pool

In GUI:
1. Select your pool from the list
2. Click "Test"
3. View results

Or programmatically:
```go
result, err := poolManager.TestPool("my_first_pool")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Found %d accounts\n", result.AccountsFound)
```

### 3. Use the Pool

```go
pool, err := poolManager.GetPool("my_first_pool")
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()
account, err := pool.GetNext(ctx)
if err != nil {
    log.Fatal(err)
}

// Use account...
fmt.Printf("Using account: %s\n", account.DeviceAccount)
```

---

## GUI Wizard

### Step 1: Basic Information
- Enter pool name (auto-filled)
- Add description
- Preview supported features

### Step 2: SQL Queries (Optional)
- Add multiple queries
- Edit query name and SQL
- Remove unwanted queries
- All query results are combined

**Query Requirements:**
- Must return: `device_account`, `device_password`, `shinedust`, `packs_opened`, `last_used_at`
- Use `SELECT` statements only
- Parameterized queries supported

### Step 3: Manual Inclusions (Optional)
- Enter device accounts (one per line)
- Accounts fetched from database
- Added to pool after query results

### Step 4: Manual Exclusions (Optional)
- Enter device accounts to exclude
- Applied LAST (after all other sources)
- Useful for blacklisting specific accounts

### Step 5: Configuration
- **Sort Method**: How to order accounts
  - `packs_desc` - Most packs first
  - `packs_asc` - Fewest packs first
  - `modified_desc` - Recently used first
  - `modified_asc` - Oldest used first
- **Retry Failed**: Re-queue failed accounts
- **Max Failures**: Attempts before permanent failure
- **Refresh Interval**: Auto-refresh seconds (0 = manual only)
- **Watched Paths**: Folders to monitor for XML imports

---

## YAML Configuration

### Complete Example

```yaml
pool_name: "production_pool"
description: "Production accounts with multiple sources"

# SQL Queries (optional)
queries:
  - name: "high_value_accounts"
    sql: |
      SELECT device_account, device_password, shinedust, packs_opened, last_used_at
      FROM accounts
      WHERE packs_opened >= 50
        AND shinedust >= 1000
        AND is_active = 1
        AND is_banned = 0
      ORDER BY packs_opened DESC

  - name: "recently_active"
    sql: |
      SELECT device_account, device_password, shinedust, packs_opened, last_used_at
      FROM accounts
      WHERE last_used_at >= datetime('now', '-3 days')
        AND is_active = 1
      ORDER BY last_used_at DESC

# Manual Inclusions (optional)
include:
  - "premium_account_1@example.com"
  - "premium_account_2@example.com"
  - "test_account@example.com"

# Watched Paths (optional)
watched_paths:
  - "C:/shared/premium_accounts"
  - "D:/backup/accounts"
  - "./imported_accounts"

# Manual Exclusions (optional) - applied LAST
exclude:
  - "banned_account@example.com"
  - "maintenance_account@example.com"
  - "broken_account@example.com"

# Pool Configuration
config:
  sort_method: "packs_desc"
  retry_failed: true
  max_failures: 3
  refresh_interval: 300  # 5 minutes
```

### Minimal Example

```yaml
pool_name: "simple_pool"
description: "Just a basic query"

queries:
  - name: "all_active"
    sql: |
      SELECT device_account, device_password, shinedust, packs_opened, last_used_at
      FROM accounts
      WHERE is_active = 1

config:
  sort_method: "modified_asc"
  retry_failed: false
  max_failures: 1
  refresh_interval: 0
```

---

## Account Resolution

### Resolution Order

1. **Execute Queries** - All queries run, results combined
2. **Add Inclusions** - Manual includes added
3. **Scan Watched Paths** - Import XMLs from folders
4. **Apply Exclusions** - Remove excluded accounts

### Example Flow

```yaml
queries:
  - sql: "SELECT ... WHERE packs > 10"  # Returns: acc1, acc2, acc3

include:
  - acc4
  - acc5

watched_paths:
  - "./imports"  # Contains: acc6.xml, acc7.xml

exclude:
  - acc3  # Remove this one

# Final pool: acc1, acc2, acc4, acc5, acc6, acc7
```

### Conflict Handling

**Duplicate Accounts:**
- If account appears in multiple queries: included once
- If account in both query and include: included once
- If account in both watched path and query: included once

**Exclusion Priority:**
- Exclusions are ALWAYS applied last
- Overrides all other sources
- Useful for temporary blacklisting

---

## Global XML Storage

### Location
- All account XMLs stored in `./account_xmls/`
- Named by device_account: `account@example.com.xml`
- Automatically created as needed

### Behavior

**On Account Injection:**
1. Check if XML exists in global storage
2. If exists: use cached file (fast)
3. If not exists: generate from database
4. Save to global storage for future use

**Benefits:**
- Fast account injection (no regeneration)
- Easy backup (single folder)
- Simple account trading (copy XML files)
- Persistent across pool changes

### Manual Management

```go
// Get XML (from cache or generate)
xml, err := poolManager.GetAccountXML("account@example.com")

// Ensure XML exists
err := poolManager.EnsureXMLExists("account@example.com")

// Export single account
err := poolManager.ExportAccountXML("account@example.com", "./export")
```

---

## Import/Export Operations

### Import from Folder

**GUI:**
1. Account Pools → Actions
2. Click "Import Folder"
3. Select folder with XML files
4. View import results

**Programmatically:**
```go
imported, err := poolManager.ImportFolder("C:/accounts/batch1")
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Imported %d accounts\n", len(imported))
```

**What Happens:**
1. XML files parsed
2. Accounts inserted/updated in database
3. XMLs copied to global storage
4. Returns list of imported device_accounts

### Export Pool

**GUI:**
1. Select pool
2. Click "Export Pool"
3. Choose destination folder
4. All pool accounts exported

**Programmatically:**
```go
err := poolManager.ExportPoolXMLs("my_pool", "./export")
if err != nil {
    log.Fatal(err)
}
```

**What Happens:**
1. All accounts listed from pool
2. XMLs retrieved (from cache or generated)
3. Saved to destination folder
4. Named by device_account

### Export Single Account

```go
err := poolManager.ExportAccountXML("acc@example.com", "./export")
```

---

## API Reference

### PoolManager

```go
// Create pool manager
poolManager := accountpool.NewPoolManager(
    poolsDir string,      // e.g., "pools"
    db *sql.DB,           // Database connection
    xmlStorageDir string, // e.g., "account_xmls"
)

// Discover pools from YAML files
err := poolManager.DiscoverPools()

// List all pools
pools := poolManager.ListPools()

// Get pool definition
poolDef, err := poolManager.GetPoolDefinition(name string)

// Get or create pool instance
pool, err := poolManager.GetPool(name string)

// Test pool (dry run)
result, err := poolManager.TestPool(name string)

// Create pool
err := poolManager.CreatePool(poolDef *PoolDefinition)

// Update pool
err := poolManager.UpdatePool(name string, poolDef *PoolDefinition)

// Delete pool
err := poolManager.DeletePool(name string)

// Import/Export
imported, err := poolManager.ImportFolder(folderPath string)
err := poolManager.ExportPoolXMLs(poolName, destFolder string)
err := poolManager.ExportAccountXML(deviceAccount, destFolder string)
xml, err := poolManager.GetAccountXML(deviceAccount string)
err := poolManager.EnsureXMLExists(deviceAccount string)

// Refresh pool
err := poolManager.RefreshPool(name string)

// Close pool instance
err := poolManager.ClosePool(name string)

// Close all pools
err := poolManager.CloseAll()
```

### AccountPool Interface

```go
// Get next available account
account, err := pool.GetNext(ctx context.Context)

// Return account to pool
err := pool.Return(account *Account)

// Mark account as used (successfully processed)
err := pool.MarkUsed(account *Account, result AccountResult)

// Mark account as failed
err := pool.MarkFailed(account *Account, reason string)

// Get account by ID
account, err := pool.GetByID(id string)

// Get pool statistics
stats := pool.GetStats()

// Refresh pool from sources
err := pool.Refresh()

// List all accounts
accounts := pool.ListAccounts()

// Close pool
err := pool.Close()
```

---

## Best Practices

### 1. Use Descriptive Names
```yaml
pool_name: "high_value_premium_accounts"  # Good
pool_name: "pool1"                        # Bad
```

### 2. Add Descriptions
```yaml
description: "Premium accounts with 50+ packs for event farming"  # Good
description: "Accounts"                                           # Bad
```

### 3. Name Your Queries
```yaml
queries:
  - name: "event_ready_accounts"  # Good
    sql: "SELECT ..."
  - name: "query1"                # Bad
    sql: "SELECT ..."
```

### 4. Use Appropriate Refresh Intervals
```yaml
# High-activity pool
refresh_interval: 300  # 5 minutes

# Low-activity pool
refresh_interval: 3600  # 1 hour

# Manual-only pool
refresh_interval: 0  # No auto-refresh
```

### 5. Leverage Watched Paths for Shared Accounts
```yaml
# Network share for team
watched_paths:
  - "//server/shared/premium_accounts"

# Multiple team members can drop XMLs here
# Pool auto-imports them
```

### 6. Use Exclusions for Temporary Blacklisting
```yaml
# Don't delete accounts, just exclude them
exclude:
  - "maintenance_account@example.com"  # Under maintenance
  - "broken_account@example.com"       # Investigating issue
```

### 7. Combine Multiple Queries for Complex Logic
```yaml
queries:
  # High priority accounts
  - name: "premium"
    sql: "SELECT ... WHERE premium = 1"

  # Fill remaining slots with active accounts
  - name: "active_fallback"
    sql: "SELECT ... WHERE last_used < '7 days ago' LIMIT 50"
```

### 8. Backup Your Pools
```bash
# Backup pool definitions
cp -r pools/ backups/pools-$(date +%Y%m%d)/

# Backup global XML storage
cp -r account_xmls/ backups/account_xmls-$(date +%Y%m%d)/

# Backup database
cp bot.db backups/bot-$(date +%Y%m%d).db
```

### 9. Monitor Pool Health
```go
stats := pool.GetStats()
fmt.Printf("Available: %d\n", stats.Available)
fmt.Printf("Failed: %d\n", stats.Failed)
fmt.Printf("Completed: %d\n", stats.Completed)

// Alert if too many failures
if stats.Failed > stats.Total / 10 {
    log.Warn("Pool has high failure rate")
}
```

### 10. Test Before Production
```go
// Always test new pools
result, err := poolManager.TestPool("new_pool")
if err != nil {
    log.Fatal(err)
}

if result.AccountsFound == 0 {
    log.Fatal("Pool returned no accounts!")
}

fmt.Printf("Pool test successful: %d accounts\n", result.AccountsFound)
```

---

## Troubleshooting

### Pool Returns No Accounts

**Check:**
1. SQL queries return results (test in DB browser)
2. Manual includes exist in database
3. Watched paths are valid and accessible
4. Exclusions aren't filtering everything
5. Accounts meet min/max pack requirements

**Debug:**
```go
// Get pool instance
pool, _ := poolManager.GetPool("my_pool")

// Check stats
stats := pool.GetStats()
fmt.Printf("Total: %d, Available: %d\n", stats.Total, stats.Available)

// List all accounts
accounts := pool.ListAccounts()
for _, acc := range accounts {
    fmt.Printf("%s - %s\n", acc.DeviceAccount, acc.Status)
}
```

### XML Generation Fails

**Check:**
1. `account_xmls/` directory exists and is writable
2. Database has `device_account` and `device_password`
3. Account data is not corrupted

**Fix:**
```bash
# Recreate directory
mkdir -p account_xmls
chmod 755 account_xmls

# Verify database
sqlite3 bot.db "SELECT COUNT(*) FROM accounts WHERE device_account IS NOT NULL"
```

### Watched Paths Not Syncing

**Check:**
1. Paths are absolute or relative from bot root
2. Paths exist and are accessible
3. XMLs have correct format
4. Auto-refresh is enabled

**Test:**
```go
// Manual sync
pool, _ := poolManager.GetPool("my_pool")
err := pool.Refresh()  // Force sync
```

### Pool Performance Issues

**Optimize:**
1. Add database indexes
2. Limit query results
3. Increase refresh interval
4. Reduce watched path count

```sql
-- Add indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_accounts_active ON accounts(is_active);
CREATE INDEX IF NOT EXISTS idx_accounts_packs ON accounts(packs_opened);
CREATE INDEX IF NOT EXISTS idx_accounts_last_used ON accounts(last_used_at);
```

---

## See Also

- [Unified Pool Migration Guide](UNIFIED_POOL_MIGRATION.md)
- [Example Unified Pool](../pools/example_unified_pool.yaml.example)
- [Test Pool](../pools/test_unified_pool.yaml.example)
- [Database Schema](DATABASE_SCHEMA.md)
