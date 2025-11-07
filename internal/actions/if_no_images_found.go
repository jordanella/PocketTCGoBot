package actions

import (
	"fmt"
)

type IfNoImagesFound struct {
	Templates []string     `yaml:"templates"`
	Actions   []ActionStep `yaml:"actions"`
}

// UnmarshalYAML implements custom unmarshaling for IfNoImagesFound to handle polymorphic Actions field
func (a *IfNoImagesFound) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	// Extract fields
	if temps, ok := raw["templates"].([]interface{}); ok {
		a.Templates = make([]string, len(temps))
		for i, t := range temps {
			if str, ok := t.(string); ok {
				a.Templates[i] = str
			}
		}
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

func (a *IfNoImagesFound) Validate(ab *ActionBuilder) error {
	// Template name is required
	if len(a.Templates) == 0 {
		return fmt.Errorf("at least one template is required")
	}

	// Validate template exists in registry (if registry is available)
	if ab.templateRegistry != nil {
		for _, tmpl := range a.Templates {
			// Validate template exists in registry
			if !ab.templateRegistry.Has(tmpl) {
				return fmt.Errorf("template '%s' not found in registry", tmpl)
			}
		}
	}

	if len(a.Actions) == 0 {
		return fmt.Errorf("actions cannot be empty")
	}

	// Validate nested actions with better error context
	for i, action := range a.Actions {
		if err := action.Validate(ab); err != nil {
			return fmt.Errorf("IfNoImagesFound (%s) -> nested action %d: %w", a.Templates, i+1, err)
		}
	}

	return nil
}

func (a *IfNoImagesFound) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("IfNoImagesFound (%s)", a.Templates),
		execute: func(bot BotInterface) error {
			// Build the nested actions into a concrete slice of executable steps
			nestedSteps := ab.buildSteps(a.Actions)

			// Check if any template exists
			bot.CV().InvalidateCache()
			templateFound := false
			for _, tmpl := range a.Templates {
				template, config, err := buildTemplateConfiguration(bot, tmpl, nil, nil)
				if err != nil {
					return fmt.Errorf("failed to build template configuration: %w", err)
				}

				result, err := bot.CV().FindTemplate(tmpl, config)
				if err != nil {
					return fmt.Errorf("error checking template %s existence: %w", template.Name, err)
				}
				if result.Found {
					templateFound = true
				}
			}
			if templateFound {
				return nil
			}

			// Re-execute the action builder's steps
			subBuilder := &ActionBuilder{
				steps: nestedSteps,
			}

			// Call the internal execution function with the bot
			if err := subBuilder.executeSteps(bot.Context(), bot); err != nil {
				return fmt.Errorf("IfNoImagesFound (%d) failed: %w", len(a.Templates), err)
			}

			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
