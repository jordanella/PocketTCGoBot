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
		bot:              botInterface,
		ctx:              ctx,
		cancel:           cancel,
		lastActivityTime: time.Now(),
		stuckCount:       0,
		stuckThreshold:   3,
		stuckTimeout:     30 * time.Second,
		checkInterval:    10 * time.Second, // Check every 10 seconds
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
	// Check ADB connection
	if err := hc.CheckADBConnection(); err != nil {
		if hc.onUnhealthy != nil {
			hc.onUnhealthy("adb_connection_lost", err)
		}
		return
	}

	// Check device responsiveness
	if err := hc.CheckDeviceResponsive(); err != nil {
		if hc.onUnhealthy != nil {
			hc.onUnhealthy("device_unresponsive", err)
		}
		return
	}
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
