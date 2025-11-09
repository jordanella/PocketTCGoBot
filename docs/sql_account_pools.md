# SQL-Based Account Pools

## Overview
SQL-based account pools allow users to define dynamic account selection criteria using SQL queries stored in YAML format. This provides powerful filtering, sorting, and selection capabilities beyond static file scanning.

---

## Architecture

### Query Definition Format (YAML)

```yaml
# pools/premium_accounts.yaml
name: "Premium Farmers Pool"
description: "Accounts with 10+ packs, sorted by pack count descending"
type: "sql"
version: "1.0"

# Query configuration
query:
  # Main SELECT query (parameterized for safety)
  select: |
    SELECT
      id,
      xml_path,
      pack_count,
      last_modified,
      status,
      failure_count,
      last_error
    FROM accounts
    WHERE status IN (?, ?)
      AND pack_count >= ?
      AND failure_count < ?
    ORDER BY pack_count DESC, last_modified ASC
    LIMIT ?

  # Parameters for the query (values substituted safely)
  parameters:
    - name: "status_available"
      value: "available"
      type: "string"

    - name: "status_skipped"
      value: "skipped"
      type: "string"

    - name: "min_packs"
      value: 10
      type: "int"

    - name: "max_failures"
      value: 3
      type: "int"

    - name: "limit"
      value: 100
      type: "int"

# Pool behavior settings
pool_config:
  retry_failed: true
  max_failures: 3
  wait_for_accounts: true
  max_wait_time: "5m"
  buffer_size: 50
  refresh_interval: "30s"  # Auto-refresh query results

# GUI builder configuration (for visual query builder)
gui_config:
  filters:
    - field: "status"
      operator: "in"
      values: ["available", "skipped"]
      display: "Status is Available or Skipped"

    - field: "pack_count"
      operator: ">="
      value: 10
      display: "Pack Count >= 10"

    - field: "failure_count"
      operator: "<"
      value: 3
      display: "Failure Count < 3"

  sort:
    - field: "pack_count"
      direction: "DESC"
      display: "Most Packs First"

    - field: "last_modified"
      direction: "ASC"
      display: "Then Oldest Modified"

  limit: 100
```

---

## Implementation

### 1. SQLAccountPool Type

```go
package accountpool

import (
    "context"
    "database/sql"
    "fmt"
    "sync"
    "time"
)

// SQLAccountPool implements AccountPool using SQL queries
type SQLAccountPool struct {
    mu           sync.RWMutex
    db           *sql.DB
    queryDef     *QueryDefinition
    accounts     map[string]*Account
    available    chan *Account
    config       PoolConfig
    closed       bool
    stopRefresh  chan struct{}
    lastRefresh  time.Time
}

// QueryDefinition defines a SQL query for account selection
type QueryDefinition struct {
    Name        string                 `yaml:"name"`
    Description string                 `yaml:"description"`
    Type        string                 `yaml:"type"`
    Version     string                 `yaml:"version"`
    Query       QueryConfig            `yaml:"query"`
    PoolConfig  PoolConfig             `yaml:"pool_config"`
    GUIConfig   *GUIQueryConfig        `yaml:"gui_config,omitempty"`
}

// QueryConfig defines the SQL query and parameters
type QueryConfig struct {
    Select     string       `yaml:"select"`
    Parameters []Parameter  `yaml:"parameters"`
}

// Parameter represents a query parameter
type Parameter struct {
    Name  string      `yaml:"name"`
    Value interface{} `yaml:"value"`
    Type  string      `yaml:"type"` // "string", "int", "float", "bool"
}

// GUIQueryConfig stores visual query builder configuration
type GUIQueryConfig struct {
    Filters []FilterConfig `yaml:"filters"`
    Sort    []SortConfig   `yaml:"sort"`
    Limit   int            `yaml:"limit"`
}

// FilterConfig represents a single filter condition
type FilterConfig struct {
    Field    string        `yaml:"field"`
    Operator string        `yaml:"operator"` // "=", "!=", "<", ">", "<=", ">=", "in", "like"
    Value    interface{}   `yaml:"value,omitempty"`
    Values   []interface{} `yaml:"values,omitempty"`
    Display  string        `yaml:"display"`
}

// SortConfig represents a sort order
type SortConfig struct {
    Field     string `yaml:"field"`
    Direction string `yaml:"direction"` // "ASC", "DESC"
    Display   string `yaml:"display"`
}
```

### 2. Pool Creation

```go
// NewSQLAccountPool creates a pool from a SQL query definition
func NewSQLAccountPool(db *sql.DB, queryDefPath string) (*SQLAccountPool, error) {
    // Load query definition from YAML
    queryDef, err := loadQueryDefinition(queryDefPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load query definition: %w", err)
    }

    pool := &SQLAccountPool{
        db:          db,
        queryDef:    queryDef,
        accounts:    make(map[string]*Account),
        available:   make(chan *Account, queryDef.PoolConfig.BufferSize),
        config:      queryDef.PoolConfig,
        stopRefresh: make(chan struct{}),
    }

    // Initial query execution
    if err := pool.refresh(); err != nil {
        return nil, fmt.Errorf("initial query execution failed: %w", err)
    }

    // Start auto-refresh if configured
    if queryDef.PoolConfig.RefreshInterval > 0 {
        go pool.autoRefresh()
    }

    return pool, nil
}
```

### 3. Query Execution

```go
// refresh executes the SQL query and updates the account pool
func (p *SQLAccountPool) refresh() error {
    p.mu.Lock()
    defer p.mu.Unlock()

    // Build parameter array
    params := make([]interface{}, len(p.queryDef.Query.Parameters))
    for i, param := range p.queryDef.Query.Parameters {
        params[i] = param.Value
    }

    // Execute query
    rows, err := p.db.Query(p.queryDef.Query.Select, params...)
    if err != nil {
        return fmt.Errorf("query execution failed: %w", err)
    }
    defer rows.Close()

    // Parse results into accounts
    newAccounts := make(map[string]*Account)
    for rows.Next() {
        account := &Account{}
        err := rows.Scan(
            &account.ID,
            &account.XMLPath,
            &account.PackCount,
            &account.LastModified,
            &account.Status,
            &account.FailureCount,
            &account.LastError,
        )
        if err != nil {
            return fmt.Errorf("failed to scan account row: %w", err)
        }

        newAccounts[account.ID] = account
    }

    // Update accounts map
    p.accounts = newAccounts

    // Refill available channel
    p.refillAvailableChannel()

    p.lastRefresh = time.Now()
    return nil
}

// refillAvailableChannel repopulates the buffered channel
func (p *SQLAccountPool) refillAvailableChannel() {
    // Drain existing channel
    for len(p.available) > 0 {
        <-p.available
    }

    // Refill with current available accounts
    for _, account := range p.accounts {
        if account.Status == AccountStatusAvailable {
            select {
            case p.available <- account:
            default:
                // Channel full, remaining accounts will be fetched on demand
                return
            }
        }
    }
}
```

---

## User Workflow

### 1. Pre-Launch Refresh Prompt

```
User clicks "Launch" on a group using SQL pool
    ↓
[Dialog Appears]
┌─────────────────────────────────────────────┐
│ Refresh Account Pool?                       │
├─────────────────────────────────────────────┤
│ Pool: "Premium Farmers Pool"                │
│ Last refreshed: 2 minutes ago               │
│                                              │
│ Re-run query to get latest accounts?        │
│                                              │
│ [ ] Remember my choice                      │
│                                              │
│     [Refresh]  [Use Cached]  [Cancel]       │
└─────────────────────────────────────────────┘
```

### 2. Query Definition UI

```
[Create/Edit SQL Pool]
┌─────────────────────────────────────────────┐
│ Pool Name: [Premium Farmers Pool         ] │
│ Description: [Accounts with 10+ packs...  ] │
├─────────────────────────────────────────────┤
│ Query Builder                               │
│                                              │
│ Filters:                                    │
│   [+] Status         IN      [Available ▼]  │
│   [+] Pack Count     >=      [10        ]   │
│   [+] Failure Count  <       [3         ]   │
│                                              │
│ Sort:                                       │
│   [↑] Pack Count     Descending             │
│   [↑] Last Modified  Ascending              │
│                                              │
│ Limit: [100     ]                           │
│                                              │
│ [Preview Query]  [Test Query]               │
├─────────────────────────────────────────────┤
│ Advanced (SQL Editor)                       │
│ ┌─────────────────────────────────────────┐ │
│ │ SELECT id, xml_path, pack_count...     │ │
│ │ FROM accounts                           │ │
│ │ WHERE status IN (?, ?)                  │ │
│ │   AND pack_count >= ?                   │ │
│ │ ...                                     │ │
│ └─────────────────────────────────────────┘ │
│                                              │
│     [Save]  [Cancel]                        │
└─────────────────────────────────────────────┘
```

---

## Query Builder Components

### Available Filters

| Field | Type | Operators | Description |
|-------|------|-----------|-------------|
| `status` | enum | `=`, `!=`, `IN` | Account status |
| `pack_count` | int | `=`, `!=`, `<`, `>`, `<=`, `>=` | Number of packs |
| `last_modified` | timestamp | `<`, `>`, `<=`, `>=`, `BETWEEN` | Last modification time |
| `failure_count` | int | `=`, `!=`, `<`, `>`, `<=`, `>=` | Number of failures |
| `cards_found` | int | `=`, `!=`, `<`, `>`, `<=`, `>=` | Total cards found |
| `stars_total` | int | `=`, `!=`, `<`, `>`, `<=`, `>=` | Total stars collected |
| `completed_at` | timestamp | `IS NULL`, `IS NOT NULL`, `<`, `>` | Completion time |
| `custom_metadata` | json | `LIKE`, `=` | Custom metadata fields |

### Query Validation

Before saving, validate:
- ✓ SQL syntax is correct
- ✓ Only SELECT queries allowed (no INSERT/UPDATE/DELETE)
- ✓ Required columns are present in SELECT
- ✓ Parameters match query placeholders
- ✓ Test execution returns valid results

---

## Safety Features

### 1. Query Sanitization
```go
// Only allow SELECT queries
func validateQuery(query string) error {
    upper := strings.ToUpper(strings.TrimSpace(query))
    if !strings.HasPrefix(upper, "SELECT") {
        return fmt.Errorf("only SELECT queries are allowed")
    }

    // Check for dangerous keywords
    dangerous := []string{"DROP", "DELETE", "UPDATE", "INSERT", "ALTER", "CREATE"}
    for _, keyword := range dangerous {
        if strings.Contains(upper, keyword) {
            return fmt.Errorf("query contains forbidden keyword: %s", keyword)
        }
    }

    return nil
}
```

### 2. Parameterized Queries
All user inputs are passed as parameters, never concatenated into SQL:

```go
// SAFE - Uses parameterized query
query := "SELECT * FROM accounts WHERE pack_count >= ? AND status = ?"
rows, _ := db.Query(query, minPacks, status)

// UNSAFE - Never do this!
query := fmt.Sprintf("SELECT * FROM accounts WHERE pack_count >= %d", minPacks)
```

### 3. Query Timeout
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

rows, err := db.QueryContext(ctx, query, params...)
```

---

## Example Queries

### 1. Fresh Accounts (Never Used)
```yaml
name: "Fresh Account Pool"
query:
  select: |
    SELECT id, xml_path, pack_count, last_modified, status
    FROM accounts
    WHERE status = ?
      AND completed_at IS NULL
      AND failure_count = 0
    ORDER BY pack_count DESC
    LIMIT ?
  parameters:
    - {name: "status", value: "available", type: "string"}
    - {name: "limit", value: 50, type: "int"}
```

### 2. High-Value Retry Pool
```yaml
name: "High Value Retry Pool"
query:
  select: |
    SELECT id, xml_path, pack_count, last_modified, status
    FROM accounts
    WHERE status IN (?, ?)
      AND pack_count >= ?
      AND failure_count BETWEEN ? AND ?
      AND last_error NOT LIKE '%banned%'
    ORDER BY pack_count DESC, failure_count ASC
    LIMIT ?
  parameters:
    - {name: "status_failed", value: "failed", type: "string"}
    - {name: "status_skipped", value: "skipped", type: "string"}
    - {name: "min_packs", value: 15, type: "int"}
    - {name: "min_failures", value: 1, type: "int"}
    - {name: "max_failures", value: 2, type: "int"}
    - {name: "limit", value: 25, type: "int"}
```

### 3. Recently Modified
```yaml
name: "Recently Updated Accounts"
query:
  select: |
    SELECT id, xml_path, pack_count, last_modified, status
    FROM accounts
    WHERE status = ?
      AND last_modified >= datetime('now', '-7 days')
    ORDER BY last_modified DESC
    LIMIT ?
  parameters:
    - {name: "status", value: "available", type: "string"}
    - {name: "limit", value: 100, type: "int"}
```

### 4. Star-Rich Accounts
```yaml
name: "High Star Accounts"
query:
  select: |
    SELECT id, xml_path, pack_count, last_modified, status, stars_total
    FROM accounts
    WHERE status = ?
      AND stars_total >= ?
    ORDER BY stars_total DESC, pack_count DESC
    LIMIT ?
  parameters:
    - {name: "status", value: "available", type: "string"}
    - {name: "min_stars", value: 50, type: "int"}
    - {name: "limit", value: 30, type: "int"}
```

---

## Integration with Manager Groups

### Group Configuration
```yaml
group:
  name: "Premium Farmers"
  routine: "farm_premium_packs.yaml"
  instances: [0, 1, 2, 3]
  bot_count: 3

  # SQL-based account pool
  account_pool:
    type: "sql"
    query_file: "./pools/premium_accounts.yaml"
    refresh_on_launch: true  # Prompt user to refresh
    auto_refresh: true       # Auto-refresh during execution
    refresh_interval: "5m"   # Refresh every 5 minutes
```

### Launch Flow with SQL Pool
```
1. User clicks "Launch Group"
    ↓
2. Orchestrator detects SQL pool
    ↓
3. Check last refresh time
    ↓
4. IF refresh_on_launch AND (last_refresh > 5min OR user_setting = "always_ask"):
     Show refresh prompt dialog

   User chooses:
     - Refresh: Execute query, update pool
     - Use Cached: Use existing results
     - Cancel: Abort launch
    ↓
5. Proceed with normal launch flow
```

---

## Database Schema Requirements

The SQL pools query from the existing `accounts` table:

```sql
CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    xml_path TEXT NOT NULL,
    pack_count INTEGER DEFAULT 0,
    last_modified TIMESTAMP,
    status TEXT DEFAULT 'available',
    failure_count INTEGER DEFAULT 0,
    last_error TEXT,
    completed_at TIMESTAMP,
    cards_found INTEGER DEFAULT 0,
    stars_total INTEGER DEFAULT 0,
    keep_count INTEGER DEFAULT 0,
    custom_metadata TEXT  -- JSON blob for extensibility
);

CREATE INDEX idx_accounts_status ON accounts(status);
CREATE INDEX idx_accounts_pack_count ON accounts(pack_count);
CREATE INDEX idx_accounts_modified ON accounts(last_modified);
```

---

## Benefits

1. **Dynamic Selection**: Query fresh data every time instead of static file lists
2. **Complex Criteria**: Use SQL's full power (joins, aggregations, subqueries if needed)
3. **GUI-Friendly**: Visual query builder for non-technical users
4. **Reusable**: Save query definitions as templates
5. **Safe**: Parameterized queries prevent SQL injection
6. **Transparent**: Users can see and understand the query
7. **Testable**: Test queries before using them
8. **Flexible**: Switch between file-based and SQL-based pools easily

---

## Implementation Priority

1. **Phase 1**: Query definition YAML format and parser
2. **Phase 2**: SQLAccountPool implementation
3. **Phase 3**: Pre-launch refresh prompt
4. **Phase 4**: Basic query builder GUI (filters, sorts, limit)
5. **Phase 5**: Advanced SQL editor mode
6. **Phase 6**: Query templates and sharing

---

## Future Enhancements

- **Query Templates**: Pre-built query templates for common use cases
- **Multi-Database Support**: PostgreSQL, MySQL, etc.
- **Query Chaining**: Run multiple queries and merge results
- **Scheduled Refresh**: Auto-refresh at specific times
- **Query Analytics**: Track which queries perform best
- **Export Results**: Save query results to CSV/JSON
- **Query Sharing**: Import/export query definitions
