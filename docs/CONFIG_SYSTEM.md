# Configuration System

The configuration system allows you to define user-configurable parameters in your routines, making them flexible and reusable without needing to create multiple copies of the same script.

## Overview

Config parameters are defined at the top of a routine YAML file and are automatically initialized as variables when the routine starts. These can then be referenced throughout the routine using variable interpolation syntax `${variable_name}`.

## Config Definition

Config parameters are defined in the `config` section of a routine:

```yaml
routine_name: "My Routine"
config:
  - name: target_count
    label: "Target Count"
    type: number
    default: "10"
    description: "Number of iterations to run"
    min: 1
    max: 100
```

## Config Parameter Fields

### Required Fields

- **name**: Variable name that will be created (must be valid identifier)
- **label**: Human-readable label for GUI display
- **type**: Parameter type - one of: `text`, `number`, `checkbox`, `dropdown`
- **default**: Default value (as string)

### Optional Fields

- **description**: Explanation of what this parameter does
- **required**: Whether this parameter must be set (default: false)
- **options**: Array of choices (required for `dropdown` type)
- **min**: Minimum value (for `number` type)
- **max**: Maximum value (for `number` type)

## Parameter Types

### Text

Free-form text input:

```yaml
- name: player_name
  label: "Player Name"
  type: text
  default: "Player1"
  description: "Name to use in game"
```

### Number

Numeric input with optional min/max constraints:

```yaml
- name: delay_seconds
  label: "Delay (seconds)"
  type: number
  default: "2"
  min: 0
  max: 10
  description: "Delay between actions"
```

### Checkbox

Boolean on/off toggle:

```yaml
- name: enable_sound
  label: "Enable Sound"
  type: checkbox
  default: "true"
  description: "Play sound effects"
```

Checkbox values are stored as `"true"` or `"false"` strings.

### Dropdown

Select from predefined options:

```yaml
- name: difficulty
  label: "Difficulty"
  type: dropdown
  options: ["Easy", "Normal", "Hard"]
  default: "Normal"
  description: "Game difficulty level"
```

## Using Config Values

Config parameters are automatically initialized as variables at routine start. You can reference them using variable interpolation:

```yaml
config:
  - name: farm_type
    type: dropdown
    options: ["Gold", "Experience", "Materials"]
    default: "Gold"

steps:
  # Use config value in condition
  - action: While
    condition:
      type: VariableLessThan
      variable: counter
      value: ${target_count}
    actions:
      # Use config value in template name
      - action: ClickImage
        template: ${farm_type}_button

      # Use config value with suffix
      - action: WaitForImage
        template: ${farm_type}_screen
```

## Variable Interpolation

The `${variable_name}` syntax works in:

- Template names in image-based actions
- Values in variable conditions
- Values in SetVariable action

Examples:

```yaml
# Dynamic template selection
- action: ClickImage
  template: ${button_type}_button

# Dynamic threshold
- action: SetVariable
  name: result
  value: ${computed_value}

# In conditions
- action: If
  condition:
    type: VariableEquals
    variable: mode
    value: ${target_mode}
```

## Type Defaults

If a config parameter has no default value, these type defaults are used:

- **text**: `""` (empty string)
- **number**: `"0"`
- **checkbox**: `"false"`
- **dropdown**: First option in the list

## Validation

Config parameters are validated:

- **Name**: Must not be empty
- **Type**: Must be one of: text, number, checkbox, dropdown
- **Dropdown**: Must have at least one option
- **Number**: Min must be less than max (if both specified)
- **Default**: Must be a valid option (for dropdown type)

## Runtime Initialization

When a routine starts, config parameters are initialized in this order:

1. User-provided override values (from GUI or API)
2. Default value specified in config
3. Type default

This is handled by `InitializeConfigVariables()` function:

```go
InitializeConfigVariables(bot, routine.Config, overrides)
```

The `overrides` map allows the GUI or calling code to provide user-selected values.

## Complete Example

See `bin/routines/example_config.yaml` for a comprehensive example showing:

- All config parameter types
- Variable interpolation in templates
- Using config in conditions
- Combining config values with other strings
- Conditional logic based on config values

## Integration with Variables

Config parameters are just variables that are initialized at routine start. You can:

- Read them with GetVariable
- Modify them with SetVariable
- Use them in all variable conditions
- Increment/Decrement numeric config values
- Combine them with runtime variables

Example:

```yaml
config:
  - name: max_runs
    type: number
    default: "10"

steps:
  - action: SetVariable
    name: counter
    value: "0"

  - action: While
    condition:
      type: VariableLessThan
      variable: counter
      value: ${max_runs}  # Config value
    actions:
      - action: Increment
        name: counter  # Runtime variable
```

## Future Enhancements

Planned features:

- **Profile Save/Load**: Save and load sets of config values as named profiles
- **Config Validation UI**: Real-time validation in GUI
- **Config Presets**: Common preset configurations
- **Advanced Types**: Color picker, file path, range slider
- **Conditional Configs**: Show/hide config based on other config values
- **Config Groups**: Organize related configs into collapsible sections

## Best Practices

1. **Use Descriptive Names**: Choose clear variable names (e.g., `max_iterations` not `max`)
2. **Provide Defaults**: Always set sensible default values
3. **Add Descriptions**: Help users understand what each parameter does
4. **Set Constraints**: Use min/max for numbers to prevent invalid values
5. **Limit Options**: For dropdowns, keep option lists reasonably short
6. **Use Checkboxes for Booleans**: Don't use dropdown for true/false
7. **Group Related Configs**: Organize configs logically
8. **Test Defaults**: Ensure routine works with all default values

## Troubleshooting

**Variable not found error**:
- Ensure config parameter name matches the interpolation variable
- Check that InitializeConfigVariables is called before routine execution

**Invalid value error**:
- Verify default value is valid for the type
- For dropdowns, ensure default is in options list
- For numbers, ensure default is within min/max range

**Interpolation not working**:
- Verify you're using `${var}` syntax, not `{var}` or `$var`
- Check that the action supports interpolation (currently: templates, variable values)
- Ensure variable is initialized before use
