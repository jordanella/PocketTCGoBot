package actions

import (
	"fmt"

	"jordanella.com/pocket-tcg-go/internal/cv"
)

type IfImageFound struct {
	Template  string       `yaml:"template"`            // Template lookup by name (required)
	Threshold *float64     `yaml:"threshold,omitempty"` // Optional: override template's threshold
	Region    *cv.Region   `yaml:"region,omitempty"`    // Optional: override template's region
	Actions   []ActionStep `yaml:"actions"`
}

// UnmarshalYAML implements custom unmarshaling for IfImageFound to handle polymorphic Actions field
func (a *IfImageFound) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	// Extract fields
	if val, ok := raw["template"].(string); ok {
		a.Template = val
	}
	if val, ok := raw["threshold"].(float64); ok {
		f := val
		a.Threshold = &f
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
		a.Region = &r
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

func (a *IfImageFound) Validate(ab *ActionBuilder) error {
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
			return fmt.Errorf("IfImageFound (%s) -> nested action %d: %w", a.Template, i+1, err)
		}
	}

	return nil
}

func (a *IfImageFound) Build(ab *ActionBuilder) *ActionBuilder {

	step := Step{
		name: fmt.Sprintf("IfImageFound (%s)", a.Template),
		execute: func(bot BotInterface) error {
			// Build the nested actions into a concrete slice of executable steps
			nestedSteps := ab.buildSteps(a.Actions)

			template, config, err := buildTemplateConfiguration(bot, a.Template, a.Threshold, a.Region)
			if err != nil {
				return fmt.Errorf("failed to build template configuration: %w", err)
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
				return fmt.Errorf("IfImageFound (%s) -> nested action failed: %w", a.Template, err)
			}

			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
