package bot

import (
	"context"
	"fmt"
	"time"

	"jordanella.com/pocket-tcg-go/internal/database"
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

// LaunchGroup starts all bots in a group with full orchestration
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
		// Account pool is already set on group, no need for manager
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
		o.releaseAllInstances(groupName)
		return result, fmt.Errorf("failed to launch any bots")
	}

	// Mark group as running
	group.runningMu.Lock()
	group.running = true
	group.runningMu.Unlock()

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

	// Discover running instances before checking availability
	if err := o.emulatorManager.DiscoverInstances(); err != nil {
		result.LaunchErrors = append(result.LaunchErrors,
			fmt.Sprintf("failed to discover emulator instances: %v", err))
		// Continue anyway - instances might still be launchable
	}

	// Check all available instances
	for _, instanceID := range group.AvailableInstances {
		// Stop if we have enough
		if len(result.AcquiredInstances) >= group.RequestedBotCount {
			break
		}

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

		var emulatorPID int
		if !running {
			// Launch emulator
			pid, err := o.launchEmulator(instanceID)
			if err != nil {
				result.LaunchErrors = append(result.LaunchErrors,
					fmt.Sprintf("failed to launch instance %d: %v", instanceID, err))
				continue
			}
			emulatorPID = pid

			// Wait for emulator to be ready
			if err := o.waitForEmulatorReady(instanceID, options.EmulatorTimeout); err != nil {
				result.LaunchErrors = append(result.LaunchErrors,
					fmt.Sprintf("instance %d failed to become ready: %v", instanceID, err))
				continue
			}
		}

		// Reserve instance
		if err := o.reserveInstance(instanceID, group.Name, instanceID, emulatorPID); err != nil {
			result.LaunchErrors = append(result.LaunchErrors,
				fmt.Sprintf("failed to reserve instance %d: %v", instanceID, err))
			continue
		}

		// Successfully acquired
		result.AcquiredInstances = append(result.AcquiredInstances, instanceID)
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
	// Update status
	botInfo.Status = BotStatusRunning

	// Execute with restart policy
	err := group.executeWithRestart(botInfo.InstanceID, group.RoutineName, policy)

	// Update status based on result
	if err != nil {
		botInfo.Status = BotStatusFailed
		botInfo.Error = err
	} else {
		botInfo.Status = BotStatusCompleted
	}

	// Remove from active bots
	group.activeBotsMu.Lock()
	delete(group.ActiveBots, botInfo.InstanceID)
	group.activeBotsMu.Unlock()

	// Release instance
	o.releaseInstance(botInfo.InstanceID, group.Name)

	// If all bots have finished, mark group as not running
	if group.GetActiveBotCount() == 0 {
		group.runningMu.Lock()
		group.running = false
		group.runningMu.Unlock()
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

	return nil
}
