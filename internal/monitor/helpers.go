package monitor

import (
	"context"
	"time"
)

// CheckForErrors checks the error channel non-blockingly
// Returns an error event if one is available, nil otherwise
func CheckForErrors(errorChan <-chan ErrorEvent) *ErrorEvent {
	select {
	case event, ok := <-errorChan:
		if !ok {
			return nil // Channel closed
		}
		return &event
	default:
		return nil // No error waiting
	}
}

// CheckForErrorsWithContext checks for errors with context cancellation
func CheckForErrorsWithContext(ctx context.Context, errorChan <-chan ErrorEvent) (*ErrorEvent, error) {
	select {
	case event, ok := <-errorChan:
		if !ok {
			return nil, nil // Channel closed
		}
		return &event, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, nil // No error waiting
	}
}

// HandleError is a convenience function that sends a response back to the monitor
func HandleError(event *ErrorEvent, handled bool, action ErrorAction, message string, err error) {
	if event == nil || event.ResponseChan == nil {
		return
	}

	response := ErrorResponse{
		Handled: handled,
		Action:  action,
		Message: message,
		Error:   err,
	}

	// Non-blocking send with timeout
	select {
	case event.ResponseChan <- response:
		// Response sent successfully
	case <-time.After(5 * time.Second):
		// Timeout - monitor may have stopped listening
	}
}

// HandleErrorWithRecoveryTime is like HandleError but includes recovery timing
func HandleErrorWithRecoveryTime(event *ErrorEvent, handled bool, action ErrorAction, message string, err error, recoveryTime time.Duration) {
	if event == nil || event.ResponseChan == nil {
		return
	}

	response := ErrorResponse{
		Handled:      handled,
		Action:       action,
		Message:      message,
		Error:        err,
		RecoveryTime: recoveryTime,
	}

	select {
	case event.ResponseChan <- response:
	case <-time.After(5 * time.Second):
	}
}

// ErrorHandlerFunc is a function signature for error handling callbacks
type ErrorHandlerFunc func(event *ErrorEvent) ErrorResponse

// HandleWithCallback handles an error using a callback function
// This is useful for defining error handling logic inline
func HandleWithCallback(event *ErrorEvent, handler ErrorHandlerFunc) {
	if event == nil || handler == nil {
		return
	}

	startTime := time.Now()
	response := handler(event)
	response.RecoveryTime = time.Since(startTime)

	select {
	case event.ResponseChan <- response:
	case <-time.After(5 * time.Second):
	}
}

// ShouldAbortRoutine checks if the error action requires aborting the routine
func ShouldAbortRoutine(action ErrorAction) bool {
	return action == ActionAbort || action == ActionStop || action == ActionRestart
}

// ShouldStopBot checks if the error action requires stopping the bot entirely
func ShouldStopBot(action ErrorAction) bool {
	return action == ActionStop
}

// CreateSimpleResponse creates a standard success response
func CreateSimpleResponse(action ErrorAction, message string) ErrorResponse {
	return ErrorResponse{
		Handled: true,
		Action:  action,
		Message: message,
	}
}

// CreateErrorResponse creates a failure response
func CreateErrorResponse(err error, message string) ErrorResponse {
	return ErrorResponse{
		Handled: false,
		Action:  ActionAbort,
		Message: message,
		Error:   err,
	}
}
