package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"jordanella.com/pocket-tcg-go/internal/database"
	"jordanella.com/pocket-tcg-go/internal/events"
)

// LaunchResult contains the results of a group launch
type LaunchResult struct {
	Success        bool
	LaunchedBots   int
	RequestedBots  int
	Errors         []string
	Conflicts      []InstanceConflict
	SkippedInstances []int
}

// LaunchOverrides allows runtime modification of group parameters without changing stored definition
type LaunchOverrides struct {
	// Core parameter overrides (nil = use definition value)
	RequestedBotCount  *int              // Override number of bots to launch
	AvailableInstances []int             // Override which instances to use
	RoutineConfig      map[string]string // Merge with or override routine config
	AccountPoolName    *string           // Override account pool

	// Account limiting
	MaxAccounts *int // Limit number of accounts to use from pool (e.g., only use 50 accounts)

	// Launch options override
	LaunchOptions *LaunchOptions // Override launch options
	RestartPolicy *RestartPolicy // Override restart policy
}

// LaunchGroupWithOverrides starts all bots in a group with runtime parameter overrides
// This allows launching with different settings without modifying the stored definition
func (o *Orchestrator) LaunchGroupWithOverrides(groupName string, overrides *LaunchOverrides) (*LaunchResult, error) {
	// Get the stored definition
	definition, err := o.LoadGroupDefinition(groupName)
	if err != nil {
		return nil, fmt.Errorf("group definition '%s' not found: %w", groupName, err)
	}

	// Create a modified definition for this launch
	runtimeDef := definition.Clone()

	// Apply overrides to create runtime configuration
	if overrides != nil {
		if overrides.RequestedBotCount != nil {
			runtimeDef.RequestedBotCount = *overrides.RequestedBotCount
		}
		if len(overrides.AvailableInstances) > 0 {
			runtimeDef.AvailableInstances = overrides.AvailableInstances
		}
		if overrides.AccountPoolName != nil {
			runtimeDef.AccountPoolName = *overrides.AccountPoolName
		}
		if len(overrides.RoutineConfig) > 0 {
			// Merge with existing config
			if runtimeDef.RoutineConfig == nil {
				runtimeDef.RoutineConfig = make(map[string]string)
			}
			for k, v := range overrides.RoutineConfig {
				runtimeDef.RoutineConfig[k] = v
			}
		}
		if overrides.LaunchOptions != nil {
			runtimeDef.LaunchOptions = *overrides.LaunchOptions
		}
		if overrides.RestartPolicy != nil {
			runtimeDef.RestartPolicy = *overrides.RestartPolicy
		}
	}

	// Validate the runtime definition
	validationResult := ValidateGroupDefinition(runtimeDef)
	if !validationResult.Valid {
		return nil, fmt.Errorf("runtime configuration validation failed:\n%s", validationResult.FormatValidationErrors())
	}

	// Create a temporary runtime group (don't save to definitions)
	// Use a unique runtime name to avoid conflicts
	runtimeName := groupName + "_runtime_" + fmt.Sprintf("%d", time.Now().Unix())

	// Create runtime group from modified definition
	runtimeGroup, err := o.createTempRuntimeGroup(runtimeName, runtimeDef)
	if err != nil {
		return nil, fmt.Errorf("failed to create runtime group: %w", err)
	}

	// If MaxAccounts is specified, create a limited pool view
	if overrides != nil && overrides.MaxAccounts != nil && *overrides.MaxAccounts > 0 {
		// TODO: Implement account pool limiting wrapper
		// For now, just note it in the result
		fmt.Printf("Note: MaxAccounts override (%d) requested but not yet implemented\n", *overrides.MaxAccounts)
	}

	// Launch the runtime group
	result, err := o.launchGroupInternal(runtimeGroup, runtimeDef.LaunchOptions)

	// Store original group name in metadata for display purposes
	// (The runtime group will have the modified name internally)

	return result, err
}

// LaunchGroup starts all bots in a group with full orchestration
// Uses the stored group definition without modifications
func (o *Orchestrator) LaunchGroup(groupName string, options LaunchOptions) (*LaunchResult, error) {
	// Get group
	group, exists := o.GetGroup(groupName)
	if !exists {
		return nil, fmt.Errorf("group '%s' not found", groupName)
	}

	// Check if already running
	if group.IsRunning() {
		return nil, fmt.Errorf("group '%s' is already running", groupName)
	}

	// Launch with stored configuration
	return o.launchGroupInternal(group, options)
}

// launchGroupInternal is the shared launch implementation used by both LaunchGroup and LaunchGroupWithOverrides
func (o *Orchestrator) launchGroupInternal(group *BotGroup, options LaunchOptions) (*LaunchResult, error) {
	// Validate launch options
	validationResult := ValidateLaunchOptions(&options)
	if !validationResult.Valid {
		return nil, fmt.Errorf("launch options validation failed:\n%s", validationResult.FormatValidationErrors())
	}

	result := &LaunchResult{
		Success:       true,
		RequestedBots: group.RequestedBotCount,
		Errors:        make([]string, 0),
		Conflicts:     make([]InstanceConflict, 0),
		SkippedInstances: make([]int, 0),
	}

	// Phase 0: Resolve and setup account pool if needed
	if group.AccountPoolName != "" && group.AccountPool == nil {
		pool, err := o.resolveAccountPool(group.AccountPoolName)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("failed to resolve account pool: %v", err))
			return result, fmt.Errorf("failed to resolve account pool '%s': %w", group.AccountPoolName, err)
		}
		group.AccountPool = pool
	}

	// Phase 1: Routine Validation
	if options.ValidateRoutine {
		validationResult := o.ValidateRoutine(group.RoutineName, group.RoutineConfig)
		if !validationResult.Valid {
			result.Success = false
			result.Errors = append(result.Errors, validationResult.FormatValidationErrors())
			return result, fmt.Errorf("routine validation failed")
		}
	}

	// Phase 2: Acquire Emulator Instances
	acquiredInstances, acquireResult := o.acquireInstances(group, options)
	result.Conflicts = acquireResult.Conflicts
	result.SkippedInstances = acquireResult.SkippedInstances

	if len(acquiredInstances) == 0 {
		result.Success = false
		result.Errors = append(result.Errors, "no emulator instances available")
		return result, fmt.Errorf("failed to acquire any emulator instances")
	}

	if len(acquiredInstances) < group.RequestedBotCount {
		// Warn but continue with what we have
		result.Errors = append(result.Errors,
			fmt.Sprintf("only acquired %d of %d requested instances",
				len(acquiredInstances), group.RequestedBotCount))
	}

	// Phase 3: Launch Bots with Stagger
	launchedCount, launchErrors := o.launchBotsStaggered(group, acquiredInstances, options)
	result.LaunchedBots = launchedCount
	result.Errors = append(result.Errors, launchErrors...)

	if launchedCount == 0 {
		result.Success = false
		// Release all acquired instances since no bots launched
		o.releaseAllInstances(group.Name)
		return result, fmt.Errorf("failed to launch any bots")
	}

	// Mark group as running
	group.runningMu.Lock()
	group.running = true
	group.runningMu.Unlock()

	// Publish group launched event
	if o.eventBus != nil {
		o.eventBus.PublishAsync(events.NewGroupLaunchedEvent(
			group.Name,
			launchedCount,
			group.RequestedBotCount,
			acquiredInstances,
		))
	}

	return result, nil
}

// InstanceAcquisitionResult contains results of instance acquisition
type InstanceAcquisitionResult struct {
	AcquiredInstances []int
	Conflicts         []InstanceConflict
	SkippedInstances  []int
	LaunchErrors      []string
}

// acquireInstances attempts to acquire emulator instances for a group
func (o *Orchestrator) acquireInstances(group *BotGroup, options LaunchOptions) ([]int, *InstanceAcquisitionResult) {
	result := &InstanceAcquisitionResult{
		AcquiredInstances: make([]int, 0, group.RequestedBotCount),
		Conflicts:         make([]InstanceConflict, 0),
		SkippedInstances:  make([]int, 0),
		LaunchErrors:      make([]string, 0),
	}

	fmt.Printf("[AcquireInstances] Group '%s': Requested=%d, Available instances=%v\n",
		group.Name, group.RequestedBotCount, group.AvailableInstances)

	// Discover running instances before checking availability
	if err := o.emulatorManager.DiscoverInstances(); err != nil {
		result.LaunchErrors = append(result.LaunchErrors,
			fmt.Sprintf("failed to discover emulator instances: %v", err))
		// Continue anyway - instances might still be launchable
	}

	// Phase 1: Determine which instances to use (check availability and conflicts)
	type instancePlan struct {
		instanceID int
		isRunning  bool
	}
	instancesPlanned := make([]instancePlan, 0, group.RequestedBotCount)

	// Refresh instance discovery before planning to get current state
	if err := o.emulatorManager.DiscoverInstances(); err != nil {
		fmt.Printf("[AcquireInstances] Warning: Failed to refresh instance discovery: %v\n", err)
	}

	for _, instanceID := range group.AvailableInstances {
		// Stop if we have enough planned
		if len(instancesPlanned) >= group.RequestedBotCount {
			fmt.Printf("[AcquireInstances] Planned enough instances (%d/%d)\n",
				len(instancesPlanned), group.RequestedBotCount)
			break
		}

		fmt.Printf("[AcquireInstances] Evaluating instance %d (planned=%d, needed=%d)\n",
			instanceID, len(instancesPlanned), group.RequestedBotCount)

		// Check availability
		available, conflictingGroup, err := o.checkInstanceAvailability(instanceID, group.Name)
		if err != nil {
			result.LaunchErrors = append(result.LaunchErrors,
				fmt.Sprintf("error checking instance %d: %v", instanceID, err))
			continue
		}

		// Handle conflicts
		if !available {
			conflict := InstanceConflict{
				InstanceID:       instanceID,
				CurrentGroupName: conflictingGroup,
				RequestedBy:      group.Name,
			}
			result.Conflicts = append(result.Conflicts, conflict)

			// Handle based on conflict resolution strategy
			switch options.OnConflict {
			case ConflictResolutionCancel:
				// Stop the other group's bot on this instance
				if err := o.stopBotOnInstance(conflictingGroup, instanceID); err != nil {
					result.LaunchErrors = append(result.LaunchErrors,
						fmt.Sprintf("failed to cancel instance %d from group '%s': %v",
							instanceID, conflictingGroup, err))
					result.SkippedInstances = append(result.SkippedInstances, instanceID)
					continue
				}
				// Instance is now available, proceed
			case ConflictResolutionSkip:
				// Skip this instance, try next
				result.SkippedInstances = append(result.SkippedInstances, instanceID)
				continue
			case ConflictResolutionAbort:
				// Abort entire launch
				return result.AcquiredInstances, result
			case ConflictResolutionAsk:
				// This should be handled by caller (GUI) before calling LaunchGroup
				// For now, treat as skip
				result.SkippedInstances = append(result.SkippedInstances, instanceID)
				continue
			}
		}

		// Check if emulator is running
		running, err := o.isEmulatorRunning(instanceID)
		if err != nil {
			result.LaunchErrors = append(result.LaunchErrors,
				fmt.Sprintf("error checking if instance %d is running: %v", instanceID, err))
			continue
		}

		// Add to plan
		instancesPlanned = append(instancesPlanned, instancePlan{
			instanceID: instanceID,
			isRunning:  running,
		})
		fmt.Printf("[AcquireInstances] Added instance %d to plan (running=%v)\n", instanceID, running)
	}

	if len(instancesPlanned) == 0 {
		result.LaunchErrors = append(result.LaunchErrors, "no instances available after conflict resolution")
		return result.AcquiredInstances, result
	}

	// Phase 2: Launch all instances that need launching
	for _, plan := range instancesPlanned {
		if !plan.isRunning {
			fmt.Printf("[AcquireInstances] Launching instance %d...\n", plan.instanceID)
			if _, err := o.launchEmulator(plan.instanceID); err != nil {
				result.LaunchErrors = append(result.LaunchErrors,
					fmt.Sprintf("failed to launch instance %d: %v", plan.instanceID, err))
				// Don't continue - we'll try to wait for it anyway in case it partially launched
			}
		}
	}

	// Phase 3: Wait for all instances to be ready
	for _, plan := range instancesPlanned {
		instanceID := plan.instanceID
		fmt.Printf("[AcquireInstances] Waiting for instance %d to be ready...\n", instanceID)

		// Refresh discovery one more time to ensure health monitor has current state
		if err := o.emulatorManager.DiscoverInstances(); err != nil {
			fmt.Printf("[AcquireInstances] Warning: Failed to refresh before wait: %v\n", err)
		}

		if err := o.waitForEmulatorReady(instanceID, options.EmulatorTimeout); err != nil {
			result.LaunchErrors = append(result.LaunchErrors,
				fmt.Sprintf("instance %d failed to become ready: %v", instanceID, err))
			continue
		}

		// Reserve instance
		if err := o.reserveInstance(instanceID, group.Name, instanceID, 0); err != nil {
			result.LaunchErrors = append(result.LaunchErrors,
				fmt.Sprintf("failed to reserve instance %d: %v", instanceID, err))
			continue
		}

		// Successfully acquired
		result.AcquiredInstances = append(result.AcquiredInstances, instanceID)
		fmt.Printf("[AcquireInstances] Successfully acquired instance %d (total: %d/%d)\n",
			instanceID, len(result.AcquiredInstances), group.RequestedBotCount)
	}

	return result.AcquiredInstances, result
}

// launchBotsStaggered launches bots with a staggered delay
func (o *Orchestrator) launchBotsStaggered(group *BotGroup, instances []int, options LaunchOptions) (int, []string) {
	launchedCount := 0
	errors := make([]string, 0)

	staggerDelay := options.StaggerDelay
	if staggerDelay == 0 {
		staggerDelay = o.staggerDelay
	}

	for i, instanceID := range instances {
		// Create bot for this instance
		bot, err := group.createBot(instanceID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to create bot for instance %d: %v", instanceID, err))
			// Release this instance
			o.releaseInstance(instanceID, group.Name)
			continue
		}

		// Create bot info
		botCtx, botCancel := context.WithCancel(group.ctx)
		botInfo := &BotInfo{
			Bot:           bot,
			InstanceID:    instanceID,
			StartedAt:     time.Now(),
			Status:        BotStatusStarting,
			routineCtx:    botCtx,
			routineCancel: botCancel,
		}

		// Add to active bots
		group.activeBotsMu.Lock()
		group.ActiveBots[instanceID] = botInfo
		group.activeBotsMu.Unlock()

		// Launch bot routine in background
		go o.runBotRoutine(group, botInfo, options.RestartPolicy)

		launchedCount++

		// Stagger next launch (except for last bot)
		if i < len(instances)-1 {
			time.Sleep(staggerDelay)
		}
	}

	return launchedCount, errors
}

// runBotRoutine executes a bot's routine with restart policy
func (o *Orchestrator) runBotRoutine(group *BotGroup, botInfo *BotInfo, policy RestartPolicy) {
	instanceID := botInfo.InstanceID

	// Guarantee cleanup runs regardless of panic or early return
	defer func() {
		// Recover from panics to ensure cleanup always runs
		if r := recover(); r != nil {
			fmt.Printf("[RunBotRoutine] PANIC in bot routine for instance %d: %v\n", instanceID, r)
			botInfo.Status = BotStatusFailed
			botInfo.Error = fmt.Errorf("panic: %v", r)
		}

		// Stop tracking this instance in health monitor
		o.healthMonitor.UntrackInstance(instanceID)
		fmt.Printf("[RunBotRoutine] Stopped health monitoring for instance %d\n", instanceID)

		// Remove from active bots
		group.activeBotsMu.Lock()
		delete(group.ActiveBots, instanceID)
		group.activeBotsMu.Unlock()

		// Release instance
		o.releaseInstance(instanceID, group.Name)

		// If all bots have finished, mark group as not running
		if group.GetActiveBotCount() == 0 {
			group.runningMu.Lock()
			group.running = false
			group.runningMu.Unlock()
		}
	}()

	// Register health callback to stop bot if instance becomes unhealthy
	o.healthMonitor.OnHealthChange(instanceID, func(id int, isReady, wasReady bool) {
		if wasReady && !isReady {
			// Instance went from healthy → unhealthy
			fmt.Printf("[BotGroup '%s'] Instance %d became unhealthy - stopping bot\n", group.Name, id)

			// Cancel the routine context to stop the bot gracefully
			botInfo.Status = BotStatusStopping
			botInfo.routineCancel()
		}
	})

	// Update status
	botInfo.Status = BotStatusRunning

	// Publish bot started event
	if o.eventBus != nil {
		o.eventBus.PublishAsync(events.NewBotStartedEvent(group.Name, instanceID))
	}

	// Execute with restart policy
	err := group.executeWithRestart(instanceID, group.RoutineName, policy)

	// Update status based on result and publish appropriate event
	if err != nil {
		botInfo.Status = BotStatusFailed
		botInfo.Error = err

		// Publish bot failed event
		if o.eventBus != nil {
			o.eventBus.PublishAsync(events.NewBotFailedEvent(group.Name, instanceID, err))
		}
	} else {
		botInfo.Status = BotStatusCompleted

		// Publish bot completed event
		if o.eventBus != nil {
			o.eventBus.PublishAsync(events.NewBotCompletedEvent(group.Name, instanceID))
		}
	}
}

// StopGroup stops all bots in a group
func (o *Orchestrator) StopGroup(groupName string) error {
	group, exists := o.GetGroup(groupName)
	if !exists {
		return fmt.Errorf("group '%s' not found", groupName)
	}

	if !group.IsRunning() {
		return fmt.Errorf("group '%s' is not running", groupName)
	}

	// Cancel all bot routines
	group.activeBotsMu.Lock()
	for _, botInfo := range group.ActiveBots {
		botInfo.Status = BotStatusStopping
		botInfo.routineCancel()
	}
	group.activeBotsMu.Unlock()

	// Shutdown all bots
	group.shutdownAllBots()

	// Release all account checkouts for this orchestration
	if o.db != nil && group.OrchestrationID != "" {
		released, err := database.ReleaseAllAccountsForOrchestration(o.db, group.OrchestrationID)
		if err != nil {
			fmt.Printf("Warning: Failed to release accounts for orchestration %s: %v\n", group.OrchestrationID, err)
		} else if released > 0 {
			fmt.Printf("Released %d account checkout(s) for orchestration %s\n", released, group.OrchestrationID)
		}
	}

	// Release all instances
	o.releaseAllInstances(groupName)

	// Clear active bots
	group.activeBotsMu.Lock()
	group.ActiveBots = make(map[int]*BotInfo)
	group.activeBotsMu.Unlock()

	// Mark as not running
	group.runningMu.Lock()
	group.running = false
	group.runningMu.Unlock()

	// Publish group stopped event
	if o.eventBus != nil {
		o.eventBus.PublishAsync(events.NewGroupStoppedEvent(groupName))
	}

	return nil
}

// stopBotOnInstance stops a specific bot instance from another group
func (o *Orchestrator) stopBotOnInstance(groupName string, instanceID int) error {
	group, exists := o.GetGroup(groupName)
	if !exists {
		return fmt.Errorf("group '%s' not found", groupName)
	}

	// Find and stop the bot
	botInfo, exists := group.GetBotInfo(instanceID)
	if !exists {
		// Bot not running on this instance
		return nil
	}

	// Cancel bot routine
	botInfo.Status = BotStatusStopping
	botInfo.routineCancel()

	// Shutdown bot
	group.shutdownBot(instanceID)

	// Release instance
	o.releaseInstance(instanceID, groupName)

	// Remove from active bots
	group.activeBotsMu.Lock()
	delete(group.ActiveBots, instanceID)
	group.activeBotsMu.Unlock()

	// Publish bot stopped event
	if o.eventBus != nil {
		o.eventBus.PublishAsync(events.NewBotStoppedEvent(groupName, instanceID))
	}

	return nil
}

// createTempRuntimeGroup creates a temporary runtime group from a definition
// This group is not stored in groupDefinitions and is meant for single-use execution
func (o *Orchestrator) createTempRuntimeGroup(runtimeName string, def *BotGroupDefinition) (*BotGroup, error) {
	o.groupsMu.Lock()
	defer o.groupsMu.Unlock()

	// Check if runtime name conflicts
	if _, exists := o.activeGroups[runtimeName]; exists {
		return nil, fmt.Errorf("runtime group '%s' already exists", runtimeName)
	}

	// Generate unique orchestration ID
	orchestrationID := uuid.New().String()

	// Create group
	ctx, cancel := context.WithCancel(context.Background())
	group := &BotGroup{
		Name:               runtimeName,
		OrchestrationID:    orchestrationID,
		orchestrator:       o,
		bots:               make(map[int]*Bot),
		RoutineName:        def.RoutineName,
		RoutineConfig:      def.RoutineConfig,
		AvailableInstances: def.AvailableInstances,
		RequestedBotCount:  def.RequestedBotCount,
		ActiveBots:         make(map[int]*BotInfo),
		AccountPoolName:    def.AccountPoolName,
		running:            false,
		ctx:                ctx,
		cancelFunc:         cancel,
	}

	fmt.Printf("Created temporary runtime group '%s' with orchestration ID: %s\n", runtimeName, orchestrationID)

	o.activeGroups[runtimeName] = group
	return group, nil
}
