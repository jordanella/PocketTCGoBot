# Condition System Documentation

## Overview

The condition system provides a flexible, composable way to define control flow in your bot routines. Instead of having separate actions for each specific condition (like `WhileImageFound`, `UntilImageFound`), you now have generic control flow actions (`If`, `While`, `Until`) that work with any condition.

## Architecture

### Core Components

1. **Condition Interface** - All conditions implement this interface:
   - `Evaluate(bot BotInterface) (bool, error)` - Returns true/false
   - `Validate(ab *ActionBuilder) error` - Validates at build time

2. **Control Flow Actions** - Generic actions that use conditions:
   - `If` - Execute one of two action sequences based on a condition
   - `While` - Repeat actions while a condition is true
   - `Until` - Repeat actions until a condition becomes true

3. **Boolean Evaluators** - Conditions that check state:
   - `ImageExists` - Check if a template is visible on screen
   - `ImageNotExists` - Check if a template is NOT visible

4. **Logical Operators** - Combine conditions with boolean logic:
   - `Not` - Negate a condition
   - `All` - AND logic (all conditions must be true)
   - `Any` - OR logic (at least one condition must be true)
   - `None` - NOR logic (no conditions can be true)

## YAML Syntax

### Basic If Statement

```yaml
- action: If
  condition:
    type: ImageExists
    template: "OK"
  then:
    - action: Click
      x: 500
      y: 300
```

### If with Else

```yaml
- action: If
  condition:
    type: ImageNotExists
    template: "Error"
  then:
    - action: Click
      x: 100
      y: 100
  else:
    - action: Click
      x: 200
      y: 200
```

### If with Else-If Chains (NEW!)

You can now chain multiple conditions using `elseif`:

```yaml
- action: If
  condition:
    type: ImageExists
    template: "Victory"
  then:
    - action: Click
      x: 500
      y: 300

  elseif:
    # First else-if branch
    - condition:
        type: ImageExists
        template: "Defeat"
      then:
        - action: Click
          x: 600
          y: 300

    # Second else-if branch
    - condition:
        type: ImageExists
        template: "Timeout"
      then:
        - action: Click
          x: 700
          y: 300

  else:
    # Final fallback if nothing matches
    - action: Delay
      count: 1
```

### While Loop

```yaml
- action: While
  max_attempts: 10
  condition:
    type: ImageExists
    template: "Enemy"
  actions:
    - action: Click
      x: 500
      y: 500
    - action: Delay
      count: 1
```

### Until Loop

```yaml
- action: Until
  max_attempts: 20
  condition:
    type: ImageExists
    template: "Victory"
  actions:
    - action: Click
      x: 400
      y: 400
    - action: Delay
      count: 2
```

## Logical Operators

### Not - Negate a condition

```yaml
- action: While
  condition:
    type: Not
    condition:
      type: ImageExists
      template: "Error"
  actions:
    - action: Click
      x: 300
      y: 300
```

### All - AND logic (all must be true)

```yaml
- action: While
  max_attempts: 10
  condition:
    type: All
    conditions:
      - type: ImageExists
        template: "Enemy"
      - type: ImageNotExists
        template: "Victory"
  actions:
    - action: Click
      x: 500
      y: 500
```

### Any - OR logic (at least one must be true)

```yaml
- action: Until
  max_attempts: 20
  condition:
    type: Any
    conditions:
      - type: ImageExists
        template: "Victory"
      - type: ImageExists
        template: "Defeat"
  actions:
    - action: Click
      x: 400
      y: 400
```

### None - NOR logic (none can be true)

```yaml
- action: Until
  condition:
    type: None
    conditions:
      - type: ImageExists
        template: "Enemy1"
      - type: ImageExists
        template: "Enemy2"
  actions:
    - action: Click
      x: 300
      y: 300
```

## Complex Nested Conditions

You can nest conditions arbitrarily deep for complex logic:

```yaml
- action: If
  condition:
    type: Any
    conditions:
      # First possibility: Ready AND not Busy
      - type: All
        conditions:
          - type: ImageExists
            template: "Ready"
          - type: ImageNotExists
            template: "Busy"
      # Second possibility: ForceStart button is present
      - type: ImageExists
        template: "ForceStart"
  then:
    - action: Click
      x: 600
      y: 600
```

## Template Configuration

All image-based conditions support the same template configuration as existing actions:

```yaml
condition:
  type: ImageExists
  template: "OK"           # Required: template name from registry
  threshold: 0.95          # Optional: override template's threshold
  region:                  # Optional: override template's search region
    x1: 100
    y1: 200
    x2: 500
    y2: 600
```

## Break Action (NEW!)

The `Break` action allows you to exit a loop early, regardless of the loop condition. This works with all loop types: `While`, `Until`, `Repeat`, `WhileImageFound`, `UntilImageFound`, etc.

### Basic Break Usage

```yaml
- action: While
  max_attempts: 20
  condition:
    type: ImageExists
    template: "Target"
  actions:
    # Check if we should stop early
    - action: If
      condition:
        type: ImageExists
        template: "Stop"
      then:
        - action: Break

    # Otherwise continue with normal actions
    - action: Click
      template: "Target"
```

### Break with Conditional Logic

```yaml
# Search until success, but break if error appears
- action: Until
  max_attempts: 15
  condition:
    type: ImageExists
    template: "Success"
  actions:
    # Check for error condition
    - action: If
      condition:
        type: ImageExists
        template: "Error"
      then:
        - action: Click
          template: "OK"
        - action: Break

    # Normal retry action
    - action: Click
      x: 300
      y: 300
```

### Nested Loops with Break

When `Break` is used in nested loops, it only breaks the innermost loop:

```yaml
- action: While
  max_attempts: 5
  condition:
    type: ImageExists
    template: "OuterCondition"
  actions:
    # Inner loop
    - action: While
      max_attempts: 3
      condition:
        type: ImageExists
        template: "InnerCondition"
      actions:
        - action: Click
          x: 100
          y: 100

        # This breaks only the inner loop
        - action: If
          condition:
            type: ImageExists
            template: "InnerTarget"
          then:
            - action: Break

    # This breaks the outer loop
    - action: If
      condition:
        type: ImageExists
        template: "OuterTarget"
      then:
        - action: Break
```

## Backward Compatibility

Your existing actions continue to work:
- `WhileImageFound` - Still works exactly as before (now supports Break!)
- `UntilImageFound` - Still works exactly as before (now supports Break!)
- `IfImageFound` - Still works exactly as before
- `Repeat` - Still works exactly as before (now supports Break!)
- All other existing actions remain unchanged

The new condition system is additive - it provides more flexibility for complex scenarios while keeping the simple cases simple.

## When to Use Each Approach

### Use Legacy Actions When:
- Simple, single-condition loops
- You want concise YAML
- Example: `WhileImageFound` for "keep clicking while X is visible"

### Use New Condition System When:
- Complex boolean logic (AND, OR, NOT)
- Multiple conditions to check
- Reusable condition patterns
- More readable for complex scenarios

## Examples

- [example_condition_system.yaml](../bin/routines/example_condition_system.yaml) - Basic condition types and control flow patterns
- [example_advanced_conditions.yaml](../bin/routines/example_advanced_conditions.yaml) - Else-if chains, Break action, and complex nested conditions

## Implementation Details

### Files Added
- `internal/actions/conditions.go` - Condition interface and implementations
- `internal/actions/if.go` - If control flow action with else-if support
- `internal/actions/while.go` - While control flow action
- `internal/actions/until.go` - Until control flow action
- `internal/actions/break.go` - Break action for early loop termination
- `internal/actions/unmarshal_helpers.go` - YAML unmarshaling utilities
- `internal/actions/conditions_test.go` - Unit tests

### Registry Updates
- Added `if`, `while`, `until`, `break` to action registry
- Created condition registry for polymorphic unmarshaling

### Files Modified
- Updated all loop actions (`While`, `Until`, `WhileImageFound`, `UntilImageFound`, `Repeat`) to support `Break`

## Future Extensions

The condition system is designed to be extensible. Future condition types could include:

- **State-based conditions**: `VariableEquals`, `VariableGreaterThan`, etc.
- **Time-based conditions**: `TimeElapsed`, `TimeOfDay`, etc.
- **Count-based conditions**: `IterationCount`, `MatchCount`, etc.
- **Network conditions**: `IsConnected`, `ResponseTimeBelow`, etc.
- **Custom conditions**: Implement the `Condition` interface for domain-specific logic

To add a new condition:
1. Create a struct implementing the `Condition` interface
2. Add it to `conditionRegistry` in `unmarshal_helpers.go`
3. Document the YAML syntax
