# Sentry Control Actions

Sentry control actions allow sentry routines to explicitly control the main routine's execution state. These actions provide fine-grained control over when the main routine should be paused for remediation.

## Overview

By default, **sentries run in parallel with the main routine**. They only interrupt the main routine when explicitly commanded to do so using `SentryHalt` and `SentryResume` actions.

This design allows:
- Quick checks (like error dialog detection) to run without disrupting the main routine
- Sentries to only halt execution when remediation is actually needed
- Better performance through parallel execution

## Actions

### SentryHalt

**Purpose**: Pauses the main routine execution

**Usage**: Call this action when the sentry detects a condition that requires remediation (e.g., error dialog found, unexpected state detected)

**Properties**: None

**Example**:
```yaml
- action: SentryHalt
```

**Behavior**:
- Pauses the main routine at its next checkpoint (between action steps)
- The sentry routine continues executing after the halt
- If the main routine is already paused or stopped, this is a no-op

### SentryResume

**Purpose**: Resumes the main routine execution

**Usage**: Call this action after the sentry has successfully remediated the issue

**Properties**: None

**Example**:
```yaml
- action: SentryResume
```

**Behavior**:
- Resumes the main routine from where it was paused
- The sentry routine continues executing after the resume
- If the main routine is not paused, this is a no-op

## Execution Flow

### Without Explicit Halt (Parallel Execution)

```yaml
routine_name: "Quick Check Sentry"
description: "Runs parallel checks without halting"

steps:
  - action: IfImageFound
    template: optional_popup
    then:
      - action: Click
        x: 540
        y: 800
  # No halt - sentry completes, main routine never paused
```

**Flow**:
1. Main routine continues running
2. Sentry checks for optional popup
3. If found, sentry clicks it
4. Sentry completes (triggers `on_success` action)
5. Main routine was never interrupted

### With Explicit Halt (Remediation)

```yaml
routine_name: "Error Handler Sentry"
description: "Halts for error remediation"

steps:
  - action: IfImageFound
    template: error_dialog
    then:
      - action: SentryHalt       # Pause main routine
      - action: Click
        x: 540
        y: 1200
      - action: Delay
        count: 2
      - action: SentryResume     # Resume main routine
```

**Flow**:
1. Main routine is running
2. Sentry detects error dialog
3. **SentryHalt** pauses main routine
4. Sentry dismisses error dialog
5. **SentryResume** resumes main routine
6. Both continue executing

## Sentry Actions (on_success/on_failure)

The `on_success` and `on_failure` properties define what happens **after** the sentry routine completes:

- **`resume`** (default for on_success): Does nothing if main routine wasn't halted, or resumes if it was halted by the sentry
- **`pause`**: Keeps the main routine paused (sentry must have called SentryHalt)
- **`stop`**: Stops the main routine gracefully at the end of its current step
- **`force_stop`** (default for on_failure): Immediately stops the main routine

### Example Configuration

```yaml
sentries:
  - routine: error_handler
    frequency: 5
    severity: high
    on_success: resume      # Main routine continues/resumes
    on_failure: force_stop  # Critical error, stop immediately
```

## Best Practices

### 1. Only Halt When Necessary

```yaml
# Good: Only halts if remediation is needed
- action: IfImageFound
  template: error_dialog
  then:
    - action: SentryHalt
    # ... remediation steps ...
    - action: SentryResume

# Bad: Always halts even for quick checks
- action: SentryHalt
- action: IfImageFound
  template: optional_popup
  then:
    - action: Click
      x: 540
      y: 800
- action: SentryResume
```

### 2. Always Resume After Successful Remediation

```yaml
- action: IfImageFound
  template: error_dialog
  then:
    - action: SentryHalt
    - action: Click           # Dismiss error
      x: 540
      y: 1200
    - action: Delay
      count: 2
    - action: SentryResume   # ✓ Resume after fixing
```

### 3. Let Sentry Fail for Unrecoverable Errors

```yaml
- action: IfImageFound
  template: critical_error
  then:
    - action: SentryHalt
    - action: Click
      x: -1
      y: -1  # Force error to trigger on_failure: force_stop
```

### 4. Use Severity Appropriately

```yaml
sentries:
  # High-frequency, low-severity: Quick checks
  - routine: optional_popup_check
    frequency: 3
    severity: low
    on_success: resume
    on_failure: resume  # Even if check fails, continue

  # Low-frequency, high-severity: Critical checks
  - routine: error_dialog_check
    frequency: 10
    severity: high
    on_success: resume
    on_failure: force_stop
```

## Technical Details

### Sentry Execution Flag

Sentry routines are marked with an internal `isSentryExecution` flag that prevents them from being affected by their own halt commands. This ensures:

- Sentries can check `CheckPauseOrStop()` without blocking themselves
- Loops within sentry routines work correctly
- Remediation logic executes uninterrupted

### Routine Controller State

The routine controller manages execution state:
- **StateRunning**: Main routine is executing normally
- **StatePaused**: Main routine is paused (by SentryHalt or user)
- **StateStopped**: Main routine was force stopped
- **StateCompleted**: Main routine completed normally

Sentry actions modify this shared state, but sentries themselves ignore it during their execution.

## Migration from Old Behavior

**Old behavior**: Sentries automatically paused the main routine before executing

**New behavior**: Sentries run in parallel and only pause when explicitly commanded

**Migration steps**:
1. Identify sentries that need to halt the main routine
2. Add `SentryHalt` at the start of remediation logic
3. Add `SentryResume` after successful remediation
4. Remove any workarounds that assumed automatic pausing

**Example migration**:

```yaml
# Old (automatic pause, assumed):
steps:
  - action: Click
    x: 540
    y: 1200
  - action: Delay
    count: 2

# New (explicit control):
steps:
  - action: IfImageFound
    template: error_dialog
    then:
      - action: SentryHalt      # Explicit pause
      - action: Click
        x: 540
        y: 1200
      - action: Delay
        count: 2
      - action: SentryResume    # Explicit resume
```

## See Also

- [Sentry System Documentation](SENTRIES.md)
- [Routine Controller](ROUTINE_CONTROLLER.md)
- [Action Reference](ACTIONS.md)
