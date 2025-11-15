# Orchestration Refactoring - Implementation Plan

This document outlines the step-by-step plan to refactor from the current Manager-based architecture to the cleaner Orchestrator-only architecture.

## Current State vs Target State

### Current (Confusing)
```
Orchestrator
  └─ BotGroup
       └─ Manager  ← Redundant!
            └─ map[int]*Bot
```

### Target (Clean)
```
Orchestrator (global coordinator)
  ├─ groupDefinitions: map[string]*BotGroupDefinition (saved configs)
  └─ activeGroups: map[string]*BotGroup (running instances)
       └─ bots: map[int]*Bot
```

## Implementation Steps

### ✅ Step 1: Create BotGroupDefinition Type

**File:** `internal/bot/orchestrator_definition.go` (new file)

```go
package bot

import "time"

// BotGroupDefinition represents a saved orchestration group configuration
// This is the persistent blueprint that can be saved, loaded, and edited
type BotGroupDefinition struct {
    // Identity
    Name        string
    Description string

    // Routine configuration
    RoutineName   string
    RoutineConfig map[string]string // Variable overrides

    // Emulator configuration
    AvailableInstances []int
    RequestedBotCount  int

    // Account pool configuration
    AccountPoolName string

    // Launch options
    LaunchOptions LaunchOptions

    // Restart policy
    RestartPolicy RestartPolicy

    // Metadata
    CreatedAt time.Time
    UpdatedAt time.Time
    Tags      []string // For categorization/filtering
}

// Clone creates a deep copy of the definition
func (d *BotGroupDefinition) Clone() *BotGroupDefinition {
    clone := *d
    clone.AvailableInstances = append([]int{}, d.AvailableInstances...)
    clone.RoutineConfig = make(map[string]string)
    for k, v := range d.RoutineConfig {
        clone.RoutineConfig[k] = v
    }
    clone.Tags = append([]string{}, d.Tags...)
    return &clone
}

// Validate checks if the definition is valid
func (d *BotGroupDefinition) Validate() error {
    if d.Name == "" {
        return fmt.Errorf("name is required")
    }
    if d.RoutineName == "" {
        return fmt.Errorf("routine name is required")
    }
    if len(d.AvailableInstances) == 0 {
        return fmt.Errorf("at least one emulator instance is required")
    }
    if d.RequestedBotCount <= 0 {
        return fmt.Errorf("requested bot count must be positive")
    }
    if d.RequestedBotCount > len(d.AvailableInstances) {
        return fmt.Errorf("requested bot count (%d) exceeds available instances (%d)",
            d.RequestedBotCount, len(d.AvailableInstances))
    }
    return nil
}
```

**Why separate file?** Keeps orchestrator.go focused on runtime logic.

### ✅ Step 2: Add Definition Management to Orchestrator

**File:** `internal/bot/orchestrator.go`

Add fields to Orchestrator:
```go
type Orchestrator struct {
    // ... existing fields ...

    // Group management (NEW)
    groupDefinitions map[string]*BotGroupDefinition // Saved configurations
    activeGroups     map[string]*BotGroup           // Runtime instances (rename from 'groups')
    groupsMu         sync.RWMutex
}
```

Add methods:
```go
// Definition management
func (o *Orchestrator) SaveGroupDefinition(def *BotGroupDefinition) error
func (o *Orchestrator) LoadGroupDefinition(name string) (*BotGroupDefinition, error)
func (o *Orchestrator) ListGroupDefinitions() []*BotGroupDefinition
func (o *Orchestrator) UpdateGroupDefinition(def *BotGroupDefinition) error
func (o *Orchestrator) DeleteGroupDefinition(name string) error

// Group lifecycle (updated)
func (o *Orchestrator) CreateGroupFromDefinition(defName string) (*BotGroup, error)
func (o *Orchestrator) GetActiveGroup(name string) (*BotGroup, bool)
func (o *Orchestrator) ListActiveGroups() []*BotGroup
```

### ✅ Step 3: Move Bot Management into BotGroup

**File:** `internal/bot/orchestrator.go`

Update BotGroup:
```go
type BotGroup struct {
    // ... existing fields ...

    // Remove Manager reference
    // Manager *Manager  ← DELETE THIS

    // Add bot management directly (NEW)
    bots              map[int]*Bot
    botsMu            sync.RWMutex

    // Add orchestrator reference for registry access (NEW)
    orchestrator *Orchestrator
}
```

Add bot management methods to BotGroup:
```go
// createBot creates a bot instance (moved from Manager)
func (g *BotGroup) createBot(instanceID int) (*Bot, error) {
    g.botsMu.Lock()
    defer g.botsMu.Unlock()

    // Check if bot already exists
    if _, exists := g.bots[instanceID]; exists {
        return nil, fmt.Errorf("bot instance %d already exists", instanceID)
    }

    // Create bot with shared config
    bot, err := New(instanceID, g.orchestrator.config)
    if err != nil {
        return nil, fmt.Errorf("failed to create bot %d: %w", instanceID, err)
    }

    // Inject shared registries from orchestrator
    bot.templateRegistry = g.orchestrator.templateRegistry
    bot.routineRegistry = g.orchestrator.routineRegistry
    bot.SetOrchestrationID(g.OrchestrationID)

    // Initialize the bot
    if err := bot.InitializeWithSharedRegistries(); err != nil {
        return nil, fmt.Errorf("failed to initialize bot %d: %w", instanceID, err)
    }

    g.bots[instanceID] = bot
    return bot, nil
}

// shutdownBot shuts down a specific bot (moved from Manager)
func (g *BotGroup) shutdownBot(instanceID int) error {
    g.botsMu.Lock()
    defer g.botsMu.Unlock()

    bot, exists := g.bots[instanceID]
    if !exists {
        return fmt.Errorf("bot instance %d not found", instanceID)
    }

    bot.ShutdownWithSharedRegistries()
    delete(g.bots, instanceID)
    return nil
}

// GetBot retrieves a bot instance
func (g *BotGroup) GetBot(instanceID int) (*Bot, bool) {
    g.botsMu.RLock()
    defer g.botsMu.RUnlock()
    bot, exists := g.bots[instanceID]
    return bot, exists
}
```

### ✅ Step 4: Update CreateGroup in Orchestrator

**File:** `internal/bot/orchestrator.go`

```go
// CreateGroupFromDefinition creates a runtime group from a saved definition
func (o *Orchestrator) CreateGroupFromDefinition(defName string) (*BotGroup, error) {
    o.groupsMu.Lock()
    defer o.groupsMu.Unlock()

    // Load definition
    def, exists := o.groupDefinitions[defName]
    if !exists {
        return nil, fmt.Errorf("group definition '%s' not found", defName)
    }

    // Check if already running
    if _, exists := o.activeGroups[defName]; exists {
        return nil, fmt.Errorf("group '%s' is already running", defName)
    }

    // Validate definition
    if err := def.Validate(); err != nil {
        return nil, fmt.Errorf("invalid definition: %w", err)
    }

    // Generate orchestration ID
    orchestrationID := uuid.New().String()

    // Create context for lifecycle management
    ctx, cancel := context.WithCancel(context.Background())

    // Create group
    group := &BotGroup{
        Name:               def.Name,
        OrchestrationID:    orchestrationID,
        RoutineName:        def.RoutineName,
        RoutineConfig:      def.RoutineConfig,
        AvailableInstances: def.AvailableInstances,
        RequestedBotCount:  def.RequestedBotCount,
        AccountPoolName:    def.AccountPoolName,
        bots:               make(map[int]*Bot),
        ActiveBots:         make(map[int]*BotInfo),
        ctx:                ctx,
        cancelFunc:         cancel,
        orchestrator:       o,  // Link back to orchestrator
    }

    // Resolve account pool if specified
    if def.AccountPoolName != "" {
        pool, err := o.poolManager.GetPool(def.AccountPoolName)
        if err != nil {
            cancel()
            return nil, fmt.Errorf("failed to get pool: %w", err)
        }
        group.AccountPool = pool
        stats := pool.GetStats()
        group.InitialAccountCount = stats.Total
    }

    // Add to active groups
    o.activeGroups[defName] = group

    return group, nil
}
```

### ✅ Step 5: Update BotGroup.Start()

**File:** `internal/bot/orchestrator.go`

Update to use `g.createBot()` instead of `g.Manager.CreateBot()`:

```go
func (g *BotGroup) Start(options LaunchOptions) error {
    g.runningMu.Lock()
    if g.running {
        g.runningMu.Unlock()
        return fmt.Errorf("group is already running")
    }
    g.running = true
    g.runningMu.Unlock()

    // Launch bots
    for i := 0; i < g.RequestedBotCount; i++ {
        instanceID := g.AvailableInstances[i]

        // Create bot (using NEW method)
        bot, err := g.createBot(instanceID)
        if err != nil {
            // Handle error...
            continue
        }

        // Launch bot goroutine...
        // (rest of start logic)
    }

    return nil
}
```

### ✅ Step 6: Remove Manager Type

**File:** `internal/bot/manager.go`

**ACTION:** Delete this file entirely (or mark as deprecated)

All functionality now lives in:
- `Orchestrator` - Global coordination
- `BotGroup` - Bot management

### ✅ Step 7: Add Persistence (Optional but Recommended)

**File:** `internal/bot/orchestrator_persistence.go` (new file)

```go
package bot

import (
    "encoding/json"
    "os"
    "path/filepath"
)

// SaveDefinitionToFile saves a group definition to disk
func (o *Orchestrator) SaveDefinitionToFile(def *BotGroupDefinition, dir string) error {
    if err := def.Validate(); err != nil {
        return err
    }

    data, err := json.MarshalIndent(def, "", "  ")
    if err != nil {
        return err
    }

    filename := filepath.Join(dir, def.Name+".json")
    return os.WriteFile(filename, data, 0644)
}

// LoadDefinitionFromFile loads a group definition from disk
func (o *Orchestrator) LoadDefinitionFromFile(filename string) (*BotGroupDefinition, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }

    var def BotGroupDefinition
    if err := json.Unmarshal(data, &def); err != nil {
        return nil, err
    }

    if err := def.Validate(); err != nil {
        return nil, err
    }

    return &def, nil
}

// LoadAllDefinitions loads all definitions from a directory
func (o *Orchestrator) LoadAllDefinitions(dir string) error {
    files, err := filepath.Glob(filepath.Join(dir, "*.json"))
    if err != nil {
        return err
    }

    for _, file := range files {
        def, err := o.LoadDefinitionFromFile(file)
        if err != nil {
            // Log error but continue
            continue
        }

        o.groupsMu.Lock()
        o.groupDefinitions[def.Name] = def
        o.groupsMu.Unlock()
    }

    return nil
}
```

### ✅ Step 8: Update GUI Controller

**File:** `internal/gui/controller.go`

```go
type Controller struct {
    // ... existing fields ...

    // Replace any Manager references with Orchestrator
    orchestrator *bot.Orchestrator  // This is all you need!
}

func NewController(...) *Controller {
    // Initialize orchestrator
    orchestrator := bot.NewOrchestrator(
        config,
        templateRegistry,
        routineRegistry,
        emulatorManager,
        poolManager,
    )

    // Load saved definitions
    orchestrator.LoadAllDefinitions("./orchestration_groups")

    ctrl := &Controller{
        orchestrator: orchestrator,
        // ...
    }

    return ctrl
}
```

### ✅ Step 9: Update Orchestration Tab

**File:** `internal/gui/tabs/orchestration.go`

```go
func (t *OrchestrationTab) showCreateGroupDialog() {
    // ... collect form data ...

    // Create definition
    definition := &bot.BotGroupDefinition{
        Name:               name,
        Description:        description,
        RoutineName:        routineName,
        AvailableInstances: instances,
        RequestedBotCount:  botCount,
        AccountPoolName:    poolName,
        CreatedAt:          time.Now(),
        UpdatedAt:          time.Now(),
    }

    // Save definition
    if err := t.orchestrator.SaveGroupDefinition(definition); err != nil {
        dialog.ShowError(err, t.window)
        return
    }

    // Optionally save to file
    t.orchestrator.SaveDefinitionToFile(definition, "./orchestration_groups")

    // Create card (shows stopped state with definition)
    card := components.NewOrchestrationCard(nil, definition, callbacks)
    t.cards[name] = card
    t.container.Add(card.GetContainer())
}

func (t *OrchestrationTab) handleStart(def *bot.BotGroupDefinition) {
    // Create runtime group from definition
    group, err := t.orchestrator.CreateGroupFromDefinition(def.Name)
    if err != nil {
        dialog.ShowError(err, t.window)
        return
    }

    // Start the group
    if err := group.Start(options); err != nil {
        dialog.ShowError(err, t.window)
        return
    }

    // Update card to show running state
    card := t.cards[def.Name]
    card.SetGroup(group)  // Now shows live data
}
```

### ✅ Step 10: Update OrchestrationCard

**File:** `internal/gui/components/orchestration_card.go`

Update to handle both definition and runtime group:

```go
type OrchestrationCard struct {
    data *OrchestrationCardData

    // Either definition (stopped) or group (running)
    definition *bot.BotGroupDefinition
    group      *bot.BotGroup

    // ...
}

func NewOrchestrationCard(
    group *bot.BotGroup,
    definition *bot.BotGroupDefinition,
    callbacks OrchestrationCardCallbacks,
) *OrchestrationCard {
    card := &OrchestrationCard{
        group:      group,
        definition: definition,
        // ...
    }
    // Build UI based on what's available
    return card
}

func (c *OrchestrationCard) UpdateFromGroup() {
    if c.group != nil {
        // Show live runtime data
        c.data.UpdateFromGroup(c.group)
    } else if c.definition != nil {
        // Show static definition data
        c.data.UpdateFromDefinition(c.definition)
    }
}
```

## Testing Strategy

### Unit Tests

```go
func TestBotGroupDefinition_Validate(t *testing.T)
func TestOrchestrator_SaveLoadDefinition(t *testing.T)
func TestBotGroup_CreateBot(t *testing.T)
func TestBotGroup_Start(t *testing.T)
```

### Integration Tests

```go
func TestFullWorkflow(t *testing.T) {
    // 1. Create definition
    // 2. Save to disk
    // 3. Load from disk
    // 4. Create runtime group
    // 5. Start group
    // 6. Verify bots running
    // 7. Stop group
    // 8. Verify cleanup
}
```

### Manual Testing Checklist

- [ ] Create a group definition via UI
- [ ] Save and reload definitions
- [ ] Start a group
- [ ] Verify bots are created and running
- [ ] Stop a group
- [ ] Verify bots are shut down
- [ ] Edit a stopped group
- [ ] Delete a group
- [ ] Handle errors gracefully

## Migration Strategy

### Option A: Big Bang (Faster but Riskier)
1. Implement all steps at once
2. Test thoroughly
3. Switch over

### Option B: Gradual (Safer)
1. Add BotGroupDefinition alongside existing code
2. Update BotGroup to have both Manager and direct bots
3. Add deprecation warnings to Manager
4. Update callers to use new API
5. Remove Manager when all callers updated

**Recommended:** Option B for production code

## Rollback Plan

If issues arise:
```bash
# Revert changes
git revert <commit-range>

# Or restore from backup
cp orchestrator.go.backup orchestrator.go
```

## Success Criteria

- [ ] ✅ No Manager references in codebase
- [ ] ✅ BotGroup manages its own bots
- [ ] ✅ Definitions can be saved/loaded
- [ ] ✅ UI shows both stopped and running states
- [ ] ✅ All tests pass
- [ ] ✅ No regressions in bot functionality
- [ ] ✅ Clear separation of concerns

## Estimated Effort

- **Step 1-2:** 1-2 hours (new types and methods)
- **Step 3-5:** 2-3 hours (move bot management)
- **Step 6:** 30 minutes (delete Manager)
- **Step 7:** 1 hour (persistence, optional)
- **Step 8-10:** 2-3 hours (GUI updates)
- **Testing:** 2-4 hours

**Total:** 8-12 hours (spread over 2-3 sessions)

## Future Enhancements

Once refactoring is complete, you can add:
- Group templates (pre-configured definitions)
- Scheduling (start groups at specific times)
- Group dependencies (start B after A completes)
- Resource quotas (limit concurrent bots)
- Monitoring dashboard (real-time stats)
- Export/import definitions (share configs)

## Questions?

This plan provides a clear path forward. Would you like me to:
1. Start implementing Step 1 (BotGroupDefinition type)?
2. Create example code for any specific step?
3. Clarify any part of the architecture?
