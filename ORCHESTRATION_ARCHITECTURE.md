# Orchestration Architecture - Design Clarification

## Problem Statement

Currently there's confusion between `Manager` and `Orchestrator`:

**Current Issues:**
1. ❌ `Manager` manages individual bots (line 20: `bots map[int]*Bot`)
2. ❌ `BotGroup` contains a `Manager` (line 48 in orchestrator.go)
3. ❌ Overlapping responsibilities between Manager and Orchestrator
4. ❌ `Manager.orchestrationID` suggests it belongs to ONE group, but it also manages multiple bots

## Proposed Clear Architecture

### Layer 1: Global Resources (Orchestrator)

**Role:** Global coordination and resource management

```go
type Orchestrator struct {
    // Global registries (shared across ALL groups)
    templateRegistry *templates.TemplateRegistry
    routineRegistry  *actions.RoutineRegistry
    poolManager      *accountpool.PoolManager

    // Global configuration
    config *Config

    // Emulator manager
    emulatorManager *emulator.Manager

    // Orchestration group management
    groupDefinitions map[string]*BotGroupDefinition // Saved configurations
    activeGroups     map[string]*BotGroup           // Currently running
    groupsMu         sync.RWMutex

    // Global emulator instance tracking
    instanceRegistry   map[int]*InstanceAssignment
    instanceRegistryMu sync.RWMutex
}
```

**Responsibilities:**
- ✅ Initialize and manage global registries (ONE instance per app)
- ✅ Create and destroy BotGroups
- ✅ Track emulator instance assignments
- ✅ Persist/load group definitions
- ✅ Prevent instance conflicts

### Layer 2: Orchestration Group (BotGroup)

**Role:** Manages a coordinated set of bots with shared configuration

```go
type BotGroup struct {
    // Identity
    Name            string
    OrchestrationID string // UUID for this execution instance

    // Configuration
    RoutineName   string
    RoutineConfig map[string]string

    // Bot management (THIS is where bots live)
    bots              map[int]*Bot    // Active bot instances
    botsMu            sync.RWMutex

    // Emulator pool
    AvailableInstances []int
    RequestedBotCount  int

    // Account pool
    AccountPoolName     string
    AccountPool         accountpool.AccountPool
    InitialAccountCount int

    // Runtime state
    running   bool
    runningMu sync.RWMutex

    // Context for lifecycle
    ctx        context.Context
    cancelFunc context.CancelFunc

    // Back-reference to orchestrator (for registry access)
    orchestrator *Orchestrator
}
```

**Responsibilities:**
- ✅ Create and manage bots for THIS group
- ✅ Execute routines on bots
- ✅ Manage account pool for THIS group
- ✅ Track bot status
- ✅ Handle group lifecycle (start/stop/pause)

### Layer 3: Individual Bot

**Role:** Execute actions on a single emulator instance

```go
type Bot struct {
    instance         int
    config           *Config
    templateRegistry actions.TemplateRegistryInterface
    routineRegistry  actions.RoutineRegistryInterface
    orchestrationID  string // Which group do I belong to?

    // ... rest of bot fields
}
```

**Responsibilities:**
- ✅ Execute actions on ONE emulator
- ✅ Use registries from orchestrator
- ✅ Report status back to group

### ❌ REMOVE: Manager

**The `Manager` type should be eliminated.** Its responsibilities are:
1. Registry management → Move to `Orchestrator`
2. Bot creation → Move to `BotGroup`
3. Multi-bot coordination → Already in `BotGroup`

## Proposed New Structure

### Orchestrator Methods

```go
// Group definition management (save/load configurations)
func (o *Orchestrator) SaveGroupDefinition(def *BotGroupDefinition) error
func (o *Orchestrator) LoadGroupDefinition(name string) (*BotGroupDefinition, error)
func (o *Orchestrator) ListGroupDefinitions() []string
func (o *Orchestrator) DeleteGroupDefinition(name string) error

// Active group management (runtime instances)
func (o *Orchestrator) CreateGroup(def *BotGroupDefinition) (*BotGroup, error)
func (o *Orchestrator) GetGroup(name string) (*BotGroup, bool)
func (o *Orchestrator) ListActiveGroups() []*BotGroup
func (o *Orchestrator) ShutdownGroup(name string) error

// Registry access (for groups to use)
func (o *Orchestrator) TemplateRegistry() *templates.TemplateRegistry
func (o *Orchestrator) RoutineRegistry() *actions.RoutineRegistry
func (o *Orchestrator) PoolManager() *accountpool.PoolManager
```

### BotGroup Methods

```go
// Bot lifecycle (internal to group)
func (g *BotGroup) createBot(instanceID int) (*Bot, error)
func (g *BotGroup) shutdownBot(instanceID int) error
func (g *BotGroup) GetBot(instanceID int) (*Bot, bool)
func (g *BotGroup) GetAllBots() map[int]*BotInfo

// Group control (public)
func (g *BotGroup) Start(launchOptions LaunchOptions) error
func (g *BotGroup) Stop() error
func (g *BotGroup) Pause() error
func (g *BotGroup) Resume() error
func (g *BotGroup) IsRunning() bool
```

### BotGroupDefinition (Saveable Configuration)

```go
type BotGroupDefinition struct {
    Name            string
    Description     string
    RoutineName     string
    RoutineConfig   map[string]string

    // Emulator configuration
    AvailableInstances []int
    RequestedBotCount  int

    // Account pool configuration
    AccountPoolName string

    // Launch options
    LaunchOptions LaunchOptions

    // Metadata
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**This is what you save to disk/database!**

## Data Flow

### Creating and Running a Group

```go
// 1. User creates a group definition (via UI)
definition := &BotGroupDefinition{
    Name:              "Premium Farmers",
    Description:       "Farm premium packs",
    RoutineName:       "farm_premium.yaml",
    AvailableInstances: []int{1, 2, 3, 4},
    RequestedBotCount:  2,
    AccountPoolName:   "premium_accounts",
}

// 2. Save definition (persisted)
orchestrator.SaveGroupDefinition(definition)

// 3. Later, start the group (creates runtime instance)
group, err := orchestrator.CreateGroup(definition)

// 4. Launch bots
err = group.Start(LaunchOptions{...})

// Now: group.bots contains the running bot instances
// Now: orchestrator.activeGroups contains this group
```

### Stopping a Group

```go
// 1. Stop the group
group.Stop()

// 2. Group shuts down all its bots
// 3. Group removed from orchestrator.activeGroups
// 4. Definition still exists in orchestrator.groupDefinitions
```

### Restarting a Group

```go
// 1. Load definition
definition, err := orchestrator.LoadGroupDefinition("Premium Farmers")

// 2. Create new runtime instance
group, err := orchestrator.CreateGroup(definition)

// 3. Start it
group.Start(options)
```

## GUI Integration

### Orchestration Tab UI

```
┌─────────────────────────────────────────────────────┐
│ Orchestration Groups                     [+ Create] │
├─────────────────────────────────────────────────────┤
│                                                      │
│ ┌──────────────────────────────────────────────┐   │
│ │ Premium Farmers                 ● Running     │   │
│ │ Farm premium packs                            │   │
│ │ Routine: farm_premium.yaml                    │   │
│ │ Pool: 45/100 accounts                         │   │
│ │ Active: Instance 1, Instance 2                │   │
│ │ [ Pause ] [ Stop ] [ Edit ] [ Delete ]        │   │
│ └──────────────────────────────────────────────┘   │
│                                                      │
│ ┌──────────────────────────────────────────────┐   │
│ │ Wonder Pick Farmers             ○ Stopped     │   │
│ │ Automated wonder pick farming                 │   │
│ │ Routine: wonder_pick.yaml                     │   │
│ │ Pool: 20/50 accounts                          │   │
│ │ Available: 4 instances                        │   │
│ │ [ Start ] [ Edit ] [ Delete ]                 │   │
│ └──────────────────────────────────────────────┘   │
│                                                      │
└─────────────────────────────────────────────────────┘
```

### Data Model

```go
// Controller holds the orchestrator
type Controller struct {
    orchestrator *bot.Orchestrator
    // ...
}

// Tab displays groups
type OrchestrationTab struct {
    orchestrator *bot.Orchestrator

    // UI elements
    cards map[string]*components.OrchestrationCard
    // ...
}

// Card displays one group
type OrchestrationCard struct {
    group *bot.BotGroup  // Runtime instance (if running)
    def   *bot.BotGroupDefinition  // Saved configuration
    // ...
}
```

## Benefits of This Architecture

### ✅ Clear Separation of Concerns

1. **Orchestrator** = Global coordinator
   - ONE instance per application
   - Manages registries
   - Tracks all groups and emulators

2. **BotGroupDefinition** = Configuration (persisted)
   - Can be saved/loaded
   - Can be edited while stopped
   - Template for creating groups

3. **BotGroup** = Runtime instance
   - Created from definition
   - Manages its own bots
   - Can be started/stopped

4. **Bot** = Single executor
   - Uses registries from orchestrator
   - Reports to its group

### ✅ Answers Your Questions

> Does the manager even need a reference to bots?

**No!** The `Manager` should be removed. `BotGroup` manages bots.

> Just a reference to all the orchestration groups?

**Yes!** `Orchestrator` has:
- `groupDefinitions` - Saved configurations
- `activeGroups` - Currently running groups

> Do we need distinct types for these?

**Yes!**
- `BotGroupDefinition` - Saveable configuration
- `BotGroup` - Runtime instance

They serve different purposes:
- **Definition** = Blueprint (saved to disk/DB)
- **Group** = Running instance (in memory, has active bots)

### ✅ Clean Data Flow

```
User creates definition
         ↓
  Save to disk/DB (BotGroupDefinition)
         ↓
  Load definition
         ↓
  Create runtime group (BotGroup)
         ↓
  Start group
         ↓
  Group creates bots
         ↓
  Bots execute routines
         ↓
  Stop group
         ↓
  Bots shut down
         ↓
  Group removed from activeGroups
         ↓
  Definition still saved (can restart later)
```

## Migration Path

### Phase 1: Create New Types

```go
// Add to orchestrator.go
type BotGroupDefinition struct { /* ... */ }

// Update Orchestrator to have both maps
type Orchestrator struct {
    groupDefinitions map[string]*BotGroupDefinition
    activeGroups     map[string]*BotGroup
    // ...
}
```

### Phase 2: Move Bot Management to BotGroup

```go
// Remove Manager, add to BotGroup:
type BotGroup struct {
    bots     map[int]*Bot
    botsMu   sync.RWMutex

    orchestrator *Orchestrator  // For registry access
    // ...
}

func (g *BotGroup) createBot(instanceID int) (*Bot, error) {
    // Bot creation logic here (moved from Manager)
}
```

### Phase 3: Update Orchestrator Methods

```go
func (o *Orchestrator) SaveGroupDefinition(def *BotGroupDefinition) error
func (o *Orchestrator) CreateGroup(def *BotGroupDefinition) (*BotGroup, error)
```

### Phase 4: Update GUI

```go
// Controller uses Orchestrator
ctrl.orchestrator.SaveGroupDefinition(def)
group, _ := ctrl.orchestrator.CreateGroup(def)
group.Start(options)
```

## Summary

### Current (Confusing)
```
Orchestrator
  └─ BotGroup
       └─ Manager  ← Redundant!
            └─ Bots
```

### Proposed (Clear)
```
Orchestrator (global coordinator)
  ├─ Group Definitions (saved configs)
  │    └─ BotGroupDefinition
  │
  └─ Active Groups (running instances)
       └─ BotGroup
            └─ Bots
```

### Key Changes

1. ❌ **Remove** `Manager` type entirely
2. ✅ **Add** `BotGroupDefinition` type (saveable config)
3. ✅ **Move** bot management into `BotGroup`
4. ✅ **Separate** definitions (config) from groups (runtime)
5. ✅ **Orchestrator** only manages groups, not bots directly

This gives you:
- ✅ Clear responsibilities
- ✅ Ability to save/load group configs
- ✅ Clean runtime management
- ✅ No confusion between definition and instance
- ✅ Perfect UI abstraction (cards show groups)
