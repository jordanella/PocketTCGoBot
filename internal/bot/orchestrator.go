package bot

import (
	"context"
	"fmt"
	"sync"
	"time"

	"jordanella.com/pocket-tcg-go/internal/accountpool"
	"jordanella.com/pocket-tcg-go/internal/actions"
	"jordanella.com/pocket-tcg-go/internal/emulator"
	"jordanella.com/pocket-tcg-go/pkg/templates"
)

// Orchestrator manages multiple bot groups with emulator instance coordination
type Orchestrator struct {
	// Global registries (shared across all groups)
	templateRegistry *templates.TemplateRegistry
	routineRegistry  *actions.RoutineRegistry
	poolManager      *accountpool.PoolManager

	// Global configuration
	config *Config

	// Emulator manager for instance lifecycle
	emulatorManager *emulator.Manager

	// Groups management
	groups   map[string]*BotGroup
	groupsMu sync.RWMutex

	// Global emulator instance tracking
	instanceRegistry   map[int]*InstanceAssignment
	instanceRegistryMu sync.RWMutex

	// Stagger delay for bot launches
	staggerDelay time.Duration
}

// BotGroup represents a coordinated set of bots with shared configuration
type BotGroup struct {
	// Identity
	Name string

	// Bot manager for this group
	Manager *Manager

	// Routine configuration
	RoutineName   string
	RoutineConfig map[string]string // Variable overrides

	// Emulator instance pool
	AvailableInstances []int              // Pool of instances this group can use
	RequestedBotCount  int                // How many bots user wants running
	ActiveBots         map[int]*BotInfo   // Currently running bots (key = instance ID)
	activeBotsMu       sync.RWMutex

	// Account pool (optional - can be set by name or direct instance)
	AccountPoolName string                      // Name of pool (resolved via PoolManager)
	AccountPool     accountpool.AccountPool     // Direct pool instance (if not using named pool)

	// Runtime state
	running   bool
	runningMu sync.RWMutex

	// Context for cancellation
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// BotInfo tracks a single bot instance
type BotInfo struct {
	Bot        *Bot
	InstanceID int
	StartedAt  time.Time
	Status     BotStatus
	Error      error

	// Routine execution context
	routineCtx    context.Context
	routineCancel context.CancelFunc
}

// BotStatus represents the current state of a bot
type BotStatus string

const (
	BotStatusStarting  BotStatus = "starting"
	BotStatusRunning   BotStatus = "running"
	BotStatusStopping  BotStatus = "stopping"
	BotStatusStopped   BotStatus = "stopped"
	BotStatusFailed    BotStatus = "failed"
	BotStatusCompleted BotStatus = "completed"
)

// InstanceAssignment tracks which group/bot is using an emulator instance
type InstanceAssignment struct {
	InstanceID   int
	GroupName    string
	BotInstance  int
	AssignedAt   time.Time
	IsRunning    bool
	EmulatorPID  int // Process ID of emulator if we launched it
}

// ConflictResolution defines how to handle instance conflicts
type ConflictResolution int

const (
	ConflictResolutionAsk    ConflictResolution = iota // Ask user what to do
	ConflictResolutionCancel                            // Cancel the other group
	ConflictResolutionSkip                              // Skip this instance
	ConflictResolutionAbort                             // Abort launch
)

// LaunchOptions configures how a group is launched
type LaunchOptions struct {
	// Validation options
	ValidateRoutine      bool
	ValidateTemplates    bool
	ValidateEmulators    bool

	// Conflict handling
	OnConflict ConflictResolution

	// Launch behavior
	StaggerDelay    time.Duration
	EmulatorTimeout time.Duration

	// Restart policy for bots
	RestartPolicy RestartPolicy
}

// NewOrchestrator creates a new bot orchestrator
func NewOrchestrator(
	config *Config,
	templateRegistry *templates.TemplateRegistry,
	routineRegistry *actions.RoutineRegistry,
	emulatorManager *emulator.Manager,
	poolManager *accountpool.PoolManager,
) *Orchestrator {
	return &Orchestrator{
		config:             config,
		templateRegistry:   templateRegistry,
		routineRegistry:    routineRegistry,
		emulatorManager:    emulatorManager,
		poolManager:        poolManager,
		groups:             make(map[string]*BotGroup),
		instanceRegistry:   make(map[int]*InstanceAssignment),
		staggerDelay:       5 * time.Second, // Default 5 second stagger
	}
}

// SetStaggerDelay sets the delay between bot launches
func (o *Orchestrator) SetStaggerDelay(delay time.Duration) {
	o.staggerDelay = delay
}

// CreateGroup creates a new bot group
func (o *Orchestrator) CreateGroup(
	name string,
	routineName string,
	availableInstances []int,
	requestedBotCount int,
	routineConfig map[string]string,
	accountPoolName string, // Name of pool (empty string if not using pool)
) (*BotGroup, error) {
	o.groupsMu.Lock()
	defer o.groupsMu.Unlock()

	// Check if group already exists
	if _, exists := o.groups[name]; exists {
		return nil, fmt.Errorf("group '%s' already exists", name)
	}

	// Validate parameters
	if name == "" {
		return nil, fmt.Errorf("group name is required")
	}
	if routineName == "" {
		return nil, fmt.Errorf("routine name is required")
	}
	if requestedBotCount <= 0 {
		return nil, fmt.Errorf("requested bot count must be positive")
	}
	if len(availableInstances) == 0 {
		return nil, fmt.Errorf("at least one emulator instance must be specified")
	}
	if requestedBotCount > len(availableInstances) {
		return nil, fmt.Errorf("requested bot count (%d) exceeds available instances (%d)",
			requestedBotCount, len(availableInstances))
	}

	// Create manager for this group
	manager := NewManagerWithRegistries(o.config, o.templateRegistry, o.routineRegistry)

	// Create group
	ctx, cancel := context.WithCancel(context.Background())
	group := &BotGroup{
		Name:               name,
		Manager:            manager,
		RoutineName:        routineName,
		RoutineConfig:      routineConfig,
		AvailableInstances: availableInstances,
		RequestedBotCount:  requestedBotCount,
		ActiveBots:         make(map[int]*BotInfo),
		AccountPoolName:    accountPoolName,
		running:            false,
		ctx:                ctx,
		cancelFunc:         cancel,
	}

	o.groups[name] = group
	return group, nil
}

// DeleteGroup removes a group (must be stopped first)
func (o *Orchestrator) DeleteGroup(name string) error {
	o.groupsMu.Lock()
	defer o.groupsMu.Unlock()

	group, exists := o.groups[name]
	if !exists {
		return fmt.Errorf("group '%s' not found", name)
	}

	// Check if group is running
	group.runningMu.RLock()
	running := group.running
	group.runningMu.RUnlock()

	if running {
		return fmt.Errorf("cannot delete running group '%s' - stop it first", name)
	}

	// Cancel group context
	group.cancelFunc()

	// Close account pool if present
	if group.AccountPool != nil {
		group.AccountPool.Close()
	}

	// Remove from groups map
	delete(o.groups, name)
	return nil
}

// GetGroup retrieves a group by name
func (o *Orchestrator) GetGroup(name string) (*BotGroup, bool) {
	o.groupsMu.RLock()
	defer o.groupsMu.RUnlock()
	group, exists := o.groups[name]
	return group, exists
}

// ListGroups returns all group names
func (o *Orchestrator) ListGroups() []string {
	o.groupsMu.RLock()
	defer o.groupsMu.RUnlock()

	names := make([]string, 0, len(o.groups))
	for name := range o.groups {
		names = append(names, name)
	}
	return names
}

// IsRunning returns whether the group is currently running
func (g *BotGroup) IsRunning() bool {
	g.runningMu.RLock()
	defer g.runningMu.RUnlock()
	return g.running
}

// GetActiveBotCount returns the number of currently active bots
func (g *BotGroup) GetActiveBotCount() int {
	g.activeBotsMu.RLock()
	defer g.activeBotsMu.RUnlock()
	return len(g.ActiveBots)
}

// GetBotInfo retrieves information about a specific bot
func (g *BotGroup) GetBotInfo(instanceID int) (*BotInfo, bool) {
	g.activeBotsMu.RLock()
	defer g.activeBotsMu.RUnlock()
	info, exists := g.ActiveBots[instanceID]
	return info, exists
}

// GetAllBotInfo returns information about all active bots
func (g *BotGroup) GetAllBotInfo() map[int]*BotInfo {
	g.activeBotsMu.RLock()
	defer g.activeBotsMu.RUnlock()

	// Return copy to avoid race conditions
	bots := make(map[int]*BotInfo, len(g.ActiveBots))
	for id, info := range g.ActiveBots {
		bots[id] = info
	}
	return bots
}

// ===== Account Pool Management =====

// GetPoolManager returns the pool manager
func (o *Orchestrator) GetPoolManager() *accountpool.PoolManager {
	return o.poolManager
}

// SetGroupAccountPool sets a group's account pool by name (resolves via PoolManager)
func (o *Orchestrator) SetGroupAccountPool(groupName, poolName string) error {
	group, exists := o.GetGroup(groupName)
	if !exists {
		return fmt.Errorf("group '%s' not found", groupName)
	}

	// Resolve pool
	pool, err := o.resolveAccountPool(poolName)
	if err != nil {
		return fmt.Errorf("failed to resolve pool '%s': %w", poolName, err)
	}

	// Update group
	group.AccountPoolName = poolName
	group.AccountPool = pool
	group.Manager.SetAccountPool(pool)

	return nil
}

// resolveAccountPool gets an account pool instance by name
func (o *Orchestrator) resolveAccountPool(poolName string) (accountpool.AccountPool, error) {
	if poolName == "" {
		return nil, nil
	}

	if o.poolManager == nil {
		return nil, fmt.Errorf("pool manager not configured")
	}

	pool, err := o.poolManager.GetPool(poolName)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool: %w", err)
	}

	return pool, nil
}

// RefreshGroupAccountPool manually refreshes a group's account pool
func (o *Orchestrator) RefreshGroupAccountPool(groupName string) error {
	group, exists := o.GetGroup(groupName)
	if !exists {
		return fmt.Errorf("group '%s' not found", groupName)
	}

	if group.AccountPool == nil {
		return fmt.Errorf("group '%s' has no account pool configured", groupName)
	}

	return group.AccountPool.Refresh()
}
