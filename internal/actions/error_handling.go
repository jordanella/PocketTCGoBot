package actions

import (
	"jordanella.com/pocket-tcg-go/internal/monitor"
)

// CheckForErrors checks for errors manually and handles them with the provided callback
// This can be called manually within a routine for fine-grained error checking
func (ab *ActionBuilder) CheckForErrors(handler monitor.ErrorHandlerFunc) error {
	if ab.bot.ErrorMonitor() == nil {
		return nil
	}

	errorChan := ab.bot.ErrorMonitor().GetErrorChannel()
	event, err := monitor.CheckForErrorsWithContext(ab.bot.Context(), errorChan)
	if err != nil {
		return err
	}

	if event == nil {
		return nil // No error
	}

	response := handler(event)
	monitor.HandleErrorWithRecoveryTime(event, response.Handled, response.Action, response.Message, response.Error, response.RecoveryTime)

	if monitor.ShouldAbortRoutine(response.Action) {
		return &ErrorInterrupt{Action: response.Action, Message: response.Message}
	}

	return nil
}
