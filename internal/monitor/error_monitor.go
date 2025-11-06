package monitor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ErrorMonitor watches for game errors and communicates them to executing routines
type ErrorMonitor struct {
	bot              interface{} // Use interface to avoid circular import
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	errorEvents      chan ErrorEvent      // Main channel for sending errors to routines
	detectionEnabled atomic.Bool          // Can be disabled during critical operations
	handlers         []ErrorHandler       // Registered error detection handlers
	mu               sync.RWMutex         // Protects handlers slice
	checkInterval    map[Priority]time.Duration // How often to check each priority level
}

// NewErrorMonitor creates a new error monitor
func NewErrorMonitor(bot interface{}) *ErrorMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	em := &ErrorMonitor{
		bot:           bot,
		ctx:           ctx,
		cancel:        cancel,
		errorEvents:   make(chan ErrorEvent, 100), // Buffered to prevent blocking
		handlers:      make([]ErrorHandler, 0),
		checkInterval: map[Priority]time.Duration{
			PriorityCritical: 1 * time.Second,  // Fast polling for critical errors
			PriorityHigh:     3 * time.Second,  // Medium polling for popups
			PriorityMedium:   5 * time.Second,  // Slower polling for warnings
			PriorityLow:      10 * time.Second, // Slowest for health checks
		},
	}
	em.detectionEnabled.Store(true)
	return em
}

// Start begins monitoring for errors with multiple polling goroutines
func (em *ErrorMonitor) Start() {
	em.wg.Add(3)
	go em.monitorCriticalErrors()
	go em.monitorHighPriorityErrors()
	go em.monitorMediumPriorityErrors()
}

// Stop gracefully shuts down the error monitor
func (em *ErrorMonitor) Stop() {
	em.cancel()
	em.wg.Wait()
	close(em.errorEvents)
}

// RegisterHandler adds an error detection handler
func (em *ErrorMonitor) RegisterHandler(handler ErrorHandler) {
	em.mu.Lock()
	defer em.mu.Unlock()
	em.handlers = append(em.handlers, handler)
}

// GetErrorChannel returns the error events channel for routines to listen on
func (em *ErrorMonitor) GetErrorChannel() <-chan ErrorEvent {
	return em.errorEvents
}

// EnableDetection enables error detection
func (em *ErrorMonitor) EnableDetection() {
	em.detectionEnabled.Store(true)
}

// DisableDetection temporarily disables error detection
// Useful during critical operations where interruption is not desired
func (em *ErrorMonitor) DisableDetection() {
	em.detectionEnabled.Store(false)
}

// IsDetectionEnabled returns whether error detection is currently enabled
func (em *ErrorMonitor) IsDetectionEnabled() bool {
	return em.detectionEnabled.Load()
}

// TriggerError manually sends an error event (for external error detection)
// Returns true if the event was sent, false if channel is full
func (em *ErrorMonitor) TriggerError(errorType ErrorType, severity ErrorSeverity, message string, template interface{}) bool {
	if !em.detectionEnabled.Load() {
		return false
	}

	event := ErrorEvent{
		Type:         errorType,
		Severity:     severity,
		Template:     template,
		DetectedAt:   time.Now(),
		Message:      message,
		Context:      make(map[string]interface{}),
		ResponseChan: make(chan ErrorResponse, 1), // Buffered for async response
	}

	select {
	case em.errorEvents <- event:
		return true
	default:
		// Channel full - error is dropped
		// TODO: Log this situation
		return false
	}
}

// TriggerErrorWithContext manually sends an error event with additional context
func (em *ErrorMonitor) TriggerErrorWithContext(errorType ErrorType, severity ErrorSeverity, message string, template interface{}, context map[string]interface{}) bool {
	if !em.detectionEnabled.Load() {
		return false
	}

	event := ErrorEvent{
		Type:         errorType,
		Severity:     severity,
		Template:     template,
		DetectedAt:   time.Now(),
		Message:      message,
		Context:      context,
		ResponseChan: make(chan ErrorResponse, 1),
	}

	select {
	case em.errorEvents <- event:
		return true
	default:
		return false
	}
}

// WaitForResponse waits for a response on an error event's response channel
// Returns the response or an error if timeout occurs
func (em *ErrorMonitor) WaitForResponse(event ErrorEvent, timeout time.Duration) (ErrorResponse, error) {
	select {
	case response := <-event.ResponseChan:
		return response, nil
	case <-time.After(timeout):
		return ErrorResponse{}, fmt.Errorf("timeout waiting for error response")
	case <-em.ctx.Done():
		return ErrorResponse{}, em.ctx.Err()
	}
}

// Monitoring loops - each runs at different intervals based on priority

func (em *ErrorMonitor) monitorCriticalErrors() {
	defer em.wg.Done()

	ticker := time.NewTicker(em.checkInterval[PriorityCritical])
	defer ticker.Stop()

	for {
		select {
		case <-em.ctx.Done():
			return
		case <-ticker.C:
			if !em.detectionEnabled.Load() {
				continue
			}

			em.mu.RLock()
			handlers := em.handlers
			em.mu.RUnlock()

			for _, handler := range handlers {
				if handler.Priority == PriorityCritical {
					// TODO: Actual error detection will be implemented by you
					// This is where you would:
					// 1. Check template existence using handler.Template
					// 2. If detected, create and send ErrorEvent
					// 3. Optionally wait for response depending on severity
				}
			}
		}
	}
}

func (em *ErrorMonitor) monitorHighPriorityErrors() {
	defer em.wg.Done()

	ticker := time.NewTicker(em.checkInterval[PriorityHigh])
	defer ticker.Stop()

	for {
		select {
		case <-em.ctx.Done():
			return
		case <-ticker.C:
			if !em.detectionEnabled.Load() {
				continue
			}

			em.mu.RLock()
			handlers := em.handlers
			em.mu.RUnlock()

			for _, handler := range handlers {
				if handler.Priority == PriorityHigh {
					// TODO: Actual error detection implementation
					// Example for popup detection:
					// if bot.CV().FindTemplate(handler.Template, config) {
					//     em.TriggerError(handler.ErrorType, SeverityHigh, "Popup detected", handler.Template)
					// }
				}
			}
		}
	}
}

func (em *ErrorMonitor) monitorMediumPriorityErrors() {
	defer em.wg.Done()

	ticker := time.NewTicker(em.checkInterval[PriorityMedium])
	defer ticker.Stop()

	for {
		select {
		case <-em.ctx.Done():
			return
		case <-ticker.C:
			if !em.detectionEnabled.Load() {
				continue
			}

			em.mu.RLock()
			handlers := em.handlers
			em.mu.RUnlock()

			for _, handler := range handlers {
				if handler.Priority == PriorityMedium || handler.Priority == PriorityLow {
					// TODO: Actual error detection implementation
				}
			}
		}
	}
}
