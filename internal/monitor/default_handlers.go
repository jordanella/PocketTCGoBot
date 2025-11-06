package monitor

import (
	"fmt"
	"time"
)

// DefaultErrorHandlers provides built-in handlers for common error types
// These handlers focus on diagnosable, recoverable errors

// HandleCommunicationError handles ADB disconnection and emulator crashes
// These are critical errors that require immediate attention
func HandleCommunicationError(event *ErrorEvent) ErrorResponse {
	startTime := time.Now()

	// Communication errors are critical - we can't continue without ADB
	return ErrorResponse{
		Handled:      false, // Can't handle this automatically
		Action:       ActionStop,
		RecoveryTime: time.Since(startTime),
		Error:        fmt.Errorf("ADB communication lost"),
		Message:      "ADB connection failed - bot must stop for manual intervention",
	}
}

// HandleMaintenanceError handles maintenance mode detection
// Bot should wait and retry periodically
func HandleMaintenanceError(event *ErrorEvent) ErrorResponse {
	startTime := time.Now()

	// Game is in maintenance - we should abort current routine
	// Caller should implement retry logic with exponential backoff
	return ErrorResponse{
		Handled:      true,
		Action:       ActionAbort,
		RecoveryTime: time.Since(startTime),
		Error:        nil,
		Message:      "Game in maintenance mode - aborting routine. Caller should retry later.",
	}
}

// HandleUpdateRequiredError handles forced update detection
// Bot must stop as the game version is incompatible
func HandleUpdateRequiredError(event *ErrorEvent) ErrorResponse {
	startTime := time.Now()

	return ErrorResponse{
		Handled:      false,
		Action:       ActionStop,
		RecoveryTime: time.Since(startTime),
		Error:        fmt.Errorf("game update required"),
		Message:      "Game requires update - bot cannot continue until game is updated",
	}
}

// HandleBannedError handles account ban detection
// Mark account as banned and stop using it
func HandleBannedError(event *ErrorEvent) ErrorResponse {
	startTime := time.Now()

	// This is a critical error - account is unusable
	// Caller should mark account as banned in database
	return ErrorResponse{
		Handled:      false,
		Action:       ActionStop,
		RecoveryTime: time.Since(startTime),
		Error:        fmt.Errorf("account banned"),
		Message:      "Account has been banned - mark as banned in database and stop using this account",
	}
}

// HandleTitleScreenError handles unexpected return to title screen
// This usually means session expired or connection lost
func HandleTitleScreenError(event *ErrorEvent) ErrorResponse {
	startTime := time.Now()

	// Abort current routine - caller can restart from title screen
	return ErrorResponse{
		Handled:      true,
		Action:       ActionAbort,
		RecoveryTime: time.Since(startTime),
		Error:        nil,
		Message:      "Returned to title screen - abort routine. Caller should re-login and restart.",
	}
}

// HandleNoResponseError handles game freeze or hang detection
// Try to recover by going home or restarting
func HandleNoResponseError(event *ErrorEvent) ErrorResponse {
	startTime := time.Now()

	// Game is frozen - we should abort and let caller decide to restart
	return ErrorResponse{
		Handled:      false,
		Action:       ActionAbort,
		RecoveryTime: time.Since(startTime),
		Error:        fmt.Errorf("game not responding"),
		Message:      "Game appears frozen - aborting routine. Caller should consider restarting emulator.",
	}
}

// HandleTimeoutError handles execution timeout
// Abort the current routine when max runtime is exceeded
func HandleTimeoutError(event *ErrorEvent) ErrorResponse {
	startTime := time.Now()

	return ErrorResponse{
		Handled:      true,
		Action:       ActionAbort,
		RecoveryTime: time.Since(startTime),
		Error:        nil,
		Message:      "Routine exceeded maximum runtime - aborting for safety",
	}
}

// DefaultErrorHandler is a comprehensive handler that routes to specific handlers
// This is the recommended default handler for most use cases
func DefaultErrorHandler(event *ErrorEvent) ErrorResponse {
	switch event.Type {
	case ErrorCommunication:
		return HandleCommunicationError(event)

	case ErrorMaintenance:
		return HandleMaintenanceError(event)

	case ErrorUpdate:
		return HandleUpdateRequiredError(event)

	case ErrorBanned:
		return HandleBannedError(event)

	case ErrorTitleScreen:
		return HandleTitleScreenError(event)

	case ErrorNoResponse:
		return HandleNoResponseError(event)

	case ErrorTimeout:
		return HandleTimeoutError(event)

	case ErrorPopup:
		// Popups are complex - require template matching and clicking
		// For now, just log and continue (caller can implement custom handler)
		return ErrorResponse{
			Handled:      false,
			Action:       ActionContinue,
			RecoveryTime: 0,
			Error:        nil,
			Message:      "Popup detected but not handled - implement custom popup handler",
		}

	case ErrorStuck:
		// Stuck detection is complex - requires screen history analysis
		// For now, just abort (caller can implement custom recovery)
		return ErrorResponse{
			Handled:      false,
			Action:       ActionAbort,
			RecoveryTime: 0,
			Error:        fmt.Errorf("bot appears stuck"),
			Message:      "Bot stuck on same screen - implement custom recovery logic",
		}

	default:
		// Unknown error type - abort to be safe
		return ErrorResponse{
			Handled:      false,
			Action:       ActionAbort,
			RecoveryTime: 0,
			Error:        fmt.Errorf("unknown error type: %v", event.Type),
			Message:      fmt.Sprintf("Unknown error type %v - aborting routine", event.Type),
		}
	}
}

// GetDefaultHandler returns the default error handler
func GetDefaultHandler() ErrorHandlerFunc {
	return DefaultErrorHandler
}

// GetHandlerForType returns a specific handler for an error type
func GetHandlerForType(errorType ErrorType) ErrorHandlerFunc {
	switch errorType {
	case ErrorCommunication:
		return HandleCommunicationError
	case ErrorMaintenance:
		return HandleMaintenanceError
	case ErrorUpdate:
		return HandleUpdateRequiredError
	case ErrorBanned:
		return HandleBannedError
	case ErrorTitleScreen:
		return HandleTitleScreenError
	case ErrorNoResponse:
		return HandleNoResponseError
	default:
		return DefaultErrorHandler
	}
}
