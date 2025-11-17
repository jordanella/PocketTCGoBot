package bot

import (
	"context"
	"fmt"
	"sync"
	"time"

	"jordanella.com/pocket-tcg-go/internal/emulator"
	"jordanella.com/pocket-tcg-go/internal/events"
	"jordanella.com/pocket-tcg-go/internal/monitor"
)

// InstanceHealthStatus represents the health state of an emulator instance
type InstanceHealthStatus struct {
	InstanceID       int
	WindowDetected   bool
	ADBConnected     bool
	IsReady          bool
	LastCheckTime    time.Time
	ConsecutiveFails int
}

// HealthStatusCallback is called when an instance's health status changes
type HealthStatusCallback func(instanceID int, isReady bool, previousReady bool)

// OrchestratorHealthMonitor provides health monitoring for orchestrator instance launching
// It wraps the existing HealthChecker system to avoid duplicating polling logic
type OrchestratorHealthMonitor struct {
	emulatorManager *emulator.Manager

	// Instance health tracking
	instances   map[int]*InstanceHealthStatus
	instancesMu sync.RWMutex

	// Ready notifications
	readyChannels   map[int][]chan bool
	readyChannelsMu sync.Mutex

	// Health change callbacks
	callbacks   map[int][]HealthStatusCallback
	callbacksMu sync.RWMutex

	// Event bus for publishing health events
	eventBus events.EventBus

	// Background monitoring
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewOrchestratorHealthMonitor creates a new orchestrator health monitor
func NewOrchestratorHealthMonitor(emulatorManager *emulator.Manager) *OrchestratorHealthMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	return &OrchestratorHealthMonitor{
		emulatorManager: emulatorManager,
		instances:       make(map[int]*InstanceHealthStatus),
		readyChannels:   make(map[int][]chan bool),
		callbacks:       make(map[int][]HealthStatusCallback),
		eventBus:        nil, // Will be set by orchestrator via SetEventBus
		ctx:             ctx,
		cancel:          cancel,
	}
}

// SetEventBus sets the event bus for publishing health events
func (ohm *OrchestratorHealthMonitor) SetEventBus(eventBus events.EventBus) {
	ohm.eventBus = eventBus
}

// Start begins background health monitoring
func (ohm *OrchestratorHealthMonitor) Start() {
	ohm.wg.Add(1)
	go ohm.monitorInstances()
}

// Stop stops background monitoring
func (ohm *OrchestratorHealthMonitor) Stop() {
	ohm.cancel()
	ohm.wg.Wait()

	// Close all ready channels
	ohm.readyChannelsMu.Lock()
	defer ohm.readyChannelsMu.Unlock()

	for instanceID, channels := range ohm.readyChannels {
		for _, ch := range channels {
			close(ch)
		}
		delete(ohm.readyChannels, instanceID)
	}
}

// WaitForInstanceReady waits for an instance to become ready (window detected + ADB connected)
// This replaces the polling logic in waitForEmulatorReady
func (ohm *OrchestratorHealthMonitor) WaitForInstanceReady(instanceID int, timeout time.Duration) error {
	// Check if already ready
	if ohm.IsInstanceReady(instanceID) {
		return nil
	}

	// Create ready notification channel
	readyChan := make(chan bool, 1)

	ohm.readyChannelsMu.Lock()
	ohm.readyChannels[instanceID] = append(ohm.readyChannels[instanceID], readyChan)
	ohm.readyChannelsMu.Unlock()

	// Wait for ready signal or timeout
	select {
	case <-readyChan:
		return nil
	case <-time.After(timeout):
		// Remove channel from list
		ohm.readyChannelsMu.Lock()
		ohm.removeReadyChannel(instanceID, readyChan)
		ohm.readyChannelsMu.Unlock()

		return fmt.Errorf("timeout waiting for instance %d to be ready after %v", instanceID, timeout)
	case <-ohm.ctx.Done():
		// Remove channel from list to prevent leak
		ohm.readyChannelsMu.Lock()
		ohm.removeReadyChannel(instanceID, readyChan)
		ohm.readyChannelsMu.Unlock()

		return fmt.Errorf("health monitor stopped while waiting for instance %d", instanceID)
	}
}

// IsInstanceReady checks if an instance is currently ready
func (ohm *OrchestratorHealthMonitor) IsInstanceReady(instanceID int) bool {
	ohm.instancesMu.RLock()
	defer ohm.instancesMu.RUnlock()

	status, exists := ohm.instances[instanceID]
	if !exists {
		return false
	}

	return status.IsReady
}

// GetInstanceStatus returns the current health status of an instance
func (ohm *OrchestratorHealthMonitor) GetInstanceStatus(instanceID int) *InstanceHealthStatus {
	ohm.instancesMu.RLock()
	defer ohm.instancesMu.RUnlock()

	if status, exists := ohm.instances[instanceID]; exists {
		// Return a copy to avoid race conditions
		statusCopy := *status
		return &statusCopy
	}

	return nil
}

// OnHealthChange registers a callback for health status changes
// The callback is invoked when the instance's health status changes (ready <-> not ready)
func (ohm *OrchestratorHealthMonitor) OnHealthChange(instanceID int, callback HealthStatusCallback) {
	ohm.callbacksMu.Lock()
	defer ohm.callbacksMu.Unlock()

	ohm.callbacks[instanceID] = append(ohm.callbacks[instanceID], callback)
	fmt.Printf("[HealthMonitor] Registered health callback for instance %d\n", instanceID)
}

// monitorInstances runs in background and checks instance health periodically
func (ohm *OrchestratorHealthMonitor) monitorInstances() {
	defer ohm.wg.Done()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ohm.ctx.Done():
			return
		case <-ticker.C:
			ohm.checkAllInstances()
		}
	}
}

// checkAllInstances checks the health of all tracked instances
func (ohm *OrchestratorHealthMonitor) checkAllInstances() {
	// Rediscover instances to get updated window handles
	if err := ohm.emulatorManager.DiscoverInstances(); err != nil {
		// Log but don't stop monitoring
		fmt.Printf("Warning: Failed to discover instances during health check: %v\n", err)
	}

	// Notification queue to process outside of lock
	type notification struct {
		instanceID    int
		becameReady   bool
		statusChanged bool
		isReady       bool
		previousReady bool
	}
	notifications := make([]notification, 0)

	ohm.instancesMu.Lock()
	// Check each tracked instance
	for instanceID, status := range ohm.instances {
		previousReady := status.IsReady

		// Check window detection
		instance, err := ohm.emulatorManager.GetInstance(instanceID)
		status.WindowDetected = (err == nil && instance.MuMu != nil && instance.MuMu.WindowHandle != 0)

		// Check ADB connection - try to connect if not connected
		status.ADBConnected = false
		if status.WindowDetected {
			if instance.ADB == nil {
				// Try to connect ADB
				fmt.Printf("[HealthMonitor] Instance %d: Window detected, attempting ADB connection...\n", instanceID)
				if err := ohm.emulatorManager.ConnectInstance(instanceID); err != nil {
					fmt.Printf("[HealthMonitor] Instance %d: ADB connection failed: %v\n", instanceID, err)
				} else {
					// Re-fetch instance to get updated ADB connection
					instance, err = ohm.emulatorManager.GetInstance(instanceID)
					if err == nil && instance.ADB != nil {
						fmt.Printf("[HealthMonitor] Instance %d: ADB connection successful\n", instanceID)
					}
				}
			}

			// Test ADB connection if we have it
			if instance.ADB != nil {
				_, err := instance.ADB.Shell("echo test")
				status.ADBConnected = (err == nil)
			}
		}

		// Update ready state
		status.IsReady = status.WindowDetected && status.ADBConnected
		status.LastCheckTime = time.Now()

		// Track consecutive failures
		if !status.IsReady {
			status.ConsecutiveFails++
		} else {
			status.ConsecutiveFails = 0
		}

		// Queue notifications to process outside lock
		becameReady := !previousReady && status.IsReady
		statusChanged := previousReady != status.IsReady

		if becameReady || statusChanged {
			notifications = append(notifications, notification{
				instanceID:    instanceID,
				becameReady:   becameReady,
				statusChanged: statusChanged,
				isReady:       status.IsReady,
				previousReady: previousReady,
			})
		}
	}
	ohm.instancesMu.Unlock()

	// Process notifications without holding instancesMu
	// This prevents deadlock when callbacks or notifyInstanceReady acquire other locks
	for _, n := range notifications {
		if n.becameReady {
			ohm.notifyInstanceReady(n.instanceID)
		}
		if n.statusChanged {
			ohm.invokeHealthCallbacks(n.instanceID, n.isReady, n.previousReady)

			// Publish health change event
			if ohm.eventBus != nil {
				// Get current status for event data
				ohm.instancesMu.RLock()
				status := ohm.instances[n.instanceID]
				ohm.instancesMu.RUnlock()

				if status != nil {
					ohm.eventBus.PublishAsync(events.NewInstanceHealthChangedEvent(
						n.instanceID,
						n.isReady,
						n.previousReady,
						status.WindowDetected,
						status.ADBConnected,
					))
				}
			}
		}
	}
}

// TrackInstance starts tracking an instance's health
func (ohm *OrchestratorHealthMonitor) TrackInstance(instanceID int) {
	ohm.instancesMu.Lock()
	defer ohm.instancesMu.Unlock()

	if _, exists := ohm.instances[instanceID]; !exists {
		ohm.instances[instanceID] = &InstanceHealthStatus{
			InstanceID:       instanceID,
			WindowDetected:   false,
			ADBConnected:     false,
			IsReady:          false,
			LastCheckTime:    time.Now(),
			ConsecutiveFails: 0,
		}
	}
}

// UntrackInstance stops tracking an instance's health
func (ohm *OrchestratorHealthMonitor) UntrackInstance(instanceID int) {
	ohm.instancesMu.Lock()
	delete(ohm.instances, instanceID)
	ohm.instancesMu.Unlock()

	// Close any waiting channels
	ohm.readyChannelsMu.Lock()
	if channels, exists := ohm.readyChannels[instanceID]; exists {
		for _, ch := range channels {
			close(ch)
		}
		delete(ohm.readyChannels, instanceID)
	}
	ohm.readyChannelsMu.Unlock()

	// Remove callbacks
	ohm.callbacksMu.Lock()
	delete(ohm.callbacks, instanceID)
	ohm.callbacksMu.Unlock()
}

// notifyInstanceReady notifies all waiting goroutines that an instance is ready
// Must be called with instancesMu locked
func (ohm *OrchestratorHealthMonitor) notifyInstanceReady(instanceID int) {
	ohm.readyChannelsMu.Lock()
	defer ohm.readyChannelsMu.Unlock()

	if channels, exists := ohm.readyChannels[instanceID]; exists {
		for _, ch := range channels {
			select {
			case ch <- true:
			default:
				// Channel already has a value, skip
			}
		}
		// Clear the channels list
		delete(ohm.readyChannels, instanceID)
	}
}

// removeReadyChannel removes a specific channel from the ready channels list
// Must be called with readyChannelsMu locked
func (ohm *OrchestratorHealthMonitor) removeReadyChannel(instanceID int, ch chan bool) {
	if channels, exists := ohm.readyChannels[instanceID]; exists {
		// Find and remove the channel
		for i, c := range channels {
			if c == ch {
				ohm.readyChannels[instanceID] = append(channels[:i], channels[i+1:]...)
				break
			}
		}

		// If no more channels, remove the entry
		if len(ohm.readyChannels[instanceID]) == 0 {
			delete(ohm.readyChannels, instanceID)
		}
	}
}

// invokeHealthCallbacks invokes all registered callbacks for an instance
// Must be called with instancesMu locked (read or write)
func (ohm *OrchestratorHealthMonitor) invokeHealthCallbacks(instanceID int, isReady, previousReady bool) {
	// Get callbacks with read lock
	ohm.callbacksMu.RLock()
	callbacks, exists := ohm.callbacks[instanceID]
	ohm.callbacksMu.RUnlock()

	if !exists || len(callbacks) == 0 {
		return
	}

	// Log the health change
	if previousReady && !isReady {
		fmt.Printf("[HealthMonitor] Instance %d: Health changed: READY → UNHEALTHY\n", instanceID)
	} else if !previousReady && isReady {
		fmt.Printf("[HealthMonitor] Instance %d: Health changed: UNHEALTHY → READY\n", instanceID)
	}

	// Invoke callbacks in goroutines to avoid blocking health monitor
	for _, callback := range callbacks {
		go callback(instanceID, isReady, previousReady)
	}
}

// CreateBotHealthChecker creates a HealthChecker for a bot instance
// This integrates with the existing health monitoring system for runtime checks
func CreateBotHealthChecker(bot monitor.BotInterface) *monitor.HealthChecker {
	return monitor.NewHealthChecker(bot).
		WithCheckInterval(10 * time.Second).
		WithUnhealthyCallback(func(reason string, err error) {
			fmt.Printf("Bot %d: Health check failed - %s: %v\n", bot.Instance(), reason, err)
			// Recovery actions are handled by the bot's executeRecoveryAction
		})
}
