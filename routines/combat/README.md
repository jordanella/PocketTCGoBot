# Combat Routines

This directory contains routines related to battle and combat mechanics in Pokemon TCG Pocket.

## Purpose

Combat routines handle:
- Battle initiation and execution
- Card selection and play decisions
- Turn management
- Win/loss detection
- Battle rewards collection

## Naming Conventions

- Use lowercase with underscores: `battle_loop.yaml`
- Prefix with context when needed: `pvp_battle.yaml`, `ai_battle.yaml`
- Use descriptive names that indicate the routine's purpose

## Common Patterns

### Battle Loop Structure
```yaml
routine_name: "Battle Routine Name"
description: "Description of what this battle routine does"

actions:
  - action: WaitForTemplate
    template: battle_start_button

  - action: Click
    template: battle_start_button

  # ... battle logic ...

  - action: WaitForTemplate
    template: victory_screen
    timeout: 300000  # 5 minutes max battle time
```

### Recommended Templates

- `battle_start_button` - Initiates a battle
- `card_hand_*` - Cards in player's hand
- `attack_button` - Confirm attack action
- `victory_screen` - Battle won
- `defeat_screen` - Battle lost
- `battle_rewards` - Post-battle reward screen

## Testing

Before deploying combat routines:
1. Test in AI battles first (low risk)
2. Verify timeout handling (battles can take 5+ minutes)
3. Test both victory and defeat paths
4. Ensure proper reward collection

## Related Domains

- **navigation/** - Getting to battle screens
- **error_handling/** - Handling disconnects mid-battle
- **farming/** - Automated battle farming for resources
