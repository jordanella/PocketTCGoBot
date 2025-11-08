# Health Monitoring System

## Overview

The health monitoring system provides comprehensive runtime health checks for bot instances, detecting issues like missing windows, frozen screens, ADB connection problems, and device unresponsiveness. It complements the routine-defined sentry system by providing intrinsic, always-on monitoring that is independent of routine execution.

## Architecture

### Components

1. **HealthChecker** ([internal/monitor/health_checker.go](../internal/monitor/health_checker.go))
   - Performs periodic health checks on bot instances
   - Tracks consecutive failures and frozen screen detection
   - Triggers unhealthy callbacks when issues are detected

2. **RecoveryConfig** ([internal/bot/bot.go](../internal/bot/bot.go))
   - Configures automatic recovery actions for different issue types
   - Tracks recovery attempts per issue type
   - Enforces maximum recovery attempt limits

3. **Recovery Actions** ([internal/bot/bot.go](../internal/bot/bot.go))
   - Executes appropriate recovery strategies based on health check failures
   - Provides multiple recovery action types (log, pause, restart, reconnect, etc.)

## Health Checks

### 1. ADB Connection Check
**Method:** `CheckADBConnection()`
**Purpose:** Verifies ADB connection is alive and responsive
**Implementation:** Pings device using `getprop` command
**Failure Action:** Configured via `RecoveryConfig.ADBConnectionLost` (default: ReconnectADB)

### 2. Instance Window Check
**Method:** `CheckInstanceWindow()`
**Purpose:** Verifies the emulator/device window exists and is accessible
**Implementation:** Checks device serial and queries Android version
**Failure Action:** Configured via `RecoveryConfig.InstanceWindowMissing` (default: Stop)

### 3. Device Responsiveness Check
**Method:** `CheckDeviceResponsive()`
**Purpose:** Ensures device is responding to commands in a timely manner
**Implementation:** Executes test command with timeout
**Failure Action:** Configured via `RecoveryConfig.DeviceUnresponsive` (default: RestartApp)

### 4. Screen Frozen Detection
**Method:** `CheckScreenFrozen()`
**Purpose:** Detects if the screen appears frozen (no focus changes)
**Implementation:** Monitors `dumpsys window` focus output for changes
**Failure Action:** Configured via `RecoveryConfig.ScreenFrozen` (default: RestartApp)
**Threshold:** Default 5 consecutive checks (configurable via `WithFrozenThreshold()`)

### 5. Process Running Check
**Method:** `CheckProcessRunning()`
**Purpose:** Verifies target application process is running
**Implementation:** Checks for process via `ps` command
**Note:** Currently stubbed, can be enhanced with specific package checks

## Recovery Actions

### Available Actions

| Action | Description | Use Case |
|--------|-------------|----------|
| `RecoveryActionNone` | Do nothing | For monitoring-only mode |
| `RecoveryActionLog` | Log the issue only | Minor issues, information gathering |
| `RecoveryActionPause` | Pause the bot | Manual intervention needed |
| `RecoveryActionRestart` | Restart last routine | Transient errors, bot stuck |
| `RecoveryActionReconnectADB` | Reconnect ADB | ADB connection lost |
| `RecoveryActionRestartApp` | Restart target app | App frozen or unresponsive |
| `RecoveryActionStop` | Stop the bot | Fatal errors |

### Default Recovery Configuration

```go
RecoveryConfig{
    ADBConnectionLost:     RecoveryActionReconnectADB,
    InstanceWindowMissing: RecoveryActionStop,
    DeviceUnresponsive:    RecoveryActionRestartApp,
    ScreenFrozen:          RecoveryActionRestartApp,
    BotStuck:              RecoveryActionRestart,
    MaxRecoveryAttempts:   3,
}
```

## Usage

### Basic Setup (Automatic)

Health monitoring is automatically initialized for all bot instances:

```go
bot, err := bot.New(instance, config)
if err != nil {
    return err
}

if err := bot.Initialize(); err != nil {
    return err
}
// Health monitoring is now active with default configuration
```

### Custom Configuration

```go
// Create bot
bot, err := bot.New(instance, config)

// Customize recovery behavior
recoveryConfig := bot.RecoveryConfig{
    ADBConnectionLost:     bot.RecoveryActionReconnectADB,
    InstanceWindowMissing: bot.RecoveryActionStop,
    DeviceUnresponsive:    bot.RecoveryActionRestartApp,
    ScreenFrozen:          bot.RecoveryActionLog, // Just log, don't recover
    BotStuck:              bot.RecoveryActionRestart,
    MaxRecoveryAttempts:   5, // Allow more attempts
}

// Apply custom config (must be done before Initialize)
bot.recoveryConfig = recoveryConfig

if err := bot.Initialize(); err != nil {
    return err
}
```

### Custom Unhealthy Callback

```go
bot.SetUnhealthyAction(func() {
    // Custom action when bot becomes unhealthy
    log.Printf("Bot %d is unhealthy, notifying admin...", bot.Instance())
    // Send notification, update dashboard, etc.
})
```

## Health Check Behavior

### Check Interval
- Default: 10 seconds (configurable via `WithCheckInterval()`)
- Runs continuously in background goroutine
- Stops automatically when bot shuts down

### Failure Tracking
- **Consecutive Failures**: Counted per health check type
- **Failure Threshold**: Default 3 failures before triggering recovery
- **Reset on Success**: Failure count resets when checks pass

### Recovery Attempt Tracking
- Tracked per issue type (e.g., "adb_connection_lost", "screen_frozen")
- Independent counters for each issue type
- Reset to 0 on successful recovery
- Bot stops when `MaxRecoveryAttempts` exceeded

## Health Check Flow

```
┌─────────────────────────────────────────┐
│     Health Checker (every 10s)          │
└──────────────┬──────────────────────────┘
               │
               ▼
    ┌──────────────────────┐
    │  Check ADB Connection │
    └──────────┬────────────┘
               │ OK
               ▼
    ┌──────────────────────┐
    │ Check Instance Window │
    └──────────┬────────────┘
               │ OK
               ▼
    ┌──────────────────────┐
    │Check Device Responsive│
    └──────────┬────────────┘
               │ OK
               ▼
    ┌──────────────────────┐
    │  Check Screen Frozen  │
    └──────────┬────────────┘
               │ OK
               ▼
    ┌──────────────────────┐
    │   All Checks Pass     │
    │  Reset Failure Count  │
    └───────────────────────┘

         (On Failure)
               │
               ▼
    ┌──────────────────────┐
    │ Increment Failures    │
    │ Trigger Unhealthy     │
    │     Callback          │
    └──────────┬────────────┘
               │
               ▼
    ┌──────────────────────┐
    │ Execute Recovery      │
    │      Action           │
    └──────────┬────────────┘
               │
        ┌──────┴────────┐
        │               │
        ▼               ▼
  [Success]       [Max Attempts]
  Reset Count      Stop Bot
```

## Recovery Action Implementation

### RecoveryActionReconnectADB
1. Disconnects current ADB connection
2. Attempts to reconnect to instance
3. On success: Resets recovery attempts
4. On failure: Stops bot

### RecoveryActionRestartApp
1. Force stops Pokemon TCG Pocket app (`am force-stop jp.pokemon.pokemontcgp`)
2. Waits 2 seconds
3. Restarts app using `monkey` command
4. On success: Resets recovery attempts
5. On failure: Stops bot

### RecoveryActionRestart
1. Stops current routine execution
2. Relies on Manager's `RestartBot()` to restart with last routine
3. Note: Manager must handle the restart externally

## Integration with Sentry System

The health monitoring system is **complementary** to the routine-defined sentry system:

### Health Monitoring (Intrinsic)
- **Always Active**: Runs regardless of routine execution
- **System-Level**: Monitors bot infrastructure (ADB, windows, devices)
- **Automatic**: No configuration in routine YAML required
- **Recovery Actions**: Automatically attempts recovery
- **Examples**: ADB disconnect, window missing, device frozen

### Sentry System (Routine-Defined)
- **Routine-Specific**: Defined in routine YAML files
- **Domain-Level**: Monitors game state and domain logic
- **Configurable**: Frequency, severity, actions customizable per routine
- **Error Detection**: Detects domain-specific errors (popups, battles, etc.)
- **Examples**: Ad popups, error screens, battle completion

Both systems work together to provide comprehensive monitoring:
- Health monitoring handles infrastructure issues
- Sentries handle domain/game-specific issues

## Monitoring and Debugging

### Log Output

Health check failures are logged with instance number and reason:

```
Bot 2: Health check failed - adb_connection_lost: device not responding
Bot 2: Executing recovery action 'reconnect_adb' for 'adb_connection_lost' (attempt 1/3)
Bot 2: Attempting to reconnect ADB
Bot 2: ADB reconnected successfully
```

### Health Status Query

Get comprehensive health metrics:

```go
status := healthChecker.GetHealthStatus()
fmt.Printf("Last Check: %v\n", status.LastCheckTime)
fmt.Printf("Last Activity: %v\n", status.LastActivityTime)
fmt.Printf("Consecutive Failures: %d\n", status.ConsecutiveFailures)
fmt.Printf("Frozen Check Count: %d\n", status.FrozenCheckCount)
```

## Configuration Reference

### HealthChecker Configuration

```go
healthChecker := monitor.NewHealthChecker(bot).
    WithCheckInterval(10 * time.Second).        // How often to check
    WithFailureThreshold(3).                    // Failures before action
    WithFrozenThreshold(5).                     // Frozen checks before action
    WithUnhealthyCallback(func(reason string, err error) {
        // Custom callback
    })
```

### RecoveryConfig Fields

```go
type RecoveryConfig struct {
    ADBConnectionLost     RecoveryAction  // Action for ADB disconnect
    InstanceWindowMissing RecoveryAction  // Action for missing window
    DeviceUnresponsive    RecoveryAction  // Action for unresponsive device
    ScreenFrozen          RecoveryAction  // Action for frozen screen
    BotStuck              RecoveryAction  // Action for stuck bot
    MaxRecoveryAttempts   int             // Max attempts before stopping
}
```

## Best Practices

1. **Check Interval**: 10-30 seconds is recommended
   - Too frequent: Adds overhead
   - Too infrequent: Slow detection

2. **Failure Threshold**: 2-5 failures recommended
   - Prevents false positives from transient issues
   - Balances quick detection with stability

3. **Frozen Threshold**: 3-10 checks recommended
   - Depends on your routine's typical behavior
   - Higher for routines with long waiting periods

4. **Max Recovery Attempts**: 3-5 recommended
   - Prevents infinite restart loops
   - Allows time to resolve transient issues

5. **Recovery Actions**: Choose based on issue severity
   - Transient issues: Restart or RestartApp
   - Fatal issues: Stop
   - Connection issues: ReconnectADB

## Troubleshooting

### Health checks failing frequently
- Check ADB connection stability
- Verify emulator instances are stable
- Increase failure threshold to reduce false positives
- Review device resource usage (CPU, memory)

### Recovery actions not working
- Check log output for error messages
- Verify ADB commands execute successfully
- Ensure app package name is correct (jp.pokemon.pokemontcgp)
- Check recovery attempt limits aren't exceeded

### Screen frozen detection triggering incorrectly
- Increase frozen threshold for routines with long waits
- Verify `dumpsys window` command works on device
- Check if focus actually changes during routine execution

## Future Enhancements

Potential improvements for future versions:

1. **Process Monitoring**: Implement full `CheckProcessRunning()` with package name
2. **Memory/CPU Monitoring**: Detect resource exhaustion
3. **Network Connectivity**: Monitor network connection status
4. **Performance Metrics**: Track check execution times
5. **Configurable Package Name**: Support different target apps
6. **Recovery History**: Track recovery attempts over time
7. **Health Dashboard**: GUI display of health metrics
8. **Alert System**: Notifications for critical health issues

## Related Documentation

- [Sentry System](SENTRIES.md) - Routine-defined error monitoring
- [Bot Lifecycle](../ROADMAP_V2.md) - Bot instance management
- [Manager Documentation](MANAGER.md) - Multi-instance coordination

---

**Last Updated:** 2025-11-08
**Version:** v0.2.0
**Status:** Production Ready
