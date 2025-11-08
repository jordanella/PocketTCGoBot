package actions

import (
	"sync"
	"time"
)

// SentryMetrics tracks execution statistics for a sentry routine
type SentryMetrics struct {
	mu sync.RWMutex

	// Execution counts
	TotalExecutions   int64
	SuccessCount      int64
	FailureCount      int64
	LastExecutionTime time.Time

	// Timing statistics
	TotalDuration    time.Duration
	MinDuration      time.Duration
	MaxDuration      time.Duration
	AverageDuration  time.Duration

	// Action counts
	ResumeActions     int64
	PauseActions      int64
	StopActions       int64
	ForceStopActions  int64

	// Error tracking
	LastError         error
	LastErrorTime     time.Time
	ConsecutiveErrors int64
}

// NewSentryMetrics creates a new metrics tracker
func NewSentryMetrics() *SentryMetrics {
	return &SentryMetrics{
		MinDuration: time.Duration(1<<63 - 1), // Max duration initially
	}
}

// RecordExecution records a sentry execution with timing and result
func (sm *SentryMetrics) RecordExecution(duration time.Duration, err error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.TotalExecutions++
	sm.LastExecutionTime = time.Now()

	// Update timing stats
	sm.TotalDuration += duration
	if duration < sm.MinDuration {
		sm.MinDuration = duration
	}
	if duration > sm.MaxDuration {
		sm.MaxDuration = duration
	}
	sm.AverageDuration = sm.TotalDuration / time.Duration(sm.TotalExecutions)

	// Update success/failure counts
	if err == nil {
		sm.SuccessCount++
		sm.ConsecutiveErrors = 0
	} else {
		sm.FailureCount++
		sm.ConsecutiveErrors++
		sm.LastError = err
		sm.LastErrorTime = time.Now()
	}
}

// RecordAction records which action was taken after execution
func (sm *SentryMetrics) RecordAction(action SentryAction) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	switch action {
	case SentryActionResume:
		sm.ResumeActions++
	case SentryActionPause:
		sm.PauseActions++
	case SentryActionStop:
		sm.StopActions++
	case SentryActionForceStop:
		sm.ForceStopActions++
	}
}

// GetErrorRate returns the error rate as a percentage (0-100)
func (sm *SentryMetrics) GetErrorRate() float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.TotalExecutions == 0 {
		return 0.0
	}

	return float64(sm.FailureCount) / float64(sm.TotalExecutions) * 100.0
}

// GetSuccessRate returns the success rate as a percentage (0-100)
func (sm *SentryMetrics) GetSuccessRate() float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.TotalExecutions == 0 {
		return 0.0
	}

	return float64(sm.SuccessCount) / float64(sm.TotalExecutions) * 100.0
}

// IsHealthy returns whether the sentry is considered healthy
// A sentry is unhealthy if it has consecutive errors exceeding the threshold
func (sm *SentryMetrics) IsHealthy(consecutiveErrorThreshold int64) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if consecutiveErrorThreshold <= 0 {
		consecutiveErrorThreshold = 3 // Default threshold
	}

	return sm.ConsecutiveErrors < consecutiveErrorThreshold
}

// GetStats returns a snapshot of current metrics (thread-safe)
func (sm *SentryMetrics) GetStats() SentryStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return SentryStats{
		TotalExecutions:   sm.TotalExecutions,
		SuccessCount:      sm.SuccessCount,
		FailureCount:      sm.FailureCount,
		LastExecutionTime: sm.LastExecutionTime,
		AverageDuration:   sm.AverageDuration,
		MinDuration:       sm.MinDuration,
		MaxDuration:       sm.MaxDuration,
		ErrorRate:         float64(sm.FailureCount) / float64(max(sm.TotalExecutions, 1)) * 100.0,
		SuccessRate:       float64(sm.SuccessCount) / float64(max(sm.TotalExecutions, 1)) * 100.0,
		ConsecutiveErrors: sm.ConsecutiveErrors,
		LastError:         sm.LastError,
		LastErrorTime:     sm.LastErrorTime,
	}
}

// SentryStats is a snapshot of sentry metrics (immutable)
type SentryStats struct {
	TotalExecutions   int64
	SuccessCount      int64
	FailureCount      int64
	LastExecutionTime time.Time
	AverageDuration   time.Duration
	MinDuration       time.Duration
	MaxDuration       time.Duration
	ErrorRate         float64
	SuccessRate       float64
	ConsecutiveErrors int64
	LastError         error
	LastErrorTime     time.Time
}

// max returns the maximum of two int64 values
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
