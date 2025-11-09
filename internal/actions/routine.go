package actions

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"jordanella.com/pocket-tcg-go/internal/cv"
)

// Routine holds the entire routine definition from the YAML file
type Routine struct {
	RoutineName string        `yaml:"routine_name"`
	Description string        `yaml:"description,omitempty"` // Optional description of the routine's purpose
	Tags        []string      `yaml:"tags,omitempty"`        // Optional tags for organization (e.g., "sentry", "navigation", "combat")
	Config      []ConfigParam `yaml:"config,omitempty"`      // Optional user-configurable parameters
	Steps       []ActionStep  `yaml:"steps"`                 // ActionStep is the interface you already defined
	Sentries    []Sentry      `yaml:"sentries,omitempty"`    // Sentry definitions for error handling
}

// StepMetadata holds timeout and retry configuration for a step
type StepMetadata struct {
	Timeout     time.Duration // Timeout for the step (0 = no timeout)
	MaxAttempts int           // Maximum number of attempts (0 or 1 = no retries)
	RetryDelay  time.Duration // Delay between retries (0 = use default)
}

// HasMetadata returns true if any metadata is set
func (sm StepMetadata) HasMetadata() bool {
	return sm.Timeout > 0 || sm.MaxAttempts > 1 || sm.RetryDelay > 0
}

// ActionWithMetadata wraps an ActionStep with execution metadata
type ActionWithMetadata struct {
	Action   ActionStep
	Metadata StepMetadata
}

// Validate delegates to the wrapped action
func (a *ActionWithMetadata) Validate(ab *ActionBuilder) error {
	return a.Action.Validate(ab)
}

// Build delegates to the wrapped action and applies metadata to the built step
func (a *ActionWithMetadata) Build(ab *ActionBuilder) *ActionBuilder {
	// Build the action normally
	ab = a.Action.Build(ab)

	// Apply metadata to the last added step
	if len(ab.steps) > 0 {
		lastStep := &ab.steps[len(ab.steps)-1]
		if a.Metadata.Timeout > 0 {
			lastStep.timeout = a.Metadata.Timeout
		}
		if a.Metadata.MaxAttempts > 1 {
			lastStep.maxAttempts = a.Metadata.MaxAttempts
		}
		if a.Metadata.RetryDelay > 0 {
			lastStep.retryDelay = a.Metadata.RetryDelay
		}
	}

	return ab
}

// Custom Unmarshaler for polymorphic actions (Steps)
// This is required because 'steps' is a list of interfaces (ActionStep),
// and YAML doesn't know which concrete struct to use without help.
func (r *Routine) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Unmarshal into a raw map to handle the fields manually
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	// Extract the routine_name
	if name, ok := raw["routine_name"].(string); ok {
		r.RoutineName = name
	}

	// Extract the description
	if desc, ok := raw["description"].(string); ok {
		r.Description = desc
	}

	// Extract the tags
	if tagsRaw, ok := raw["tags"].([]interface{}); ok {
		r.Tags = make([]string, len(tagsRaw))
		for i, tag := range tagsRaw {
			if tagStr, ok := tag.(string); ok {
				r.Tags[i] = tagStr
			}
		}
	}

	// Extract sentries (will be unmarshaled separately)
	if sentriesRaw, ok := raw["sentries"].([]interface{}); ok {
		r.Sentries = make([]Sentry, len(sentriesRaw))
		for i, sentryRaw := range sentriesRaw {
			// Marshal back to YAML and unmarshal into Sentry struct
			sentryBytes, err := yaml.Marshal(sentryRaw)
			if err != nil {
				return fmt.Errorf("sentry %d: error marshaling: %w", i+1, err)
			}
			if err := yaml.Unmarshal(sentryBytes, &r.Sentries[i]); err != nil {
				return fmt.Errorf("sentry %d: error unmarshaling: %w", i+1, err)
			}
		}
	}

	// Now, handle the 'steps' as a raw slice
	stepsRaw, ok := raw["steps"]
	if !ok || stepsRaw == nil {
		// No steps is valid - just return
		return nil
	}

	stepsSlice, ok := stepsRaw.([]interface{})
	if !ok {
		return fmt.Errorf("'steps' field must be a list")
	}

	// Convert each step to a map[string]interface{}
	rawSteps := make([]map[string]interface{}, len(stepsSlice))
	for i, step := range stepsSlice {
		stepMap, ok := step.(map[string]interface{})
		if !ok {
			return fmt.Errorf("step %d: must be a map/object", i+1)
		}
		rawSteps[i] = stepMap
	}

	// Iterate through the raw steps and map them to concrete ActionStep types
	r.Steps = make([]ActionStep, len(rawSteps))
	for i, rawStep := range rawSteps {
		actionType, ok := rawStep["action"].(string)
		if !ok || actionType == "" {
			return fmt.Errorf("step %d: missing or invalid 'action' field", i+1)
		}

		// Extract step metadata (timeout, max_attempts, retry_delay) before unmarshaling
		var stepMetadata StepMetadata
		if timeoutMs, ok := rawStep["timeout"].(int); ok {
			stepMetadata.Timeout = time.Duration(timeoutMs) * time.Millisecond
		}
		if maxAttempts, ok := rawStep["max_attempts"].(int); ok {
			stepMetadata.MaxAttempts = maxAttempts
		}
		if retryDelayMs, ok := rawStep["retry_delay"].(int); ok {
			stepMetadata.RetryDelay = time.Duration(retryDelayMs) * time.Millisecond
		}

		// Look up the concrete struct type in the registry
		stepType, found := actionRegistry[strings.ToLower(actionType)]
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

		// Wrap the action with metadata if any was specified
		if stepMetadata.HasMetadata() {
			r.Steps[i] = &ActionWithMetadata{
				Action:   action,
				Metadata: stepMetadata,
			}
		} else {
			r.Steps[i] = action
		}
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

// unmarshalActionWithNestedSteps is a generic helper for unmarshaling actions that contain nested ActionStep slices
// This handles the polymorphic unmarshaling of the nested actions field
func unmarshalActionWithNestedSteps(unmarshal func(interface{}) error, maxAttempts *int, template *string, threshold **float64, region **cv.Region, actions *[]ActionStep) error {
	// Unmarshal into a raw map to handle the fields manually
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	// Extract fields
	if val, ok := raw["max_attempts"].(int); ok {
		*maxAttempts = val
	}
	if val, ok := raw["template"].(string); ok {
		*template = val
	}
	if val, ok := raw["threshold"].(float64); ok {
		f := val
		*threshold = &f
	}

	// Handle region if present
	if regionRaw, ok := raw["region"].(map[string]interface{}); ok {
		var r cv.Region
		if x1, ok := regionRaw["x1"].(int); ok {
			r.X1 = x1
		}
		if y1, ok := regionRaw["y1"].(int); ok {
			r.Y1 = y1
		}
		if x2, ok := regionRaw["x2"].(int); ok {
			r.X2 = x2
		}
		if y2, ok := regionRaw["y2"].(int); ok {
			r.Y2 = y2
		}
		*region = &r
	}

	// Handle the nested actions
	actionsRaw, ok := raw["actions"]
	if !ok || actionsRaw == nil {
		return nil
	}

	unmarshaledActions, err := unmarshalNestedActions(actionsRaw)
	if err != nil {
		return err
	}
	*actions = unmarshaledActions

	return nil
}

// unmarshalNestedActions is a helper that unmarshals a polymorphic actions field
func unmarshalNestedActions(actionsRaw interface{}) ([]ActionStep, error) {
	if actionsRaw == nil {
		return nil, nil
	}

	actionsSlice, ok := actionsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("'actions' field must be a list")
	}

	// Convert each action to a map[string]interface{}
	rawSteps := make([]map[string]interface{}, len(actionsSlice))
	for i, step := range actionsSlice {
		stepMap, ok := step.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("action %d: must be a map/object", i+1)
		}
		rawSteps[i] = stepMap
	}

	// Unmarshal each action
	actions := make([]ActionStep, len(rawSteps))
	for i, rawStep := range rawSteps {
		actionType, ok := rawStep["action"].(string)
		if !ok || actionType == "" {
			return nil, fmt.Errorf("action %d: missing or invalid 'action' field", i+1)
		}

		// Look up the concrete struct type in the registry
		stepType, found := actionRegistry[strings.ToLower(actionType)]
		if !found {
			return nil, fmt.Errorf("action %d: unknown action type '%s' (available types: %v)", i+1, actionType, getRegisteredActions())
		}

		// Create an instance of the concrete type (which implements ActionStep)
		action := reflect.New(stepType).Interface().(ActionStep)

		// Marshal the raw map back to YAML, then unmarshal it into the concrete struct
		stepBytes, err := yaml.Marshal(rawStep)
		if err != nil {
			return nil, fmt.Errorf("action %d (%s): error marshaling raw step: %w", i+1, actionType, err)
		}
		if err := yaml.Unmarshal(stepBytes, action); err != nil {
			return nil, fmt.Errorf("action %d (%s): error unmarshaling into %T: %w", i+1, actionType, action, err)
		}

		actions[i] = action
	}

	return actions, nil
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
