package bot

import (
	"fmt"
	"time"
)

// InstanceConflict represents a conflict where an instance is already in use
type InstanceConflict struct {
	InstanceID       int
	CurrentGroupName string
	CurrentBotID     int
	RequestedBy      string
}

// checkInstanceAvailability checks if an emulator instance is available
// Returns (available, conflictingGroup, error)
func (o *Orchestrator) checkInstanceAvailability(instanceID int, requestingGroup string) (bool, string, error) {
	o.instanceRegistryMu.RLock()
	defer o.instanceRegistryMu.RUnlock()

	assignment, exists := o.instanceRegistry[instanceID]
	if !exists {
		// Instance not assigned to anyone
		return true, "", nil
	}

	if assignment.GroupName == requestingGroup {
		// Already assigned to requesting group (probably re-launch)
		return true, "", nil
	}

	// Instance is assigned to another group
	return false, assignment.GroupName, nil
}

// reserveInstance marks an instance as in use by a specific group/bot
func (o *Orchestrator) reserveInstance(instanceID int, groupName string, botID int, emulatorPID int) error {
	o.instanceRegistryMu.Lock()
	defer o.instanceRegistryMu.Unlock()

	// Check if already reserved
	if assignment, exists := o.instanceRegistry[instanceID]; exists {
		if assignment.GroupName != groupName {
			return fmt.Errorf("instance %d is already reserved by group '%s'",
				instanceID, assignment.GroupName)
		}
	}

	// Reserve instance
	o.instanceRegistry[instanceID] = &InstanceAssignment{
		InstanceID:  instanceID,
		GroupName:   groupName,
		BotInstance: botID,
		AssignedAt:  time.Now(),
		IsRunning:   true,
		EmulatorPID: emulatorPID,
	}

	return nil
}

// releaseInstance frees an instance from the registry
func (o *Orchestrator) releaseInstance(instanceID int, groupName string) error {
	o.instanceRegistryMu.Lock()
	defer o.instanceRegistryMu.Unlock()

	assignment, exists := o.instanceRegistry[instanceID]
	if !exists {
		// Already released, not an error
		return nil
	}

	// Verify the group releasing matches the group that reserved
	if assignment.GroupName != groupName {
		return fmt.Errorf("instance %d is reserved by group '%s', cannot release from group '%s'",
			instanceID, assignment.GroupName, groupName)
	}

	delete(o.instanceRegistry, instanceID)
	return nil
}

// releaseAllInstances releases all instances held by a group
func (o *Orchestrator) releaseAllInstances(groupName string) {
	o.instanceRegistryMu.Lock()
	defer o.instanceRegistryMu.Unlock()

	// Find and remove all instances for this group
	for instanceID, assignment := range o.instanceRegistry {
		if assignment.GroupName == groupName {
			delete(o.instanceRegistry, instanceID)
		}
	}
}

// getInstanceAssignment retrieves the current assignment for an instance
func (o *Orchestrator) getInstanceAssignment(instanceID int) (*InstanceAssignment, bool) {
	o.instanceRegistryMu.RLock()
	defer o.instanceRegistryMu.RUnlock()

	assignment, exists := o.instanceRegistry[instanceID]
	if !exists {
		return nil, false
	}

	// Return copy to avoid race conditions
	assignmentCopy := *assignment
	return &assignmentCopy, true
}

// getAllInstanceAssignments returns all current instance assignments
func (o *Orchestrator) getAllInstanceAssignments() map[int]*InstanceAssignment {
	o.instanceRegistryMu.RLock()
	defer o.instanceRegistryMu.RUnlock()

	// Return copy to avoid race conditions
	assignments := make(map[int]*InstanceAssignment, len(o.instanceRegistry))
	for id, assignment := range o.instanceRegistry {
		assignmentCopy := *assignment
		assignments[id] = &assignmentCopy
	}
	return assignments
}

// findConflicts identifies all instances that would conflict with a launch request
func (o *Orchestrator) findConflicts(requestedInstances []int, requestingGroup string) []InstanceConflict {
	o.instanceRegistryMu.RLock()
	defer o.instanceRegistryMu.RUnlock()

	conflicts := make([]InstanceConflict, 0)

	for _, instanceID := range requestedInstances {
		assignment, exists := o.instanceRegistry[instanceID]
		if !exists {
			continue
		}

		// Skip if already assigned to requesting group
		if assignment.GroupName == requestingGroup {
			continue
		}

		// Found a conflict
		conflicts = append(conflicts, InstanceConflict{
			InstanceID:       instanceID,
			CurrentGroupName: assignment.GroupName,
			CurrentBotID:     assignment.BotInstance,
			RequestedBy:      requestingGroup,
		})
	}

	return conflicts
}

// isEmulatorRunning checks if an emulator instance is currently running
func (o *Orchestrator) isEmulatorRunning(instanceID int) (bool, error) {
	if o.emulatorManager == nil {
		return false, fmt.Errorf("emulator manager not configured")
	}

	// Use emulator manager to detect if instance is running
	instance, err := o.emulatorManager.GetInstance(instanceID)
	if err != nil {
		return false, fmt.Errorf("failed to get emulator instance %d: %w", instanceID, err)
	}

	// Check if window is detectable via MuMu instance
	if instance.MuMu == nil {
		return false, nil
	}

	// Check if window handle exists (indicates emulator is running)
	return instance.MuMu.WindowHandle != 0, nil
}

// launchEmulator starts an emulator instance
func (o *Orchestrator) launchEmulator(instanceID int) (int, error) {
	if o.emulatorManager == nil {
		return 0, fmt.Errorf("emulator manager not configured")
	}

	_, err := o.emulatorManager.GetInstance(instanceID)
	if err != nil {
		return 0, fmt.Errorf("failed to get emulator instance %d: %w", instanceID, err)
	}

	// TODO: Implement emulator launch
	// For now, return error as this requires OS-specific process launching
	return 0, fmt.Errorf("emulator launch not yet implemented - instance %d must be started manually", instanceID)
}

// waitForEmulatorReady waits for an emulator to be ready for bot operations
func (o *Orchestrator) waitForEmulatorReady(instanceID int, timeout time.Duration) error {
	if o.emulatorManager == nil {
		return fmt.Errorf("emulator manager not configured")
	}

	instance, err := o.emulatorManager.GetInstance(instanceID)
	if err != nil {
		return fmt.Errorf("failed to get emulator instance %d: %w", instanceID, err)
	}

	// Create timeout context
	startTime := time.Now()

	// Poll until window is detected and ADB is responsive
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Check timeout
		if time.Since(startTime) > timeout {
			return fmt.Errorf("timeout waiting for emulator instance %d to be ready", instanceID)
		}

		// Check if window handle exists
		if instance.MuMu == nil || instance.MuMu.WindowHandle == 0 {
			continue
		}

		// Check if ADB is responsive
		adb := instance.ADB
		if adb == nil {
			continue
		}

		// Try a simple ADB command
		if err := adb.Connect(); err == nil {
			// Emulator is ready
			return nil
		}
	}

	return fmt.Errorf("emulator instance %d never became ready", instanceID)
}
