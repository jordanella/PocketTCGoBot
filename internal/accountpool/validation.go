package accountpool

import (
	"fmt"
)

// ValidationResult contains the results of a validation check
type ValidationResult struct {
	Valid  bool
	Errors []ValidationError
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string
	Message string
}

// AddError adds a validation error
func (vr *ValidationResult) AddError(field, message string) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}

// FormatErrors returns a formatted string of all validation errors
func (vr *ValidationResult) FormatErrors() string {
	if vr.Valid {
		return ""
	}

	result := "Validation failed:\n"
	for _, err := range vr.Errors {
		result += fmt.Sprintf("  - %s: %s\n", err.Field, err.Message)
	}
	return result
}

// ValidatePoolDefinition validates a unified pool definition
func ValidatePoolDefinition(def *UnifiedPoolDefinition) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: make([]ValidationError, 0),
	}

	// Validate pool name
	if def.PoolName == "" {
		result.AddError("PoolName", "pool name is required")
	}

	// Validate that at least one source is defined
	hasSource := len(def.Queries) > 0 || len(def.Include) > 0 || len(def.WatchedPaths) > 0
	if !hasSource {
		result.AddError("Sources", "at least one source (queries, include, or watched_paths) must be defined")
	}

	// Validate queries
	for i, query := range def.Queries {
		if query.Name == "" {
			result.AddError(fmt.Sprintf("Queries[%d].Name", i), "query name is required")
		}

		if len(query.Filters) == 0 {
			result.AddError(fmt.Sprintf("Queries[%d].Filters", i), "at least one filter must be defined")
		}

		// Validate filters
		for j, filter := range query.Filters {
			if filter.Column == "" {
				result.AddError(fmt.Sprintf("Queries[%d].Filters[%d].Column", i, j), "column name is required")
			}

			if filter.Comparator == "" {
				result.AddError(fmt.Sprintf("Queries[%d].Filters[%d].Comparator", i, j), "comparator is required")
			}

			// Validate comparator is a known operator
			validComparators := map[string]bool{
				"=": true, "!=": true, ">": true, ">=": true, "<": true, "<=": true,
				"LIKE": true, "NOT LIKE": true, "IN": true, "NOT IN": true,
			}
			if !validComparators[filter.Comparator] {
				result.AddError(fmt.Sprintf("Queries[%d].Filters[%d].Comparator", i, j),
					fmt.Sprintf("invalid comparator '%s'", filter.Comparator))
			}
		}

		// Validate sorts
		for j, sort := range query.Sort {
			if sort.Column == "" {
				result.AddError(fmt.Sprintf("Queries[%d].Sort[%d].Column", i, j), "column name is required")
			}

			if sort.Direction != "asc" && sort.Direction != "desc" && sort.Direction != "ASC" && sort.Direction != "DESC" {
				result.AddError(fmt.Sprintf("Queries[%d].Sort[%d].Direction", i, j),
					fmt.Sprintf("direction must be 'asc' or 'desc', got '%s'", sort.Direction))
			}
		}

		// Validate limit
		if query.Limit < 0 {
			result.AddError(fmt.Sprintf("Queries[%d].Limit", i), "limit cannot be negative")
		}
	}

	// Validate configuration
	if def.Config.MaxFailures < 0 {
		result.AddError("Config.MaxFailures", "max failures cannot be negative")
	}

	if def.Config.RefreshInterval < 0 {
		result.AddError("Config.RefreshInterval", "refresh interval cannot be negative")
	}

	validSortMethods := map[string]bool{
		"packs_asc": true, "packs_desc": true,
		"modified_asc": true, "modified_desc": true,
		"random": true, "": true, // empty is valid (no sorting)
	}
	if !validSortMethods[def.Config.SortMethod] {
		result.AddError("Config.SortMethod",
			fmt.Sprintf("invalid sort method '%s'", def.Config.SortMethod))
	}

	return result
}
