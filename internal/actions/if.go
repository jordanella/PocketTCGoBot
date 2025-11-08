package actions

import (
	"fmt"
)

// ElseIf represents an else-if branch with its own condition
type ElseIf struct {
	Condition Condition    `yaml:"condition"`
	Then      []ActionStep `yaml:"then"`
}

// If executes one of multiple action sequences based on conditions
type If struct {
	Condition   Condition    `yaml:"condition"`
	ThenActions []ActionStep `yaml:"then"`
	ElseIfs     []ElseIf     `yaml:"elseif,omitempty"` // Optional else-if branches
	ElseActions []ActionStep `yaml:"else,omitempty"`   // Optional final else
}

// UnmarshalYAML implements custom unmarshaling for If to handle polymorphic Condition and Actions
func (a *If) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// First, unmarshal into a raw map to inspect the structure
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	// Unmarshal the condition
	if conditionRaw, ok := raw["condition"]; ok {
		condition, err := unmarshalCondition(conditionRaw)
		if err != nil {
			return fmt.Errorf("failed to unmarshal condition: %w", err)
		}
		a.Condition = condition
	}

	// Unmarshal then actions
	if thenRaw, ok := raw["then"]; ok {
		thenActions, err := unmarshalActions(thenRaw)
		if err != nil {
			return fmt.Errorf("failed to unmarshal then actions: %w", err)
		}
		a.ThenActions = thenActions
	}

	// Unmarshal elseif branches (optional)
	if elseIfsRaw, ok := raw["elseif"]; ok {
		if elseIfsSlice, ok := elseIfsRaw.([]interface{}); ok {
			a.ElseIfs = make([]ElseIf, len(elseIfsSlice))
			for i, elseIfRaw := range elseIfsSlice {
				elseIfMap, ok := elseIfRaw.(map[string]interface{})
				if !ok {
					return fmt.Errorf("elseif %d must be a map/object", i+1)
				}

				// Unmarshal condition
				if condRaw, ok := elseIfMap["condition"]; ok {
					condition, err := unmarshalCondition(condRaw)
					if err != nil {
						return fmt.Errorf("elseif %d: failed to unmarshal condition: %w", i+1, err)
					}
					a.ElseIfs[i].Condition = condition
				}

				// Unmarshal then actions
				if thenRaw, ok := elseIfMap["then"]; ok {
					thenActions, err := unmarshalActions(thenRaw)
					if err != nil {
						return fmt.Errorf("elseif %d: failed to unmarshal then actions: %w", i+1, err)
					}
					a.ElseIfs[i].Then = thenActions
				}
			}
		}
	}

	// Unmarshal else actions (optional)
	if elseRaw, ok := raw["else"]; ok {
		elseActions, err := unmarshalActions(elseRaw)
		if err != nil {
			return fmt.Errorf("failed to unmarshal else actions: %w", err)
		}
		a.ElseActions = elseActions
	}

	return nil
}

func (a *If) Validate(ab *ActionBuilder) error {
	if a.Condition == nil {
		return fmt.Errorf("If: condition is required")
	}

	if err := a.Condition.Validate(ab); err != nil {
		return fmt.Errorf("If: invalid condition: %w", err)
	}

	if len(a.ThenActions) == 0 {
		return fmt.Errorf("If: then actions cannot be empty")
	}

	// Validate then actions
	for i, action := range a.ThenActions {
		if err := action.Validate(ab); err != nil {
			return fmt.Errorf("If -> then action %d: %w", i+1, err)
		}
	}

	// Validate elseif branches
	for i, elseIf := range a.ElseIfs {
		if elseIf.Condition == nil {
			return fmt.Errorf("If -> elseif %d: condition is required", i+1)
		}
		if err := elseIf.Condition.Validate(ab); err != nil {
			return fmt.Errorf("If -> elseif %d: invalid condition: %w", i+1, err)
		}
		if len(elseIf.Then) == 0 {
			return fmt.Errorf("If -> elseif %d: then actions cannot be empty", i+1)
		}
		for j, action := range elseIf.Then {
			if err := action.Validate(ab); err != nil {
				return fmt.Errorf("If -> elseif %d -> then action %d: %w", i+1, j+1, err)
			}
		}
	}

	// Validate else actions if present
	for i, action := range a.ElseActions {
		if err := action.Validate(ab); err != nil {
			return fmt.Errorf("If -> else action %d: %w", i+1, err)
		}
	}

	return nil
}

func (a *If) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "If",
		execute: func(bot BotInterface) error {
			// Evaluate the main condition
			result, err := a.Condition.Evaluate(bot)
			if err != nil {
				return fmt.Errorf("If: condition evaluation failed: %w", err)
			}

			var actionsToExecute []ActionStep

			// If main condition is true, execute then branch
			if result {
				actionsToExecute = a.ThenActions
			} else {
				// Try each else-if branch in order
				matched := false
				for i, elseIf := range a.ElseIfs {
					result, err := elseIf.Condition.Evaluate(bot)
					if err != nil {
						return fmt.Errorf("If: elseif %d condition evaluation failed: %w", i+1, err)
					}
					if result {
						actionsToExecute = elseIf.Then
						matched = true
						break // First matching else-if wins
					}
				}

				// If no else-if matched, try final else
				if !matched && len(a.ElseActions) > 0 {
					actionsToExecute = a.ElseActions
				}
			}

			// If no actions to execute, we're done
			if len(actionsToExecute) == 0 {
				return nil
			}

			// Build and execute the chosen actions
			steps := ab.buildSteps(actionsToExecute)
			subBuilder := &ActionBuilder{
				steps: steps,
			}

			if err := subBuilder.executeSteps(bot.Context(), bot); err != nil {
				return fmt.Errorf("If: execution failed: %w", err)
			}

			return nil
		},
		issue: a.Validate(ab),
	}

	ab.steps = append(ab.steps, step)
	return ab
}
