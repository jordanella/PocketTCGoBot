package actions

import (
	"fmt"
	"time"

	"jordanella.com/pocket-tcg-go/internal/cv"
)

type WhileImageFound struct {
	MaxAttempts int          `yaml:"max_attempts"`
	Template    string       `yaml:"template"`            // Template lookup by name (required)
	Threshold   *float64     `yaml:"threshold,omitempty"` // Optional: override template's threshold
	Region      *cv.Region   `yaml:"region,omitempty"`    // Optional: override template's region
	Actions     []ActionStep `yaml:"actions"`
}

// UnmarshalYAML implements custom unmarshaling for WhileImageFound to handle polymorphic Actions field
func (a *WhileImageFound) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshalActionWithNestedSteps(unmarshal, &a.MaxAttempts, &a.Template, &a.Threshold, &a.Region, &a.Actions)
}

func (a *WhileImageFound) Validate(ab *ActionBuilder) error {
	if a.MaxAttempts < 0 {
		return fmt.Errorf("max_attempts must be non-negative")
	}

	// Template name is required
	if a.Template == "" {
		return fmt.Errorf("template is required")
	}

	// Validate template exists in registry (if registry is available)
	if ab.templateRegistry != nil {
		if !ab.templateRegistry.Has(a.Template) {
			return fmt.Errorf("template '%s' not found in registry", a.Template)
		}
	}

	if len(a.Actions) == 0 {
		return fmt.Errorf("actions cannot be empty")
	}

	// Validate nested actions with better error context
	for i, action := range a.Actions {
		if err := action.Validate(ab); err != nil {
			return fmt.Errorf("WhileImageFound (%s) -> nested action %d: %w", a.Template, i+1, err)
		}
	}

	return nil
}

func (a *WhileImageFound) Build(ab *ActionBuilder) *ActionBuilder {

	step := Step{
		name: fmt.Sprintf("WhileImageFound (%s)", a.Template),
		execute: func(bot BotInterface) error {
			// Build the nested actions into a concrete slice of executable steps
			nestedSteps := ab.buildSteps(a.Actions)

			template, config, err := buildTemplateConfiguration(bot, a.Template, a.Threshold, a.Region)
			if err != nil {
				return fmt.Errorf("failed to build template configuration: %w", err)
			}

			attempt := 0
			for {
				if a.MaxAttempts > 0 && attempt >= a.MaxAttempts {
					return fmt.Errorf("template %s still exists after %d attempts", template.Name, a.MaxAttempts)
				}

				bot.CV().InvalidateCache()

				// Exit if template no longer exists
				// Use template name for registry cache lookup
				result, err := bot.CV().FindTemplate(template.Name, config)
				if err != nil {
					return fmt.Errorf("error checking template %s existence: %w", template.Name, err)
				}
				if !result.Found {
					return nil
				}

				// 2. Execute the pre-built nested steps
				subBuilder := &ActionBuilder{
					steps: nestedSteps,
				}

				// Call the internal execution function with the bot
				if err := subBuilder.executeSteps(bot.Context(), bot); err != nil {
					// Check if this is a Break signal
					if _, isBreak := err.(*BreakLoop); isBreak {
						return nil // Break loop normally
					}
					return fmt.Errorf("loop iteration %d failed: %w", attempt+1, err)
				}

				attempt++
				time.Sleep(100 * time.Millisecond)

				if !ab.checkExecutionState(bot) {
					return fmt.Errorf("routine stopped by controller during loop")
				}
			}
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
