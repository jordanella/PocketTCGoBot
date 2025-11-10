# Unified Account Pool Migration Guide

## Overview

The account pool system has been redesigned with a new **Unified Pool** type that replaces both file-based and SQL-based pools. This new system provides:

- **Flexible Account Resolution**: Combine SQL queries, manual inclusions/exclusions, and watched paths
- **Global XML Storage**: Centralized XML file management at `./account_xmls/`
- **Database as Source of Truth**: All account data stored in SQLite database
- **Import/Export Operations**: Easy account management across different sources
- **Watched Paths**: Automatic folder monitoring for account imports

## Key Changes

### 1. New Unified Pool Type

The unified pool type supports multiple sources for populating accounts:

1. **SQL Queries** - Execute database queries to select accounts
2. **Manual Inclusions** - Explicitly add specific accounts
3. **Watched Paths** - Monitor folders for XML imports
4. **Manual Exclusions** - Remove specific accounts (applied last)

**Resolution Order**: Queries → Include → Watched Paths → Exclude

### 2. Global XML Storage

All account XMLs are now stored in a single global directory: `./account_xmls/`

- XMLs named by `deviceAccount` (e.g., `account@example.com.xml`)
- Generated on-demand from database if not present
- Used as performance cache for account injection
- Simplifies backup and account trading

### 3. Database-Backed

All account data is stored in the SQLite database (`bot.db`):

- `accounts` table is the source of truth
- XMLs are cache/backup only
- Import operations update database first
- Query-based account selection

### 4. Watched Paths (Read-Only)

Watched paths are folders that are monitored for account XMLs:

- **Read-only** - never written to
- XMLs are:
  1. Parsed for account data
  2. Imported to database (if not exists)
  3. Copied to global storage
  4. Added to pool's aggregated account list
- Useful for:
  - Shared network folders
  - Backup locations
  - Account trading workflows

## YAML Configuration

### Basic Example

```yaml
pool_name: "my_premium_pool"
description: "Premium accounts with 10+ packs"

queries:
  - name: "high_pack_accounts"
    sql: |
      SELECT device_account, device_password, shinedust, packs_opened, last_used_at
      FROM accounts
      WHERE packs_opened >= 10
      AND is_active = 1
      ORDER BY packs_opened DESC

config:
  sort_method: "packs_desc"
  retry_failed: true
  max_failures: 3
  refresh_interval: 300
```

### Full Example with All Features

```yaml
pool_name: "hybrid_pool"
description: "Hybrid pool combining queries, inclusions, exclusions, and watched paths"

# Multiple queries (results are combined)
queries:
  - name: "active_accounts"
    sql: |
      SELECT device_account, device_password, shinedust, packs_opened, last_used_at
      FROM accounts
      WHERE is_active = 1
      AND is_banned = 0

  - name: "recent_accounts"
    sql: |
      SELECT device_account, device_password, shinedust, packs_opened, last_used_at
      FROM accounts
      WHERE last_used_at >= datetime('now', '-7 days')

# Manually include specific accounts
include:
  - "premium1@example.com"
  - "test_account@example.com"

# Watch folders for automatic import
watched_paths:
  - "C:/shared/premium_accounts"
  - "./imported"

# Exclude specific accounts (applied last)
exclude:
  - "banned_account@example.com"

config:
  sort_method: "packs_desc"  # packs_asc, packs_desc, modified_asc, modified_desc
  retry_failed: true
  max_failures: 3
  refresh_interval: 300  # seconds (0 = disabled)
```

## Migration from Legacy Pools

### File Pools → Unified Pools

**Old File Pool** (`pools/my_file_pool.yaml`):
```yaml
name: "my_file_pool"
type: "file"
directory: "C:/accounts/manual"
pool_config:
  retry_failed: false
  max_failures: 1
```

**New Unified Pool** (`pools/my_file_pool.yaml`):
```yaml
pool_name: "my_file_pool"
description: "Migrated from file pool"

watched_paths:
  - "C:/accounts/manual"

config:
  sort_method: "modified_asc"
  retry_failed: false
  max_failures: 1
  refresh_interval: 0
```

**Migration Steps**:
1. Use `ImportFolder()` to import existing XMLs to database
2. Create new unified pool with `watched_paths` pointing to old directory
3. Delete old file pool definition

### SQL Pools → Unified Pools

**Old SQL Pool** (`pools/my_sql_pool.yaml`):
```yaml
name: "premium_accounts"
type: "sql"
query:
  select: |
    SELECT id, device_account, device_password, packs_opened, last_used_at, pool_status, failure_count, last_error
    FROM accounts
    WHERE packs_opened >= 50
```

**New Unified Pool** (`pools/my_sql_pool.yaml`):
```yaml
pool_name: "premium_accounts"
description: "Migrated from SQL pool"

queries:
  - name: "main_query"
    sql: |
      SELECT device_account, device_password, shinedust, packs_opened, last_used_at
      FROM accounts
      WHERE packs_opened >= 50

config:
  sort_method: "packs_desc"
  retry_failed: true
  max_failures: 3
  refresh_interval: 300
```

**Migration Steps**:
1. Update query to use simplified column selection
2. Change `type` from "sql" to "unified"
3. Restructure YAML to match unified format

## API Changes

### PoolManager Constructor

**Old**:
```go
poolManager := accountpool.NewPoolManager(poolsDir, db)
```

**New**:
```go
poolManager := accountpool.NewPoolManager(poolsDir, db, xmlStorageDir)
```

The `xmlStorageDir` parameter specifies the global XML storage location (e.g., `"account_xmls"`).

### New PoolManager Methods

```go
// Import accounts from arbitrary folder
imported, err := poolManager.ImportFolder("C:/accounts/batch1")

// Export single account XML
err := poolManager.ExportAccountXML("account@example.com", "./export")

// Get account XML content (from storage or generates from DB)
xml, err := poolManager.GetAccountXML("account@example.com")

// Ensure XML exists in global storage
err := poolManager.EnsureXMLExists("account@example.com")
```

## Directory Structure

```
PocketTCGoBot/
├── account_xmls/              # Global XML storage (auto-managed)
│   ├── account1@email.xml
│   ├── account2@email.xml
│   └── ...
├── pools/                     # Pool YAML definitions
│   ├── example_unified_pool.yaml.example
│   ├── premium_pool.yaml
│   ├── farm_pool.yaml
│   └── ...
├── bot.db                     # Database (source of truth)
└── ...
```

## GUI Changes

### Pool Creation

1. Open "Account Pools" tab
2. Click "+ New Pool"
3. Select "Unified Pool" (now default)
4. Follow instructions to create YAML file manually

The GUI now labels legacy pool types with "(Legacy)" to indicate they're deprecated.

### Pool Types in GUI

- **Unified Pool** - New recommended type
- **SQL Pool (Legacy)** - Old query-based pools (deprecated)
- **File Pool (Legacy)** - Old file-based pools (deprecated)

## Workflows

### Importing Accounts from Multiple Sources

```go
poolManager := accountpool.NewPoolManager("pools", db, "account_xmls")

// Import from various folders
poolManager.ImportFolder("C:/accounts/batch1")
poolManager.ImportFolder("D:/backup/accounts")
poolManager.ImportFolder("./imported")

// All imported accounts now in database and global storage
```

### Creating a Pool with Watched Paths

1. Create pool YAML:
```yaml
pool_name: "watched_pool"
description: "Pool with automatic imports"

watched_paths:
  - "C:/shared/accounts"
  - "./imported"

config:
  sort_method: "packs_desc"
  retry_failed: true
  max_failures: 3
  refresh_interval: 300  # Sync every 5 minutes
```

2. Place XMLs in watched folders
3. Pool automatically syncs during refresh intervals
4. Accounts imported to database and added to pool

### Exporting Accounts

```go
// Export single account
poolManager.ExportAccountXML("account@example.com", "./export")

// Export entire pool (TODO: needs implementation)
// poolManager.ExportPoolXMLs("premium_pool", "./export")
```

## Benefits

### For Users

1. **Flexibility** - Combine multiple account sources in one pool
2. **Simplicity** - One pool type instead of multiple types
3. **Reliability** - Database-backed with automatic XML generation
4. **Convenience** - Watched paths for automatic imports

### For Developers

1. **Single Implementation** - One pool type to maintain
2. **Testability** - Database-backed makes testing easier
3. **Extensibility** - Easy to add new query sources
4. **Performance** - Global XML storage reduces generation overhead

## Backward Compatibility

Legacy pool types (file and SQL) are still supported but marked as deprecated:

- Existing legacy pools will continue to work
- GUI labels them as "(Legacy)"
- Recommended to migrate to unified pools
- Legacy implementations may be removed in future versions

## Future Enhancements

Potential improvements to the unified pool system:

1. **GUI Pool Editor** - Visual editor for unified pool YAML
2. **Export Pool XMLs** - Complete implementation of pool export
3. **Pool Templates** - Pre-configured pool templates
4. **Advanced Filters** - More complex account selection logic
5. **Pool Groups** - Organize pools into hierarchical groups
6. **Real-time Watched Paths** - File system watcher for instant imports

## Troubleshooting

### Pool Not Loading

- Check YAML syntax (use `example_unified_pool.yaml.example` as reference)
- Ensure database is initialized
- Verify pool file has `.yaml` extension (not `.yaml.example`)
- Check logs for validation errors

### Accounts Not Appearing

- Verify SQL queries return results (test in database tool)
- Check account exclusions (may be filtering out accounts)
- Refresh pool manually to force reload
- Verify accounts exist in database

### XML Generation Fails

- Ensure `account_xmls/` directory is writable
- Check database has `device_account` and `device_password` columns
- Verify account data is not corrupted

### Watched Paths Not Syncing

- Check folder paths are valid and accessible
- Verify XMLs have correct format (`<account>` and `<password>` tags)
- Enable auto-refresh with `refresh_interval` > 0
- Manually refresh pool to trigger sync

## See Also

- [Example Unified Pool](../pools/example_unified_pool.yaml.example)
- [Database Schema](DATABASE_SCHEMA.md)
- [SQL Account Pools Documentation](sql_account_pools.md)
