package actions

import (
	"fmt"
)

// RoutineExecutor handles execution of routines with sentry support
type RoutineExecutor struct {
	routine       *ActionBuilder
	sentries      []Sentry
	sentryEngine  *SentryEngine
	routineLoader *RoutineLoader
}

// NewRoutineExecutor creates a new routine executor
func NewRoutineExecutor(routine *ActionBuilder, sentries []Sentry) *RoutineExecutor {
	return &RoutineExecutor{
		routine:  routine,
		sentries: sentries,
	}
}

// WithRoutineLoader sets the routine loader for loading sentry routines
func (re *RoutineExecutor) WithRoutineLoader(loader *RoutineLoader) *RoutineExecutor {
	re.routineLoader = loader
	return re
}

// LoadSentryRoutines loads and validates all sentry routine builders
func (re *RoutineExecutor) LoadSentryRoutines(bot BotInterface) error {
	if len(re.sentries) == 0 {
		return nil // No sentries to load
	}

	routineRegistry := bot.Routines()
	if routineRegistry == nil {
		return fmt.Errorf("routine registry not available on bot")
	}

	// Load each sentry routine
	for i := range re.sentries {
		sentry := &re.sentries[i]

		// Get the routine builder from the registry
		builder, err := routineRegistry.Get(sentry.Routine)
		if err != nil {
			return fmt.Errorf("sentry routine '%s' not found in registry: %w", sentry.Routine, err)
		}

		// Cache the builder
		sentry.SetRoutineBuilder(builder)
	}

	return nil
}

// Execute runs the main routine with sentry monitoring
func (re *RoutineExecutor) Execute(bot BotInterface) error {
	// Load sentry routines if not already loaded
	if len(re.sentries) > 0 && re.sentries[0].GetRoutineBuilder() == nil {
		if err := re.LoadSentryRoutines(bot); err != nil {
			return fmt.Errorf("failed to load sentry routines: %w", err)
		}
	}

	// Initialize routine controller state
	controller := bot.RoutineController()
	if controller != nil {
		controller.Reset()
		controller.SetRunning()
		defer func() {
			controller.SetCompleted()
			// Reset to idle after completion for next execution
			controller.SetIdle()
		}()
	}

	// Start sentry engine if sentries are configured
	if len(re.sentries) > 0 {
		re.sentryEngine = NewSentryEngine(bot, re.sentries)
		if err := re.sentryEngine.Start(); err != nil {
			return fmt.Errorf("failed to start sentry engine: %w", err)
		}
		defer re.sentryEngine.Stop()
	}

	// Execute the main routine
	err := re.routine.Execute(bot)

	// Sentry engine will be stopped by defer
	return err
}

// ExecuteRoutineWithSentries is a convenience function to load and execute a routine with sentries
func ExecuteRoutineWithSentries(bot BotInterface, routineName string) error {
	// Get the routine from the registry
	routineRegistry := bot.Routines()
	if routineRegistry == nil {
		return fmt.Errorf("routine registry not available on bot")
	}

	// Get routine with sentries
	builder, sentries, err := routineRegistry.GetWithSentries(routineName)
	if err != nil {
		return fmt.Errorf("routine '%s' not found in registry: %w", routineName, err)
	}

	// Create executor and run
	executor := NewRoutineExecutor(builder, sentries)
	return executor.Execute(bot)
}
