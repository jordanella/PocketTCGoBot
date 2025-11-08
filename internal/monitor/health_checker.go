package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BotInterface defines the minimal interface needed for health checking
type BotInterface interface {
	ADB() ADBInterface
	Instance() int
}

// ADBInterface defines the minimal ADB interface needed for health checking
type ADBInterface interface {
	Shell(ctx context.Context, command string) (string, error)
	ScreenBounds() (int, int, error)
	GetDeviceSerial() string
}

// UnhealthyCallback is called when the bot becomes unhealthy
type UnhealthyCallback func(reason string, err error)

// HealthChecker type
type HealthChecker struct {
	bot              BotInterface
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	lastActivityTime time.Time
	stuckCount       int
	stuckThreshold   int
	stuckTimeout     time.Duration
	checkInterval    time.Duration
	onUnhealthy      UnhealthyCallback
	mu               sync.RWMutex

	// Enhanced monitoring
	consecutiveFailures int
	failureThreshold    int
	lastScreenHash      string
	frozenCheckCount    int
	frozenThreshold     int
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(bot interface{}) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())

	// Type assert to BotInterface
	botInterface, ok := bot.(BotInterface)
	if !ok {
		// Fallback: create a health checker that won't do ADB checks
		return &HealthChecker{
			bot:              nil,
			ctx:              ctx,
			cancel:           cancel,
			lastActivityTime: time.Now(),
			stuckCount:       0,
			stuckThreshold:   3,
			stuckTimeout:     30 * time.Second,
			checkInterval:    10 * time.Second,
		}
	}

	return &HealthChecker{
		bot:                 botInterface,
		ctx:                 ctx,
		cancel:              cancel,
		lastActivityTime:    time.Now(),
		stuckCount:          0,
		stuckThreshold:      3,
		stuckTimeout:        30 * time.Second,
		checkInterval:       10 * time.Second, // Check every 10 seconds
		consecutiveFailures: 0,
		failureThreshold:    3,
		frozenCheckCount:    0,
		frozenThreshold:     3, // Consider frozen after 3 identical screens
	}
}

// WithUnhealthyCallback sets the callback for unhealthy events
func (hc *HealthChecker) WithUnhealthyCallback(callback UnhealthyCallback) *HealthChecker {
	hc.onUnhealthy = callback
	return hc
}

// WithCheckInterval sets the health check interval
func (hc *HealthChecker) WithCheckInterval(interval time.Duration) *HealthChecker {
	hc.checkInterval = interval
	return hc
}

// Start begins health monitoring
func (hc *HealthChecker) Start() {
	hc.wg.Add(2)
	go hc.monitorStuck()
	go hc.monitorHealth()
}

// Stop stops health monitoring
func (hc *HealthChecker) Stop() {
	hc.cancel()
	hc.wg.Wait()
}

// RecordActivity records bot activity to prevent stuck detection
func (hc *HealthChecker) RecordActivity() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.lastActivityTime = time.Now()
	hc.stuckCount = 0
}

// Private
func (hc *HealthChecker) monitorStuck() {
	defer hc.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-ticker.C:
			hc.checkIfStuck()
		}
	}
}

func (hc *HealthChecker) checkIfStuck() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	timeSinceActivity := time.Since(hc.lastActivityTime)

	if timeSinceActivity > hc.stuckTimeout {
		hc.stuckCount++

		// If stuck count exceeds threshold, trigger recovery
		if hc.stuckCount >= hc.stuckThreshold {
			if hc.onUnhealthy != nil {
				hc.onUnhealthy("bot_stuck", fmt.Errorf("no activity for %v", timeSinceActivity))
			}
			// Reset counter after triggering
			hc.stuckCount = 0
		}
	} else {
		// Activity detected, reset counter
		hc.stuckCount = 0
	}
}

// monitorHealth performs periodic health checks
func (hc *HealthChecker) monitorHealth() {
	defer hc.wg.Done()

	// Skip monitoring if bot interface is nil
	if hc.bot == nil {
		return
	}

	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-ticker.C:
			hc.performHealthChecks()
		}
	}
}

// performHealthChecks runs all health checks
func (hc *HealthChecker) performHealthChecks() {
	hc.mu.Lock()
	hasFailure := false
	hc.mu.Unlock()

	// Check ADB connection
	if err := hc.CheckADBConnection(); err != nil {
		hasFailure = true
		hc.incrementFailureCount()
		if hc.onUnhealthy != nil {
			hc.onUnhealthy("adb_connection_lost", err)
		}
		return
	}

	// Check instance window exists
	if err := hc.CheckInstanceWindow(); err != nil {
		hasFailure = true
		hc.incrementFailureCount()
		if hc.onUnhealthy != nil {
			hc.onUnhealthy("instance_window_missing", err)
		}
		return
	}

	// Check device responsiveness
	if err := hc.CheckDeviceResponsive(); err != nil {
		hasFailure = true
		hc.incrementFailureCount()
		if hc.onUnhealthy != nil {
			hc.onUnhealthy("device_unresponsive", err)
		}
		return
	}

	// Check if screen is frozen
	if err := hc.CheckScreenFrozen(); err != nil {
		hasFailure = true
		hc.incrementFailureCount()
		if hc.onUnhealthy != nil {
			hc.onUnhealthy("screen_frozen", err)
		}
		return
	}

	// All checks passed - reset failure count
	if !hasFailure {
		hc.ResetFailureCount()
	}
}

// incrementFailureCount increments the consecutive failure counter
func (hc *HealthChecker) incrementFailureCount() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.consecutiveFailures++
}

// CheckADBConnection verifies the ADB connection is alive
func (hc *HealthChecker) CheckADBConnection() error {
	if hc.bot == nil {
		return nil // Skip check if no bot
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Simple command to test ADB connectivity
	_, err := hc.bot.ADB().Shell(ctx, "echo test")
	if err != nil {
		return fmt.Errorf("ADB connection check failed for instance %d: %w", hc.bot.Instance(), err)
	}

	return nil
}

// CheckDeviceResponsive verifies the device is responding to commands
func (hc *HealthChecker) CheckDeviceResponsive() error {
	if hc.bot == nil {
		return nil // Skip check if no bot
	}

	// Try to get screen bounds - if device is frozen, this will fail or timeout
	_, _, err := hc.bot.ADB().ScreenBounds()
	if err != nil {
		return fmt.Errorf("device responsiveness check failed for instance %d: %w", hc.bot.Instance(), err)
	}

	return nil
}

// CheckInstanceWindow verifies the emulator/device window exists
func (hc *HealthChecker) CheckInstanceWindow() error {
	if hc.bot == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if device is still connected via ADB
	serial := hc.bot.ADB().GetDeviceSerial()
	output, err := hc.bot.ADB().Shell(ctx, "getprop ro.build.version.release")
	if err != nil {
		return fmt.Errorf("instance window check failed for %s (instance %d): device not responding", serial, hc.bot.Instance())
	}

	if output == "" {
		return fmt.Errorf("instance window check failed for %s (instance %d): empty response from device", serial, hc.bot.Instance())
	}

	return nil
}

// CheckScreenFrozen detects if the screen appears to be frozen
func (hc *HealthChecker) CheckScreenFrozen() error {
	if hc.bot == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get current screen state via dumpsys
	output, err := hc.bot.ADB().Shell(ctx, "dumpsys window | grep mCurrentFocus")
	if err != nil {
		// Don't treat this as frozen, just unable to check
		return nil
	}

	hc.mu.Lock()
	defer hc.mu.Unlock()

	// Check if screen state hasn't changed
	if output == hc.lastScreenHash && output != "" {
		hc.frozenCheckCount++
		if hc.frozenCheckCount >= hc.frozenThreshold {
			hc.frozenCheckCount = 0 // Reset for next detection
			return fmt.Errorf("screen appears frozen for instance %d: same focus for %d checks", hc.bot.Instance(), hc.frozenThreshold)
		}
	} else {
		hc.frozenCheckCount = 0
		hc.lastScreenHash = output
	}

	return nil
}

// CheckProcessRunning verifies the target app process is running
func (hc *HealthChecker) CheckProcessRunning(packageName string) error {
	if hc.bot == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if package is running
	output, err := hc.bot.ADB().Shell(ctx, fmt.Sprintf("pidof %s", packageName))
	if err != nil {
		return fmt.Errorf("failed to check process for package %s on instance %d: %w", packageName, hc.bot.Instance(), err)
	}

	if output == "" || output == "error: closed" {
		return fmt.Errorf("process not running for package %s on instance %d", packageName, hc.bot.Instance())
	}

	return nil
}

// GetHealthStatus returns a comprehensive health status
func (hc *HealthChecker) GetHealthStatus() map[string]interface{} {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	return map[string]interface{}{
		"last_activity":        hc.lastActivityTime,
		"time_since_activity":  time.Since(hc.lastActivityTime),
		"stuck_count":          hc.stuckCount,
		"consecutive_failures": hc.consecutiveFailures,
		"frozen_check_count":   hc.frozenCheckCount,
		"is_healthy":           hc.consecutiveFailures < hc.failureThreshold,
	}
}

// ResetFailureCount resets the consecutive failure counter
func (hc *HealthChecker) ResetFailureCount() {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.consecutiveFailures = 0
}
