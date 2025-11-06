package actions

import (
	"context"
	"time"

	"jordanella.com/pocket-tcg-go/internal/monitor"
)

// ActionBuilder type and core methods
type ActionBuilder struct {
	bot                BotInterface // Reference to bot
	steps              []Step
	timeout            time.Duration
	retries            int
	ignoreErrors       bool
	ctx                context.Context
	errorCheckEnabled  bool                         // Whether to check for errors during execution
	errorCheckInterval time.Duration                // How often to check for errors
	errorHandler       monitor.ErrorHandlerFunc     // Custom error handler for this action
}

type Step struct {
	name         string
	execute      func() error
	recover      func(error) error
	timeout      time.Duration
	canInterrupt bool
}

// Builder configuration methods

func (ab *ActionBuilder) WithTimeout(d time.Duration) *ActionBuilder {
	ab.timeout = d
	return ab
}

// Timeout sets a timeout in seconds for the entire action sequence
// This is a convenience method that calls WithTimeout with seconds converted to duration
// If the action exceeds this timeout, it will be aborted with a timeout error
func (ab *ActionBuilder) Timeout(seconds int) *ActionBuilder {
	ab.timeout = time.Duration(seconds) * time.Second
	return ab
}

func (ab *ActionBuilder) WithRetries(n int) *ActionBuilder {
	ab.retries = n
	return ab
}

func (ab *ActionBuilder) IgnoreErrors() *ActionBuilder {
	if ab.steps[len(ab.steps)-1].recover == nil {
		ab.steps[len(ab.steps)-1].recover = func(error) error { return nil }
	}
	ab.ignoreErrors = true
	return ab
}

func (ab *ActionBuilder) Interruptible() *ActionBuilder {
	// This would be a step-level property, not builder-level
	// For now, just return ab
	ab.steps[len(ab.steps)-1].canInterrupt = true
	return ab
}

// WithErrorHandler sets a custom error handler for this action
// If not set, a default handler will be used that continues on most errors
func (ab *ActionBuilder) WithErrorHandler(handler monitor.ErrorHandlerFunc) *ActionBuilder {
	ab.errorHandler = handler
	ab.errorCheckEnabled = true
	if ab.errorCheckInterval == 0 {
		ab.errorCheckInterval = 1 * time.Second // Default check interval
	}
	return ab
}

// WithErrorChecking enables error checking with the specified interval
// Uses the default error handler if no custom handler is set
func (ab *ActionBuilder) WithErrorChecking(interval time.Duration) *ActionBuilder {
	ab.errorCheckEnabled = true
	ab.errorCheckInterval = interval
	if ab.errorHandler == nil {
		ab.errorHandler = defaultErrorHandler
	}
	return ab
}

// DisableErrorChecking disables automatic error checking for this action
// Useful for critical operations that shouldn't be interrupted
func (ab *ActionBuilder) DisableErrorChecking() *ActionBuilder {
	ab.errorCheckEnabled = false
	return ab
}

// defaultErrorHandler is used when no custom handler is provided
var defaultErrorHandler = func(event *monitor.ErrorEvent) monitor.ErrorResponse {
	switch event.Severity {
	case monitor.SeverityCritical:
		// Critical errors should stop the bot
		return monitor.CreateErrorResponse(nil, "Critical error, stopping bot")
	case monitor.SeverityHigh:
		// High priority errors like popups - continue after logging
		return monitor.CreateSimpleResponse(monitor.ActionContinue, "Handled high priority error")
	default:
		// Medium/low severity - just continue
		return monitor.CreateSimpleResponse(monitor.ActionContinue, "Error noted, continuing")
	}
}

// Execution

func (ab *ActionBuilder) Execute() error {
	ctx := ab.bot.Context()
	if ab.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ab.timeout)
		defer cancel()
	}

	// If error checking is enabled, execute with monitoring
	if ab.errorCheckEnabled && ab.bot.ErrorMonitor() != nil {
		return ab.executeWithErrorMonitoring(ctx)
	}

	// Otherwise execute normally
	return ab.executeSteps(ctx)
}

func (ab *ActionBuilder) ExecuteOnce() error {
	ctx := context.Background()
	return ab.executeSteps(ctx)
}

// Internal

func (ab *ActionBuilder) executeSteps(ctx context.Context) error {
	for _, step := range ab.steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := step.execute(); err != nil {
			if !ab.ignoreErrors {
				return err
			}
		}
	}
	return nil
}

// executeWithErrorMonitoring executes steps while checking for errors
func (ab *ActionBuilder) executeWithErrorMonitoring(ctx context.Context) error {
	errorChan := ab.bot.ErrorMonitor().GetErrorChannel()
	ticker := time.NewTicker(ab.errorCheckInterval)
	defer ticker.Stop()

	// Execute steps in goroutine
	done := make(chan error, 1)
	go func() {
		done <- ab.executeSteps(ctx)
	}()

	// Monitor for errors while executing
	for {
		select {
		case err := <-done:
			return err // Execution completed

		case <-ticker.C:
			// Check for errors periodically
			event := monitor.CheckForErrors(errorChan)
			if event != nil {
				startTime := time.Now()
				response := ab.errorHandler(event)
				response.RecoveryTime = time.Since(startTime)

				// Send response back to monitor
				monitor.HandleErrorWithRecoveryTime(event, response.Handled, response.Action, response.Message, response.Error, response.RecoveryTime)

				// Check if we should abort
				if monitor.ShouldAbortRoutine(response.Action) {
					return &ErrorInterrupt{Action: response.Action, Message: response.Message}
				}
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (ab *ActionBuilder) shouldRetry(err error) bool {
	err = ab.steps[len(ab.steps)-1].recover(err)
	return err != nil
}

// ErrorInterrupt is a special error type that indicates an error was handled
// but requires interrupting the current routine
type ErrorInterrupt struct {
	Action  monitor.ErrorAction
	Message string
}

func (e *ErrorInterrupt) Error() string {
	return e.Message
}
