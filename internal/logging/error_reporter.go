package logging

import (
	"fmt"
	"sync"
	"time"
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	ErrorCategoryOrchestration ErrorCategory = "orchestration"
	ErrorCategoryBot           ErrorCategory = "bot"
	ErrorCategoryEmulator      ErrorCategory = "emulator"
	ErrorCategoryAccountPool   ErrorCategory = "account_pool"
	ErrorCategoryNetwork       ErrorCategory = "network"
	ErrorCategoryDatabase      ErrorCategory = "database"
	ErrorCategoryValidation    ErrorCategory = "validation"
	ErrorCategorySystem        ErrorCategory = "system"
)

// ErrorSeverity represents the severity of an error
type ErrorSeverity string

const (
	ErrorSeverityLow      ErrorSeverity = "low"
	ErrorSeverityMedium   ErrorSeverity = "medium"
	ErrorSeverityHigh     ErrorSeverity = "high"
	ErrorSeverityCritical ErrorSeverity = "critical"
)

// ErrorReport represents a detailed error report
type ErrorReport struct {
	Timestamp   time.Time              `json:"timestamp"`
	Category    ErrorCategory          `json:"category"`
	Severity    ErrorSeverity          `json:"severity"`
	Component   string                 `json:"component"`
	Message     string                 `json:"message"`
	Error       error                  `json:"error"`
	Context     map[string]interface{} `json:"context,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
	Recoverable bool                   `json:"recoverable"`
}

// ErrorReporter provides centralized error reporting
type ErrorReporter struct {
	logger         *Logger
	errorHistory   []*ErrorReport
	errorHistoryMu sync.RWMutex
	maxHistory     int

	// Error callbacks for different severities
	callbacks   map[ErrorSeverity][]ErrorCallback
	callbacksMu sync.RWMutex
}

// ErrorCallback is called when an error is reported
type ErrorCallback func(report *ErrorReport)

// NewErrorReporter creates a new error reporter
func NewErrorReporter() *ErrorReporter {
	return &ErrorReporter{
		logger:       NewLogger("ErrorReporter"),
		errorHistory: make([]*ErrorReport, 0),
		maxHistory:   1000, // Keep last 1000 errors
		callbacks:    make(map[ErrorSeverity][]ErrorCallback),
	}
}

// SetLogger sets the logger for the error reporter
func (er *ErrorReporter) SetLogger(logger *Logger) {
	er.logger = logger
}

// Report reports an error with full details
func (er *ErrorReporter) Report(report *ErrorReport) {
	report.Timestamp = time.Now()

	// Log the error
	er.logError(report)

	// Store in history
	er.addToHistory(report)

	// Invoke callbacks
	er.invokeCallbacks(report)
}

// ReportError reports a simple error
func (er *ErrorReporter) ReportError(category ErrorCategory, severity ErrorSeverity, component, message string, err error) {
	er.Report(&ErrorReport{
		Category:    category,
		Severity:    severity,
		Component:   component,
		Message:     message,
		Error:       err,
		Recoverable: true,
	})
}

// ReportErrorWithContext reports an error with additional context
func (er *ErrorReporter) ReportErrorWithContext(category ErrorCategory, severity ErrorSeverity, component, message string, err error, context map[string]interface{}) {
	er.Report(&ErrorReport{
		Category:    category,
		Severity:    severity,
		Component:   component,
		Message:     message,
		Error:       err,
		Context:     context,
		Recoverable: true,
	})
}

// ReportCriticalError reports a critical, non-recoverable error
func (er *ErrorReporter) ReportCriticalError(category ErrorCategory, component, message string, err error, context map[string]interface{}) {
	er.Report(&ErrorReport{
		Category:    category,
		Severity:    ErrorSeverityCritical,
		Component:   component,
		Message:     message,
		Error:       err,
		Context:     context,
		Recoverable: false,
	})
}

// logError logs an error report
func (er *ErrorReporter) logError(report *ErrorReport) {
	context := map[string]interface{}{
		"category":    string(report.Category),
		"severity":    string(report.Severity),
		"component":   report.Component,
		"recoverable": report.Recoverable,
	}

	// Add custom context
	if report.Context != nil {
		for k, v := range report.Context {
			context[k] = v
		}
	}

	// Log based on severity
	switch report.Severity {
	case ErrorSeverityCritical:
		er.logger.FatalWithContext(report.Message, report.Error, context)
	case ErrorSeverityHigh:
		er.logger.ErrorWithContext(report.Message, report.Error, context)
	case ErrorSeverityMedium:
		er.logger.WarnWithContext(report.Message, context)
	case ErrorSeverityLow:
		er.logger.InfoWithContext(report.Message, context)
	}
}

// addToHistory adds an error to the history
func (er *ErrorReporter) addToHistory(report *ErrorReport) {
	er.errorHistoryMu.Lock()
	defer er.errorHistoryMu.Unlock()

	er.errorHistory = append(er.errorHistory, report)

	// Trim history if it exceeds max size
	if len(er.errorHistory) > er.maxHistory {
		er.errorHistory = er.errorHistory[len(er.errorHistory)-er.maxHistory:]
	}
}

// invokeCallbacks invokes registered callbacks for the error severity
func (er *ErrorReporter) invokeCallbacks(report *ErrorReport) {
	er.callbacksMu.RLock()
	callbacks := er.callbacks[report.Severity]
	er.callbacksMu.RUnlock()

	for _, callback := range callbacks {
		go callback(report)
	}
}

// OnError registers a callback for a specific error severity
func (er *ErrorReporter) OnError(severity ErrorSeverity, callback ErrorCallback) {
	er.callbacksMu.Lock()
	defer er.callbacksMu.Unlock()

	er.callbacks[severity] = append(er.callbacks[severity], callback)
}

// GetRecentErrors returns the N most recent errors
func (er *ErrorReporter) GetRecentErrors(n int) []*ErrorReport {
	er.errorHistoryMu.RLock()
	defer er.errorHistoryMu.RUnlock()

	if n > len(er.errorHistory) {
		n = len(er.errorHistory)
	}

	// Return last N errors
	start := len(er.errorHistory) - n
	result := make([]*ErrorReport, n)
	copy(result, er.errorHistory[start:])

	return result
}

// GetErrorsByCategory returns errors filtered by category
func (er *ErrorReporter) GetErrorsByCategory(category ErrorCategory, limit int) []*ErrorReport {
	er.errorHistoryMu.RLock()
	defer er.errorHistoryMu.RUnlock()

	result := make([]*ErrorReport, 0)
	for i := len(er.errorHistory) - 1; i >= 0 && len(result) < limit; i-- {
		if er.errorHistory[i].Category == category {
			result = append(result, er.errorHistory[i])
		}
	}

	return result
}

// GetErrorStats returns statistics about errors
func (er *ErrorReporter) GetErrorStats() map[string]int {
	er.errorHistoryMu.RLock()
	defer er.errorHistoryMu.RUnlock()

	stats := map[string]int{
		"total":                       len(er.errorHistory),
		"severity_critical":           0,
		"severity_high":               0,
		"severity_medium":             0,
		"severity_low":                0,
		"category_orchestration":     0,
		"category_bot":                0,
		"category_emulator":           0,
		"category_account_pool":       0,
		"category_network":            0,
		"category_database":           0,
		"category_validation":         0,
		"category_system":             0,
		"recoverable":                 0,
		"non_recoverable":             0,
	}

	for _, report := range er.errorHistory {
		stats[fmt.Sprintf("severity_%s", report.Severity)]++
		stats[fmt.Sprintf("category_%s", report.Category)]++
		if report.Recoverable {
			stats["recoverable"]++
		} else {
			stats["non_recoverable"]++
		}
	}

	return stats
}

// Clear clears the error history
func (er *ErrorReporter) Clear() {
	er.errorHistoryMu.Lock()
	defer er.errorHistoryMu.Unlock()

	er.errorHistory = make([]*ErrorReport, 0)
}
