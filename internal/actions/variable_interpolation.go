package actions

import (
	"fmt"
	"regexp"
	"strings"
)

// Variable interpolation pattern: ${variable_name}
var interpolationPattern = regexp.MustCompile(`\$\{([a-zA-Z0-9_]+)\}`)

// InterpolateVariables replaces ${variable_name} patterns with their values
// Returns the interpolated string and any error if variables are not found
func InterpolateVariables(input string, vars VariableStoreInterface) (string, error) {
	if !strings.Contains(input, "${") {
		// Fast path: no interpolation needed
		return input, nil
	}

	var missingVars []string
	result := interpolationPattern.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name (remove ${ and })
		varName := match[2 : len(match)-1]

		value, ok := vars.Get(varName)
		if !ok {
			missingVars = append(missingVars, varName)
			return match // Keep the original ${var} if not found
		}
		return value
	})

	if len(missingVars) > 0 {
		return result, fmt.Errorf("undefined variables: %v", missingVars)
	}

	return result, nil
}

// InterpolateVariablesWithDefault is like InterpolateVariables but returns a default value for missing variables
func InterpolateVariablesWithDefault(input string, vars VariableStoreInterface, defaultValue string) string {
	if !strings.Contains(input, "${") {
		return input
	}

	return interpolationPattern.ReplaceAllStringFunc(input, func(match string) string {
		varName := match[2 : len(match)-1]
		value, ok := vars.Get(varName)
		if !ok {
			return defaultValue
		}
		return value
	})
}

// HasInterpolation checks if a string contains variable interpolation syntax
func HasInterpolation(input string) bool {
	return strings.Contains(input, "${")
}

// ExtractVariableNames returns all variable names referenced in the string
func ExtractVariableNames(input string) []string {
	matches := interpolationPattern.FindAllStringSubmatch(input, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			names = append(names, match[1])
		}
	}
	return names
}

// InterpolateString is a convenience wrapper for runtime interpolation
// Used by actions during execution to resolve variable references
func InterpolateString(input string, bot BotInterface) (string, error) {
	return InterpolateVariables(input, bot.Variables())
}

// MustInterpolateString interpolates or panics - use when variables are required
func MustInterpolateString(input string, bot BotInterface) string {
	result, err := InterpolateString(input, bot)
	if err != nil {
		panic(fmt.Sprintf("variable interpolation failed: %v", err))
	}
	return result
}
