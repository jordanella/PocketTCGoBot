package actions

import (
	"fmt"

	"jordanella.com/pocket-tcg-go/internal/cv"
)

type ClickIfImageFound struct {
	Template  string     `yaml:"template"`            // Template lookup by name (required)
	Threshold *float64   `yaml:"threshold,omitempty"` // Optional: override template's threshold
	Region    *cv.Region `yaml:"region,omitempty"`    // Optional: override template's region
	Point     *cv.Point  `yaml:"point,omitempty"`
	Offset    *cv.Point  `yaml:"offset,omitempty"`
}

func (a *ClickIfImageFound) Validate(ab *ActionBuilder) error {
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

	if a.Point != nil && a.Offset != nil {
		return fmt.Errorf("cannot specify both 'point' and 'offset'")
	}

	return nil
}

func (a *ClickIfImageFound) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("ClickIfImageFound (%s)", a.Template),
		execute: func(bot BotInterface) error {
			template, config, err := buildTemplateConfiguration(bot, a.Template, a.Threshold, a.Region)
			if err != nil {
				return fmt.Errorf("failed to build template configuration: %w", err)
			}

			result, err := bot.CV().FindTemplate(template.Name, config)
			if err != nil {
				return fmt.Errorf("failed to find template: %w", err)
			}

			if !result.Found {
				return nil
			}

			clickX := result.Location.X + (template.Region.X2-template.Region.X1)/2
			clickY := result.Location.X + (template.Region.X2-template.Region.X1)/2

			if a.Point != nil {
				clickX = a.Point.X
				clickY = a.Point.Y
			} else if a.Offset != nil {
				clickX += a.Offset.X
				clickY += a.Offset.Y
			}

			return bot.ADB().Click(clickX, clickY)
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
