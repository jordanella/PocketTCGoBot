package bot

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"jordanella.com/pocket-tcg-go/internal/accountpool"
	"jordanella.com/pocket-tcg-go/internal/actions"
	"jordanella.com/pocket-tcg-go/internal/database"
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

	// Database connection for routine tracking (optional)
	db *sql.DB

	// Emulator manager for instance lifecycle
	emulatorManager *emulator.Manager

	// Health monitoring for instance launching
	healthMonitor *OrchestratorHealthMonitor

	// Group management
	groupDefinitions map[string]*BotGroupDefinition // Saved configurations
	activeGroups     map[string]*BotGroup           // Running instances
	groupsMu         sync.RWMutex

	// Global emulator instance tracking
	instanceRegistry   map[int]*InstanceAssignment
	instanceRegistryMu sync.RWMutex

	// Stagger delay for bot launches
	staggerDelay time.Duration

	// Configuration directory for saving group definitions
	groupConfigDir string
}

// BotGroup represents a coordinated set of bots with shared configuration
type BotGroup struct {
	// Identity
	Name            string
	OrchestrationID string // UUID for this execution context

	// Bot management (bots now managed directly by BotGroup)
	bots   map[int]*Bot // Active bot instances (key = instance ID)
	botsMu sync.RWMutex

	// Reference to orchestrator for registry access
	orchestrator *Orchestrator

	// Routine configuration
	RoutineName   string
	RoutineConfig map[string]string // Variable overrides

	// Emulator instance pool
	AvailableInstances []int            // Pool of instances this group can use
	RequestedBotCount  int              // How many bots user wants running
	ActiveBots         map[int]*BotInfo // Currently running bots (key = instance ID)
	activeBotsMu       sync.RWMutex

	// Account pool (optional - can be set by name or direct instance)
	AccountPoolName     string                  // Name of pool definition (resolved via PoolManager)
	AccountPool         accountpool.AccountPool // Execution-specific pool instance for this orchestration
	InitialAccountCount int                     // Total accounts when pool first populated (for progress monitoring)

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
	InstanceID  int
	GroupName   string
	BotInstance int
	AssignedAt  time.Time
	IsRunning   bool
	EmulatorPID int // Process ID of emulator if we launched it
}

// ConflictResolution defines how to handle instance conflicts
type ConflictResolution int

const (
	ConflictResolutionAsk    ConflictResolution = iota // Ask user what to do
	ConflictResolutionCancel                           // Cancel the other group
	ConflictResolutionSkip                             // Skip this instance
	ConflictResolutionAbort                            // Abort launch
)

// LaunchOptions configures how a group is launched
type LaunchOptions struct {
	// Validation options
	ValidateRoutine   bool `yaml:"validate_routine" json:"validate_routine"`
	ValidateTemplates bool `yaml:"validate_templates" json:"validate_templates"`
	ValidateEmulators bool `yaml:"validate_emulators" json:"validate_emulators"`

	// Conflict handling
	OnConflict ConflictResolution `yaml:"on_conflict" json:"on_conflict"`

	// Launch behavior
	StaggerDelay    time.Duration `yaml:"stagger_delay" json:"stagger_delay"`
	EmulatorTimeout time.Duration `yaml:"emulator_timeout" json:"emulator_timeout"`

	// Restart policy for bots
	RestartPolicy RestartPolicy `yaml:"restart_policy" json:"restart_policy"`
}

// NewOrchestrator creates a new bot orchestrator
func NewOrchestrator(
	config *Config,
	templateRegistry *templates.TemplateRegistry,
	routineRegistry *actions.RoutineRegistry,
	emulatorManager *emulator.Manager,
	poolManager *accountpool.PoolManager,
	db *sql.DB,
) *Orchestrator {
	// Default groups config directory
	groupConfigDir := "data/groups"
	if config != nil && config.FolderPath != "" {
		groupConfigDir = config.FolderPath + "/groups"
	}

	// Create and start health monitor
	healthMonitor := NewOrchestratorHealthMonitor(emulatorManager)
	healthMonitor.Start()

	return &Orchestrator{
		config:           config,
		templateRegistry: templateRegistry,
		routineRegistry:  routineRegistry,
		emulatorManager:  emulatorManager,
		healthMonitor:    healthMonitor,
		poolManager:      poolManager,
		db:               db,
		groupDefinitions: make(map[string]*BotGroupDefinition),
		activeGroups:     make(map[string]*BotGroup),
		instanceRegistry: make(map[int]*InstanceAssignment),
		staggerDelay:     5 * time.Second, // Default 5 second stagger
		groupConfigDir:   groupConfigDir,
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
	if _, exists := o.activeGroups[name]; exists {
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

	// Generate unique orchestration ID for this bot group execution
	orchestrationID := uuid.New().String()

	// Create group with orchestrator reference for registry access
	ctx, cancel := context.WithCancel(context.Background())
	group := &BotGroup{
		Name:               name,
		OrchestrationID:    orchestrationID,
		orchestrator:       o, // Link back to orchestrator for registries
		bots:               make(map[int]*Bot),
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

	fmt.Printf("Created bot group '%s' with orchestration ID: %s\n", name, orchestrationID)

	o.activeGroups[name] = group
	return group, nil
}

// CreateGroupFromDefinition creates a runtime group from a saved definition
func (o *Orchestrator) CreateGroupFromDefinition(def *BotGroupDefinition) (*BotGroup, error) {
	if def == nil {
		return nil, fmt.Errorf("definition cannot be nil")
	}

	// Validate definition
	if err := def.Validate(); err != nil {
		return nil, fmt.Errorf("invalid definition: %w", err)
	}

	// Use the existing CreateGroup method
	return o.CreateGroup(
		def.Name,
		def.RoutineName,
		def.AvailableInstances,
		def.RequestedBotCount,
		def.RoutineConfig,
		def.AccountPoolName,
	)
}

// DeleteGroup removes a group (must be stopped first)
func (o *Orchestrator) DeleteGroup(name string) error {
	o.groupsMu.Lock()
	defer o.groupsMu.Unlock()

	group, exists := o.activeGroups[name]
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
	delete(o.activeGroups, name)
	return nil
}

// GetGroup retrieves a group by name
func (o *Orchestrator) GetGroup(name string) (*BotGroup, bool) {
	o.groupsMu.RLock()
	defer o.groupsMu.RUnlock()
	group, exists := o.activeGroups[name]
	return group, exists
}

// ListGroups returns all group names
func (o *Orchestrator) ListGroups() []string {
	o.groupsMu.RLock()
	defer o.groupsMu.RUnlock()

	names := make([]string, 0, len(o.activeGroups))
	for name := range o.activeGroups {
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

// GetRoutineRegistry returns the routine registry
func (o *Orchestrator) GetRoutineRegistry() *actions.RoutineRegistry {
	return o.routineRegistry
}

// SetGroupAccountPool sets a group's account pool by name (resolves via PoolManager)
// This creates an execution-specific pool instance for this orchestration
func (o *Orchestrator) SetGroupAccountPool(groupName, poolName string) error {
	group, exists := o.GetGroup(groupName)
	if !exists {
		return fmt.Errorf("group '%s' not found", groupName)
	}

	// Resolve pool definition and create execution-specific instance
	pool, err := o.resolveAccountPool(poolName)
	if err != nil {
		return fmt.Errorf("failed to resolve pool '%s': %w", poolName, err)
	}

	// Get initial account count for progress monitoring
	stats := pool.GetStats()
	initialCount := stats.Total

	// Update group
	group.AccountPoolName = poolName
	group.AccountPool = pool
	group.InitialAccountCount = initialCount
	// Account pool is already set on the group

	fmt.Printf("Bot Group '%s' (orchestration %s): Populated pool '%s' with %d accounts\n",
		group.Name, group.OrchestrationID, poolName, initialCount)

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

// GetGroupAccountProgress returns progress information for account processing
func (o *Orchestrator) GetGroupAccountProgress(groupName string) (processed int, total int, err error) {
	group, exists := o.GetGroup(groupName)
	if !exists {
		return 0, 0, fmt.Errorf("group '%s' not found", groupName)
	}

	if group.AccountPool == nil {
		return 0, 0, fmt.Errorf("group '%s' has no account pool configured", groupName)
	}

	stats := group.AccountPool.GetStats()
	remaining := stats.Available
	total = group.InitialAccountCount
	processed = total - remaining

	return processed, total, nil
}

// ===== Group Definition Management =====

// SaveGroupDefinition saves a group definition to memory and disk
func (o *Orchestrator) SaveGroupDefinition(def *BotGroupDefinition) error {
	if err := def.Validate(); err != nil {
		return fmt.Errorf("invalid definition: %w", err)
	}

	o.groupsMu.Lock()
	defer o.groupsMu.Unlock()

	// Save to memory
	o.groupDefinitions[def.Name] = def

	// Save to disk
	if err := def.SaveToYAML(o.groupConfigDir); err != nil {
		return fmt.Errorf("failed to save to disk: %w", err)
	}

	fmt.Printf("Saved group definition '%s' to %s\n", def.Name, o.groupConfigDir)
	return nil
}

// LoadGroupDefinition retrieves a group definition by name
func (o *Orchestrator) LoadGroupDefinition(name string) (*BotGroupDefinition, error) {
	o.groupsMu.RLock()
	defer o.groupsMu.RUnlock()

	def, exists := o.groupDefinitions[name]
	if !exists {
		return nil, fmt.Errorf("group definition '%s' not found", name)
	}

	return def.Clone(), nil
}

// ListGroupDefinitions returns all saved group definitions
func (o *Orchestrator) ListGroupDefinitions() []*BotGroupDefinition {
	o.groupsMu.RLock()
	defer o.groupsMu.RUnlock()

	definitions := make([]*BotGroupDefinition, 0, len(o.groupDefinitions))
	for _, def := range o.groupDefinitions {
		definitions = append(definitions, def.Clone())
	}

	return definitions
}

// LoadGroupDefinitionsFromDisk loads all group definitions from disk
func (o *Orchestrator) LoadGroupDefinitionsFromDisk() error {
	definitions, err := LoadAllFromYAML(o.groupConfigDir)
	if err != nil {
		return fmt.Errorf("failed to load definitions: %w", err)
	}

	o.groupsMu.Lock()
	defer o.groupsMu.Unlock()

	for _, def := range definitions {
		o.groupDefinitions[def.Name] = def
		fmt.Printf("Loaded group definition '%s' from disk\n", def.Name)
	}

	fmt.Printf("Loaded %d group definition(s) from %s\n", len(definitions), o.groupConfigDir)
	return nil
}

// UpdateGroupDefinition updates an existing group definition
func (o *Orchestrator) UpdateGroupDefinition(def *BotGroupDefinition) error {
	o.groupsMu.Lock()
	defer o.groupsMu.Unlock()

	existing, exists := o.groupDefinitions[def.Name]
	if !exists {
		return fmt.Errorf("group definition '%s' not found", def.Name)
	}

	// Check if group is currently running
	if _, running := o.activeGroups[def.Name]; running {
		return fmt.Errorf("cannot update definition while group is running")
	}

	if err := existing.Update(def); err != nil {
		return err
	}

	return nil
}

// DeleteGroupDefinition removes a group definition from memory and disk
func (o *Orchestrator) DeleteGroupDefinition(name string) error {
	o.groupsMu.Lock()
	defer o.groupsMu.Unlock()

	// Check if group is currently running
	if _, running := o.activeGroups[name]; running {
		return fmt.Errorf("cannot delete definition while group is running")
	}

	def, exists := o.groupDefinitions[name]
	if !exists {
		return fmt.Errorf("group definition '%s' not found", name)
	}

	// Delete from disk
	if err := def.DeleteYAML(o.groupConfigDir); err != nil {
		fmt.Printf("Warning: failed to delete YAML file for '%s': %v\n", name, err)
	}

	// Delete from memory
	delete(o.groupDefinitions, name)
	fmt.Printf("Deleted group definition '%s'\n", name)
	return nil
}

// GetActiveGroup retrieves a running group by name
func (o *Orchestrator) GetActiveGroup(name string) (*BotGroup, bool) {
	o.groupsMu.RLock()
	defer o.groupsMu.RUnlock()

	group, exists := o.activeGroups[name]
	return group, exists
}

// ListActiveGroups returns all currently running groups
func (o *Orchestrator) ListActiveGroups() []*BotGroup {
	o.groupsMu.RLock()
	defer o.groupsMu.RUnlock()

	groups := make([]*BotGroup, 0, len(o.activeGroups))
	for _, group := range o.activeGroups {
		groups = append(groups, group)
	}

	return groups
}

// NOTE: GetGroup already exists earlier in the file at line ~262
// This duplicate has been removed to avoid conflicts

// ===== BotGroup Bot Management Methods =====

// createBot creates a bot instance for this group
func (g *BotGroup) createBot(instanceID int) (*Bot, error) {
	g.botsMu.Lock()
	defer g.botsMu.Unlock()

	// Check if bot already exists
	if _, exists := g.bots[instanceID]; exists {
		return nil, fmt.Errorf("bot instance %d already exists in group '%s'", instanceID, g.Name)
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

// shutdownBot shuts down a specific bot in this group
func (g *BotGroup) shutdownBot(instanceID int) error {
	g.botsMu.Lock()
	defer g.botsMu.Unlock()

	bot, exists := g.bots[instanceID]
	if !exists {
		return fmt.Errorf("bot instance %d not found in group '%s'", instanceID, g.Name)
	}

	bot.ShutdownWithSharedRegistries()
	delete(g.bots, instanceID)
	return nil
}

// GetBot retrieves a bot instance by ID
func (g *BotGroup) GetBot(instanceID int) (*Bot, bool) {
	g.botsMu.RLock()
	defer g.botsMu.RUnlock()
	bot, exists := g.bots[instanceID]
	return bot, exists
}

// GetBotCount returns the number of bots in this group
func (g *BotGroup) GetBotCount() int {
	g.botsMu.RLock()
	defer g.botsMu.RUnlock()
	return len(g.bots)
}

// shutdownAllBots shuts down all bots in this group
func (g *BotGroup) shutdownAllBots() {
	g.botsMu.Lock()
	defer g.botsMu.Unlock()

	for instanceID, bot := range g.bots {
		bot.ShutdownWithSharedRegistries()
		delete(g.bots, instanceID)
	}
}

// executeWithRestart executes a routine on a specific bot with restart policy
func (g *BotGroup) executeWithRestart(instanceID int, routineName string, policy RestartPolicy) error {
	g.botsMu.RLock()
	bot, exists := g.bots[instanceID]
	db := g.orchestrator.db
	g.botsMu.RUnlock()

	if !exists {
		return fmt.Errorf("bot instance %d not found", instanceID)
	}

	// Track the routine name for restart capability
	bot.SetLastRoutine(routineName)

	// Get routine with sentries from registry
	routineBuilder, sentries, err := bot.Routines().GetWithSentries(routineName)
	if err != nil {
		return fmt.Errorf("failed to get routine '%s': %w", routineName, err)
	}

	// Get routine metadata for config parameters
	routineMetadata := bot.Routines().GetMetadata(routineName + ".yaml")
	var configParams []actions.ConfigParam
	if routineMetadata != nil {
		if metadata, ok := routineMetadata.(map[string]interface{}); ok {
			if config, ok := metadata["config"].([]actions.ConfigParam); ok {
				configParams = config
			}
		}
	}

	// Start routine execution tracking if database is available and account is injected
	var executionID int64
	var accountID int64
	if db != nil {
		// Check if bot has device_account_id variable set (indicates account was injected)
		if deviceAccountStr, exists := bot.Variables().Get("device_account_id"); exists && deviceAccountStr != "" {
			// Parse account ID from variable
			fmt.Sscanf(deviceAccountStr, "%d", &accountID)

			// Record routine start
			executionID, err = database.StartRoutineExecution(db, accountID, routineName, bot.OrchestrationID(), instanceID)
			if err != nil {
				fmt.Printf("Bot %d: Warning - failed to start routine tracking: %v\n", instanceID, err)
			} else {
				// Store execution_id in bot variables for UpdateRoutineMetrics action
				bot.Variables().Set("execution_id", fmt.Sprintf("%d", executionID))
				fmt.Printf("Bot %d: Started routine execution tracking (ID: %d)\n", instanceID, executionID)
			}
		}
	}

	// Create routine executor with sentries
	executor := actions.NewRoutineExecutor(routineBuilder, sentries)

	// Helper function to execute one iteration with proper initialization
	executeIteration := func() error {
		// Clear non-persistent variables before each iteration
		if vs, ok := bot.Variables().(*actions.VariableStore); ok {
			vs.ClearNonPersistent()
		}

		// Reinitialize config variables
		if len(configParams) > 0 {
			if err := actions.InitializeConfigVariables(bot, configParams, nil); err != nil {
				return fmt.Errorf("failed to initialize config variables: %w", err)
			}
		}

		// Execute the routine with sentries
		return executor.Execute(bot)
	}

	// If restart is not enabled, execute once and return
	if !policy.Enabled {
		err := executeIteration()

		// Update routine execution tracking
		if db != nil && executionID > 0 {
			if err == nil {
				if completeErr := database.CompleteRoutineExecution(db, executionID, 0, 0); completeErr != nil {
					fmt.Printf("Bot %d: Warning - failed to mark routine as completed: %v\n", instanceID, completeErr)
				}
			} else {
				if failErr := database.FailRoutineExecution(db, executionID, err.Error()); failErr != nil {
					fmt.Printf("Bot %d: Warning - failed to mark routine as failed: %v\n", instanceID, failErr)
				}
			}
		}

		return err
	}

	// Execute with retry logic
	retryCount := 0
	currentDelay := policy.InitialDelay

	for {
		// Execute the routine (with variable reinitialization)
		err := executeIteration()

		// Success - reset retry counter and restart routine
		if err == nil {
			// Update routine execution tracking
			if db != nil && executionID > 0 {
				if completeErr := database.CompleteRoutineExecution(db, executionID, 0, 0); completeErr != nil {
					fmt.Printf("Bot %d: Warning - failed to mark routine as completed: %v\n", instanceID, completeErr)
				} else {
					fmt.Printf("Bot %d: Routine execution completed and tracked (ID: %d)\n", instanceID, executionID)
				}
			}

			if policy.ResetOnSuccess && retryCount > 0 {
				fmt.Printf("Bot %d: Routine '%s' succeeded after %d retries\n", instanceID, routineName, retryCount)
			}

			// Reset retry counter for next iteration
			retryCount = 0
			currentDelay = policy.InitialDelay

			// Start new execution tracking for next iteration
			if db != nil {
				if deviceAccountStr, exists := bot.Variables().Get("device_account_id"); exists && deviceAccountStr != "" {
					fmt.Sscanf(deviceAccountStr, "%d", &accountID)
					executionID, err = database.StartRoutineExecution(db, accountID, routineName, bot.OrchestrationID(), instanceID)
					if err != nil {
						fmt.Printf("Bot %d: Warning - failed to start routine tracking: %v\n", instanceID, err)
						executionID = 0
					} else {
						bot.Variables().Set("execution_id", fmt.Sprintf("%d", executionID))
						fmt.Printf("Bot %d: Restarting routine from beginning (new execution ID: %d)\n", instanceID, executionID)
					}
				}
			}

			// Continue to next iteration (infinite loop until stopped)
			continue
		}

		// Check if we've exceeded max retries
		if policy.MaxRetries > 0 && retryCount >= policy.MaxRetries {
			// Update routine execution tracking on final failure
			if db != nil && executionID > 0 {
				if failErr := database.FailRoutineExecution(db, executionID, err.Error()); failErr != nil {
					fmt.Printf("Bot %d: Warning - failed to mark routine as failed: %v\n", instanceID, failErr)
				}
			}

			return fmt.Errorf("bot %d routine '%s' failed after %d retries: %w", instanceID, routineName, retryCount, err)
		}

		// Failure - log and retry after delay
		fmt.Printf("Bot %d: Routine '%s' failed (attempt %d/%d): %v\n",
			instanceID, routineName, retryCount+1, policy.MaxRetries, err)

		// Update routine execution tracking on failure (but continuing retries)
		if db != nil && executionID > 0 {
			if failErr := database.FailRoutineExecution(db, executionID, err.Error()); failErr != nil {
				fmt.Printf("Bot %d: Warning - failed to mark routine as failed: %v\n", instanceID, failErr)
			}
		}

		// Calculate delay with backoff
		retryCount++
		if retryCount > 1 {
			currentDelay = time.Duration(float64(currentDelay) * policy.BackoffFactor)
			if currentDelay > policy.MaxDelay {
				currentDelay = policy.MaxDelay
			}
		}

		// Wait before retrying
		fmt.Printf("Bot %d: Waiting %v before retry %d...\n", instanceID, currentDelay, retryCount+1)
		time.Sleep(currentDelay)

		// Start new execution tracking for retry
		if db != nil {
			if deviceAccountStr, exists := bot.Variables().Get("device_account_id"); exists && deviceAccountStr != "" {
				fmt.Sscanf(deviceAccountStr, "%d", &accountID)
				executionID, err = database.StartRoutineExecution(db, accountID, routineName, bot.OrchestrationID(), instanceID)
				if err != nil {
					fmt.Printf("Bot %d: Warning - failed to start routine tracking: %v\n", instanceID, err)
					executionID = 0
				} else {
					bot.Variables().Set("execution_id", fmt.Sprintf("%d", executionID))
				}
			}
		}
	}
}

// GetInstanceAssignment returns the assignment for a specific instance
func (o *Orchestrator) GetInstanceAssignment(instanceID int) (*InstanceAssignment, bool) {
	o.instanceRegistryMu.RLock()
	defer o.instanceRegistryMu.RUnlock()

	assignment, exists := o.instanceRegistry[instanceID]
	return assignment, exists
}

// GetAllInstanceAssignments returns all instance assignments
func (o *Orchestrator) GetAllInstanceAssignments() map[int]*InstanceAssignment {
	o.instanceRegistryMu.RLock()
	defer o.instanceRegistryMu.RUnlock()

	// Return copy to avoid race conditions
	assignments := make(map[int]*InstanceAssignment, len(o.instanceRegistry))
	for k, v := range o.instanceRegistry {
		assignmentCopy := *v // Copy the struct
		assignments[k] = &assignmentCopy
	}
	return assignments
}

// GetAllInstanceIDsFromGroups returns all instance IDs that are in any group (active or not)
func (o *Orchestrator) GetAllInstanceIDsFromGroups() map[int][]string {
	o.groupsMu.RLock()
	defer o.groupsMu.RUnlock()

	// Map of instance ID -> list of group names
	instanceToGroups := make(map[int][]string)

	for groupName, group := range o.activeGroups {
		for _, instanceID := range group.AvailableInstances {
			instanceToGroups[instanceID] = append(instanceToGroups[instanceID], groupName)
		}
	}

	return instanceToGroups
}

// GetEmulatorManager returns the emulator manager for discovering instances
func (o *Orchestrator) GetEmulatorManager() *emulator.Manager {
	return o.emulatorManager
}
