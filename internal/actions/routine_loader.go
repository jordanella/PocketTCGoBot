package actions

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	// ... other necessary imports
)

type RoutineLoader struct {
	templateRegistry TemplateRegistryInterface // Optional: for build-time validation
}

func NewRoutineLoader() *RoutineLoader {
	return &RoutineLoader{}
}

// WithTemplateRegistry sets the template registry for build-time validation
func (rl *RoutineLoader) WithTemplateRegistry(registry TemplateRegistryInterface) *RoutineLoader {
	rl.templateRegistry = registry
	return rl
}

// LoadFromFile reads a YAML file, unmarshals the Routine, validates all actions,
// and builds the final executable ActionBuilder that can be executed on any bot.
func (rl *RoutineLoader) LoadFromFile(filepath string) (*ActionBuilder, error) {
	// 1. Read the File
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read routine file %s: %w", filepath, err)
	}

	var routine Routine
	// 2. Unmarshal the YAML (using the custom UnmarshalYAML handler for polymorphism)
	if err := yaml.Unmarshal(data, &routine); err != nil {
		return nil, fmt.Errorf("failed to unmarshal routine YAML: %w", err)
	}

	// 3. Create the new ActionBuilder with optional template registry
	ab := NewActionBuilder()
	if rl.templateRegistry != nil {
		ab.WithTemplateRegistry(rl.templateRegistry)
	}

	// 4. Validate and Build all steps
	// Note: We use the *same* ActionBuilder (ab) for both validation and building
	// to ensure nested builders (like WhileTemplateExists) get a valid reference.
	for i, action := range routine.Steps {
		// Validation Check (Fails fast if invalid configuration)
		if err := action.Validate(ab); err != nil {
			return nil, fmt.Errorf("routine '%s' step %d validation failed: %w", routine.RoutineName, i+1, err)
		}

		// Build the step (appends the executable Step to ab.steps and captures
		// the 'issue' error if validation passed but was captured in 'issue')
		ab = action.Build(ab)
	}

	// The ab.steps slice now holds the entire executable routine
	return ab, nil
}
