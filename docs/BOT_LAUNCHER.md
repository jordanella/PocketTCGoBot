# Bot Launcher GUI

The Bot Launcher provides a graphical interface for launching multiple bot instances with routine selection and automatic account injection via the coordinator.

## Features

- **Multi-Bot Configuration**: Launch 1-20 bot instances simultaneously
- **Routine Selection**: Choose different routines for each bot or set all to the same
- **Account Injection**: Automatic account assignment via coordinator
- **Shared Registries**: All bots share template and routine registries for memory efficiency
- **Centralized Management**: Start/stop all bots with one click
- **Real-time Status**: Monitor each bot's current state

## User Interface

### Main Components

1. **Number of Bots Input**: Specify how many bots to launch (1-20)
2. **Generate Button**: Creates configuration UI for each bot
3. **Set All Button**: Quickly assign the same routine to all bots
4. **Bot Configuration Cards**: Individual routine selection for each bot
5. **Launch/Stop Buttons**: Control all bots simultaneously
6. **Status Display**: Shows overall launcher status

### Bot Configuration Card

Each bot gets a configuration card with:
- **Bot Instance Number**: Identifies the bot (Bot 1, Bot 2, etc.)
- **Routine Dropdown**: Select which routine to run
- **Status Label**: Shows current state (Ready/Running/Stopped/Error)

## Workflow

### 1. Configure Bots

```
1. Enter number of bots (e.g., 6)
2. Click "Generate Bot Configs"
3. Individual config cards appear for each bot
```

### 2. Select Routines

**Option A: Individual Selection**
```
1. Click dropdown for each bot
2. Select desired routine from list
3. "<none>" means no routine (manual mode)
```

**Option B: Bulk Selection**
```
1. Click "Set All to..." button
2. Select routine from dialog
3. Click "Apply"
4. All bots now have same routine selected
```

### 3. Launch Bots

```
1. Click "Launch All Bots"
2. Coordinator creates shared registries
3. Each bot gets account injected
4. Bots start executing selected routines
5. Status updates to "Running: {routine_name}"
```

### 4. Monitor Execution

```
- Check status labels on each card
- View logs in Event Log tab
- Watch real-time progress
```

### 5. Stop Bots

```
1. Click "Stop All Bots"
2. Coordinator stops all bots gracefully
3. Shared registries cleaned up
4. Status updates to "Stopped"
```

## Architecture

### Component Integration

```
Bot Launcher GUI
    ↓
Creates Bot Manager (with shared registries)
    ↓
Creates Bot Coordinator (for account injection)
    ↓
For each bot:
    1. Manager creates bot instance
    2. Coordinator injects account
    3. Coordinator executes routine
```

### Coordinator Flow

```
BotRequest {
    Instance: 1
    RoutineName: "daily_missions"
    Bot: *bot.Bot
}
    ↓
Coordinator Queue
    ↓
Account Manager
    ↓
Account Injected
    ↓
Routine Executed
```

## Routine Selection

### Available Routines

The dropdown populates from `{FolderPath}/routines/*.yaml`:

```
routines/
├── startup.yaml           → "startup"
├── daily_missions.yaml    → "daily_missions"
├── pack_opening.yaml      → "pack_opening"
└── wonder_pick.yaml       → "wonder_pick"
```

### Special Option

**`<none>`**: Bot runs without a routine (manual/default behavior)

## Account Injection

The coordinator automatically:

1. **Loads Account**: Gets next eligible account from account manager
2. **Injects**: Assigns account to bot request
3. **Marks Used**: Prevents reuse during cooldown
4. **Logs**: Records account assignment

Account selection criteria (configurable):
- Pack count threshold
- Last used time
- Beginner status
- Custom filters

## Error Handling

### Bot Creation Failure

```
Error: Failed to create bot 3: ADB connection failed

Status: "Error: ADB connection failed"
Log: [ERROR] Bot 3: Failed to launch: ADB connection failed
```

### Routine Not Found

```
Error: Failed to get routine 'invalid_routine': routine file not found

Status: "Error: routine file not found"
Log: [ERROR] Bot 2: Failed to get routine 'invalid_routine'
```

### Account Injection Failure

```
Warning: Failed to inject account for bot 4: no eligible accounts

Bot continues without account injection
Log: [WARNING] Bot 4: No account injected
```

## Best Practices

### 1. Start Small

```
First time:
- Start with 1-2 bots
- Test routine execution
- Verify account injection
- Scale up to 6-8 bots
```

### 2. Routine Organization

```
Create focused routines:
✓ startup.yaml          - Initial setup
✓ daily_missions.yaml   - Daily tasks
✓ pack_opening.yaml     - Pack management
✓ cleanup.yaml          - End of session

Avoid monolithic routines:
✗ everything.yaml       - Too complex, hard to debug
```

### 3. Account Management

```
Before launching:
1. Ensure accounts directory has files
2. Set appropriate pack count thresholds
3. Configure cooldown periods
4. Test with 1 bot first
```

### 4. Monitoring

```
During execution:
- Check Event Log tab regularly
- Monitor status labels
- Watch for error patterns
- Adjust routines as needed
```

### 5. Graceful Shutdown

```
Always use "Stop All Bots" button:
✓ Allows routines to complete current step
✓ Properly saves state
✓ Cleans up shared registries
✓ Releases resources

Don't:
✗ Close window without stopping
✗ Kill process forcefully
```

## Configuration

### Routine Directory

```
Default: {FolderPath}/routines/

Override in config:
config.RoutinesPath = "C:/custom/routines"
```

### Account Directory

```
Default: {FolderPath}/accounts/

Override in config:
config.AccountsPath = "C:/custom/accounts"
```

### Number of Bots

```
Minimum: 1
Maximum: 20
Recommended: 6-8

Memory usage:
- 6 bots: ~1.5GB
- 8 bots: ~2GB
```

## Troubleshooting

### "No routines available"

```
Problem: Dropdown only shows "<none>"

Solutions:
1. Check routines/ folder exists
2. Verify .yaml files present
3. Ensure files have correct format
4. Click "Refresh" in Routines tab
```

### "Bot instance already running"

```
Problem: Can't launch bot, says already running

Solutions:
1. Click "Stop All Bots" first
2. Wait for all bots to fully stop
3. Check Event Log for stuck bots
4. Restart application if needed
```

### "Request queue is full"

```
Problem: Too many launch requests

Solutions:
1. Wait for current bots to start
2. Launch in smaller batches
3. Reduce number of bots
4. Check coordinator isn't stuck
```

### Memory Issues

```
Problem: System running slow with many bots

Solutions:
1. Reduce number of bots
2. Close unused templates
3. Clear routine cache
4. Verify shared registries enabled
5. Monitor RAM usage
```

## Advanced Usage

### Custom Account Filtering

```go
// In coordinator setup
coordinator.AccountManager.SetFilter(func(acc *Account) bool {
    return acc.PackCount >= 10 && acc.Metadata.BeginnerDone
})
```

### Pre-Launch Validation

```go
// Validate all routines before launching
for _, routine := range selectedRoutines {
    if err := manager.RoutineRegistry().Validate(routine); err != nil {
        log.Printf("Invalid routine %s: %v", routine, err)
        return
    }
}
```

### Batch Launching

```go
// Launch bots in groups
for batch := 1; batch <= 3; batch++ {
    launchBotsInRange(batch*2-1, batch*2)
    time.Sleep(30 * time.Second)
}
```

## Keyboard Shortcuts

```
Ctrl+L: Focus launcher tab
Ctrl+R: Refresh routine list
Ctrl+S: Stop all bots (when active)
Ctrl+G: Generate configs
```

## Integration with Other Tabs

### Event Log Tab
- Shows all bot launch/stop events
- Displays routine execution logs
- Records account injection status
- Error messages appear here

### Dashboard Tab
- Shows active bot count
- Displays MuMu instances
- Real-time bot status

### Routines Tab
- Browse available routines
- Validate routine structure
- Edit routines (external)
- Refresh to update launcher dropdown

### Accounts Tab
- View available accounts
- Check eligibility
- Monitor usage history
- Configure filters

### Database Tab
- Track bot activity
- View pack results
- Monitor errors
- Analyze performance
