package actions

import (
	"fmt"
	"strconv"
	"strings"
)

// VariableEquals checks if a variable equals a specific value
type VariableEquals struct {
	Variable string `yaml:"variable"`
	Value    string `yaml:"value"`
}

func (c *VariableEquals) Validate(ab *ActionBuilder) error {
	if c.Variable == "" {
		return fmt.Errorf("VariableEquals: variable name is required")
	}
	return nil
}

func (c *VariableEquals) Evaluate(bot BotInterface) (bool, error) {
	value, ok := bot.Variables().Get(c.Variable)
	if !ok {
		return false, fmt.Errorf("VariableEquals: variable '%s' not found", c.Variable)
	}
	return value == c.Value, nil
}

// VariableNotEquals checks if a variable does not equal a specific value
type VariableNotEquals struct {
	Variable string `yaml:"variable"`
	Value    string `yaml:"value"`
}

func (c *VariableNotEquals) Validate(ab *ActionBuilder) error {
	if c.Variable == "" {
		return fmt.Errorf("VariableNotEquals: variable name is required")
	}
	return nil
}

func (c *VariableNotEquals) Evaluate(bot BotInterface) (bool, error) {
	value, ok := bot.Variables().Get(c.Variable)
	if !ok {
		return false, fmt.Errorf("VariableNotEquals: variable '%s' not found", c.Variable)
	}
	return value != c.Value, nil
}

// VariableGreaterThan checks if a variable is greater than a value (numeric comparison)
type VariableGreaterThan struct {
	Variable string `yaml:"variable"`
	Value    string `yaml:"value"`
}

func (c *VariableGreaterThan) Validate(ab *ActionBuilder) error {
	if c.Variable == "" {
		return fmt.Errorf("VariableGreaterThan: variable name is required")
	}
	if _, err := strconv.ParseFloat(c.Value, 64); err != nil {
		return fmt.Errorf("VariableGreaterThan: value must be a valid number, got '%s'", c.Value)
	}
	return nil
}

func (c *VariableGreaterThan) Evaluate(bot BotInterface) (bool, error) {
	varValue, ok := bot.Variables().Get(c.Variable)
	if !ok {
		return false, fmt.Errorf("VariableGreaterThan: variable '%s' not found", c.Variable)
	}

	varNum, err := strconv.ParseFloat(varValue, 64)
	if err != nil {
		return false, fmt.Errorf("VariableGreaterThan: variable '%s' is not a valid number: %s", c.Variable, varValue)
	}

	compareNum, err := strconv.ParseFloat(c.Value, 64)
	if err != nil {
		return false, fmt.Errorf("VariableGreaterThan: value is not a valid number: %s", c.Value)
	}

	return varNum > compareNum, nil
}

// VariableLessThan checks if a variable is less than a value (numeric comparison)
type VariableLessThan struct {
	Variable string `yaml:"variable"`
	Value    string `yaml:"value"`
}

func (c *VariableLessThan) Validate(ab *ActionBuilder) error {
	if c.Variable == "" {
		return fmt.Errorf("VariableLessThan: variable name is required")
	}
	if _, err := strconv.ParseFloat(c.Value, 64); err != nil {
		return fmt.Errorf("VariableLessThan: value must be a valid number, got '%s'", c.Value)
	}
	return nil
}

func (c *VariableLessThan) Evaluate(bot BotInterface) (bool, error) {
	varValue, ok := bot.Variables().Get(c.Variable)
	if !ok {
		return false, fmt.Errorf("VariableLessThan: variable '%s' not found", c.Variable)
	}

	varNum, err := strconv.ParseFloat(varValue, 64)
	if err != nil {
		return false, fmt.Errorf("VariableLessThan: variable '%s' is not a valid number: %s", c.Variable, varValue)
	}

	compareNum, err := strconv.ParseFloat(c.Value, 64)
	if err != nil {
		return false, fmt.Errorf("VariableLessThan: value is not a valid number: %s", c.Value)
	}

	return varNum < compareNum, nil
}

// VariableGreaterThanOrEqual checks if a variable is >= a value (numeric comparison)
type VariableGreaterThanOrEqual struct {
	Variable string `yaml:"variable"`
	Value    string `yaml:"value"`
}

func (c *VariableGreaterThanOrEqual) Validate(ab *ActionBuilder) error {
	if c.Variable == "" {
		return fmt.Errorf("VariableGreaterThanOrEqual: variable name is required")
	}
	if _, err := strconv.ParseFloat(c.Value, 64); err != nil {
		return fmt.Errorf("VariableGreaterThanOrEqual: value must be a valid number, got '%s'", c.Value)
	}
	return nil
}

func (c *VariableGreaterThanOrEqual) Evaluate(bot BotInterface) (bool, error) {
	varValue, ok := bot.Variables().Get(c.Variable)
	if !ok {
		return false, fmt.Errorf("VariableGreaterThanOrEqual: variable '%s' not found", c.Variable)
	}

	varNum, err := strconv.ParseFloat(varValue, 64)
	if err != nil {
		return false, fmt.Errorf("VariableGreaterThanOrEqual: variable '%s' is not a valid number: %s", c.Variable, varValue)
	}

	compareNum, err := strconv.ParseFloat(c.Value, 64)
	if err != nil {
		return false, fmt.Errorf("VariableGreaterThanOrEqual: value is not a valid number: %s", c.Value)
	}

	return varNum >= compareNum, nil
}

// VariableLessThanOrEqual checks if a variable is <= a value (numeric comparison)
type VariableLessThanOrEqual struct {
	Variable string `yaml:"variable"`
	Value    string `yaml:"value"`
}

func (c *VariableLessThanOrEqual) Validate(ab *ActionBuilder) error {
	if c.Variable == "" {
		return fmt.Errorf("VariableLessThanOrEqual: variable name is required")
	}
	if _, err := strconv.ParseFloat(c.Value, 64); err != nil {
		return fmt.Errorf("VariableLessThanOrEqual: value must be a valid number, got '%s'", c.Value)
	}
	return nil
}

func (c *VariableLessThanOrEqual) Evaluate(bot BotInterface) (bool, error) {
	varValue, ok := bot.Variables().Get(c.Variable)
	if !ok {
		return false, fmt.Errorf("VariableLessThanOrEqual: variable '%s' not found", c.Variable)
	}

	varNum, err := strconv.ParseFloat(varValue, 64)
	if err != nil {
		return false, fmt.Errorf("VariableLessThanOrEqual: variable '%s' is not a valid number: %s", c.Variable, varValue)
	}

	compareNum, err := strconv.ParseFloat(c.Value, 64)
	if err != nil {
		return false, fmt.Errorf("VariableLessThanOrEqual: value is not a valid number: %s", c.Value)
	}

	return varNum <= compareNum, nil
}

// VariableContains checks if a variable contains a substring
type VariableContains struct {
	Variable  string `yaml:"variable"`
	Substring string `yaml:"substring"`
}

func (c *VariableContains) Validate(ab *ActionBuilder) error {
	if c.Variable == "" {
		return fmt.Errorf("VariableContains: variable name is required")
	}
	return nil
}

func (c *VariableContains) Evaluate(bot BotInterface) (bool, error) {
	value, ok := bot.Variables().Get(c.Variable)
	if !ok {
		return false, fmt.Errorf("VariableContains: variable '%s' not found", c.Variable)
	}
	return strings.Contains(value, c.Substring), nil
}

// VariableStartsWith checks if a variable starts with a prefix
type VariableStartsWith struct {
	Variable string `yaml:"variable"`
	Prefix   string `yaml:"prefix"`
}

func (c *VariableStartsWith) Validate(ab *ActionBuilder) error {
	if c.Variable == "" {
		return fmt.Errorf("VariableStartsWith: variable name is required")
	}
	return nil
}

func (c *VariableStartsWith) Evaluate(bot BotInterface) (bool, error) {
	value, ok := bot.Variables().Get(c.Variable)
	if !ok {
		return false, fmt.Errorf("VariableStartsWith: variable '%s' not found", c.Variable)
	}
	return strings.HasPrefix(value, c.Prefix), nil
}

// VariableEndsWith checks if a variable ends with a suffix
type VariableEndsWith struct {
	Variable string `yaml:"variable"`
	Suffix   string `yaml:"suffix"`
}

func (c *VariableEndsWith) Validate(ab *ActionBuilder) error {
	if c.Variable == "" {
		return fmt.Errorf("VariableEndsWith: variable name is required")
	}
	return nil
}

func (c *VariableEndsWith) Evaluate(bot BotInterface) (bool, error) {
	value, ok := bot.Variables().Get(c.Variable)
	if !ok {
		return false, fmt.Errorf("VariableEndsWith: variable '%s' not found", c.Variable)
	}
	return strings.HasSuffix(value, c.Suffix), nil
}
