package actions

import (
	"fmt"
	"time"
)

// While executes actions repeatedly while a condition is true
type While struct {
	Condition   Condition    `yaml:"condition"`
	Actions     []ActionStep `yaml:"actions"`
	MaxAttempts int          `yaml:"max_attempts,omitempty"` // Optional: 0 means infinite
}

// UnmarshalYAML implements custom unmarshaling for While to handle polymorphic Condition and Actions
func (a *While) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First, unmarshal into a raw map to inspect the structure
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	// Unmarshal max_attempts
	if maxAttempts, ok := raw["max_attempts"]; ok {
		if val, ok := maxAttempts.(int); ok {
			a.MaxAttempts = val
		}
	}

	// Unmarshal the condition
	if conditionRaw, ok := raw["condition"]; ok {
		condition, err := unmarshalCondition(conditionRaw)
		if err != nil {
			return fmt.Errorf("failed to unmarshal condition: %w", err)
		}
		a.Condition = condition
	}

	// Unmarshal actions
	if actionsRaw, ok := raw["actions"]; ok {
		actions, err := unmarshalActions(actionsRaw)
		if err != nil {
			return fmt.Errorf("failed to unmarshal actions: %w", err)
		}
		a.Actions = actions
	}

	return nil
}

func (a *While) Validate(ab *ActionBuilder) error {
	if a.Condition == nil {
		return fmt.Errorf("While: condition is required")
	}

	if err := a.Condition.Validate(ab); err != nil {
		return fmt.Errorf("While: invalid condition: %w", err)
	}

	if a.MaxAttempts < 0 {
		return fmt.Errorf("While: max_attempts must be non-negative")
	}

	if len(a.Actions) == 0 {
		return fmt.Errorf("While: actions cannot be empty")
	}

	// Validate actions
	for i, action := range a.Actions {
		if err := action.Validate(ab); err != nil {
			return fmt.Errorf("While -> action %d: %w", i+1, err)
		}
	}

	return nil
}

func (a *While) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "While",
		execute: func(bot BotInterface) error {
			// Build the nested actions once
			nestedSteps := ab.buildSteps(a.Actions)

			attempt := 0
			for {
				// Check max attempts
				if a.MaxAttempts > 0 && attempt >= a.MaxAttempts {
					return fmt.Errorf("While: exceeded max attempts (%d)", a.MaxAttempts)
				}

				// Check pause/stop state
				if !ab.checkExecutionState(bot) {
					return fmt.Errorf("While: routine stopped by controller")
				}

				// Evaluate condition
				result, err := a.Condition.Evaluate(bot)
				if err != nil {
					return fmt.Errorf("While: condition evaluation failed: %w", err)
				}

				// Exit loop if condition is false
				if !result {
					return nil
				}

				// Execute the actions
				subBuilder := &ActionBuilder{
					steps: nestedSteps,
				}

				if err := subBuilder.executeSteps(bot.Context(), bot); err != nil {
					// Check if this is a Break signal
					if _, isBreak := err.(*BreakLoop); isBreak {
						return nil // Break loop normally
					}
					return fmt.Errorf("While: iteration %d failed: %w", attempt+1, err)
				}

				attempt++
				time.Sleep(100 * time.Millisecond)
			}
		},
		issue: a.Validate(ab),
	}

	ab.steps = append(ab.steps, step)
	return ab
}
