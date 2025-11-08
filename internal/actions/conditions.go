package actions

import (
	"fmt"

	"jordanella.com/pocket-tcg-go/internal/cv"
)

// Condition represents a boolean condition that can be evaluated
type Condition interface {
	Evaluate(bot BotInterface) (bool, error)
	Validate(ab *ActionBuilder) error
}

// ConditionType is used for YAML unmarshaling to determine which condition type to create
type ConditionType struct {
	Type string `yaml:"type"`
}

// ImageExists checks if a template image exists on screen
type ImageExists struct {
	Template  string      `yaml:"template"`            // Template lookup by name (required)
	Threshold *float64    `yaml:"threshold,omitempty"` // Optional: override template's threshold
	Region    *cv.Region  `yaml:"region,omitempty"`    // Optional: override template's region
}

func (c *ImageExists) Validate(ab *ActionBuilder) error {
	if c.Template == "" {
		return fmt.Errorf("ImageExists: template is required")
	}

	// Validate template exists in registry (if registry is available)
	if ab.templateRegistry != nil {
		if !ab.templateRegistry.Has(c.Template) {
			return fmt.Errorf("ImageExists: template '%s' not found in registry", c.Template)
		}
	}

	return nil
}

func (c *ImageExists) Evaluate(bot BotInterface) (bool, error) {
	template, config, err := buildTemplateConfiguration(bot, c.Template, c.Threshold, c.Region)
	if err != nil {
		return false, fmt.Errorf("ImageExists: failed to build template configuration: %w", err)
	}

	bot.CV().InvalidateCache()
	result, err := bot.CV().FindTemplate(template.Name, config)
	if err != nil {
		return false, fmt.Errorf("ImageExists: error checking template %s: %w", template.Name, err)
	}

	return result.Found, nil
}

// ImageNotExists checks if a template image does NOT exist on screen
type ImageNotExists struct {
	Template  string      `yaml:"template"`            // Template lookup by name (required)
	Threshold *float64    `yaml:"threshold,omitempty"` // Optional: override template's threshold
	Region    *cv.Region  `yaml:"region,omitempty"`    // Optional: override template's region
}

func (c *ImageNotExists) Validate(ab *ActionBuilder) error {
	if c.Template == "" {
		return fmt.Errorf("ImageNotExists: template is required")
	}

	// Validate template exists in registry (if registry is available)
	if ab.templateRegistry != nil {
		if !ab.templateRegistry.Has(c.Template) {
			return fmt.Errorf("ImageNotExists: template '%s' not found in registry", c.Template)
		}
	}

	return nil
}

func (c *ImageNotExists) Evaluate(bot BotInterface) (bool, error) {
	template, config, err := buildTemplateConfiguration(bot, c.Template, c.Threshold, c.Region)
	if err != nil {
		return false, fmt.Errorf("ImageNotExists: failed to build template configuration: %w", err)
	}

	bot.CV().InvalidateCache()
	result, err := bot.CV().FindTemplate(template.Name, config)
	if err != nil {
		return false, fmt.Errorf("ImageNotExists: error checking template %s: %w", template.Name, err)
	}

	return !result.Found, nil
}

// Not negates a condition
type Not struct {
	Condition Condition `yaml:"condition"`
}

// UnmarshalYAML implements custom unmarshaling for Not to handle polymorphic Condition
func (c *Not) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	if conditionRaw, ok := raw["condition"]; ok {
		condition, err := unmarshalCondition(conditionRaw)
		if err != nil {
			return fmt.Errorf("Not: failed to unmarshal condition: %w", err)
		}
		c.Condition = condition
	}

	return nil
}

func (c *Not) Validate(ab *ActionBuilder) error {
	if c.Condition == nil {
		return fmt.Errorf("Not: condition is required")
	}
	return c.Condition.Validate(ab)
}

func (c *Not) Evaluate(bot BotInterface) (bool, error) {
	result, err := c.Condition.Evaluate(bot)
	if err != nil {
		return false, fmt.Errorf("Not: %w", err)
	}
	return !result, nil
}

// All checks if ALL conditions are true (AND logic)
type All struct {
	Conditions []Condition `yaml:"conditions"`
}

// UnmarshalYAML implements custom unmarshaling for All to handle polymorphic Conditions
func (c *All) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	if conditionsRaw, ok := raw["conditions"]; ok {
		conditions, err := unmarshalConditions(conditionsRaw)
		if err != nil {
			return fmt.Errorf("All: failed to unmarshal conditions: %w", err)
		}
		c.Conditions = conditions
	}

	return nil
}

func (c *All) Validate(ab *ActionBuilder) error {
	if len(c.Conditions) == 0 {
		return fmt.Errorf("All: at least one condition is required")
	}

	for i, cond := range c.Conditions {
		if cond == nil {
			return fmt.Errorf("All: condition %d is nil", i+1)
		}
		if err := cond.Validate(ab); err != nil {
			return fmt.Errorf("All: condition %d: %w", i+1, err)
		}
	}

	return nil
}

func (c *All) Evaluate(bot BotInterface) (bool, error) {
	for i, cond := range c.Conditions {
		result, err := cond.Evaluate(bot)
		if err != nil {
			return false, fmt.Errorf("All: condition %d: %w", i+1, err)
		}
		if !result {
			return false, nil // Short-circuit: if any is false, return false
		}
	}
	return true, nil
}

// Any checks if ANY condition is true (OR logic)
type Any struct {
	Conditions []Condition `yaml:"conditions"`
}

// UnmarshalYAML implements custom unmarshaling for Any to handle polymorphic Conditions
func (c *Any) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	if conditionsRaw, ok := raw["conditions"]; ok {
		conditions, err := unmarshalConditions(conditionsRaw)
		if err != nil {
			return fmt.Errorf("Any: failed to unmarshal conditions: %w", err)
		}
		c.Conditions = conditions
	}

	return nil
}

func (c *Any) Validate(ab *ActionBuilder) error {
	if len(c.Conditions) == 0 {
		return fmt.Errorf("Any: at least one condition is required")
	}

	for i, cond := range c.Conditions {
		if cond == nil {
			return fmt.Errorf("Any: condition %d is nil", i+1)
		}
		if err := cond.Validate(ab); err != nil {
			return fmt.Errorf("Any: condition %d: %w", i+1, err)
		}
	}

	return nil
}

func (c *Any) Evaluate(bot BotInterface) (bool, error) {
	for i, cond := range c.Conditions {
		result, err := cond.Evaluate(bot)
		if err != nil {
			return false, fmt.Errorf("Any: condition %d: %w", i+1, err)
		}
		if result {
			return true, nil // Short-circuit: if any is true, return true
		}
	}
	return false, nil
}

// None checks if NONE of the conditions are true (NOR logic)
type None struct {
	Conditions []Condition `yaml:"conditions"`
}

// UnmarshalYAML implements custom unmarshaling for None to handle polymorphic Conditions
func (c *None) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw map[string]interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	if conditionsRaw, ok := raw["conditions"]; ok {
		conditions, err := unmarshalConditions(conditionsRaw)
		if err != nil {
			return fmt.Errorf("None: failed to unmarshal conditions: %w", err)
		}
		c.Conditions = conditions
	}

	return nil
}

func (c *None) Validate(ab *ActionBuilder) error {
	if len(c.Conditions) == 0 {
		return fmt.Errorf("None: at least one condition is required")
	}

	for i, cond := range c.Conditions {
		if cond == nil {
			return fmt.Errorf("None: condition %d is nil", i+1)
		}
		if err := cond.Validate(ab); err != nil {
			return fmt.Errorf("None: condition %d: %w", i+1, err)
		}
	}

	return nil
}

func (c *None) Evaluate(bot BotInterface) (bool, error) {
	for i, cond := range c.Conditions {
		result, err := cond.Evaluate(bot)
		if err != nil {
			return false, fmt.Errorf("None: condition %d: %w", i+1, err)
		}
		if result {
			return false, nil // Short-circuit: if any is true, return false
		}
	}
	return true, nil
}
