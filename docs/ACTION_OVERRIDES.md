# Action-Level Template Overrides

Actions that use templates support **action-level overrides** for threshold and region. This allows you to:
- Define templates once in the registry (single source of truth)
- Override threshold or region per-action when needed
- Keep routines simple and maintainable

## Basic Usage

### Without Overrides

Use the template with its default settings from the registry:

```yaml
steps:
  - action: WhileTemplateExists
    max_attempts: 10
    template_name: "OK"  # Uses template's default threshold and region
    actions:
      - action: Click
        x: 100
        y: 200
```

### With Threshold Override

Override the matching threshold for this specific action:

```yaml
steps:
  - action: WhileTemplateExists
    max_attempts: 10
    template: "OK"
    threshold: 0.95  # Override: more strict matching than template default
    actions:
      - action: Click
        x: 100
        y: 200
```

### With Region Override

Override the search region for this specific action:

```yaml
steps:
  - action: WhileTemplateExists
    max_attempts: 10
    template: "OK"
    region:  # Override: only search in bottom-right corner
      x1: 400
      y1: 300
      x2: 800
      y2: 600
    actions:
      - action: Click
        x: 100
        y: 200
```

### With Both Overrides

```yaml
steps:
  - action: WhileTemplateExists
    max_attempts: 10
    template: "OK"
    threshold: 0.95     # More strict matching
    region:             # Limited search area
      x1: 400
      y1: 300
      x2: 800
      y2: 600
    actions:
      - action: Click
        x: 100
        y: 200
```

## How Overrides Work

### Priority System

1. **Action-level** settings take highest priority
2. **Template-level** settings are the fallback
3. **System defaults** if neither is specified

### Threshold Priority

```yaml
# Template definition
templates:
  - name: OK
    path: ui/OK.png
    threshold: 0.8  # Template default

# Routine with override
steps:
  - action: WhileTemplateExists
    template_name: "OK"
    threshold: 0.95  # Action override - THIS VALUE IS USED
```

Result: Uses `0.95` threshold for this specific action.

### Region Priority

```yaml
# Template definition
templates:
  - name: OK
    path: ui/OK.png
    region:  # Template default search area
      x1: 0
      y1: 0
      x2: 800
      y2: 600

# Routine with override
steps:
  - action: WhileTemplateExists
    template_name: "OK"
    region:  # Action override - THIS REGION IS USED
      x1: 400
      y1: 300
      x2: 800
      y2: 600
```

Result: Uses action-level region `(400,300)-(800,600)` for this specific action.

### No Override

```yaml
# Template definition
templates:
  - name: OK
    path: ui/OK.png
    threshold: 0.8
    region:
      x1: 100
      y1: 100
      x2: 700
      y2: 500

# Routine without overrides
steps:
  - action: WhileTemplateExists
    template_name: "OK"
    # No threshold or region specified
```

Result: Uses template's defaults (`threshold: 0.8`, `region: (100,100)-(700,500)`).

## Use Cases

### Use Case 1: Strict Matching for Critical Actions

Some actions need very precise matching to avoid mistakes:

```yaml
templates:
  - name: BuyButton
    path: ui/buy.png
    threshold: 0.8  # Normal matching for most cases

steps:
  # Normal buy action - uses default
  - action: FindImage
    template_name: "BuyButton"

  # Expensive purchase - needs strict matching
  - action: FindImage
    template_name: "BuyButton"
    threshold: 0.98  # Override: very strict to avoid accidents
```

### Use Case 2: Context-Specific Search Regions

Same template appears in multiple locations, search different areas:

```yaml
templates:
  - name: CloseButton
    path: ui/close.png
    threshold: 0.8
    # No default region - appears in multiple places

steps:
  # Close button in main dialog (top-right)
  - action: WhileTemplateExists
    template_name: "CloseButton"
    region:
      x1: 600
      y1: 0
      x2: 800
      y2: 100
    actions:
      - action: Click
        x: 750
        y: 50

  # Close button in settings menu (bottom-left)
  - action: WhileTemplateExists
    template_name: "CloseButton"
    region:
      x1: 0
      y1: 500
      x2: 200
      y2: 600
    actions:
      - action: Click
        x: 100
        y: 550
```

### Use Case 3: Progressive Threshold Adjustment

Try strict matching first, then relax if not found:

```yaml
templates:
  - name: RewardClaim
    path: ui/reward.png
    threshold: 0.85  # Moderate default

steps:
  # First attempt: strict matching
  - action: FindImage
    template_name: "RewardClaim"
    threshold: 0.95

  # If not found, try with relaxed matching
  - action: FindImage
    template_name: "RewardClaim"
    threshold: 0.75
```

### Use Case 4: Per-Screen Region Optimization

Different screens need different search areas for the same template:

```yaml
templates:
  - name: NextButton
    path: ui/next.png
    threshold: 0.8

steps:
  # Tutorial screen - Next button at bottom center
  - action: WhileTemplateExists
    template_name: "NextButton"
    region:
      x1: 300
      y1: 500
      x2: 500
      y2: 600

  # Shop screen - Next button at bottom right
  - action: WhileTemplateExists
    template_name: "NextButton"
    region:
      x1: 600
      y1: 500
      x2: 800
      y2: 600
```

## Actions Supporting Overrides

### WhileTemplateExists

```yaml
- action: WhileTemplateExists
  max_attempts: 10
  template_name: "TemplateName"  # Required
  threshold: 0.95                # Optional override
  region:                        # Optional override
    x1: 100
    y1: 200
    x2: 500
    y2: 600
  actions:
    # Nested actions...
```

### FindImage

```yaml
- action: FindImage
  template_name: "TemplateName"  # Required
  threshold: 0.95                # Optional override
  region:                        # Optional override
    x1: 100
    y1: 200
    x2: 500
    y2: 600
```

## Best Practices

### 1. Define Sensible Defaults in Templates

```yaml
# templates/ui_templates.yaml
templates:
  - name: OK
    path: ui/OK.png
    threshold: 0.8      # Good default for most cases
    region:             # Common search area
      x1: 0
      y1: 400
      x2: 800
      y2: 600
```

### 2. Override Only When Necessary

```yaml
# ✅ GOOD - Most actions use template defaults
- action: WhileTemplateExists
  template_name: "OK"  # No overrides needed

# ✅ GOOD - Override for special case
- action: WhileTemplateExists
  template_name: "OK"
  threshold: 0.98  # Critical action needs strict matching
```

### 3. Document Why You're Overriding

```yaml
# ✅ GOOD - Comment explains the override
- action: WhileTemplateExists
  template_name: "BuyButton"
  threshold: 0.98  # Strict matching to prevent accidental purchases
```

### 4. Use Region Overrides for Disambiguation

```yaml
# ✅ GOOD - Region prevents false positives
templates:
  - name: BackButton
    path: ui/back.png

steps:
  # Multiple "Back" buttons on screen, search specific area
  - action: WhileTemplateExists
    template_name: "BackButton"
    region:  # Only search top-left corner for THIS back button
      x1: 0
      y1: 0
      x2: 200
      y2: 100
```

### 5. Keep Template Definitions DRY

```yaml
# ❌ BAD - Duplicating path in every routine
- action: WhileTemplateExists
  template:
    name: "OK"
    path: "templates/ui/OK.png"
    threshold: 0.8

# ✅ GOOD - Path defined once in templates, override when needed
templates:
  - name: OK
    path: ui/OK.png
    threshold: 0.8

steps:
  - action: WhileTemplateExists
    template_name: "OK"
    # threshold: 0.95  # Uncomment to override only when needed
```

## Validation

### Build-Time Validation

Templates are validated when loading the routine:

```yaml
- action: WhileTemplateExists
  template_name: "NonExistent"  # Error: template not found in registry
```

Error message:
```
routine 'DailyTasks' step 5 validation failed:
  template 'NonExistent' not found in registry
```

### Override Validation

Overrides are type-checked:

```yaml
- action: WhileTemplateExists
  template_name: "OK"
  threshold: "high"  # Error: threshold must be a number
```

```yaml
- action: WhileTemplateExists
  template_name: "OK"
  region:
    x1: "left"  # Error: coordinates must be numbers
```

## Common Patterns

### Pattern 1: Template with Multiple Contexts

One template, different usage contexts:

```yaml
templates:
  - name: Checkbox
    path: ui/checkbox.png
    threshold: 0.8

steps:
  # Settings page - checkboxes in a list
  - action: WhileTemplateExists
    template_name: "Checkbox"
    region:  # Search in settings list area
      x1: 100
      y1: 200
      x2: 700
      y2: 500

  # Popup dialog - single checkbox
  - action: FindImage
    template_name: "Checkbox"
    region:  # Search in dialog area
      x1: 300
      y1: 250
      x2: 500
      y2: 350
```

### Pattern 2: Fallback Detection

Try strict matching first, fall back to relaxed:

```yaml
templates:
  - name: Reward
    path: ui/reward.png
    threshold: 0.85

steps:
  # Try strict first
  - action: FindImage
    template_name: "Reward"
    threshold: 0.95

  # If not found (continues to next action), try relaxed
  - action: FindImage
    template_name: "Reward"
    threshold: 0.75
```

### Pattern 3: Screen-Specific Tuning

Different screens need different thresholds:

```yaml
templates:
  - name: PlayButton
    path: ui/play.png
    threshold: 0.8

steps:
  # Main menu - high quality graphics, strict matching
  - action: FindImage
    template_name: "PlayButton"
    threshold: 0.9

  # Loading screen - compressed graphics, relaxed matching
  - action: FindImage
    template_name: "PlayButton"
    threshold: 0.7
```

## Migration from Inline Templates

**Before** (inline template):
```yaml
- action: WhileTemplateExists
  template:
    name: "OK"
    path: "templates/ui/OK.png"
    threshold: 0.9
    region:
      x1: 100
      y1: 200
      x2: 500
      y2: 600
```

**After** (registry + overrides):
```yaml
# templates/ui_templates.yaml
templates:
  - name: OK
    path: ui/OK.png
    threshold: 0.8  # Default threshold

# routine.yaml
- action: WhileTemplateExists
  template_name: "OK"
  threshold: 0.9  # Override for this action
  region:         # Override for this action
    x1: 100
    y1: 200
    x2: 500
    y2: 600
```

**Benefits**:
- Template path defined once
- Default threshold can be adjusted globally
- Override only what you need per-action
- Build-time validation ensures template exists

## Summary

**Action-level overrides** provide:
- ✅ **Single source of truth** - Templates defined once in registry
- ✅ **Flexible customization** - Override threshold/region per-action
- ✅ **Maintainability** - Update defaults globally, override locally
- ✅ **Build-time validation** - Catch missing templates early
- ✅ **Image caching** - Registry provides preload/on-demand/unload-after
- ✅ **Simpler routines** - No path duplication, clear intent

Use **template defaults** for common cases and **action-level overrides** for special situations.
