package actions

import (
	"fmt"
)

// RunRoutine executes another routine by loading it from the routine registry
// This allows for modular, reusable routine composition
//
// The 'routine' field should be the filename (without extension) of the routine to run.
// For example, if you have "common_navigation.yaml", use routine: "common_navigation"
//
// The optional 'label' field can be used for readability in logs and doesn't affect execution.
type RunRoutine struct {
	Routine string `yaml:"routine"` // Filename of the routine to run (without extension)
	Label   string `yaml:"label"`   // Optional human-readable label for logging
}

func (a *RunRoutine) Validate(ab *ActionBuilder) error {
	if a.Routine == "" {
		return fmt.Errorf("routine cannot be empty")
	}

	// Optional: Validate routine exists if registry is available at build time
	// This allows early detection of missing routines
	// Note: We don't fail hard here since the registry might not be initialized yet
	// The real validation happens at execution time

	return nil
}

func (a *RunRoutine) Build(ab *ActionBuilder) *ActionBuilder {
	// Use label for display if provided, otherwise use filename
	displayName := a.Routine
	if a.Label != "" {
		displayName = fmt.Sprintf("%s (%s)", a.Label, a.Routine)
	}

	step := Step{
		name: fmt.Sprintf("RunRoutine: %s", displayName),
		execute: func(bot BotInterface) error {
			// Get the routine from the registry (eagerly loaded)
			routineBuilder, err := bot.Routines().Get(a.Routine)
			if err != nil {
				return fmt.Errorf("failed to get routine '%s': %w", a.Routine, err)
			}

			// Execute the loaded routine
			if err := routineBuilder.Execute(bot); err != nil {
				return fmt.Errorf("routine '%s' execution failed: %w", displayName, err)
			}

			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
