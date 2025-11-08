package actions

import (
	"fmt"
)

type Repeat struct {
	Iterations int          `yaml:"iterations"`
	Actions    []ActionStep `yaml:"actions"`
}

// UnmarshalYAML implements custom unmarshaling for Repeat to handle polymorphic Actions field
func (a *Repeat) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	// Extract fields
	if val, ok := raw["iterations"].(int); ok {
		a.Iterations = val
	}

	// Handle the nested actions
	if actionsRaw, ok := raw["actions"]; ok && actionsRaw != nil {
		unmarshaledActions, err := unmarshalNestedActions(actionsRaw)
		if err != nil {
			return err
		}
		a.Actions = unmarshaledActions
	}

	return nil
}

func (a *Repeat) Validate(ab *ActionBuilder) error {
	if a.Iterations <= 0 {
		return fmt.Errorf("iterations must be greater than 0")
	}

	if len(a.Actions) == 0 {
		return fmt.Errorf("actions cannot be empty")
	}

	// Validate nested actions with better error context
	for i, action := range a.Actions {
		if err := action.Validate(ab); err != nil {
			return fmt.Errorf("Repeat (%d) -> nested action %d: %w", a.Iterations, i+1, err)
		}
	}

	return nil
}

func (a *Repeat) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("Repeat (%d)", a.Iterations),
		execute: func(bot BotInterface) error {
			// Build the nested actions into a concrete slice of executable steps
			nestedSteps := ab.buildSteps(a.Actions)

			for i := 0; i < a.Iterations; i++ {
				// Re-execute the action builder's steps
				subBuilder := &ActionBuilder{
					steps: nestedSteps,
				}

				// Call the internal execution function with the bot
				if err := subBuilder.executeSteps(bot.Context(), bot); err != nil {
					// Check if this is a Break signal
					if _, isBreak := err.(*BreakLoop); isBreak {
						return nil // Break loop normally
					}
					return fmt.Errorf("repeat iteration %d failed: %w", i+1, err)
				}

				if !ab.checkExecutionState(bot) {
					return fmt.Errorf("routine stopped by controller during loop")
				}
			}
			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
