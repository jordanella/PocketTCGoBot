# Database Schema Design - PocketTCGo Bot

## Overview

SQLite database for tracking account progress, activity, pack results, and card collections.

## Database Location

Proposed: `./data/pocket_tcg_bot.db`

## Schema

### 1. accounts

Core account information and resources.

```sql
CREATE TABLE accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_account TEXT NOT NULL UNIQUE,      -- From deviceAccount XML
    device_password TEXT NOT NULL,            -- From deviceAccount XML
    username TEXT,                            -- In-game username
    friend_code TEXT,                         -- Friend code (if available)

    -- Resources
    shinedust INTEGER DEFAULT 0,              -- Dust currency
    hourglasses INTEGER DEFAULT 0,            -- Hourglass currency (pack timers)
    pokegold INTEGER DEFAULT 0,               -- Premium currency
    pack_points INTEGER DEFAULT 0,            -- Points for pack exchange

    -- Statistics
    packs_opened INTEGER DEFAULT 0,           -- Total packs opened
    wonder_picks_done INTEGER DEFAULT 0,      -- Total wonder picks
    account_level INTEGER DEFAULT 1,          -- Player level

    -- Timestamps
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_used_at DATETIME,                    -- Last time bot used this account
    stamina_recovery_time DATETIME,           -- When stamina/packs recover

    -- Metadata
    file_path TEXT,                           -- Path to deviceAccount XML file
    is_active BOOLEAN DEFAULT 1,              -- Whether account is active
    is_banned BOOLEAN DEFAULT 0,              -- Flagged if detected banned
    notes TEXT,                               -- User notes

    UNIQUE(device_account)
);

CREATE INDEX idx_accounts_device_account ON accounts(device_account);
CREATE INDEX idx_accounts_last_used ON accounts(last_used_at);
CREATE INDEX idx_accounts_active ON accounts(is_active);
```

### 2. activity_log

High-level activity tracking per session.

```sql
CREATE TABLE activity_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,

    -- Activity details
    activity_type TEXT NOT NULL,              -- 'wonder_pick', 'pack_opening', 'solo_battle', etc.
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    duration_seconds INTEGER,                 -- How long it took

    -- Status
    status TEXT DEFAULT 'running',            -- 'running', 'completed', 'failed', 'aborted'
    error_message TEXT,                       -- If status='failed'

    -- Context
    bot_version TEXT,                         -- Version of bot
    routine_name TEXT,                        -- Which routine was running

    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

CREATE INDEX idx_activity_account ON activity_log(account_id);
CREATE INDEX idx_activity_type ON activity_log(activity_type);
CREATE INDEX idx_activity_started ON activity_log(started_at);
```

### 3. error_log

Detailed error tracking for debugging and monitoring.

```sql
CREATE TABLE error_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER,                       -- NULL if not account-specific
    activity_log_id INTEGER,                  -- Link to activity if applicable

    -- Error details
    error_type TEXT NOT NULL,                 -- 'ErrorPopup', 'ErrorStuck', 'ErrorCommunication', etc.
    error_severity TEXT NOT NULL,             -- 'critical', 'high', 'medium', 'low'
    error_message TEXT NOT NULL,
    stack_trace TEXT,                         -- If available

    -- Context
    screen_state TEXT,                        -- What screen bot was on
    template_name TEXT,                       -- Template being searched
    action_name TEXT,                         -- What action was executing

    -- Recovery
    was_recovered BOOLEAN DEFAULT 0,
    recovery_action TEXT,                     -- What action was taken
    recovery_time_ms INTEGER,                 -- How long recovery took

    occurred_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE SET NULL,
    FOREIGN KEY (activity_log_id) REFERENCES activity_log(id) ON DELETE SET NULL
);

CREATE INDEX idx_error_account ON error_log(account_id);
CREATE INDEX idx_error_type ON error_log(error_type);
CREATE INDEX idx_error_occurred ON error_log(occurred_at);
CREATE INDEX idx_error_severity ON error_log(error_severity);
```

### 4. pack_results

Individual pack opening results.

```sql
CREATE TABLE pack_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    activity_log_id INTEGER,

    -- Pack details
    pack_type TEXT NOT NULL,                  -- 'genetic_apex', 'mewtwo', 'charizard', 'pikachu'
    pack_name TEXT,                           -- Specific pack name if available
    is_god_pack BOOLEAN DEFAULT 0,            -- Was it a god pack?
    card_count INTEGER DEFAULT 5,             -- Usually 5, sometimes 6

    -- Statistics
    rarity_breakdown TEXT,                    -- JSON: {"1_diamond": 3, "2_diamond": 1, "3_diamond": 1}
    pack_points_earned INTEGER DEFAULT 0,     -- Points earned from this pack

    opened_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (activity_log_id) REFERENCES activity_log(id) ON DELETE SET NULL
);

CREATE INDEX idx_pack_account ON pack_results(account_id);
CREATE INDEX idx_pack_opened ON pack_results(opened_at);
CREATE INDEX idx_pack_type ON pack_results(pack_type);
CREATE INDEX idx_pack_god_pack ON pack_results(is_god_pack) WHERE is_god_pack = 1;
```

### 5. cards_pulled

Individual cards pulled from packs.

```sql
CREATE TABLE cards_pulled (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pack_result_id INTEGER NOT NULL,
    account_id INTEGER NOT NULL,

    -- Card details
    card_id TEXT NOT NULL,                    -- Game's card ID if detectable
    card_name TEXT,                           -- Card name (OCR or lookup)
    card_number TEXT,                         -- Card number (e.g., "001/165")
    rarity TEXT NOT NULL,                     -- '1_diamond', '2_diamond', '3_diamond', '4_diamond', 'crown'
    card_type TEXT,                           -- 'pokemon', 'trainer', 'energy'
    is_full_art BOOLEAN DEFAULT 0,
    is_ex BOOLEAN DEFAULT 0,

    -- Detection
    detection_confidence REAL,               -- CV confidence score
    detected_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (pack_result_id) REFERENCES pack_results(id) ON DELETE CASCADE,
    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

CREATE INDEX idx_cards_pack ON cards_pulled(pack_result_id);
CREATE INDEX idx_cards_account ON cards_pulled(account_id);
CREATE INDEX idx_cards_rarity ON cards_pulled(rarity);
CREATE INDEX idx_cards_name ON cards_pulled(card_name);
```

### 6. account_collection

Track which cards each account owns (deduplicated).

```sql
CREATE TABLE account_collection (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,

    -- Card identity
    card_id TEXT NOT NULL,                    -- Unique card identifier
    card_name TEXT NOT NULL,
    card_number TEXT,
    rarity TEXT NOT NULL,

    -- Ownership
    quantity INTEGER DEFAULT 1,               -- How many owned (usually capped at 2-3)
    first_obtained_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_obtained_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
    UNIQUE(account_id, card_id)
);

CREATE INDEX idx_collection_account ON account_collection(account_id);
CREATE INDEX idx_collection_card_id ON account_collection(card_id);
CREATE INDEX idx_collection_rarity ON account_collection(rarity);
```

### 7. wonder_pick_results

Track wonder pick attempts and results.

```sql
CREATE TABLE wonder_pick_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,
    activity_log_id INTEGER,

    -- Wonder pick details
    card_selected TEXT,                       -- Which card was picked
    card_rarity TEXT,                         -- Rarity of picked card
    success BOOLEAN DEFAULT 1,                -- Whether pick was successful

    -- Context
    energy_cost INTEGER DEFAULT 1,            -- How much energy spent
    was_free BOOLEAN DEFAULT 0,               -- Was it a free pick?

    picked_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
    FOREIGN KEY (activity_log_id) REFERENCES activity_log(id) ON DELETE SET NULL
);

CREATE INDEX idx_wonder_account ON wonder_pick_results(account_id);
CREATE INDEX idx_wonder_picked ON wonder_pick_results(picked_at);
```

### 8. mission_completion

Track mission/quest completion.

```sql
CREATE TABLE mission_completion (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id INTEGER NOT NULL,

    -- Mission details
    mission_type TEXT NOT NULL,               -- 'beginner', 'daily', 'solo_battle', 'limited_time'
    mission_name TEXT,

    -- Rewards
    shinedust_reward INTEGER DEFAULT 0,
    hourglasses_reward INTEGER DEFAULT 0,
    pokegold_reward INTEGER DEFAULT 0,
    pack_points_reward INTEGER DEFAULT 0,

    completed_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

CREATE INDEX idx_mission_account ON mission_completion(account_id);
CREATE INDEX idx_mission_type ON mission_completion(mission_type);
CREATE INDEX idx_mission_completed ON mission_completion(completed_at);
```

### 9. bot_statistics

Overall bot performance statistics.

```sql
CREATE TABLE bot_statistics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    -- Counts
    total_accounts INTEGER DEFAULT 0,
    active_accounts INTEGER DEFAULT 0,
    banned_accounts INTEGER DEFAULT 0,

    total_packs_opened INTEGER DEFAULT 0,
    total_wonder_picks INTEGER DEFAULT 0,
    total_god_packs INTEGER DEFAULT 0,

    -- Performance
    total_runtime_hours REAL DEFAULT 0,
    total_errors INTEGER DEFAULT 0,
    total_recoveries INTEGER DEFAULT 0,

    -- Timestamps
    stats_date DATE DEFAULT CURRENT_DATE,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(stats_date)
);

CREATE INDEX idx_stats_date ON bot_statistics(stats_date);
```

## Views

### Active Accounts Summary

```sql
CREATE VIEW v_active_accounts AS
SELECT
    a.id,
    a.username,
    a.device_account,
    a.account_level,
    a.packs_opened,
    a.shinedust,
    a.hourglasses,
    a.pokegold,
    a.last_used_at,
    COUNT(DISTINCT pr.id) as total_packs,
    COUNT(DISTINCT cp.id) as total_cards_pulled,
    COUNT(DISTINCT ac.id) as unique_cards_owned
FROM accounts a
LEFT JOIN pack_results pr ON a.id = pr.account_id
LEFT JOIN cards_pulled cp ON a.id = cp.account_id
LEFT JOIN account_collection ac ON a.id = ac.account_id
WHERE a.is_active = 1 AND a.is_banned = 0
GROUP BY a.id;
```

### Recent Activity Summary

```sql
CREATE VIEW v_recent_activity AS
SELECT
    al.id,
    a.username,
    al.activity_type,
    al.started_at,
    al.completed_at,
    al.duration_seconds,
    al.status,
    al.error_message
FROM activity_log al
JOIN accounts a ON al.account_id = a.id
ORDER BY al.started_at DESC
LIMIT 100;
```

### Pack Opening Statistics

```sql
CREATE VIEW v_pack_statistics AS
SELECT
    a.id as account_id,
    a.username,
    COUNT(pr.id) as total_packs_opened,
    SUM(CASE WHEN pr.is_god_pack = 1 THEN 1 ELSE 0 END) as god_packs,
    COUNT(DISTINCT pr.pack_type) as pack_types_opened,
    MAX(pr.opened_at) as last_pack_opened
FROM accounts a
LEFT JOIN pack_results pr ON a.id = pr.account_id
GROUP BY a.id;
```

## Migration Strategy

### Phase 1: Core Tables
1. accounts
2. activity_log
3. error_log

### Phase 2: Pack Tracking
4. pack_results
5. cards_pulled

### Phase 3: Advanced Features
6. account_collection
7. wonder_pick_results
8. mission_completion
9. bot_statistics

## Database Package Structure

```
internal/database/
├── database.go          # DB connection, initialization
├── migrations.go        # Schema migrations
├── models.go            # Go structs for tables
├── accounts.go          # Account CRUD operations
├── activity.go          # Activity logging
├── errors.go            # Error logging
├── packs.go             # Pack result tracking
├── cards.go             # Card tracking
└── statistics.go        # Statistics and reporting
```

## Integration Points

### Bot Startup
```go
db, err := database.Open("./data/pocket_tcg_bot.db")
db.RunMigrations()
```

### Account Loading
```go
account, err := db.GetOrCreateAccount(deviceAccount, devicePassword)
```

### Activity Tracking
```go
activityID := db.StartActivity(accountID, "wonder_pick", "DoWonderPick")
// ... do work ...
db.CompleteActivity(activityID, "completed", nil)
```

### Pack Opening
```go
packID := db.LogPackOpening(accountID, "genetic_apex", false, 5)
db.LogCardPulled(packID, accountID, "Charizard EX", "4_diamond", 0.95)
```

### Error Logging
```go
db.LogError(accountID, activityID, "ErrorPopup", "high", "Level up detected", screenState, true, "Dismissed popup", 1500)
```

## Benefits

- ✅ **Self-contained**: Single SQLite file, easy deployment
- ✅ **Queryable**: Rich querying for analysis and debugging
- ✅ **Historical**: Full history of all bot activities
- ✅ **Statistics**: Built-in views for reporting
- ✅ **Debugging**: Detailed error tracking
- ✅ **Collection tracking**: Know exactly what cards each account has
- ✅ **Performance**: Indices for fast lookups
- ✅ **Integrity**: Foreign keys ensure data consistency

## Future Enhancements

- Export to CSV for analysis
- Web dashboard for viewing statistics
- Discord webhooks for rare card pulls
- Automatic account rotation based on stamina recovery
- Ban detection heuristics based on error patterns
