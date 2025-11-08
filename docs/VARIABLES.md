# Variable System Documentation

## Overview

The variable system allows you to store and manipulate runtime state during routine execution. All variables are stored as strings in a thread-safe `map[string]string`, making the system simple and type-safe.

## Key Features

- **Runtime state management** - Track counters, modes, status, and more
- **Dynamic control flow** - Use variables in conditions for complex logic
- **String-based storage** - All values stored as strings for simplicity
- **Thread-safe** - Safe for concurrent access
- **Scoped to routine execution** - Variables cleared between routine runs

## Variable Actions

### SetVariable

Set a variable to a specific value.

```yaml
- action: SetVariable
  name: counter
  value: "0"
```

**Parameters:**
- `name` (required): Variable name
- `value` (required): Value to set (always a string)

### GetVariable

Get a variable value. Mainly useful for copying one variable to another.

```yaml
- action: GetVariable
  name: source
  target: destination  # Optional: copy to this variable
```

**Parameters:**
- `name` (required): Variable to get
- `target` (optional): Copy value to this variable

### Increment

Increment a numeric variable.

```yaml
- action: Increment
  name: counter
  amount: "1"  # Optional, defaults to "1"
```

**Parameters:**
- `name` (required): Variable to increment
- `amount` (optional): Amount to add (default: "1")

**Notes:**
- If variable doesn't exist, initializes to 0
- Variable must be a valid integer
- Amount must be a valid integer

### Decrement

Decrement a numeric variable.

```yaml
- action: Decrement
  name: counter
  amount: "1"  # Optional, defaults to "1"
```

**Parameters:**
- `name` (required): Variable to decrement
- `amount` (optional): Amount to subtract (default: "1")

**Notes:**
- If variable doesn't exist, initializes to 0
- Variable must be a valid integer
- Amount must be a valid integer

## Variable Conditions

### Equality Checks

#### VariableEquals

Check if a variable equals a specific value.

```yaml
condition:
  type: VariableEquals
  variable: mode
  value: "farming"
```

#### VariableNotEquals

Check if a variable does NOT equal a specific value.

```yaml
condition:
  type: VariableNotEquals
  variable: status
  value: "error"
```

### Numeric Comparisons

All numeric conditions support integers and floating-point numbers.

#### VariableGreaterThan

```yaml
condition:
  type: VariableGreaterThan
  variable: health
  value: "50"
```

#### VariableLessThan

```yaml
condition:
  type: VariableLessThan
  variable: counter
  value: "10"
```

#### VariableGreaterThanOrEqual

```yaml
condition:
  type: VariableGreaterThanOrEqual
  variable: energy
  value: "100"
```

#### VariableLessThanOrEqual

```yaml
condition:
  type: VariableLessThanOrEqual
  variable: attempts
  value: "5"
```

### String Operations

#### VariableContains

Check if variable contains a substring.

```yaml
condition:
  type: VariableContains
  variable: message
  substring: "success"
```

#### VariableStartsWith

Check if variable starts with a prefix.

```yaml
condition:
  type: VariableStartsWith
  variable: status
  prefix: "battle"
```

#### VariableEndsWith

Check if variable ends with a suffix.

```yaml
condition:
  type: VariableEndsWith
  variable: filename
  suffix: ".txt"
```

## Common Patterns

### Counter Loop

```yaml
# Initialize counter
- action: SetVariable
  name: counter
  value: "0"

# Loop with counter
- action: While
  max_attempts: 20
  condition:
    type: VariableLessThan
    variable: counter
    value: "10"
  actions:
    - action: Click
      x: 100
      y: 100

    - action: Increment
      name: counter

    - action: Delay
      count: 1
```

### Mode-Based Execution

```yaml
# Set mode
- action: SetVariable
  name: mode
  value: "farming"

# Different actions based on mode
- action: If
  condition:
    type: VariableEquals
    variable: mode
    value: "farming"
  then:
    - action: Click
      template: "Farm"

elseif:
  - condition:
      type: VariableEquals
      variable: mode
      value: "battling"
    then:
      - action: Click
        template: "Battle"

else:
  - action: Click
    template: "Explore"
```

### Retry with Limit

```yaml
# Track attempts
- action: SetVariable
  name: attempts
  value: "0"

- action: SetVariable
  name: max_attempts
  value: "5"

# Retry until success or max attempts
- action: Until
  max_attempts: 10
  condition:
    type: ImageExists
    template: "Success"
  actions:
    # Increment attempts
    - action: Increment
      name: attempts

    # Check if exceeded max
    - action: If
      condition:
        type: VariableGreaterThan
        variable: attempts
        value: "5"
      then:
        - action: Break

    # Try the action
    - action: Click
      x: 300
      y: 300

    - action: Delay
      count: 2
```

### Complex State Tracking

```yaml
# Track multiple states
- action: SetVariable
  name: health
  value: "100"

- action: SetVariable
  name: energy
  value: "50"

- action: SetVariable
  name: in_battle
  value: "true"

# Complex decision making
- action: If
  condition:
    type: All
    conditions:
      - type: VariableGreaterThan
        variable: health
        value: "30"
      - type: VariableGreaterThan
        variable: energy
        value: "20"
      - type: VariableEquals
        variable: in_battle
        value: "true"
  then:
    - action: Click
      template: "Attack"

elseif:
  - condition:
      type: VariableLessThanOrEqual
      variable: health
      value: "30"
    then:
      - action: Click
        template: "Heal"

else:
  - action: Click
    template: "Defend"
```

## Best Practices

### Variable Naming

- Use descriptive names: `click_count`, `current_mode`, `max_attempts`
- Use snake_case for multi-word names
- Avoid single letters except for simple counters (`i`, `j`)

### String vs Numeric

- Always store numbers as strings: `"10"`, `"3.14"`
- The system handles conversion for numeric comparisons
- Use numeric conditions for math operations
- Use string conditions for text matching

### Initialization

- Always initialize variables before use
- Set default values at routine start
- Check existence with conditions if needed

### Error Handling

- Variables not found will cause errors in conditions
- Use `If` with `VariableEquals` to check state safely
- Initialize all variables at routine start

## Variable Lifecycle

1. **Created** - When `SetVariable` is first called
2. **Exists** - Throughout routine execution
3. **Cleared** - When routine completes (future feature)
4. **Reset** - On next routine execution

## Type Conversions

The system handles type conversions automatically:

- **Numeric comparisons**: Strings converted to float64
- **String operations**: No conversion needed
- **Equality**: Direct string comparison

```yaml
# This works - "10" is converted to number for comparison
- action: SetVariable
  name: count
  value: "10"

- action: If
  condition:
    type: VariableGreaterThan
    variable: count
    value: "5"  # Compares numerically: 10 > 5
  then:
    - action: Click
      x: 100
      y: 100
```

## Examples

See [example_variables.yaml](../bin/routines/example_variables.yaml) for comprehensive examples of all variable features.

## Future Features

Coming soon:

- **Variable interpolation**: Use `${variable_name}` in action parameters
- **Config definitions**: Define user-configurable variables in YAML
- **Profile system**: Save/load variable presets
- **Arithmetic actions**: Add, Subtract, Multiply, Divide
- **String manipulation**: Concat, Substring, Replace
- **Persistent variables**: Variables that survive routine executions

## Implementation Details

### Files
- `internal/actions/variables.go` - Variable actions (SetVariable, Increment, etc.)
- `internal/actions/variable_conditions.go` - Variable conditions
- `internal/actions/interfaces.go` - VariableStoreInterface definition
- `internal/bot/bot.go` - VariableStore integration

### Registry
- Action registry: `setvariable`, `getvariable`, `increment`, `decrement`
- Condition registry: `variableequals`, `variablegreaterthan`, etc.
