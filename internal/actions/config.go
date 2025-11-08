package actions

import "fmt"

// ConfigParam defines a user-configurable parameter for a routine
type ConfigParam struct {
	Name        string   `yaml:"name"`                   // Variable name
	Label       string   `yaml:"label"`                  // Display label for GUI
	Type        string   `yaml:"type"`                   // Type: text, number, checkbox, dropdown
	Default     string   `yaml:"default"`                // Default value
	Description string   `yaml:"description,omitempty"`  // Optional description
	Options     []string `yaml:"options,omitempty"`      // Options for dropdown type
	Min         *float64 `yaml:"min,omitempty"`          // Min value for number type
	Max         *float64 `yaml:"max,omitempty"`          // Max value for number type
	Required    bool     `yaml:"required,omitempty"`     // Whether parameter is required
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
	}
	if !validTypes[cp.Type] {
		return fmt.Errorf("config param '%s': invalid type '%s' (must be: text, number, checkbox, dropdown)", cp.Name, cp.Type)
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
	case "text":
		return ""
	default:
		return ""
	}
}
