package monitor

import "time"

// ErrorType represents the category of error detected
type ErrorType int

const (
	ErrorCommunication ErrorType = iota // ADB disconnected, emulator crashed
	ErrorStuck                           // Bot stuck on same screen
	ErrorNoResponse                      // Game not responding
	ErrorPopup                           // Unexpected popup (level up, rewards, etc.)
	ErrorMaintenance                     // Maintenance mode
	ErrorUpdate                          // Update required
	ErrorBanned                          // Account banned/suspended
	ErrorTitleScreen                     // Returned to title screen unexpectedly
	ErrorCustom                          // Custom error type
)

// ErrorSeverity determines how the error should be handled
type ErrorSeverity int

const (
	SeverityCritical ErrorSeverity = iota // Stop bot immediately
	SeverityHigh                           // Interrupt routine, handle, then decide
	SeverityMedium                         // Handle when convenient
	SeverityLow                            // Log only
)

// ErrorAction tells the routine what to do after error handling
type ErrorAction int

const (
	ActionContinue ErrorAction = iota // Continue routine execution
	ActionAbort                        // Abort current routine
	ActionRetry                        // Retry the current step
	ActionStop                         // Stop the bot entirely
	ActionRestart                      // Restart the routine from beginning
)

// Priority for monitoring loops (kept for backward compatibility with existing code)
type Priority int

const (
	PriorityCritical Priority = iota
	PriorityHigh
	PriorityMedium
	PriorityLow
)

// ErrorEvent is sent from the monitor to routines when an error is detected
type ErrorEvent struct {
	Type         ErrorType
	Severity     ErrorSeverity
	Template     interface{} // Avoid circular import - will be *templates.Template
	Context      map[string]interface{}
	DetectedAt   time.Time
	Message      string
	ResponseChan chan ErrorResponse // Channel for two-way communication
}

// ErrorResponse is sent from routines back to the monitor after handling
type ErrorResponse struct {
	Handled      bool          // Whether the error was successfully handled
	Action       ErrorAction   // What the routine will do next
	RecoveryTime time.Duration // How long recovery took
	Error        error         // Any error that occurred during recovery
	Message      string        // Optional message about recovery
}
