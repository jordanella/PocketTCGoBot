# Health Monitoring Architecture - Comprehensive Design

## Current Problems

### 1. **No Continuous Health Monitoring for Running Bots**
- Health monitor only tracks instances during `waitForEmulatorReady()`
- Once bot launches and routine starts, `UntrackInstance()` is called
- Running bots have NO connection to health monitoring
- If emulator window closes, bot continues blindly until it fails

### 2. **Stale State Detection**
- Health monitor polls every 1 second but uses cached data from `DiscoverInstances()`
- If window closes between discoveries, health monitor doesn't know for 1+ seconds
- Multiple `DiscoverInstances()` calls create inconsistency

### 3. **No Failure Remediation**
- When instance becomes unhealthy, nothing happens automatically
- Bot group doesn't know about instance health status
- No automatic stop, restart, or recovery

## Proposed Architecture

### Phase 1: Continuous Health Monitoring (Current Focus)

#### 1.1 Health Monitor Lifecycle Changes

**Current Flow:**
```
LaunchGroup → acquireInstances → waitForEmulatorReady → TrackInstance()
→ Instance ready → UntrackInstance() → runBotRoutine() → [NO MONITORING]
```

**New Flow:**
```
LaunchGroup → acquireInstances → waitForEmulatorReady → TrackInstance()
→ Instance ready → runBotRoutine() → [KEEP TRACKING UNTIL BOT STOPS]
→ Bot completes/fails → UntrackInstance()
```

**Changes Required:**
- Remove `UntrackInstance()` from `waitForEmulatorReady()`
- Add `UntrackInstance()` to bot cleanup in `runBotRoutine()` after routine completes
- Health monitor continues polling tracked instances throughout bot lifetime

#### 1.2 Health Status Callback System

**Add to `OrchestratorHealthMonitor`:**
```go
type HealthStatusCallback func(instanceID int, isReady bool, previousReady bool)

// Subscribe to health status changes
func (ohm *OrchestratorHealthMonitor) OnHealthChange(instanceID int, callback HealthStatusCallback)
```

**Usage:**
```go
// When bot starts running, register callback
healthMonitor.OnHealthChange(instanceID, func(id int, isReady, wasReady bool) {
    if wasReady && !isReady {
        // Instance went from healthy → unhealthy
        fmt.Printf("ALERT: Instance %d became unhealthy!\n", id)
        group.HandleInstanceFailure(id)
    }
})
```

#### 1.3 Bot Group Failure Handling

**Add to `BotGroup`:**
```go
// HandleInstanceFailure is called when health monitor detects instance became unhealthy
func (g *BotGroup) HandleInstanceFailure(instanceID int) {
    fmt.Printf("[BotGroup '%s'] Instance %d health check failed\n", g.Name, instanceID)

    // Phase 1: Just stop the bot gracefully
    botInfo, exists := g.GetBotInfo(instanceID)
    if !exists {
        return // Bot already stopped
    }

    // Cancel the routine
    botInfo.Status = BotStatusStopping
    botInfo.routineCancel()

    // Cleanup will happen in runBotRoutine() when context cancels
}
```

### Phase 2: Automatic Recovery (Future)

When Phase 1 is stable, add recovery capabilities:

#### 2.1 Instance Lifecycle Recovery
```go
func (g *BotGroup) HandleInstanceFailure(instanceID int) {
    // Try to restart the instance
    if err := g.orchestrator.restartInstance(instanceID); err != nil {
        // Instance can't be restarted, stop the bot
        g.stopBotOnInstance(instanceID)
        return
    }

    // Wait for instance to be ready
    if err := g.orchestrator.waitForEmulatorReady(instanceID, timeout); err != nil {
        g.stopBotOnInstance(instanceID)
        return
    }

    // Reattach bot to the restarted instance
    g.reattachBot(instanceID)
}
```

#### 2.2 Routine Restart
```go
func (g *BotGroup) reattachBot(instanceID int) {
    botInfo, exists := g.GetBotInfo(instanceID)
    if !exists {
        return
    }

    // Create new context for restarted routine
    newCtx, newCancel := context.WithCancel(g.ctx)
    botInfo.routineCtx = newCtx
    botInfo.routineCancel = newCancel
    botInfo.Status = BotStatusRestarting

    // Restart the routine
    go g.orchestrator.runBotRoutine(g, botInfo, g.restartPolicy)
}
```

### Phase 3: Advanced Features (Future)

#### 3.1 Account State Preservation
- Save bot's account checkout before restart
- Restore account checkout after restart
- Update database to mark account as "in recovery"

#### 3.2 Routine State Checkpoints
- Allow routines to save progress checkpoints
- Resume from checkpoint after recovery
- Configurable checkpoint frequency

#### 3.3 Failure Pattern Detection
```go
type FailurePattern struct {
    InstanceID       int
    FailureCount     int
    LastFailureTime  time.Time
    FailureReasons   []string
}

// If instance fails 3+ times in 10 minutes, don't restart it
func (g *BotGroup) shouldAttemptRecovery(instanceID int) bool {
    pattern := g.getFailurePattern(instanceID)
    if pattern.FailureCount >= 3 && time.Since(pattern.LastFailureTime) < 10*time.Minute {
        return false // Too many failures, give up
    }
    return true
}
```

## Implementation Plan

### Step 1: Remove Premature UntrackInstance (IMMEDIATE)
**File:** `internal/bot/orchestrator_instances.go`
- Remove `UntrackInstance()` call from `waitForEmulatorReady()`
- Keep instance tracked after it becomes ready

### Step 2: Add Health Callbacks (IMMEDIATE)
**File:** `internal/bot/orchestrator_health.go`
- Add `OnHealthChange` callback registration
- Modify `checkAllInstances()` to call callbacks when state changes
- Thread-safe callback management

### Step 3: Register Callbacks in runBotRoutine (IMMEDIATE)
**File:** `internal/bot/orchestrator_launch.go`
- In `runBotRoutine()`, register health callback for the instance
- Callback cancels bot routine context when instance becomes unhealthy
- Add `UntrackInstance()` to cleanup after routine completes

### Step 4: Bot Group Failure Handler (IMMEDIATE)
**File:** `internal/bot/orchestrator.go`
- Add `HandleInstanceFailure()` method to `BotGroup`
- Gracefully stop bot when instance fails
- Log failure for future pattern analysis

### Step 5: Testing & Validation
- Test: Close emulator window while bot is running
- Expected: Bot detects failure within 1 second, stops gracefully
- Test: Kill ADB server while bot is running
- Expected: Bot detects ADB failure, stops gracefully

### Step 6: GUI Integration
**File:** `internal/gui/tabs/orchestration_v3.go`
- Status tab shows health status for each instance
- Visual indicator when instance is unhealthy (red/yellow)
- Show failure count and last failure reason

## Benefits of This Architecture

### Immediate Benefits (Phase 1)
1. **Proactive Failure Detection**: Know immediately when instance closes
2. **Graceful Degradation**: Stop bot cleanly instead of letting it crash
3. **Better Logging**: Clear failure reasons and timestamps
4. **User Visibility**: GUI shows real-time health status

### Future Benefits (Phases 2-3)
1. **Automatic Recovery**: Bots can recover from temporary failures
2. **Reduced Babysitting**: System handles common failures automatically
3. **Account Safety**: Accounts properly released even during failures
4. **Pattern Analysis**: Learn which instances/routines are problematic

## Open Questions for Discussion

1. **Health Check Frequency**: Currently 1 second - is this appropriate?
   - Too fast = CPU overhead
   - Too slow = delayed failure detection

2. **Failure Threshold**: How many consecutive failures before declaring unhealthy?
   - Current: 1 failure = unhealthy
   - Alternative: 3 consecutive failures in 5 seconds

3. **Recovery Strategy**: Should Phase 2 be configurable per-group?
   - Some groups: never restart (account safety)
   - Other groups: always try to recover (maximize uptime)

4. **Health Check Scope**: What constitutes "healthy"?
   - Window open + ADB connected (current)
   - Add: Can execute simple ADB command?
   - Add: Pokemon TCG app is running?
   - Add: Screen is not black?

5. **Concurrency Concerns**: Health monitor runs in background
   - How do we safely stop bots from health monitor thread?
   - Should we queue failure events instead of handling immediately?

## Next Steps

1. Review this architecture design
2. Decide on Phase 1 scope (what to implement now)
3. Implement Step 1-4 above
4. Test with manual emulator closing
5. Plan Phase 2 timeline
