# Manager Groups - Bot Orchestration Flow

## Overview
Manager Groups provide a comprehensive system for organizing, configuring, and launching multiple bot instances with coordinated emulator and account management.

---

## Group Configuration

### 1. Group Properties
- **Name**: User-defined identifier for the group
- **Emulator Instances**: List of emulator instance IDs that can be used by this group
- **Bot Count**: Number of concurrent bots to run (pulls from emulator pool as needed)
- **Routine**: The YAML routine file to execute
- **Routine Configuration**: Variable overrides and settings for the routine
- **Account Pool**: Optional AccountPool for serving accounts to bots

### 2. Group Management
Users can:
- Create new groups
- Add/remove emulator instances from a group's available pool
- Modify bot count (scale up/down)
- Change routine assignment
- Configure routine parameters
- Assign/change account pool

---

## Launch Flow

### Phase 1: Pre-Launch Validation

#### Step 1.1: Routine Validation
```
✓ Check if routine file exists
✓ Parse routine YAML
✓ Validate all actions exist in registry
✓ Validate all templates referenced exist
✓ Validate routine configuration/overrides
✓ Check for syntax errors
```

**On Failure**: Display error to user, prevent launch

#### Step 1.2: Emulator Instance Assessment
For the requested bot count, assess available emulator instances:

```
For each emulator instance in group's pool:
  1. Check if emulator window is running (window detection)

  2. If RUNNING:
     a. Check if assigned to another group's running bots

     b. If OCCUPIED by another group:
        - Show warning dialog:
          "Emulator instance {id} is in use by group '{other_group_name}'"

          Options:
          [Cancel Other Group] - Stop the other group's bot on this instance
          [Skip This Instance] - Try next available instance
          [Cancel Launch]      - Abort this group's launch

     c. If FREE:
        - Add to available pool for this launch

  3. If NOT RUNNING:
     a. Launch emulator instance
     b. Wait for emulator to be ready
     c. Add to available pool for this launch

Continue until:
  - We have enough emulators for the bot count, OR
  - We've exhausted all instances in the group's pool
```

**Example Scenario**:
- Group wants 3 bots
- Group has instances [0, 1, 2, 3, 4] assigned
- Instance 0: Not running → Launch it
- Instance 1: Running, occupied by Group A → User chooses "Skip"
- Instance 2: Running, free → Use it
- Instance 3: Running, free → Use it
- Result: 3 instances available (0, 2, 3)

**On Insufficient Instances**:
- Warn user: "Only {available} of {requested} instances available"
- Options: [Launch with {available}] [Cancel]

---

### Phase 2: Bot Launch

#### Step 2.1: Staggered Launch
```
available_instances = [list of ready emulator instances]
stagger_delay = global_config.StaggerDelay (e.g., 5 seconds)

for i in range(bot_count):
  instance_id = available_instances[i]

  # Create bot with this instance
  bot = manager.CreateBot(instance_id)

  # Launch bot routine in background goroutine
  go func(bot, instance_id, routine_name):
    # Execute routine with restart policy
    manager.ExecuteWithRestart(instance_id, routine_name, restart_policy)

  # Stagger next launch
  if i < bot_count - 1:
    time.Sleep(stagger_delay)
```

#### Step 2.2: Runtime Tracking
- Track which emulator instances are in use by which group
- Update group status: "Running (3/3 bots active)"
- Allow stopping individual bots or entire group
- Monitor for bot failures and handle per restart policy

---

## Architecture Components

### Current State
```
ManagerGroupsTab
├── templateRegistry (global)
├── routineRegistry (global)
└── groups (map[string]*ManagerGroup)
    └── ManagerGroup
        ├── Name
        ├── Manager (*bot.Manager)
        ├── RoutineName
        ├── InstanceIDs ([]int)           ← Currently just a list
        ├── AccountPool
        └── running (bool)
```

### Required Changes

#### 1. ManagerGroup Structure
```go
type ManagerGroup struct {
    Name         string
    Manager      *bot.Manager
    RoutineName  string

    // NEW: Emulator instance management
    AvailableInstances []int              // Pool of instances this group can use
    ActiveBots         map[int]*BotInfo   // Currently running bots (key = instance ID)
    RequestedBotCount  int                // How many bots user wants running

    // NEW: Routine configuration
    RoutineConfig      map[string]string  // Variable overrides for routine

    // Account pool (existing)
    AccountsPath       string
    AccountPool        accountpool.AccountPool
    PoolConfig         accountpool.PoolConfig

    // UI tracking
    card               *fyne.Container
    statusLabel        *widget.Label
    startBtn           *widget.Button
    stopBtn            *widget.Button
    running            bool
}

type BotInfo struct {
    Bot          *bot.Bot
    InstanceID   int
    StartedAt    time.Time
    Status       string  // "running", "stopped", "failed"
    RoutineCtx   context.Context
    CancelFunc   context.CancelFunc
}
```

#### 2. Global Emulator Instance Registry
Track which instances are in use across all groups:

```go
type ManagerGroupsTab struct {
    // ... existing fields ...

    // NEW: Global instance tracking
    instanceRegistry   map[int]*InstanceAssignment
    instanceRegistryMu sync.RWMutex
}

type InstanceAssignment struct {
    InstanceID   int
    GroupName    string
    BotInstance  int
    AssignedAt   time.Time
    IsRunning    bool
}
```

#### 3. Emulator Manager Integration
```go
type ManagerGroup struct {
    // ... existing fields ...

    // NEW: Emulator management
    emulatorManager *emulator.Manager
}

// Methods needed:
- CheckEmulatorRunning(instanceID int) bool
- LaunchEmulator(instanceID int) error
- WaitForEmulatorReady(instanceID int, timeout time.Duration) error
```

---

## Implementation Roadmap

### Milestone 1: Emulator Instance Management
**Files to modify:**
- [x] `internal/gui/manager_groups.go`

**Tasks:**
1. [ ] Update `ManagerGroup` struct with new fields
2. [ ] Add global instance registry to `ManagerGroupsTab`
3. [ ] Implement instance conflict detection
4. [ ] Create conflict resolution dialog
5. [ ] Add emulator launch capability
6. [ ] Implement emulator ready detection

**New Methods:**
```go
// Instance management
func (t *ManagerGroupsTab) checkInstanceAvailability(instanceID int, requestingGroup string) (bool, string)
func (t *ManagerGroupsTab) reserveInstance(instanceID int, groupName string, botID int) error
func (t *ManagerGroupsTab) releaseInstance(instanceID int) error

// Emulator operations
func (t *ManagerGroupsTab) isEmulatorRunning(instanceID int) bool
func (t *ManagerGroupsTab) launchEmulator(instanceID int) error
func (t *ManagerGroupsTab) waitForEmulatorReady(instanceID int, timeout time.Duration) error
```

---

### Milestone 2: Routine Validation
**Files to modify:**
- `internal/actions/routine_registry.go`
- `internal/gui/manager_groups.go`

**Tasks:**
1. [ ] Add routine validation method to RoutineRegistry
2. [ ] Implement pre-launch routine validation
3. [ ] Create validation error display dialog
4. [ ] Add template existence checks
5. [ ] Validate variable references

**New Methods:**
```go
// In RoutineRegistry
func (r *RoutineRegistry) ValidateRoutine(name string) error

// In ManagerGroupsTab
func (t *ManagerGroupsTab) validateRoutineForLaunch(routineName string, config map[string]string) error
func (t *ManagerGroupsTab) showValidationError(err error)
```

---

### Milestone 3: Routine Configuration UI
**Files to modify:**
- `internal/gui/manager_groups.go`

**Tasks:**
1. [ ] Add routine configuration editor to group dialog
2. [ ] Support variable overrides (key-value pairs)
3. [ ] Show available variables from routine metadata
4. [ ] Persist routine config with group
5. [ ] Apply config when executing routine

**UI Components:**
```go
// Configuration editor in group dialog
- Variable list showing routine's configurable vars
- Key-Value entry fields for overrides
- Default value display
- Save/Apply buttons
```

---

### Milestone 4: Staggered Launch System
**Files to modify:**
- `internal/gui/manager_groups.go`
- `internal/bot/config.go` (add stagger delay setting)

**Tasks:**
1. [ ] Add stagger delay to global config
2. [ ] Implement staggered bot launch
3. [ ] Update startGroup() to use new launch flow
4. [ ] Add progress indication during launch
5. [ ] Handle launch failures gracefully

**New Methods:**
```go
func (t *ManagerGroupsTab) launchGroupWithValidation(group *ManagerGroup) error
func (t *ManagerGroupsTab) acquireEmulatorInstances(group *ManagerGroup, count int) ([]int, error)
func (t *ManagerGroupsTab) launchBotsStaggered(group *ManagerGroup, instances []int) error
```

---

### Milestone 5: Bot Count Management
**Files to modify:**
- `internal/gui/manager_groups.go`

**Tasks:**
1. [ ] Add bot count selector to group config
2. [ ] Implement scale up (add more bots while running)
3. [ ] Implement scale down (stop some bots while running)
4. [ ] Update UI to show active vs requested count
5. [ ] Handle dynamic instance allocation

**UI Updates:**
- Group card shows: "Running (3/5 bots active)"
- Buttons: [Scale Up] [Scale Down] [Stop All]
- Live bot status list per group

---

## Configuration Flow Diagram

```
User Creates Group
    ↓
[Group Dialog]
├── Name: "Premium Farmers"
├── Available Instances: [0, 1, 2, 3, 4]  ← Select which instances this group can use
├── Bot Count: 3                           ← How many concurrent bots
├── Routine: "farm_premium_packs.yaml"
├── Routine Config:                        ← Variable overrides
│   ├── packs_to_open: "10"
│   ├── daily_missions: "true"
│   └── stop_on_god_pack: "true"
└── Account Pool: "./accounts/premium"
    ↓
[Save Group]
    ↓
[Launch Button Pressed]
    ↓
Validation Phase
├── Validate Routine
├── Check Emulator Instances
│   ├── Instance 0: Not Running → Launch
│   ├── Instance 1: Running, Occupied → Skip
│   ├── Instance 2: Running, Free → Use
│   └── Instance 3: Running, Free → Use
└── Result: 3 instances ready [0, 2, 3]
    ↓
Launch Phase
├── Bot 1 on Instance 0 → Start routine
├── Wait 5 seconds (stagger)
├── Bot 2 on Instance 2 → Start routine
├── Wait 5 seconds (stagger)
└── Bot 3 on Instance 3 → Start routine
    ↓
[Running State]
├── Monitor bot status
├── Allow scaling
└── Handle failures per restart policy
```

---

## Error Handling

### Routine Validation Failures
- Show detailed error message
- Highlight problematic action/template
- Prevent launch until fixed

### Insufficient Emulator Instances
- Show warning with available count
- Offer partial launch option
- Suggest freeing instances from other groups

### Emulator Launch Failures
- Retry with exponential backoff
- Show error after max retries
- Skip failed instance, try next in pool

### Bot Runtime Failures
- Handled by existing restart policy
- Log to event log
- Update group status display

---

## Future Enhancements

1. **Instance Affinity**: Remember which instances work best for certain routines
2. **Load Balancing**: Automatically distribute bots across healthiest instances
3. **Priority Groups**: Allow high-priority groups to preempt lower-priority ones
4. **Group Scheduling**: Run groups at specific times or intervals
5. **Resource Monitoring**: Track CPU/memory per instance, auto-scale
6. **Group Templates**: Save group configs as templates for quick setup

---

## Summary

This flow transforms the Manager Groups tab from a simple launcher into a sophisticated orchestration system that:
- Intelligently manages emulator instance allocation
- Prevents conflicts through instance reservation
- Validates routines before execution
- Supports configurable routine parameters
- Handles scaling and failures gracefully
- Provides clear feedback through the UI

The phased roadmap allows incremental implementation while maintaining a working system at each milestone.
