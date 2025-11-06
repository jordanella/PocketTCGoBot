package monitor

import (
	"context"
	"sync"
	"time"
)

// HealthChecker type
type HealthChecker struct {
	bot              interface{}
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	lastActivityTime time.Time
	stuckCount       int
	stuckThreshold   int
	stuckTimeout     time.Duration
	mu               sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(bot interface{}) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	return &HealthChecker{
		bot:              bot,
		ctx:              ctx,
		cancel:           cancel,
		lastActivityTime: time.Now(),
		stuckCount:       0,
		stuckThreshold:   3,
		stuckTimeout:     30 * time.Second,
	}
}

// Start begins health monitoring
func (hc *HealthChecker) Start() {
	hc.wg.Add(1)
	go hc.monitorStuck()
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
			// TODO: Trigger stuck recovery
			// Reset counter after triggering
			hc.stuckCount = 0
		}
	} else {
		// Activity detected, reset counter
		hc.stuckCount = 0
	}
}
