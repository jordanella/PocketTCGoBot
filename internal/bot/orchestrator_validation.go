package bot

import (
	"fmt"
	"strings"
)

// ValidationResult contains the results of routine validation
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// ValidationError represents a specific validation error
type ValidationError struct {
	Type    ValidationErrorType
	Message string
	Context string // Additional context (e.g., action name, template name)
}

// ValidationErrorType categorizes validation errors
type ValidationErrorType string

const (
	ValidationErrorRoutineNotFound  ValidationErrorType = "routine_not_found"
	ValidationErrorRoutineParse     ValidationErrorType = "routine_parse_error"
	ValidationErrorActionNotFound   ValidationErrorType = "action_not_found"
	ValidationErrorTemplateNotFound ValidationErrorType = "template_not_found"
	ValidationErrorInvalidConfig    ValidationErrorType = "invalid_config"
	ValidationErrorMissingVariable  ValidationErrorType = "missing_variable"
	ValidationErrorInvalidField     ValidationErrorType = "invalid_field"
	ValidationErrorMissingField     ValidationErrorType = "missing_field"
)

// ValidateRoutine performs comprehensive validation of a routine
func (o *Orchestrator) ValidateRoutine(routineName string, config map[string]string) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: make([]ValidationError, 0),
	}

	// Check if routine exists
	if !o.routineRegistry.Has(routineName) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:    ValidationErrorRoutineNotFound,
			Message: fmt.Sprintf("Routine '%s' not found in registry", routineName),
			Context: routineName,
		})
		// Can't continue validation if routine doesn't exist
		return result
	}

	// Try to load the routine
	builder, err := o.routineRegistry.Get(routineName)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:    ValidationErrorRoutineParse,
			Message: fmt.Sprintf("Failed to load routine: %v", err),
			Context: routineName,
		})
		// Can't continue if we can't load the routine
		return result
	}

	// Validate actions exist (check if any issues were recorded during build)
	if builder != nil {
		// Note: ActionBuilder doesn't expose steps directly, so we rely on
		// the routine loading process to catch missing actions.
		// If we made it this far, actions are registered.
	}

	// Validate templates referenced in routine
	// This would require parsing the routine YAML and extracting template references
	// For now, we'll do a basic validation
	templateErrors := o.validateTemplates(routineName)
	if len(templateErrors) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, templateErrors...)
	}

	// Validate configuration variables
	configErrors := o.validateConfiguration(routineName, config)
	if len(configErrors) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, configErrors...)
	}

	return result
}

// validateTemplates checks if all templates referenced in the routine exist
func (o *Orchestrator) validateTemplates(routineName string) []ValidationError {
	errors := make([]ValidationError, 0)

	// Get routine metadata to find template references
	metadata := o.routineRegistry.GetMetadata(routineName)

	// Extract template references from metadata
	// The metadata structure depends on how routines store template info
	if metadataMap, ok := metadata.(map[string]interface{}); ok {
		if templates, ok := metadataMap["templates"].([]interface{}); ok {
			for _, tmpl := range templates {
				if templateName, ok := tmpl.(string); ok {
					// Check if template exists
					if !o.templateRegistry.Has(templateName) {
						errors = append(errors, ValidationError{
							Type:    ValidationErrorTemplateNotFound,
							Message: fmt.Sprintf("Template '%s' not found in registry", templateName),
							Context: templateName,
						})
					}
				}
			}
		}
	}

	return errors
}

// validateConfiguration checks if configuration variables are valid
func (o *Orchestrator) validateConfiguration(routineName string, config map[string]string) []ValidationError {
	errors := make([]ValidationError, 0)

	// Get routine metadata to find required/available variables
	metadata := o.routineRegistry.GetMetadata(routineName)

	// Extract variable definitions from metadata
	if metadataMap, ok := metadata.(map[string]interface{}); ok {
		// Check for required variables
		if requiredVars, ok := metadataMap["required_variables"].([]any); ok {
			for _, v := range requiredVars {
				if varName, ok := v.(string); ok {
					// Check if required variable is provided in config
					if _, provided := config[varName]; !provided {
						errors = append(errors, ValidationError{
							Type:    ValidationErrorMissingVariable,
							Message: fmt.Sprintf("Required variable '%s' not provided in configuration", varName),
							Context: varName,
						})
					}
				}
			}
		}

		// Check for unknown variables (config contains vars not defined in routine)
		if availableVars, ok := metadataMap["variables"].([]interface{}); ok {
			availableSet := make(map[string]bool)
			for _, v := range availableVars {
				if varName, ok := v.(string); ok {
					availableSet[varName] = true
				}
			}

			for configVar := range config {
				if !availableSet[configVar] {
					errors = append(errors, ValidationError{
						Type:    ValidationErrorInvalidConfig,
						Message: fmt.Sprintf("Unknown variable '%s' in configuration", configVar),
						Context: configVar,
					})
				}
			}
		}
	}

	return errors
}

// FormatValidationErrors returns a human-readable string of validation errors
func (vr *ValidationResult) FormatValidationErrors() string {
	if vr.Valid {
		return "Validation passed"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d validation error(s):\n", len(vr.Errors)))

	for i, err := range vr.Errors {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s", i+1, err.Type, err.Message))
		if err.Context != "" {
			sb.WriteString(fmt.Sprintf(" (context: %s)", err.Context))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetErrorsByType returns all errors of a specific type
func (vr *ValidationResult) GetErrorsByType(errorType ValidationErrorType) []ValidationError {
	errors := make([]ValidationError, 0)
	for _, err := range vr.Errors {
		if err.Type == errorType {
			errors = append(errors, err)
		}
	}
	return errors
}

// HasErrorType checks if the result contains errors of a specific type
func (vr *ValidationResult) HasErrorType(errorType ValidationErrorType) bool {
	for _, err := range vr.Errors {
		if err.Type == errorType {
			return true
		}
	}
	return false
}

// ValidateGroupDefinition validates a bot group definition
func ValidateGroupDefinition(def *BotGroupDefinition) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: make([]ValidationError, 0),
	}

	// Validate name
	if def.Name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:    ValidationErrorMissingField,
			Message: "Group name is required",
			Context: "Name",
		})
	}

	// Validate routine name
	if def.RoutineName == "" {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:    ValidationErrorMissingField,
			Message: "Routine name is required",
			Context: "RoutineName",
		})
	}

	// Validate available instances
	if len(def.AvailableInstances) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:    ValidationErrorMissingField,
			Message: "At least one instance must be specified",
			Context: "AvailableInstances",
		})
	}

	// Validate requested bot count
	if def.RequestedBotCount <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:    ValidationErrorInvalidField,
			Message: "Requested bot count must be positive",
			Context: "RequestedBotCount",
		})
	}

	// Check if requested bot count exceeds available instances
	if def.RequestedBotCount > len(def.AvailableInstances) {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type: ValidationErrorInvalidField,
			Message: fmt.Sprintf("Requested bot count (%d) exceeds available instances (%d)",
				def.RequestedBotCount, len(def.AvailableInstances)),
			Context: "RequestedBotCount",
		})
	}

	// Validate instance IDs are not negative
	for i, instanceID := range def.AvailableInstances {
		if instanceID < 0 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    ValidationErrorInvalidField,
				Message: fmt.Sprintf("Instance ID must be non-negative (got %d)", instanceID),
				Context: fmt.Sprintf("AvailableInstances[%d]", i),
			})
		}
	}

	// Check for duplicate instance IDs
	instanceSet := make(map[int]bool)
	for i, instanceID := range def.AvailableInstances {
		if instanceSet[instanceID] {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    ValidationErrorInvalidField,
				Message: fmt.Sprintf("Duplicate instance ID %d", instanceID),
				Context: fmt.Sprintf("AvailableInstances[%d]", i),
			})
		}
		instanceSet[instanceID] = true
	}

	return result
}

// ValidateLaunchOptions validates launch options
func ValidateLaunchOptions(options *LaunchOptions) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: make([]ValidationError, 0),
	}

	// Validate emulator timeout
	if options.EmulatorTimeout <= 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:    ValidationErrorInvalidField,
			Message: "Emulator timeout must be positive",
			Context: "EmulatorTimeout",
		})
	}

	// Validate stagger delay (can be 0)
	if options.StaggerDelay < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Type:    ValidationErrorInvalidField,
			Message: "Stagger delay cannot be negative",
			Context: "StaggerDelay",
		})
	}

	// Validate restart policy
	if options.RestartPolicy.Enabled {
		if options.RestartPolicy.MaxRetries < -1 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    ValidationErrorInvalidField,
				Message: "Max retries must be >= -1 (-1 for infinite)",
				Context: "RestartPolicy.MaxRetries",
			})
		}

		if options.RestartPolicy.InitialDelay <= 0 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    ValidationErrorInvalidField,
				Message: "Initial delay must be positive",
				Context: "RestartPolicy.InitialDelay",
			})
		}

		if options.RestartPolicy.MaxDelay <= 0 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    ValidationErrorInvalidField,
				Message: "Max delay must be positive",
				Context: "RestartPolicy.MaxDelay",
			})
		}

		if options.RestartPolicy.InitialDelay > options.RestartPolicy.MaxDelay {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    ValidationErrorInvalidField,
				Message: "Initial delay cannot exceed max delay",
				Context: "RestartPolicy",
			})
		}

		if options.RestartPolicy.BackoffFactor <= 0 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Type:    ValidationErrorInvalidField,
				Message: "Backoff factor must be positive",
				Context: "RestartPolicy.BackoffFactor",
			})
		}
	}

	return result
}
