package actions

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SentryEngine manages parallel execution of sentry routines
type SentryEngine struct {
	bot       BotInterface
	sentries  []Sentry
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.RWMutex
	isRunning bool

	// Metrics tracking
	metrics map[string]*SentryMetrics // Map of routine name -> metrics
}

// NewSentryEngine creates a new sentry engine
func NewSentryEngine(bot BotInterface, sentries []Sentry) *SentryEngine {
	ctx, cancel := context.WithCancel(bot.Context())

	// Initialize metrics for each sentry
	metrics := make(map[string]*SentryMetrics)
	for _, sentry := range sentries {
		metrics[sentry.Routine] = NewSentryMetrics()
	}

	return &SentryEngine{
		bot:      bot,
		sentries: sentries,
		ctx:      ctx,
		cancel:   cancel,
		metrics:  metrics,
	}
}

// Start begins all sentry monitoring routines in parallel
func (se *SentryEngine) Start() error {
	se.mu.Lock()
	defer se.mu.Unlock()

	if se.isRunning {
		return fmt.Errorf("sentry engine already running")
	}

	// Validate all sentries have their routine builders loaded
	for i, sentry := range se.sentries {
		if sentry.GetRoutineBuilder() == nil {
			return fmt.Errorf("sentry %d (%s): routine builder not loaded", i, sentry.Routine)
		}
	}

	// Start a goroutine for each sentry
	for i := range se.sentries {
		se.wg.Add(1)
		go se.runSentry(&se.sentries[i])
	}

	se.isRunning = true
	return nil
}

// Stop gracefully shuts down all sentry routines
func (se *SentryEngine) Stop() {
	se.mu.Lock()
	if !se.isRunning {
		se.mu.Unlock()
		return
	}
	se.mu.Unlock()

	se.cancel()
	se.wg.Wait()

	se.mu.Lock()
	se.isRunning = false
	se.mu.Unlock()
}

// runSentry executes a single sentry routine on its polling interval
func (se *SentryEngine) runSentry(sentry *Sentry) {
	defer se.wg.Done()

	ticker := time.NewTicker(sentry.GetFrequency())
	defer ticker.Stop()

	for {
		select {
		case <-se.ctx.Done():
			return

		case <-ticker.C:
			// Check if main routine is stopped
			if se.bot.IsStopped() {
				return
			}

			// Execute the sentry routine
			se.executeSentry(sentry)
		}
	}
}

// executeSentry executes a sentry routine and handles the result
func (se *SentryEngine) executeSentry(sentry *Sentry) {
	// Track execution start time
	startTime := time.Now()

	// Log execution start (severity-based)
	se.logSentry(sentry, "Executing sentry routine")

	// Get the routine builder
	builder := sentry.GetRoutineBuilder()
	if builder == nil {
		se.logSentry(sentry, "ERROR: routine builder is nil")
		return
	}

	// Mark builder as sentry execution so it ignores halt signals
	builder.AsSentryExecution()

	// Get controller for result handling (but don't pause yet)
	controller := se.getRoutineController()

	// Execute the sentry routine (runs in parallel with main routine)
	err := builder.Execute(se.bot)

	// Record execution metrics
	duration := time.Since(startTime)
	if metrics := se.metrics[sentry.Routine]; metrics != nil {
		metrics.RecordExecution(duration, err)
	}

	// Handle result based on success/failure
	if controller != nil {
		se.handleSentryResult(sentry, controller, err)
	}
}

// handleSentryResult processes the sentry execution result and updates routine state
func (se *SentryEngine) handleSentryResult(sentry *Sentry, controller RoutineControllerInterface, err error) {
	var action SentryAction
	if err == nil {
		// Success: routine returned nil error
		action = sentry.OnSuccess
		se.logSentry(sentry, fmt.Sprintf("Sentry succeeded, action: %s", action))
	} else {
		// Failure: routine returned error
		action = sentry.OnFailure
		se.logSentry(sentry, fmt.Sprintf("Sentry failed (%v), action: %s", err, action))
	}

	// Record action metrics
	if metrics := se.metrics[sentry.Routine]; metrics != nil {
		metrics.RecordAction(action)
	}

	// Execute the appropriate action
	switch action {
	case SentryActionResume:
		controller.Resume()

	case SentryActionPause:
		// Keep paused, do nothing

	case SentryActionStop:
		// Graceful stop - let main routine finish current step
		controller.ForceStop()

	case SentryActionForceStop:
		// Immediate stop
		controller.ForceStop()

	default:
		// Unknown action, default to resume
		se.logSentry(sentry, fmt.Sprintf("Unknown action '%s', defaulting to resume", action))
		controller.Resume()
	}
}

// getRoutineController safely retrieves the routine controller from the bot
func (se *SentryEngine) getRoutineController() RoutineControllerInterface {
	// We need to extend BotInterface to include RoutineController access
	// For now, we'll use a type assertion
	type routineControllerProvider interface {
		RoutineController() RoutineControllerInterface
	}

	if provider, ok := se.bot.(routineControllerProvider); ok {
		return provider.RoutineController()
	}
	return nil
}

// logSentry logs a message with severity-based prefix
func (se *SentryEngine) logSentry(sentry *Sentry, message string) {
	prefix := ""
	switch sentry.Severity {
	case SentrySeverityCritical:
		prefix = "[CRITICAL]"
	case SentrySeverityHigh:
		prefix = "[HIGH]"
	case SentrySeverityMedium:
		prefix = "[MEDIUM]"
	case SentrySeverityLow:
		prefix = "[LOW]"
	}

	fmt.Printf("%s [Sentry:%s] %s\n", prefix, sentry.Routine, message)
}

// GetMetrics returns the metrics for a specific sentry routine
func (se *SentryEngine) GetMetrics(routineName string) *SentryMetrics {
	se.mu.RLock()
	defer se.mu.RUnlock()

	return se.metrics[routineName]
}

// GetAllMetrics returns all sentry metrics as a map
func (se *SentryEngine) GetAllMetrics() map[string]SentryStats {
	se.mu.RLock()
	defer se.mu.RUnlock()

	stats := make(map[string]SentryStats)
	for name, metrics := range se.metrics {
		stats[name] = metrics.GetStats()
	}
	return stats
}

// CheckSentryHealth checks if all sentries are healthy
// Returns a list of unhealthy sentry names and their error counts
func (se *SentryEngine) CheckSentryHealth(consecutiveErrorThreshold int64) []string {
	se.mu.RLock()
	defer se.mu.RUnlock()

	var unhealthy []string
	for name, metrics := range se.metrics {
		if !metrics.IsHealthy(consecutiveErrorThreshold) {
			unhealthy = append(unhealthy, name)
		}
	}
	return unhealthy
}

// GetSentryCount returns the total number of sentries being monitored
func (se *SentryEngine) GetSentryCount() int {
	se.mu.RLock()
	defer se.mu.RUnlock()

	return len(se.sentries)
}
