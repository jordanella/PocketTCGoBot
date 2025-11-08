# Example Routines

This directory contains example routines that demonstrate various bot features and patterns.

## Purpose

Example routines serve as:
- Learning resources for new routine developers
- Templates for common patterns
- Test cases for bot functionality
- Reference implementations

## Naming Conventions

- Prefix with `example_`: `example_routine.yaml`, `example_sentry.yaml`
- Include feature name: `example_variables.yaml`, `example_loop.yaml`

## Available Examples

The example routines demonstrate different aspects of the bot system:

### Basic Examples
- See existing example files in the top-level routines folder
- These will be moved here for better organization

## What to Learn From Examples

### Action Usage
- How to use different action types (Click, Swipe, Wait, etc.)
- Template matching with WaitForTemplate
- Screen capture and computer vision

### Control Flow
- If/Else conditional logic
- While loops for iteration
- Break/Continue for loop control

### Variables
- Setting and getting variables
- Variable interpolation with `${variable_name}`
- Variable scoping

### Routine Composition
- Using RunRoutine to call other routines
- Passing config overrides to nested routines
- Building complex behaviors from simple routines

### Sentry System
- Defining sentry routines
- Configuring frequency and severity
- on_success and on_failure actions

### Configuration
- Defining config parameters
- Setting defaults and descriptions
- Type validation (int, bool, string)

## Creating New Examples

When creating example routines:

1. **Focus on One Concept**: Each example should demonstrate a specific feature
2. **Add Comments**: Use description fields to explain what's happening
3. **Keep It Simple**: Don't combine too many features in one example
4. **Make It Runnable**: Examples should be executable (even if no-ops)
5. **Document Thoroughly**: Explain why, not just what

## Example Template

```yaml
routine_name: "Example: Feature Name"
description: "Demonstrates how to use [specific feature]"
tags: ["example", "feature-name"]

# Config demonstrates user-configurable parameters
config:
  - name: example_param
    type: int
    default: 5
    description: "Example configuration parameter"

# Actions demonstrate the feature
actions:
  - action: Comment
    text: "This action demonstrates [feature]"

  # ... feature demonstration ...
```

## Related Domains

All example routines may reference routines from:
- **navigation/** - Screen navigation examples
- **error_handling/** - Error handling patterns
- **combat/** - Battle routine patterns
- **farming/** - Farming loop patterns

## Best Practices for Examples

- Use realistic but simple scenarios
- Include both success and failure cases
- Add logging/output so users can see what's happening
- Reference actual templates (or note that templates are for illustration)
- Keep examples short (under 50 lines when possible)
