# Navigation Routines

This directory contains routines for navigating between screens and menus in Pokemon TCG Pocket.

## Purpose

Navigation routines handle:
- Menu traversal
- Screen transitions
- Modal dismissal
- Back button handling
- Home screen navigation

## Naming Conventions

- Use descriptive paths: `navigate_to_shop.yaml`, `navigate_to_battle.yaml`
- Use `go_to_*` prefix: `go_to_home.yaml`, `go_to_deck.yaml`
- Include origin when needed: `shop_to_home.yaml`

## Common Patterns

### Simple Navigation
```yaml
routine_name: "Navigate to Screen Name"
description: "Navigate from current screen to target screen"

actions:
  - action: WaitForTemplate
    template: current_screen_indicator
    timeout: 5000

  - action: Click
    template: menu_button

  - action: Click
    template: target_screen_button

  - action: WaitForTemplate
    template: target_screen_indicator
```

### Multi-Step Navigation
```yaml
routine_name: "Navigate Through Multiple Screens"
actions:
  - action: RunRoutine
    routine: navigation/go_to_home

  - action: RunRoutine
    routine: navigation/home_to_shop

  - action: Click
    template: specific_shop_item
```

### Safe Navigation with Retries
```yaml
actions:
  - action: If
    condition_type: template_exists
    template: popup_close_button
    actions:
      - action: Click
        template: popup_close_button

  - action: WaitForTemplate
    template: target_screen
    timeout: 10000
    retries: 3
```

## Best Practices

1. **Idempotent Navigation**: Routines should work regardless of starting state
2. **Verify Arrival**: Always confirm you reached the target screen
3. **Handle Popups**: Dismiss unexpected popups before navigating
4. **Use Common Navigation**: Build reusable navigation building blocks
5. **Add Timeouts**: Don't wait forever for screens to load

## Recommended Templates

- `home_button` - Return to home screen
- `back_button` - Go back one screen
- `menu_button` - Open main menu
- `shop_tab` - Navigate to shop
- `deck_tab` - Navigate to deck builder
- `battle_tab` - Navigate to battle selection
- `popup_close_button` - Dismiss popups

## Common Navigation Paths

Create these base navigation routines:
- `go_to_home.yaml` - From anywhere to home screen
- `go_to_shop.yaml` - From home to shop
- `go_to_battle.yaml` - From home to battle selection
- `go_to_deck.yaml` - From home to deck builder
- `dismiss_popups.yaml` - Close any open popups

## Related Domains

- **error_handling/** - Handle navigation failures
- **combat/** - Navigate to battle screens
- **farming/** - Navigate to farming locations
- **examples/** - See example_routine.yaml for navigation patterns
