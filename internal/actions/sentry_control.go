package actions

import (
	"fmt"
)

// SentryHalt pauses the main routine execution
// This action should only be used within sentry routines
// It allows the sentry to halt the main routine for remediation
type SentryHalt struct{}

func (a *SentryHalt) Validate(ab *ActionBuilder) error {
	return nil // No validation needed
}

func (a *SentryHalt) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "SentryHalt",
		execute: func(bot BotInterface) error {
			// Get routine controller
			type routineControllerProvider interface {
				RoutineController() RoutineControllerInterface
			}

			provider, ok := bot.(routineControllerProvider)
			if !ok {
				return fmt.Errorf("bot does not provide RoutineController")
			}

			controller := provider.RoutineController()
			if controller == nil {
				return fmt.Errorf("routine controller not available")
			}

			// Pause the main routine
			if !controller.Pause() {
				// Routine wasn't running, this is non-fatal
				fmt.Printf("Bot %d: SentryHalt called but routine is not running\n", bot.Instance())
			} else {
				fmt.Printf("Bot %d: Sentry halted main routine execution\n", bot.Instance())
			}

			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// SentryResume resumes the main routine execution
// This action should only be used within sentry routines
// It allows the sentry to resume the main routine after remediation
type SentryResume struct{}

func (a *SentryResume) Validate(ab *ActionBuilder) error {
	return nil // No validation needed
}

func (a *SentryResume) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "SentryResume",
		execute: func(bot BotInterface) error {
			// Get routine controller
			type routineControllerProvider interface {
				RoutineController() RoutineControllerInterface
			}

			provider, ok := bot.(routineControllerProvider)
			if !ok {
				return fmt.Errorf("bot does not provide RoutineController")
			}

			controller := provider.RoutineController()
			if controller == nil {
				return fmt.Errorf("routine controller not available")
			}

			// Resume the main routine
			if !controller.Resume() {
				// Routine wasn't paused, this is non-fatal
				fmt.Printf("Bot %d: SentryResume called but routine is not paused\n", bot.Instance())
			} else {
				fmt.Printf("Bot %d: Sentry resumed main routine execution\n", bot.Instance())
			}

			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
