# Farming Routines

This directory contains routines for automated resource gathering and farming in Pokemon TCG Pocket.

## Purpose

Farming routines handle:
- Automated pack opening
- Daily mission completion
- Resource grinding (coins, cards, etc.)
- Event farming
- Bulk battle farming for XP/rewards

## Naming Conventions

- Use lowercase with underscores: `farm_daily_missions.yaml`
- Include resource type: `farm_coins.yaml`, `farm_packs.yaml`
- Use prefixes for event farming: `event_farm_*.yaml`

## Common Patterns

### Loop-Based Farming
```yaml
routine_name: "Farm Resource Name"
description: "Automated farming routine"

config:
  - name: max_runs
    type: int
    default: 10
    description: "Maximum number of farming iterations"

actions:
  - action: SetVariable
    variable: run_count
    value: 0

  - action: While
    condition: "${run_count} < ${max_runs}"
    actions:
      # ... farming logic ...

      - action: SetVariable
        variable: run_count
        value: "${run_count} + 1"
```

### Error Recovery
Always include sentry routines for farming:
```yaml
sentries:
  - routine: error_handling/connection_check
    frequency: 30
    severity: high
    on_failure: force_stop
```

## Best Practices

1. **Use Configurable Limits**: Allow max_runs to be configured
2. **Include Counters**: Track progress with variables
3. **Timeout Protection**: Add timeouts to prevent infinite loops
4. **Resource Monitoring**: Check if resources are full before farming
5. **Error Handling**: Use sentries to detect and handle errors

## Recommended Templates

- `claim_reward_button` - Collect farmed resources
- `resource_full_indicator` - Stop farming when full
- `daily_mission_complete` - Mission completion check
- `pack_available` - Pack ready to open
- `energy_empty` - No energy to continue

## Safety Considerations

- **Rate Limiting**: Add delays between runs to avoid detection
- **Random Delays**: Use random_delay to simulate human behavior
- **Session Limits**: Don't farm for more than 2-3 hours continuously
- **Error Thresholds**: Stop after N consecutive errors

## Related Domains

- **navigation/** - Navigating to farming locations
- **combat/** - Battle-based farming
- **error_handling/** - Recovery from farming errors
