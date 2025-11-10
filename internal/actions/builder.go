package actions

import (
	"context"
	"fmt"
	"time"

	"jordanella.com/pocket-tcg-go/internal/cv"
	"jordanella.com/pocket-tcg-go/internal/monitor"
)

type ActionStep interface {
	Validate(ab *ActionBuilder) error
	Build(ab *ActionBuilder) *ActionBuilder
}

// ActionBuilder type and core methods
type ActionBuilder struct {
	steps              []Step
	timeout            time.Duration
	retries            int
	ignoreErrors       bool
	errorCheckEnabled  bool                      // Whether to check for errors during execution
	errorCheckInterval time.Duration             // How often to check for errors
	errorHandler       monitor.ErrorHandlerFunc  // Custom error handler for this action
	templateRegistry   TemplateRegistryInterface // Optional: for validating template names at build time
	isSentryExecution  bool                      // If true, ignores pause/stop signals from routine controller
}

// NewActionBuilder creates a new ActionBuilder for building reusable routines
// The bot is not required at build time - it will be provided during Execute()
func NewActionBuilder() *ActionBuilder {
	return &ActionBuilder{}
}

// WithTemplateRegistry sets the template registry for build-time validation
// This allows actions to validate that template names exist during the build phase
func (ab *ActionBuilder) WithTemplateRegistry(registry TemplateRegistryInterface) *ActionBuilder {
	ab.templateRegistry = registry
	return ab
}

// AsSentryExecution marks this ActionBuilder as a sentry execution
// Sentry executions ignore pause/stop signals from the routine controller
// This prevents sentries from being blocked by their own halt commands
func (ab *ActionBuilder) AsSentryExecution() *ActionBuilder {
	ab.isSentryExecution = true
	return ab
}

// InitializeConfigVariables initializes variables from config parameters with their defaults
// This should be called before executing a routine to set up user-configurable variables
// If a variable is marked as persistent and already exists, it will NOT be overwritten
func InitializeConfigVariables(bot BotInterface, config []ConfigParam, overrides map[string]string) error {
	for _, param := range config {
		// Skip persistent variables that already have a value
		if param.Persist {
			if _, exists := bot.Variables().Get(param.Name); exists {
				// Variable already exists and is persistent, don't reinitialize
				continue
			}
		}

		// Get the value: override > default > type default
		value := param.Default
		if overrides != nil {
			if override, ok := overrides[param.Name]; ok {
				value = override
			}
		}
		if value == "" {
			value = param.GetTypeDefault()
		}

		// Set the variable
		bot.Variables().Set(param.Name, value)

		// Mark as persistent if specified
		if param.Persist {
			if vs, ok := bot.Variables().(*VariableStore); ok {
				vs.MarkPersistent(param.Name)
			}
		}
	}
	return nil
}

type Step struct {
	name         string
	execute      func(BotInterface) error // Bot is provided at execution time
	recover      func(error) error
	canInterrupt bool
	issue        error
	timeout      time.Duration // Timeout for this specific step (0 = no timeout)
	maxAttempts  int           // Maximum number of attempts for this step (0 or 1 = no retries)
	retryDelay   time.Duration // Delay between retry attempts (default: 1s)
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

// Execute runs the action sequence on the provided bot
// This allows the same ActionBuilder to be executed on multiple bots
func (ab *ActionBuilder) Execute(bot BotInterface) error {
	ctx := bot.Context()
	if ab.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ab.timeout)
		defer cancel()
	}

	// If error checking is enabled, execute with monitoring
	if ab.errorCheckEnabled && bot.ErrorMonitor() != nil {
		return ab.executeWithErrorMonitoring(ctx, bot)
	}

	// Otherwise execute normally
	return ab.executeSteps(ctx, bot)
}

// ExecuteOnce runs the action sequence once with a background context
func (ab *ActionBuilder) ExecuteOnce(bot BotInterface) error {
	ctx := context.Background()
	return ab.executeSteps(ctx, bot)
}

// Internal

func (ab *ActionBuilder) executeSteps(ctx context.Context, bot BotInterface) error {
	for _, step := range ab.steps {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check for pause/stop signals from routine controller
		if !ab.checkExecutionState(bot) {
			return fmt.Errorf("routine stopped by controller")
		}

		if step.issue != nil {
			return fmt.Errorf("build configuration error for step '%s': %w", step.name, step.issue)
		}

		// Execute step with timeout and retries
		if err := ab.executeStepWithRetries(ctx, bot, &step); err != nil {
			if !ab.ignoreErrors {
				return err
			}
		}
	}
	return nil
}

// executeStepWithRetries executes a single step with timeout and retry logic
func (ab *ActionBuilder) executeStepWithRetries(ctx context.Context, bot BotInterface, step *Step) error {
	maxAttempts := step.maxAttempts
	if maxAttempts == 0 {
		maxAttempts = 1 // Default: no retries
	}

	retryDelay := step.retryDelay
	if retryDelay == 0 {
		retryDelay = 1 * time.Second // Default: 1 second between retries
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Create step context with timeout if specified
		stepCtx := ctx
		var cancel context.CancelFunc
		if step.timeout > 0 {
			stepCtx, cancel = context.WithTimeout(ctx, step.timeout)
		}

		// Execute the step in a goroutine to handle timeout
		done := make(chan error, 1)
		go func() {
			done <- step.execute(bot)
		}()

		// Wait for execution or timeout
		select {
		case <-stepCtx.Done():
			if cancel != nil {
				cancel()
			}
			lastErr = fmt.Errorf("step '%s' timed out after %v", step.name, step.timeout)

			// If this isn't the last attempt, wait before retrying
			if attempt < maxAttempts {
				fmt.Printf("Step '%s' attempt %d/%d failed (timeout), retrying in %v...\n",
					step.name, attempt, maxAttempts, retryDelay)
				time.Sleep(retryDelay)
				continue
			}
			return lastErr

		case err := <-done:
			if cancel != nil {
				cancel()
			}

			if err == nil {
				// Success
				if attempt > 1 {
					fmt.Printf("Step '%s' succeeded on attempt %d/%d\n", step.name, attempt, maxAttempts)
				}
				return nil
			}

			lastErr = err

			// If this isn't the last attempt, wait before retrying
			if attempt < maxAttempts {
				fmt.Printf("Step '%s' attempt %d/%d failed: %v, retrying in %v...\n",
					step.name, attempt, maxAttempts, err, retryDelay)
				time.Sleep(retryDelay)
				continue
			}

			// Last attempt failed
			return fmt.Errorf("step '%s' failed after %d attempts: %w", step.name, maxAttempts, lastErr)
		}
	}

	return lastErr
}

// checkExecutionState checks if routine should pause or stop
// Returns true if execution should continue, false if stopped
func (ab *ActionBuilder) checkExecutionState(bot BotInterface) bool {
	// Sentry executions ignore halt signals to prevent deadlock
	if ab.isSentryExecution {
		return true
	}

	// Check if bot has routine controller
	type routineControllerProvider interface {
		RoutineController() RoutineControllerInterface
	}

	provider, ok := bot.(routineControllerProvider)
	if !ok {
		return true // No controller, continue normally
	}

	controller := provider.RoutineController()
	if controller == nil {
		return true // No controller, continue normally
	}

	// Use the controller's built-in pause/stop checking
	return controller.CheckPauseOrStop()
}

// executeWithErrorMonitoring executes steps while checking for errors
func (ab *ActionBuilder) executeWithErrorMonitoring(ctx context.Context, bot BotInterface) error {
	errorChan := bot.ErrorMonitor().GetErrorChannel()
	ticker := time.NewTicker(ab.errorCheckInterval)
	defer ticker.Stop()

	// Execute steps in goroutine
	done := make(chan error, 1)
	go func() {
		done <- ab.executeSteps(ctx, bot)
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

/* UNUSED
func (ab *ActionBuilder) shouldRetry(err error) bool {
	err = ab.steps[len(ab.steps)-1].recover(err)
	return err != nil
}
*/

// ErrorInterrupt is a special error type that indicates an error was handled
// but requires interrupting the current routine
type ErrorInterrupt struct {
	Action  monitor.ErrorAction
	Message string
}

func (e *ErrorInterrupt) Error() string {
	return e.Message
}

func (ab *ActionBuilder) buildSteps(actions []ActionStep) []Step {
	// Create a temporary ActionBuilder to house the new steps.
	// This is clean because the ActionStep.Build method appends to its receiver's steps field.
	tempBuilder := NewActionBuilder()

	// Propagate template registry for nested validation
	tempBuilder.templateRegistry = ab.templateRegistry

	for _, action := range actions {
		action.Build(tempBuilder)
	}

	// The steps are now in the temporary builder
	return tempBuilder.steps
}

func buildTemplateConfiguration(bot BotInterface, templateName string, actionThreshold *float64, actionRegion *cv.Region) (template cv.Template, config *cv.MatchConfig, err error) {
	template, ok := bot.Templates().Get(templateName)
	if !ok {
		return cv.Template{}, nil, fmt.Errorf("template '%s' not found in registry", templateName)
	}

	// Build match config starting with template's threshold
	threshold := template.Threshold
	if actionThreshold != nil {
		// Override with action-level threshold if provided
		threshold = *actionThreshold
	}

	config = &cv.MatchConfig{
		Threshold: threshold,
	}

	// Apply region (action-level takes precedence over template-level)
	if actionRegion != nil {
		// Action-level region override
		config.SearchRegion = actionRegion.ToImageRectangle()
	} else if template.Region != nil {
		// Fall back to template-level region
		config.SearchRegion = template.Region.ToImageRectangle()
	}
	return template, config, nil
}
