package actions

import (
	"fmt"
	"os"
	"reflect"

	"gopkg.in/yaml.v3"
)

// Routine holds the entire routine definition from the YAML file
type Routine struct {
	RoutineName string       `yaml:"routine_name"`
	Steps       []ActionStep `yaml:"steps"` // ActionStep is the interface you already defined
}

// Custom Unmarshaler for polymorphic actions (Steps)
// This is required because 'steps' is a list of interfaces (ActionStep),
// and YAML doesn't know which concrete struct to use without help.
func (r *Routine) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// A temporary struct to unmarshal everything except 'steps'
	type routineAlias Routine
	alias := routineAlias{}

	// First, unmarshal the simple fields
	if err := unmarshal(&alias); err != nil {
		return err
	}
	r.RoutineName = alias.RoutineName

	// Now, handle the 'steps' as a raw slice of map[string]interface{}
	var rawSteps []map[string]interface{}
	if err := unmarshal(&map[string]interface{}{"steps": &rawSteps}); err != nil {
		return err
	}

	// Iterate through the raw steps and map them to concrete ActionStep types
	r.Steps = make([]ActionStep, len(rawSteps))
	for i, rawStep := range rawSteps {
		actionType, ok := rawStep["action"].(string)
		if !ok || actionType == "" {
			return fmt.Errorf("step %d: missing or invalid 'action' field", i+1)
		}

		// Look up the concrete struct type in the registry
		stepType, found := actionRegistry[actionType]
		if !found {
			return fmt.Errorf("step %d: unknown action type '%s' (available types: %v)", i+1, actionType, getRegisteredActions())
		}

		// Create an instance of the concrete type (which implements ActionStep)
		action := reflect.New(stepType).Interface().(ActionStep)

		// Marshal the raw map back to YAML, then unmarshal it into the concrete struct
		stepBytes, err := yaml.Marshal(rawStep)
		if err != nil {
			return fmt.Errorf("step %d (%s): error marshaling raw step: %w", i+1, actionType, err)
		}
		if err := yaml.Unmarshal(stepBytes, action); err != nil {
			return fmt.Errorf("step %d (%s): error unmarshaling into %T: %w", i+1, actionType, action, err)
		}

		r.Steps[i] = action
	}

	return nil
}

// getRegisteredActions returns a list of all registered action types for error messages
func getRegisteredActions() []string {
	actions := make([]string, 0, len(actionRegistry))
	for name := range actionRegistry {
		actions = append(actions, name)
	}
	return actions
}

// NewActionBuilderFromRoutine loads a YAML file, unmarshals the Routine,
// validates all actions, and builds the executable ActionBuilder.
// The returned ActionBuilder can be executed on any bot by calling Execute(bot)
func NewActionBuilderFromRoutine(filepath string) (*ActionBuilder, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read routine file %s: %w", filepath, err)
	}

	var routine Routine
	if err := yaml.Unmarshal(data, &routine); err != nil {
		return nil, fmt.Errorf("failed to unmarshal routine YAML: %w", err)
	}

	// 1. Create the new ActionBuilder
	ab := NewActionBuilder()

	// 2. Validate all steps recursively
	for i, action := range routine.Steps {
		if err := action.Validate(ab); err != nil {
			return nil, fmt.Errorf("routine '%s' step %d validation failed: %w", routine.RoutineName, i+1, err)
		}
	}

	// 3. Build the steps: this recursively appends the executable Step structs
	//    to ab.steps by calling the Build method on each ActionStep.
	for _, action := range routine.Steps {
		ab = action.Build(ab)
	}

	// The ab.steps slice now holds the entire executable routine
	return ab, nil
}
