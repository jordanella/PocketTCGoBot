package actions

import (
	"fmt"
	"time"

	"jordanella.com/pocket-tcg-go/internal/cv"
)

type WaitForImage struct {
	Timeout   int        `yaml:"timeout"`
	Template  string     `yaml:"template"`            // Template lookup by name (required)
	Threshold *float64   `yaml:"threshold,omitempty"` // Optional: override template's threshold
	Region    *cv.Region `yaml:"region,omitempty"`    // Optional: override template's region
}

func (a *WaitForImage) Validate(ab *ActionBuilder) error {
	if a.Timeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
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

	return nil
}

func (a *WaitForImage) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("WaitForTemplate(%s, %v)", a.Template, a.Timeout),
		execute: func(bot BotInterface) error {
			duration := time.Second * time.Duration(a.Timeout)

			_, config, err := buildTemplateConfiguration(bot, a.Template, a.Threshold, a.Region)
			if err != nil {
				return fmt.Errorf("failed to build template configuration: %w", err)
			}

			result, err := bot.CV().WaitForTemplate(a.Template, config, duration)
			if err != nil {
				return fmt.Errorf("template wait timeout: %w", err)
			}

			if !result.Found {
				return fmt.Errorf("template not found within timeout")
			}

			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
