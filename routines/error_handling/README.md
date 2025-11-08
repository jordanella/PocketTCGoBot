# Error Handling Routines

This directory contains routines for detecting and recovering from errors in Pokemon TCG Pocket.

## Purpose

Error handling routines are primarily used as **sentry routines** that monitor for problems and take corrective action:
- Connection loss detection
- Popup/ad dismissal
- Game crash recovery
- Unexpected screen detection
- Resource exhaustion handling

## Naming Conventions

- Use descriptive error names: `connection_check.yaml`, `popup_handler.yaml`
- Prefix with check/detect: `check_connection.yaml`, `detect_error_screen.yaml`
- Include recovery action: `dismiss_ads.yaml`, `recover_from_crash.yaml`

## Common Patterns

### Sentry Routine (Error Detection)
```yaml
routine_name: "Error Detection Routine"
description: "Detects and handles specific error condition"

# Sentry routines should return error (non-nil) when problem detected
# Return nil when everything is OK

actions:
  - action: If
    condition_type: template_exists
    template: error_indicator
    actions:
      # Handle the error
      - action: Click
        template: error_dismiss_button

      # Force error to trigger on_failure action
      - action: Fail
        message: "Error detected and handled"
```

### Popup Dismissal
```yaml
routine_name: "Dismiss Popups"
description: "Close any unexpected popups or ads"

actions:
  - action: If
    condition_type: template_exists
    template: popup_close_button
    actions:
      - action: Click
        template: popup_close_button
      - action: Wait
        duration: 1000
```

### Connection Check
```yaml
routine_name: "Check Connection"
description: "Verify game is still connected"

actions:
  - action: If
    condition_type: template_exists
    template: connection_lost_screen
    actions:
      - action: Click
        template: reconnect_button
      - action: Fail
        message: "Connection lost, attempting reconnect"
```

## Sentry Configuration

Error handling routines are typically used as sentries in other routines:

```yaml
routine_name: "Main Routine"
actions:
  # ... main logic ...

sentries:
  - routine: error_handling/popup_handler
    frequency: 10
    severity: low
    on_success: resume
    on_failure: resume  # Continue even if popup found

  - routine: error_handling/connection_check
    frequency: 30
    severity: high
    on_success: resume
    on_failure: force_stop  # Stop if connection lost
```

## Severity Guidelines

- **Low**: Minor issues (ads, popups) - continue running
- **Medium**: Recoverable issues (temporary errors) - retry action
- **High**: Serious issues (connection loss) - stop routine
- **Critical**: Fatal errors (game crash) - force stop immediately

## Best Practices

1. **Keep It Simple**: Error handlers should be fast and focused
2. **Return Errors Appropriately**: Return nil when OK, error when problem detected
3. **Use Appropriate Severity**: Match severity to impact
4. **Set Proper Frequency**: Balance detection speed vs performance
5. **Test Failure Paths**: Verify error handlers work when triggered

## Recommended Templates

- `connection_lost_screen` - Network error
- `error_dialog` - Generic error popup
- `maintenance_screen` - Game under maintenance
- `popup_close_button` - Generic close button
- `ad_close_button` - Advertisement close
- `crash_recovery_screen` - Game crashed

## Common Error Handlers

Create these base error handling routines:
- `popup_handler.yaml` - Dismiss unexpected popups (low severity)
- `connection_check.yaml` - Monitor connection status (high severity)
- `error_screen_handler.yaml` - Handle error dialogs (medium severity)
- `maintenance_check.yaml` - Detect maintenance mode (critical severity)

## Testing Error Handlers

1. Manually trigger error conditions
2. Verify sentry detects the error
3. Confirm on_success/on_failure actions work correctly
4. Check logging output for severity levels
5. Test with different frequency values

## Related Domains

- **navigation/** - Navigate away from error screens
- **combat/** - Handle mid-battle errors
- **farming/** - Farming error recovery
- **examples/** - See example_sentry_popup_handler.yaml
