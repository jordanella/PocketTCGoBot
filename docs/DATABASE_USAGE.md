# Database Usage Guide

## Overview

The PocketTCGo Bot uses a SQLite database to track all bot activities, account progress, pack openings, and card collections. This guide shows how to integrate the database into your bot workflows.

## Quick Start

### 1. Initialize Database

```go
import "jordanella.com/pocket-tcg-go/internal/database"

// Open database connection
db, err := database.Open("./data/pocket_tcg_bot.db")
if err != nil {
    log.Fatalf("Failed to open database: %v", err)
}
defer db.Close()

// Run all pending migrations
err = db.RunMigrations()
if err != nil {
    log.Fatalf("Failed to run migrations: %v", err)
}
```

### 2. Load or Create Account

```go
// Get or create an account from device credentials
account, err := db.GetOrCreateAccount("deviceAccount123", "devicePassword456")
if err != nil {
    log.Fatalf("Failed to get account: %v", err)
}

fmt.Printf("Using account ID: %d\n", account.ID)
```

### 3. Track Activity

```go
// Start an activity session
activityID, err := db.StartActivity(
    account.ID,
    "wonder_pick",      // Activity type
    "DoWonderPick",     // Routine name
    "v1.0.0",           // Bot version
)
if err != nil {
    log.Fatalf("Failed to start activity: %v", err)
}

// ... perform the activity ...

// Complete the activity
err = db.CompleteActivity(activityID)
if err != nil {
    log.Errorf("Failed to complete activity: %v", err)
}
```

### 4. Log Errors

```go
// Log an error during bot execution
stackTrace := "detailed stack trace here"
screenState := "HomeScreen"
templateName := "error_popup"
actionName := "ClickOpenPack"

errorID, err := db.LogError(
    &account.ID,        // Account ID (can be nil for global errors)
    &activityID,        // Activity log ID (optional)
    "ErrorPopup",       // Error type
    "high",             // Severity: critical, high, medium, low
    "Unexpected popup", // Error message
    &stackTrace,
    &screenState,
    &templateName,
    &actionName,
)

// Mark as recovered if you handle it
err = db.MarkErrorRecovered(errorID, "Dismissed popup", 1500)
```

### 5. Track Pack Openings

```go
// Log a pack opening
rarityBreakdown := map[string]int{
    "1_diamond": 3,
    "2_diamond": 1,
    "3_diamond": 1,
}

packName := "Genetic Apex"
packID, err := db.LogPackOpening(
    account.ID,
    &activityID,
    "genetic_apex",    // Pack type
    &packName,
    false,             // is_god_pack
    5,                 // card_count
    rarityBreakdown,
    5,                 // pack_points_earned
)

// Log each card pulled from the pack
cardName := "Charizard EX"
cardNumber := "006/165"
cardType := "pokemon"
confidence := 0.95

cardID, err := db.LogCardPulled(
    packID,
    account.ID,
    "charizard_ex_006", // Unique card ID
    &cardName,
    &cardNumber,
    "4_diamond",        // Rarity
    &cardType,
    true,               // is_full_art
    true,               // is_ex
    &confidence,        // detection_confidence
)

// The card is automatically added to account_collection!
```

## Integration Examples

### Complete Wonder Pick Workflow

```go
func DoWonderPickWithDB(bot *Bot, db *database.DB, account *database.Account) error {
    // Start activity tracking
    activityID, err := db.StartActivity(
        account.ID,
        "wonder_pick",
        "DoWonderPick",
        bot.Version,
    )
    if err != nil {
        return err
    }

    // Track completion/failure
    defer func() {
        if err != nil {
            db.FailActivity(activityID, err.Error())
        } else {
            db.CompleteActivity(activityID)
        }
    }()

    // Perform wonder pick
    selectedCard, rarity, err := bot.PerformWonderPick()
    if err != nil {
        return err
    }

    // Log the result
    wasFree := true
    _, err = db.LogWonderPick(
        account.ID,
        &activityID,
        &selectedCard,
        &rarity,
        true,  // success
        1,     // energy_cost
        wasFree,
    )

    // Update account stats
    err = db.UpdateAccountStats(
        account.ID,
        account.PacksOpened,
        account.WonderPicksDone + 1,
        account.AccountLevel,
    )

    return err
}
```

### Complete Pack Opening Workflow

```go
func OpenPackWithDB(bot *Bot, db *database.DB, account *database.Account) error {
    activityID, err := db.StartActivity(
        account.ID,
        "pack_opening",
        "OpenPack",
        bot.Version,
    )
    if err != nil {
        return err
    }

    defer func() {
        if err != nil {
            db.FailActivity(activityID, err.Error())
        } else {
            db.CompleteActivity(activityID)
        }
    }()

    // Open pack and detect cards
    packResult, err := bot.OpenPack()
    if err != nil {
        return err
    }

    // Calculate rarity breakdown
    rarityBreakdown := make(map[string]int)
    for _, card := range packResult.Cards {
        rarityBreakdown[card.Rarity]++
    }

    // Log pack opening
    packID, err := db.LogPackOpening(
        account.ID,
        &activityID,
        packResult.PackType,
        &packResult.PackName,
        packResult.IsGodPack,
        len(packResult.Cards),
        rarityBreakdown,
        packResult.PointsEarned,
    )
    if err != nil {
        return err
    }

    // Log each card
    for _, card := range packResult.Cards {
        _, err = db.LogCardPulled(
            packID,
            account.ID,
            card.ID,
            &card.Name,
            &card.Number,
            card.Rarity,
            &card.Type,
            card.IsFullArt,
            card.IsEx,
            &card.Confidence,
        )
        if err != nil {
            log.Errorf("Failed to log card: %v", err)
        }
    }

    // Update account resources and stats
    err = db.UpdateAccountResources(
        account.ID,
        account.Shinedust + packResult.ShinedustEarned,
        account.Hourglasses,
        account.Pokegold,
        account.PackPoints + packResult.PointsEarned,
    )
    if err != nil {
        return err
    }

    err = db.UpdateAccountStats(
        account.ID,
        account.PacksOpened + 1,
        account.WonderPicksDone,
        account.AccountLevel,
    )

    return err
}
```

### Error Handling with Database

```go
func PerformActionWithErrorTracking(bot *Bot, db *database.DB, account *database.Account) error {
    activityID, _ := db.StartActivity(account.ID, "custom_action", "CustomRoutine", bot.Version)

    defer func() {
        if r := recover(); r != nil {
            // Log panic as critical error
            errMsg := fmt.Sprintf("Panic recovered: %v", r)
            stackTrace := string(debug.Stack())

            db.LogError(
                &account.ID,
                &activityID,
                "ErrorPanic",
                "critical",
                errMsg,
                &stackTrace,
                nil, nil, nil,
            )

            db.FailActivity(activityID, errMsg)
        }
    }()

    // Your bot logic here
    // ...

    return nil
}
```

## Querying Data

### Get Account Statistics

```go
// Get pack statistics for an account
stats, err := db.GetPackStatistics(account.ID)
if err != nil {
    log.Errorf("Failed to get stats: %v", err)
} else {
    fmt.Printf("Total packs: %d\n", stats.TotalPacksOpened)
    fmt.Printf("God packs: %d\n", stats.GodPacks)
}

// Get account collection
collection, err := db.GetAccountCollection(account.ID)
if err != nil {
    log.Errorf("Failed to get collection: %v", err)
} else {
    fmt.Printf("You own %d unique cards\n", len(collection))
    for _, item := range collection {
        fmt.Printf("- %s x%d (%s)\n", item.CardName, item.Quantity, item.Rarity)
    }
}

// Get rarity distribution
distribution, err := db.GetRarityDistribution(account.ID)
fmt.Printf("Rarity breakdown:\n")
for rarity, count := range distribution {
    fmt.Printf("  %s: %d\n", rarity, count)
}
```

### Get Recent Activity

```go
// Get recent activities for an account
activities, err := db.GetRecentActivityForAccount(account.ID, 10)
for _, activity := range activities {
    fmt.Printf("%s: %s (%s)\n",
        activity.StartedAt.Format("2006-01-02 15:04"),
        activity.ActivityType,
        activity.Status,
    )
}

// Get all running activities (across all accounts)
running, err := db.GetRunningActivities()
fmt.Printf("Currently running: %d activities\n", len(running))
```

### Error Analysis

```go
// Get unrecovered errors
errors, err := db.GetUnrecoveredErrors(50)
fmt.Printf("Found %d unrecovered errors\n", len(errors))

// Get error statistics by type
now := time.Now()
startDate := now.AddDate(0, 0, -7) // Last 7 days
errorStats, err := db.GetErrorStatsByType(&account.ID, startDate, now)
for errorType, count := range errorStats {
    fmt.Printf("%s: %d occurrences\n", errorType, count)
}

// Get recovery rate
recoveryRate, err := db.GetRecoveryRate(&account.ID, startDate, now)
fmt.Printf("Recovery rate: %.1f%%\n", recoveryRate)
```

## Database Maintenance

### Backup

```go
// Create a backup of the database
backupPath := fmt.Sprintf("./backups/bot_%s.db", time.Now().Format("2006-01-02"))
err = db.Backup(backupPath)
if err != nil {
    log.Errorf("Backup failed: %v", err)
}
```

### Cleanup Old Data

```go
// Delete activities older than 30 days
thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
deleted, err := db.DeleteOldActivities(thirtyDaysAgo)
fmt.Printf("Deleted %d old activities\n", deleted)

// Delete errors older than 30 days
deleted, err = db.DeleteOldErrors(thirtyDaysAgo)
fmt.Printf("Deleted %d old errors\n", deleted)
```

### Optimize Database

```go
// Run VACUUM to reclaim space and optimize
err = db.Vacuum()
if err != nil {
    log.Errorf("Vacuum failed: %v", err)
}
```

## Best Practices

1. **Always use transactions**: All write operations use `ExecTx()` internally for atomicity
2. **Log activities**: Start an activity at the beginning of each major routine
3. **Track errors**: Log all errors with appropriate severity levels
4. **Mark recovery**: If you handle an error, mark it as recovered with `MarkErrorRecovered()`
5. **Update resources**: Keep account resources in sync after each action
6. **Regular backups**: Schedule daily backups of the database
7. **Clean old data**: Periodically delete old activities and errors to keep database size manageable

## Schema Version Management

The database automatically tracks its schema version. When you call `RunMigrations()`, it:
- Checks the current schema version
- Runs any pending migrations in order
- Each migration runs in a transaction (all-or-nothing)

You never need to manually manage schema versions - just call `RunMigrations()` on startup.

## Performance Tips

1. **Use views for complex queries**: The database includes pre-built views like `v_active_accounts` and `v_pack_statistics`
2. **Batch operations**: Use transactions for multiple related writes
3. **Limit queries**: Always specify a limit when querying potentially large result sets
4. **Indices are pre-configured**: The migrations create all necessary indices automatically

## Troubleshooting

### Database Locked

If you get "database is locked" errors:
- Ensure you're not opening multiple connections
- Use the same `*DB` instance throughout your application
- Keep transactions short

### Foreign Key Violations

If you get foreign key errors:
- Ensure parent records exist before creating child records
- Use `GetOrCreateAccount()` instead of `CreateAccount()` to avoid duplicates

### Migration Failures

If a migration fails:
- Check the error message
- Manually inspect the database with SQLite tools
- Migrations are transactional, so partial failures rollback automatically
