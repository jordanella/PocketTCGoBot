# Orchestration ID & Account Checkout Implementation

## Problem Statement

Previously, there were two critical issues with account management:

### Issue 1: No Orchestration Context
The `routine_executions` table tracked routine execution per account using only:
- `account_id` + `routine_name` + `execution_status`

This created problems when:
1. Bot Group A starts, injects account X, routine begins (`execution_status='started'`)
2. Bot crashes/aborts without calling `CompleteAccount` or `MarkAccountFailed`
3. Account X stuck in `InUse` status in memory, but `execution_status='started'` in database
4. Bot Group B (hours/days later) creates new pool from same definition
5. Pool refresh query sees account X with `started` status from old crashed run
6. **Result**: Either incorrectly excludes account X, or includes it causing duplicate work

### Issue 2: No Global Account Mutex
- Each pool maintained its own in-memory `Account` struct with status
- Multiple pools with overlapping queries could inject the same account into different emulators simultaneously
- No database-backed source of truth for "which emulator instance currently has which account"
- Ungraceful shutdowns left no way to determine what was injected where

## Solution: Two-Tier Approach

### Part 1: Orchestration ID (Execution Context Tracking)
Add `orchestration_id` to `routine_executions` table to track distinct bot group execution contexts.

### Part 2: Account Checkout System (Global Mutex)
Add checkout tracking columns to `accounts` table as the **source of truth** for account-to-emulator assignments.

## Database Changes

### Migration 010: Orchestration ID
```sql
ALTER TABLE routine_executions ADD COLUMN orchestration_id TEXT;
CREATE INDEX idx_routine_exec_orchestration ON routine_executions(orchestration_id);
CREATE INDEX idx_routine_exec_orchestration_lookup ON routine_executions(
    orchestration_id,
    routine_name,
    execution_status,
    completed_at
);
```

### Migration 011: Account Checkout Tracking
```sql
-- Source of truth for account-to-instance mapping
ALTER TABLE accounts ADD COLUMN checked_out_to_instance INTEGER;
ALTER TABLE accounts ADD COLUMN checked_out_to_orchestration TEXT;
ALTER TABLE accounts ADD COLUMN checked_out_at DATETIME;

CREATE INDEX idx_accounts_checked_out_instance ON accounts(checked_out_to_instance);
CREATE INDEX idx_accounts_checked_out_orchestration ON accounts(checked_out_to_orchestration);
CREATE INDEX idx_accounts_checkout_lookup ON accounts(
    checked_out_to_orchestration,
    checked_out_to_instance
) WHERE checked_out_to_orchestration IS NOT NULL;
```

### Code Changes

1. **`RoutineExecution` struct** (`internal/database/routine_executions.go`):
   ```go
   type RoutineExecution struct {
       ...
       OrchestrationID  *string // UUID identifying this bot group execution context
       ...
   }
   ```

2. **`StartRoutineExecution` function signature**:
   ```go
   func StartRoutineExecution(db *sql.DB, accountID int64, routineName string, orchestrationID string, botInstance int) (int64, error)
   ```

3. **All SELECT queries** updated to include `orchestration_id`

### How It Works

#### Bot Group Startup

1. **Orchestrator creates unique UUID** for this execution (e.g., `550e8400-e29b-41d4-a716-446655440000`)
2. **UUID passed to all bot instances** in the group
3. **When routine starts**: `StartRoutineExecution(db, accountID, routineName, orchestrationID, botInstance)`
4. **Database records**:
   ```
   account_id: 123
   routine_name: "pack_opener"
   orchestration_id: "550e8400-e29b-41d4-a716-446655440000"
   execution_status: "started"
   ```

#### Pool Queries (Account Selection)

**Before** (WRONG):
```sql
SELECT * FROM accounts a
WHERE NOT EXISTS (
    SELECT 1 FROM routine_executions re
    WHERE re.account_id = a.id
    AND re.routine_name = 'pack_opener'
    AND re.execution_status IN ('started', 'completed')
)
```
This would exclude accounts with ANY `started` status, even from old crashed runs.

**After** (CORRECT):
```sql
SELECT * FROM accounts a
WHERE NOT EXISTS (
    SELECT 1 FROM routine_executions re
    WHERE re.account_id = a.id
    AND re.routine_name = 'pack_opener'
    AND re.orchestration_id = '550e8400-e29b-41d4-a716-446655440000'  -- Current group!
    AND re.execution_status IN ('started', 'completed')
)
```
This only considers executions from **this specific orchestration**, ignoring old crashed runs.

#### Lifecycle Example

**Bot Group A** (UUID: `aaa-111`):
```
10:00 AM - Account X injected → execution_status='started', orchestration_id='aaa-111'
10:05 AM - Bot crashes (no CompleteAccount called)
        → Account X remains: execution_status='started', orchestration_id='aaa-111'
```

**Bot Group B** (UUID: `bbb-222`) - Started 2 hours later:
```
12:00 PM - Pool refreshes from database
         → Query filters: orchestration_id='bbb-222'
         → Account X with orchestration_id='aaa-111' is IGNORED
         → Account X available for injection in new group
12:01 PM - Account X injected → execution_status='started', orchestration_id='bbb-222'
```

Database now has TWO rows for account X:
```
Row 1: account_id=X, routine='pack_opener', orchestration_id='aaa-111', status='started'  [STALE]
Row 2: account_id=X, routine='pack_opener', orchestration_id='bbb-222', status='started'  [ACTIVE]
```

### Benefits

1. **Isolation**: Each bot group only sees its own execution context
2. **No False Conflicts**: Old crashed runs don't block accounts
3. **Audit Trail**: Can see which specific orchestration processed which accounts
4. **Multi-Tenancy**: Multiple groups can run same routine simultaneously without conflict
5. **Historical Analysis**: Track which bot group runs were successful

## New Architecture Flow

### 1. Pool Population (Read-Only)
Account pools query the database and create an **in-memory queue** of available accounts. This list can overlap between pools - that's fine. The pool is just a queue builder, not a state manager.

### 2. Account Checkout (Mutex Control)
When a bot needs an account:
1. **Dequeue** from pool's in-memory channel
2. **Check database**: Is account already checked out? (`checked_out_to_orchestration IS NOT NULL`)
3. **If checked out to different orchestration**:
   - Check if that orchestration is still active
   - If active: Defer/requeue account, try next
   - If stale (>10min): Reclaim and checkout to current orchestration
4. **If available**: Atomically checkout via `CheckoutAccount(db, deviceAccount, orchestrationID, emulatorInstance)`
5. **Inject** into emulator

### 3. Emulator Startup Verification
When emulator instance starts:
1. Establish ADB connection
2. **Extract current account** from emulator
3. Call `VerifyAndUpdateAccountCheckout(db, orchestrationID, emulatorInstance, actualDeviceAccount)`
4. If mismatch: Update database to match reality (handles crashes/ungraceful shutdowns)

### 4. Account Release
When routine completes:
1. `CompleteAccount`/`MarkAccountFailed`: Update `routine_executions`, then call `ReleaseAccount(db, deviceAccount, orchestrationID)`
2. `ReturnAccount`: Call `ReleaseAccount()` to make available for retry
3. On orchestration shutdown: `ReleaseAllAccountsForOrchestration(db, orchestrationID)`

## Implementation Checklist

### Database Layer (Complete)
- [x] Add migration 010 for `orchestration_id` column
- [x] Add migration 011 for checkout tracking columns
- [x] Update `RoutineExecution` struct with `OrchestrationID` field
- [x] Update `StartRoutineExecution()` to accept `orchestrationID` parameter
- [x] Update all SELECT queries to include `orchestration_id`
- [x] Create `account_checkout.go` with mutex functions:
  - [x] `CheckoutAccount()` - Atomic checkout with conflict detection
  - [x] `ReleaseAccount()` - Release on completion
  - [x] `IsAccountCheckedOut()` - Check status
  - [x] `GetAccountsCheckedOutByOrchestration()` - List all checkouts
  - [x] `ReleaseAllAccountsForOrchestration()` - Cleanup on shutdown
  - [x] `GetCheckedOutAccountForInstance()` - What's on this emulator?
  - [x] `VerifyAndUpdateAccountCheckout()` - Sync DB with reality

### Orchestrator Layer (Complete)
- [x] Generate UUID on bot group startup
- [x] Store orchestration_id in BotGroup struct
- [x] Pass orchestration_id to Manager via `SetOrchestrationID()`
- [x] Pass orchestration_id to Bot instances via `SetOrchestrationID()`
- [x] Implement `OrchestrationID()` method in Bot
- [x] Implement `OrchestrationID()` and `SetOrchestrationID()` in Manager

### Pool Layer (TODO)
- [ ] Modify `GetNext()` to check database checkout status before returning account
- [ ] Implement deferral/retry logic for checked-out accounts
- [ ] Remove in-memory status tracking (use DB as source of truth)

### Action Layer (Complete)
- [x] Update `InjectNextAccount` to:
  - Check database checkout status before using account
  - Defer accounts checked out to different orchestrations
  - Call `CheckoutAccount()` before injection with conflict detection
  - Call `StartRoutineExecution()` with orchestrationID after successful injection
  - Release checkout on injection failure
- [x] Update `CompleteAccount` to call `ReleaseAccount()` after marking as used
- [x] Update `ReturnAccount` to call `ReleaseAccount()` when returning to pool
- [x] Update `MarkAccountFailed` to call `ReleaseAccount()` after marking as failed

### Emulator Layer (TODO)
- [ ] Add account extraction on emulator startup
- [ ] Call `VerifyAndUpdateAccountCheckout()` after ADB connection established
- [ ] Handle mismatch scenarios (log warnings, update DB)

### Orchestrator Shutdown (Complete)
- [x] Call `ReleaseAllAccountsForOrchestration()` on graceful shutdown in `StopGroup()`
- [x] Clear checked_out_to_orchestration column (while keeping checked_out_to_instance)
- [ ] Handle SIGTERM/SIGINT to ensure cleanup (TODO)

### Next Steps

1. **Generate UUID in Orchestrator**: Use `github.com/google/uuid` or similar
2. **Store in BotGroup**: Pass orchestration_id when creating bot group
3. **Access in Bot**: Make orchestration_id available via `BotInterface`
4. **Update Actions**: Modify account actions to pass orchestration_id
5. **Update Pool Queries**: Ensure SQL queries filter by orchestration_id

### Example Pool Query Update

**Old unified_pool.go query**:
```go
query := `
    SELECT * FROM accounts a
    WHERE a.pool_status = 'available'
    AND NOT EXISTS (
        SELECT 1 FROM routine_executions re
        WHERE re.account_id = a.id
        AND re.routine_name = ?
        AND re.execution_status = 'completed'
    )
`
```

**New with orchestration_id**:
```go
query := `
    SELECT * FROM accounts a
    WHERE a.pool_status = 'available'
    AND NOT EXISTS (
        SELECT 1 FROM routine_executions re
        WHERE re.account_id = a.id
        AND re.routine_name = ?
        AND re.orchestration_id = ?  -- Add this!
        AND re.execution_status IN ('started', 'completed')
    )
`
// Pass both routine_name AND orchestration_id as parameters
```

This ensures each bot group only considers its own execution context when selecting accounts.
