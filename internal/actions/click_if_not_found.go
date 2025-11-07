package actions

import (
	"fmt"

	"jordanella.com/pocket-tcg-go/internal/cv"
)

type ClickIfImageNotFound struct {
	Template  string     `yaml:"template"`            // Template lookup by name (required)
	Threshold *float64   `yaml:"threshold,omitempty"` // Optional: override template's threshold
	Region    *cv.Region `yaml:"region,omitempty"`    // Optional: override template's region
	X         int        `yaml:"x"`
	Y         int        `yaml:"y"`
}

func (a *ClickIfImageNotFound) Validate(ab *ActionBuilder) error {
	// Template name is required
	if a.Template == "" {
		return fmt.Errorf("template_name is required")
	}

	// Validate template exists in registry (if registry is available)
	if ab.templateRegistry != nil {
		if !ab.templateRegistry.Has(a.Template) {
			return fmt.Errorf("template '%s' not found in registry", a.Template)
		}
	}

	if a.X < 0 || a.Y < 0 {
		return fmt.Errorf("coordinates (x=%d, y=%d) must be non-negative", a.X, a.Y)
	}

	return nil
}

func (a *ClickIfImageNotFound) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("ClickIfImageNotFound (%s)", a.Template),
		execute: func(bot BotInterface) error {
			template, config, err := buildTemplateConfiguration(bot, a.Template, a.Threshold, a.Region)
			if err != nil {
				return fmt.Errorf("failed to build template configuration: %w", err)
			}

			result, err := bot.CV().FindTemplate(template.Name, config)
			if err != nil {
				return fmt.Errorf("failed to find template: %w", err)
			}

			if result.Found {
				return nil
			}

			return bot.ADB().Click(a.X, a.Y)
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
