# Routine Library

This directory contains all bot routines for Pokemon TCG Pocket automation, organized by domain.

## Directory Structure

```
routines/
├── README.md (this file)
├── combat/          # Battle and combat routines
├── farming/         # Resource farming and grinding
├── navigation/      # Menu navigation and screen traversal
├── error_handling/  # Error detection and recovery (sentries)
└── examples/        # Example routines and templates
```

Each domain folder contains:
- `README.md` - Domain-specific documentation and conventions
- `_template.yaml` - Starter template for new routines in that domain
- Routine files organized by functionality

## Naming Conventions

### File Naming
- **Lowercase with underscores**: `battle_loop.yaml`, `farm_coins.yaml`
- **Descriptive names**: Name should indicate purpose
- **Domain prefixes** (when needed): `pvp_battle.yaml`, `event_farm_tokens.yaml`
- **Extension**: Always use `.yaml` (preferred) or `.yml`

### Routine Organization
Routines are now **namespaced** by their folder structure:
- Top-level: `common_navigation` → loads `routines/navigation/common_navigation.yaml`
- Namespaced: `combat/battle_loop` → loads `routines/combat/battle_loop.yaml`

### Routine Naming Best Practices

**Good Names:**
- `navigation/go_to_shop.yaml` - Clear, action-oriented
- `farming/farm_daily_missions.yaml` - Descriptive, includes domain
- `error_handling/popup_handler.yaml` - Purpose is obvious

**Poor Names:**
- `routine1.yaml` - Not descriptive
- `test.yaml` - Too generic
- `BATTLE.yaml` - Use lowercase

## Domain Guidelines

### Combat (`combat/`)
Routines for battles and combat mechanics.
- Battle execution
- Card selection logic
- Win/loss detection
- Reward collection

**Examples:** `battle_loop.yaml`, `pvp_battle.yaml`, `collect_battle_rewards.yaml`

### Farming (`farming/`)
Automated resource gathering and grinding.
- Daily mission completion
- Resource farming loops
- Event farming
- Pack opening automation

**Examples:** `farm_daily_missions.yaml`, `farm_coins.yaml`, `event_farm_tokens.yaml`

### Navigation (`navigation/`)
Screen navigation and menu traversal.
- Common navigation paths
- Screen transitions
- Modal dismissal
- Reusable navigation routines

**Examples:** `go_to_home.yaml`, `navigate_to_shop.yaml`, `dismiss_popups.yaml`

### Error Handling (`error_handling/`)
Error detection and recovery (primarily sentry routines).
- Connection monitoring
- Popup/ad dismissal
- Error screen detection
- Crash recovery

**Examples:** `popup_handler.yaml`, `connection_check.yaml`, `error_screen_handler.yaml`

### Examples (`examples/`)
Example routines demonstrating bot features.
- Learning resources
- Feature demonstrations
- Reference implementations
- Testing routines

**Examples:** `example_routine.yaml`, `example_variables.yaml`, `example_sentry.yaml`

## Creating New Routines

### Quick Start
1. Navigate to the appropriate domain folder
2. Copy the `_template.yaml` file
3. Rename to describe your routine: `cp _template.yaml my_routine.yaml`
4. Edit the routine with your logic
5. Test with a single bot instance

### Template Structure
Every routine should include:
```yaml
routine_name: "Human-Readable Routine Name"
description: "Brief description of what this routine does"
tags: ["domain", "keywords"]

# Optional: User-configurable parameters
config:
  - name: parameter_name
    type: int
    default: 5
    description: "What this parameter controls"

# Optional: Sentry routines for monitoring
sentries:
  - routine: error_handling/popup_handler
    frequency: 10
    severity: low
    on_success: resume
    on_failure: resume

# Main routine logic
actions:
  - action: ActionName
    # ... action parameters
```

## Routine Composition

Build complex routines from simpler ones using `RunRoutine`:

```yaml
# Main routine
actions:
  - action: RunRoutine
    routine: navigation/go_to_home

  - action: RunRoutine
    routine: navigation/home_to_shop

  - action: RunRoutine
    routine: farming/farm_daily_missions
    config:
      max_runs: 5  # Override default config
```

## Using Namespaces

### In YAML Files
Reference routines with their namespace:
```yaml
- action: RunRoutine
  routine: combat/battle_loop

- action: RunRoutine
  routine: navigation/go_to_home
```

### In GUI
Routines are displayed grouped by namespace:
```
── combat ──
combat/battle_loop
combat/pvp_battle

── navigation ──
navigation/common_navigation
navigation/go_to_home
```

### In Code
The routine registry automatically handles namespaces:
```go
builder, err := registry.Get("combat/battle_loop")
builder, err := registry.Get("navigation/go_to_home")
```

## Sentry Routines

Sentry routines monitor for errors and run in parallel:

### Defining a Sentry
```yaml
routine_name: "Error Detection Sentry"
tags: ["sentry", "error_handling"]

actions:
  - action: If
    condition_type: template_exists
    template: error_indicator
    actions:
      - action: Click
        template: dismiss_button
      - action: Fail
        message: "Error detected and handled"
```

### Using Sentries
```yaml
sentries:
  - routine: error_handling/popup_handler
    frequency: 10       # Check every 10 seconds
    severity: low       # Log severity
    on_success: resume  # Action when no error
    on_failure: resume  # Action when error detected
```

### Severity Levels
- **low**: Minor issues (popups, ads)
- **medium**: Recoverable issues (temporary errors)
- **high**: Serious issues (connection loss)
- **critical**: Fatal errors (crashes)

### Sentry Actions
- **resume**: Continue main routine
- **pause**: Pause main routine
- **stop**: Graceful stop
- **force_stop**: Immediate stop

## Configuration System

### Defining Config Parameters
```yaml
config:
  - name: max_runs
    type: int
    default: 10
    description: "Maximum farming iterations"

  - name: enable_logging
    type: bool
    default: true
    description: "Enable verbose logging"
```

### Using Config in Routines
```yaml
actions:
  - action: While
    condition: "${run_count} < ${max_runs}"
    actions:
      # ... loop body
```

### Overriding Config
```yaml
- action: RunRoutine
  routine: farming/farm_coins
  config:
    max_runs: 20        # Override default (10)
    enable_logging: false
```

## Variable System

### Setting Variables
```yaml
- action: SetVariable
  variable: counter
  value: 0

- action: SetVariable
  variable: message
  value: "Hello World"
```

### Using Variables
```yaml
- action: Comment
  text: "Counter is ${counter}"

- action: While
  condition: "${counter} < ${max_runs}"
  actions:
    # ... loop body
```

### Variable Interpolation
Variables can be used in:
- String values: `text: "Value is ${var}"`
- Conditions: `condition: "${counter} < 10"`
- Numeric calculations: `value: "${counter} + 1"`

## Best Practices

### 1. Modularity
Build small, reusable routines:
- `navigation/go_to_home.yaml` - Single purpose
- `error_handling/popup_handler.yaml` - Focused on one error type

### 2. Error Handling
Always include appropriate sentries:
```yaml
sentries:
  - routine: error_handling/popup_handler
    frequency: 10
    severity: low
    on_success: resume
    on_failure: resume

  - routine: error_handling/connection_check
    frequency: 30
    severity: high
    on_success: resume
    on_failure: force_stop
```

### 3. Timeouts
Add timeouts to prevent infinite waits:
```yaml
- action: WaitForTemplate
  template: button
  timeout: 10000  # 10 seconds max
```

### 4. Logging
Use comments for debugging:
```yaml
- action: Comment
  text: "Starting battle ${battle_count}"
```

### 5. Configuration
Make routines configurable:
```yaml
config:
  - name: max_retries
    type: int
    default: 3
    description: "Max retry attempts"
```

### 6. Testing
Test routines in isolation before combining:
1. Test with single bot instance
2. Verify error paths
3. Test with different config values
4. Monitor for resource leaks in long runs

## Template System

Templates are defined in `config/templates/` and referenced by name:

```yaml
- action: WaitForTemplate
  template: button_name  # References config/templates/button_name.yaml

- action: Click
  template: confirm_button
```

See `config/templates/README.md` for template documentation.

## Migration Notes

### Old Routines
Existing routines have been migrated to namespaced structure:
- `common_navigation.yaml` → `navigation/common_navigation.yaml`
- `example_*.yaml` → `examples/example_*.yaml`
- `example_sentry_popup_handler.yaml` → `error_handling/example_sentry_popup_handler.yaml`

### Backward Compatibility
The routine registry supports both:
- Old style: `common_navigation` (looks in root first)
- New style: `navigation/common_navigation` (namespace-aware)

## Development Workflow

### Adding New Domain Scripts

1. **Identify Domain**: Determine which folder (combat/farming/navigation/error_handling)
2. **Copy Template**: `cp domain/_template.yaml domain/my_routine.yaml`
3. **Edit Routine**: Add your logic, config, sentries
4. **Test Standalone**: Run with single bot to verify
5. **Integrate**: Use in larger routine compositions

### Example Development Flow

```bash
# 1. Create new combat routine
cd routines/combat
cp _template.yaml pvp_battle.yaml

# 2. Edit the routine
# ... edit pvp_battle.yaml ...

# 3. Test in bot launcher GUI
# Select "combat/pvp_battle" from dropdown

# 4. Use in farming routine
cd ../farming
# Reference in farm_pvp_rewards.yaml:
# - action: RunRoutine
#   routine: combat/pvp_battle
```

## Troubleshooting

### Routine Not Found
- Check namespace: `navigation/go_to_home` not `go_to_home`
- Verify file extension: `.yaml` or `.yml`
- Check file exists: `ls routines/navigation/`

### Validation Errors
- Check YAML syntax (indentation, colons, quotes)
- Verify action names are correct
- Ensure templates exist in template registry

### Sentry Issues
- Verify sentry routine returns error when problem detected
- Check frequency (don't poll too fast)
- Confirm on_success/on_failure actions are valid

## Resources

- **Action Documentation**: See `docs/ACTIONS.md`
- **Template Documentation**: See `config/templates/README.md`
- **Sentry Documentation**: See `docs/SENTRIES.md`
- **Variable Documentation**: See `docs/VARIABLES.md`
- **Config Documentation**: See `docs/CONFIG.md`

## Contributing

When adding new routines:
1. Follow naming conventions
2. Use appropriate domain folder
3. Include README updates if adding new patterns
4. Test thoroughly before committing
5. Document any new templates needed

---

**Last Updated:** 2025-11-08
**Structure Version:** v2.0 (Namespaced)
