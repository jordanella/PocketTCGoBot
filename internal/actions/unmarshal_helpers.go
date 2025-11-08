package actions

import (
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

// conditionRegistry maps condition type names to their concrete struct types
var conditionRegistry = map[string]reflect.Type{
	"imageexists":                reflect.TypeOf(ImageExists{}),
	"imagenotexists":             reflect.TypeOf(ImageNotExists{}),
	"not":                        reflect.TypeOf(Not{}),
	"all":                        reflect.TypeOf(All{}),
	"any":                        reflect.TypeOf(Any{}),
	"none":                       reflect.TypeOf(None{}),
	"variableequals":             reflect.TypeOf(VariableEquals{}),
	"variablenotequals":          reflect.TypeOf(VariableNotEquals{}),
	"variablegreaterthan":        reflect.TypeOf(VariableGreaterThan{}),
	"variablelessthan":           reflect.TypeOf(VariableLessThan{}),
	"variablegreaterthanorequal": reflect.TypeOf(VariableGreaterThanOrEqual{}),
	"variablelessthanorequal":    reflect.TypeOf(VariableLessThanOrEqual{}),
	"variablecontains":           reflect.TypeOf(VariableContains{}),
	"variablestartswith":         reflect.TypeOf(VariableStartsWith{}),
	"variableendswith":           reflect.TypeOf(VariableEndsWith{}),
}

// getRegisteredConditions returns a list of all registered condition types for error messages
func getRegisteredConditions() []string {
	conditions := make([]string, 0, len(conditionRegistry))
	for name := range conditionRegistry {
		conditions = append(conditions, name)
	}
	return conditions
}

// unmarshalCondition unmarshals a polymorphic condition from raw YAML data
func unmarshalCondition(conditionRaw interface{}) (Condition, error) {
	if conditionRaw == nil {
		return nil, fmt.Errorf("condition is nil")
	}

	conditionMap, ok := conditionRaw.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("condition must be a map/object")
	}

	// Get the condition type
	conditionTypeRaw, ok := conditionMap["type"]
	if !ok {
		return nil, fmt.Errorf("condition missing 'type' field")
	}

	conditionType, ok := conditionTypeRaw.(string)
	if !ok || conditionType == "" {
		return nil, fmt.Errorf("condition 'type' must be a non-empty string")
	}

	// Look up the concrete struct type in the registry
	structType, found := conditionRegistry[strings.ToLower(conditionType)]
	if !found {
		return nil, fmt.Errorf("unknown condition type '%s' (available types: %v)", conditionType, getRegisteredConditions())
	}

	// Create an instance of the concrete type
	condition := reflect.New(structType).Interface().(Condition)

	// Marshal back to YAML and unmarshal into the concrete type
	conditionBytes, err := yaml.Marshal(conditionMap)
	if err != nil {
		return nil, fmt.Errorf("error marshaling condition: %w", err)
	}

	if err := yaml.Unmarshal(conditionBytes, condition); err != nil {
		return nil, fmt.Errorf("error unmarshaling condition into %T: %w", condition, err)
	}

	return condition, nil
}

// unmarshalConditions unmarshals a slice of polymorphic conditions
func unmarshalConditions(conditionsRaw interface{}) ([]Condition, error) {
	if conditionsRaw == nil {
		return nil, nil
	}

	conditionsSlice, ok := conditionsRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("'conditions' field must be a list")
	}

	conditions := make([]Condition, len(conditionsSlice))
	for i, condRaw := range conditionsSlice {
		condition, err := unmarshalCondition(condRaw)
		if err != nil {
			return nil, fmt.Errorf("condition %d: %w", i+1, err)
		}
		conditions[i] = condition
	}

	return conditions, nil
}

// unmarshalActions unmarshals a polymorphic actions field
// This is a wrapper around unmarshalNestedActions with better error handling
func unmarshalActions(actionsRaw interface{}) ([]ActionStep, error) {
	return unmarshalNestedActions(actionsRaw)
}
