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
//
// Config overrides allow you to pass values to the nested routine's config parameters:
//   - action: RunRoutine
//     routine: "farming_loop"
//     config:
//       farm_type: "Gold"
//       target_count: "20"
type RunRoutine struct {
	Routine string            `yaml:"routine"` // Filename of the routine to run (without extension)
	Label   string            `yaml:"label"`   // Optional human-readable label for logging
	Config  map[string]string `yaml:"config"`  // Optional config parameter overrides
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
			registry := bot.Routines()

			// Get the routine builder from the registry
			routineBuilder, err := registry.Get(a.Routine)
			if err != nil {
				return fmt.Errorf("failed to get routine '%s': %w", a.Routine, err)
			}

			// Initialize config variables if the routine has config parameters
			// Try to cast to extended interface to access GetConfig
			if extRegistry, ok := registry.(*RoutineRegistry); ok {
				configParams, err := extRegistry.GetConfig(a.Routine)
				if err != nil {
					return fmt.Errorf("failed to get config for routine '%s': %w", a.Routine, err)
				}

				// Initialize config variables with overrides from this RunRoutine action
				if len(configParams) > 0 {
					if err := InitializeConfigVariables(bot, configParams, a.Config); err != nil {
						return fmt.Errorf("failed to initialize config for routine '%s': %w", a.Routine, err)
					}
				}
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
