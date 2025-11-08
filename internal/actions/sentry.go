package actions

import (
	"fmt"
	"time"

	"jordanella.com/pocket-tcg-go/internal/monitor"
)

// SentryAction defines what to do after sentry routine execution
type SentryAction string

const (
	SentryActionResume    SentryAction = "resume"     // Resume the main routine (default)
	SentryActionPause     SentryAction = "pause"      // Keep routine paused
	SentryActionStop      SentryAction = "stop"       // Stop at end of routine (graceful)
	SentryActionForceStop SentryAction = "force_stop" // Immediately stop the routine
)

// SentrySeverity defines logging severity for sentry execution
type SentrySeverity string

const (
	SentrySeverityLow      SentrySeverity = "low"
	SentrySeverityMedium   SentrySeverity = "medium"
	SentrySeverityHigh     SentrySeverity = "high"
	SentrySeverityCritical SentrySeverity = "critical"
)

// Sentry defines a monitoring routine that runs in parallel to check for errors
type Sentry struct {
	Routine    string         `yaml:"routine"`              // Name of the routine to execute
	Frequency  int            `yaml:"frequency,omitempty"`  // How often to poll in seconds (default: 5)
	Severity   SentrySeverity `yaml:"severity,omitempty"`   // Logging severity (default: medium)
	OnSuccess  SentryAction   `yaml:"on_success,omitempty"` // Action on success (nil error) (default: resume)
	OnFailure  SentryAction   `yaml:"on_failure,omitempty"` // Action on failure (non-nil error) (default: force_stop)

	// Internal fields set during validation
	routineBuilder *ActionBuilder // Cached routine builder
}

// Validate checks if the sentry configuration is valid
func (s *Sentry) Validate(ab *ActionBuilder) error {
	if s.Routine == "" {
		return fmt.Errorf("sentry routine name is required")
	}

	// Set defaults
	if s.Frequency == 0 {
		s.Frequency = 5 // Default to 5 seconds
	}
	if s.Frequency < 1 {
		return fmt.Errorf("sentry frequency must be at least 1 second")
	}

	if s.Severity == "" {
		s.Severity = SentrySeverityMedium
	}
	if !isValidSeverity(s.Severity) {
		return fmt.Errorf("invalid severity '%s': must be low, medium, high, or critical", s.Severity)
	}

	if s.OnSuccess == "" {
		s.OnSuccess = SentryActionResume
	}
	if !isValidSentryAction(s.OnSuccess) {
		return fmt.Errorf("invalid on_success action '%s': must be resume, pause, stop, or force_stop", s.OnSuccess)
	}

	if s.OnFailure == "" {
		s.OnFailure = SentryActionForceStop
	}
	if !isValidSentryAction(s.OnFailure) {
		return fmt.Errorf("invalid on_failure action '%s': must be resume, pause, stop, or force_stop", s.OnFailure)
	}

	// Validate that the routine exists in the registry (if available)
	if ab.templateRegistry != nil {
		// We need a way to access the routine registry
		// This will be handled at a higher level during routine loading
	}

	return nil
}

// GetFrequency returns the polling frequency as a duration
func (s *Sentry) GetFrequency() time.Duration {
	return time.Duration(s.Frequency) * time.Second
}

// GetMonitorSeverity converts SentrySeverity to monitor.ErrorSeverity
func (s *Sentry) GetMonitorSeverity() monitor.ErrorSeverity {
	switch s.Severity {
	case SentrySeverityLow:
		return monitor.SeverityLow
	case SentrySeverityMedium:
		return monitor.SeverityMedium
	case SentrySeverityHigh:
		return monitor.SeverityHigh
	case SentrySeverityCritical:
		return monitor.SeverityCritical
	default:
		return monitor.SeverityMedium
	}
}

// SetRoutineBuilder caches the routine builder after validation
func (s *Sentry) SetRoutineBuilder(builder *ActionBuilder) {
	s.routineBuilder = builder
}

// GetRoutineBuilder returns the cached routine builder
func (s *Sentry) GetRoutineBuilder() *ActionBuilder {
	return s.routineBuilder
}

// isValidSeverity checks if the severity is valid
func isValidSeverity(severity SentrySeverity) bool {
	switch severity {
	case SentrySeverityLow, SentrySeverityMedium, SentrySeverityHigh, SentrySeverityCritical:
		return true
	default:
		return false
	}
}

// isValidSentryAction checks if the action is valid
func isValidSentryAction(action SentryAction) bool {
	switch action {
	case SentryActionResume, SentryActionPause, SentryActionStop, SentryActionForceStop:
		return true
	default:
		return false
	}
}
