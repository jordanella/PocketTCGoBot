package actions

import (
	"fmt"
	"strconv"
)

// ConfigParam defines a user-configurable parameter for a routine
type ConfigParam struct {
	Name        string   `yaml:"name"`                   // Variable name
	Label       string   `yaml:"label"`                  // Display label for GUI
	Type        string   `yaml:"type"`                   // Type: text, number, checkbox, dropdown, hidden
	Default     string   `yaml:"default"`                // Default value
	Description string   `yaml:"description,omitempty"`  // Optional description
	Options     []string `yaml:"options,omitempty"`      // Options for dropdown type
	Min         *float64 `yaml:"min,omitempty"`          // Min value for number type
	Max         *float64 `yaml:"max,omitempty"`          // Max value for number type
	Required    bool     `yaml:"required,omitempty"`     // Whether parameter is required
	Persist     bool     `yaml:"persist,omitempty"`      // If true, won't be reset between routine iterations
}

// Validate validates the config param definition
func (cp *ConfigParam) Validate() error {
	if cp.Name == "" {
		return fmt.Errorf("config param: name is required")
	}

	if cp.Type == "" {
		return fmt.Errorf("config param '%s': type is required", cp.Name)
	}

	// Validate type
	validTypes := map[string]bool{
		"text":     true,
		"number":   true,
		"checkbox": true,
		"dropdown": true,
		"hidden":   true,
	}
	if !validTypes[cp.Type] {
		return fmt.Errorf("config param '%s': invalid type '%s' (must be: text, number, checkbox, dropdown, hidden)", cp.Name, cp.Type)
	}

	// Dropdown must have options
	if cp.Type == "dropdown" && len(cp.Options) == 0 {
		return fmt.Errorf("config param '%s': dropdown type requires options", cp.Name)
	}

	// Checkbox default must be true/false
	if cp.Type == "checkbox" {
		if cp.Default != "" && cp.Default != "true" && cp.Default != "false" {
			return fmt.Errorf("config param '%s': checkbox default must be 'true' or 'false'", cp.Name)
		}
	}

	// If dropdown, default must be in options (if specified)
	if cp.Type == "dropdown" && cp.Default != "" {
		found := false
		for _, opt := range cp.Options {
			if opt == cp.Default {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("config param '%s': default '%s' not in options %v", cp.Name, cp.Default, cp.Options)
		}
	}

	// Validate min/max for numbers
	if cp.Type == "number" {
		// Check min < max if both specified
		if cp.Min != nil && cp.Max != nil && *cp.Min > *cp.Max {
			return fmt.Errorf("config param '%s': min (%v) cannot be greater than max (%v)", cp.Name, *cp.Min, *cp.Max)
		}

		// Validate default value is a valid number and within range
		if cp.Default != "" {
			defaultVal, err := strconv.ParseFloat(cp.Default, 64)
			if err != nil {
				return fmt.Errorf("config param '%s': default value '%s' is not a valid number", cp.Name, cp.Default)
			}

			// Check against min
			if cp.Min != nil && defaultVal < *cp.Min {
				return fmt.Errorf("config param '%s': default value %v is less than min %v", cp.Name, defaultVal, *cp.Min)
			}

			// Check against max
			if cp.Max != nil && defaultVal > *cp.Max {
				return fmt.Errorf("config param '%s': default value %v is greater than max %v", cp.Name, defaultVal, *cp.Max)
			}
		}
	}

	return nil
}

// GetEffectiveValue returns the effective value (default or provided value)
func (cp *ConfigParam) GetEffectiveValue(providedValue string) string {
	if providedValue != "" {
		return providedValue
	}
	return cp.Default
}

// GetTypeDefault returns the default value for the type if no default is specified
func (cp *ConfigParam) GetTypeDefault() string {
	switch cp.Type {
	case "checkbox":
		return "false"
	case "number":
		return "0"
	case "dropdown":
		if len(cp.Options) > 0 {
			return cp.Options[0]
		}
		return ""
	case "text", "hidden":
		return ""
	default:
		return ""
	}
}

// IsHidden returns true if this config param should be hidden from GUI
func (cp *ConfigParam) IsHidden() bool {
	return cp.Type == "hidden"
}
