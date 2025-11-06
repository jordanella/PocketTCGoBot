# Inline Template Support in Actions

Actions that use templates (`WhileTemplateExists`, `FindImage`, etc.) support **two ways** to specify templates:

1. **Template Name** - Look up from registry (recommended)
2. **Inline Template** - Specify directly in YAML

## Template Name (Recommended)

Reference a template from the registry by name. This provides:
- Build-time validation (template existence checked when loading routine)
- Image caching (preload, on-demand, unload-after modes)
- Centralized configuration
- Easy updates without changing routines

### Example

**templates/ui_templates.yaml**:
```yaml
templates:
  - name: OK
    path: ui/OK.png
    threshold: 0.8
    region:
      x1: 120
      y1: 316
      x2: 143
      y2: 335
    preload: true
```

**routines/daily_tasks.yaml**:
```yaml
steps:
  - action: WhileTemplateExists
    max_attempts: 10
    template_name: "OK"  # Reference by name
    actions:
      - action: Click
        x: 100
        y: 200
```

## Inline Template

Specify the template directly in the action YAML. This provides:
- Self-contained routines (no external template dependencies)
- One-off templates that aren't reused
- Quick prototyping without updating template registry

### Example

```yaml
steps:
  - action: WhileTemplateExists
    max_attempts: 5
    template:  # Inline specification
      name: "main_menu"
      path: "templates/ui/menu/main.png"
      threshold: 0.9
      region:  # Optional: search only in this area of the frame
        x1: 100
        y1: 200
        x2: 500
        y2: 600
    actions:
      - action: Click
        x: 200
        y: 300
```

## Template Fields

### Required Fields

- **name**: Identifier for the template (used in logs and errors)
- **path**: Path to the PNG file
  - For **inline templates**: Full path like `"templates/ui/menu/main.png"`
  - For **registry templates**: Relative path like `"ui/Main.png"` (combined with registry base path)

### Optional Fields

- **threshold** (float64): Matching threshold (0.0-1.0)
  - Higher = more strict matching
  - Default: 0.8
  - Example: `0.95` for very precise matching

- **region** (object): Search region within the captured frame
  - Limits where to search for the template in the frame
  - **Frame coordinates** (not template coordinates!)
  - Example: Search only in top-left quadrant of screen
  - Fields:
    - `x1`: Left edge (pixels from left)
    - `y1`: Top edge (pixels from top)
    - `x2`: Right edge (pixels from left)
    - `y2`: Bottom edge (pixels from top)

- **scale** (float64): Scale factor for template matching
  - Default: 1.0 (no scaling)
  - Example: `1.5` to match template 1.5x larger

## Region vs SearchRegion

**Important**: The `region` field in a template defines **where to search in the frame**, NOT a crop of the template itself.

### What Region Does

```yaml
template:
  name: "button"
  path: "templates/ui/button.png"
  region:
    x1: 100  # Search starts 100px from left of frame
    y1: 200  # Search starts 200px from top of frame
    x2: 500  # Search ends 500px from left of frame
    y2: 600  # Search ends 600px from top of frame
```

This tells the CV service:
- Capture the full frame (entire window)
- **Only search within the rectangle (100,200) to (500,600)** for this template
- Ignore matches outside this region

### Common Use Cases

**Search in Top Half Only**:
```yaml
region:
  x1: 0
  y1: 0
  x2: 800     # Assuming 800px wide window
  y2: 300     # Only top 300px
```

**Search in Bottom-Right Quadrant**:
```yaml
region:
  x1: 400     # Right half (assuming 800px wide)
  y1: 300     # Bottom half (assuming 600px tall)
  x2: 800
  y2: 600
```

**Search in Specific UI Area** (e.g., inventory):
```yaml
region:
  x1: 50
  y1: 100
  x2: 350
  y2: 550
```

## Path Handling

### Registry Templates

When using `template_name`, the path comes from the registry:

```yaml
# templates/ui_templates.yaml
templates:
  - name: Main
    path: ui/Main.png  # Relative to registry base path
```

The registry is initialized with a base path:
```go
registry := templates.NewTemplateRegistry("templates")
```

Final path: `templates/ui/Main.png`

### Inline Templates

When using inline `template`, specify the full path:

```yaml
template:
  name: "custom"
  path: "templates/ui/menu/custom.png"  # Full path
```

The action automatically handles path normalization:
1. Removes `"templates/"` prefix if present
2. Removes `.png` extension if present
3. Passes normalized path to CV service

So these are equivalent:
- `"templates/ui/menu/custom.png"` → `"ui/menu/custom"`
- `"ui/menu/custom.png"` → `"ui/menu/custom"`
- `"ui/menu/custom"` → `"ui/menu/custom"`

## Validation

### Template Name Validation

When using `template_name`, the action validates:
1. **Build-time** (when loading routine):
   - Template exists in registry (if registry provided)
   - Returns clear error if not found
2. **Runtime** (when executing):
   - Template still exists in registry
   - Returns error if removed after loading

### Inline Template Validation

When using inline `template`, the action validates:
1. **Build-time**:
   - Name is not empty
   - Path is not empty
   - Cannot specify both `template` and `template_name`
2. **Runtime**:
   - Template file exists on disk
   - Template image loads successfully

## Actions Supporting Inline Templates

### WhileTemplateExists

Execute nested actions repeatedly while template is found in frame.

**With template_name**:
```yaml
- action: WhileTemplateExists
  max_attempts: 10
  template_name: "ClaimButton"
  actions:
    - action: Click
      x: 140
      y: 400
```

**With inline template**:
```yaml
- action: WhileTemplateExists
  max_attempts: 10
  template:
    name: "claim_button"
    path: "templates/ui/claim.png"
    threshold: 0.85
    region:
      x1: 100
      y1: 300
      x2: 200
      y2: 450
  actions:
    - action: Click
      x: 140
      y: 400
```

### FindImage

Find a template in the current frame.

**With template_name**:
```yaml
- action: FindImage
  template_name: "ErrorDialog"
```

**With inline template**:
```yaml
- action: FindImage
  template:
    name: "error_dialog"
    path: "templates/ui/errors/dialog.png"
    threshold: 0.9
```

## Caching Behavior

### Template Name Caching

When using `template_name`, benefits from registry's image cache:

```yaml
# Template with preload flag in registry
- name: Main
  path: ui/Main.png
  preload: true  # Loaded at startup

# Action uses cached image
- action: WhileTemplateExists
  template_name: "Main"  # <1ms lookup from cache
```

### Inline Template Caching

When using inline `template`, caching depends on CV service:

1. **First use**: Loads from disk (~10-20ms)
2. **Subsequent uses**: Service-level cache (<1ms)
3. **No registry cache**: Cannot use preload or unload_after modes

**Recommendation**: Use `template_name` for frequently used templates to benefit from registry caching.

## Best Practices

### Use Template Names for Reusable Templates

```yaml
# ✅ GOOD - Reusable templates in registry
templates:
  - name: OK
    path: ui/OK.png
    preload: true

# In routines
- action: WhileTemplateExists
  template_name: "OK"
```

### Use Inline Templates for One-Off Cases

```yaml
# ✅ GOOD - One-time special case
- action: WhileTemplateExists
  template:
    name: "special_event_banner"
    path: "templates/events/2024/winter/banner.png"
    threshold: 0.95
```

### Define Regions for Ambiguous Templates

```yaml
# ✅ GOOD - Region prevents false positives
template:
  name: "back_button"
  path: "templates/ui/back.png"
  region:  # Only search in top-left corner
    x1: 0
    y1: 0
    x2: 200
    y2: 100
```

### Use Higher Thresholds for Precise Matching

```yaml
# ✅ GOOD - High threshold for exact matches
template:
  name: "gold_coin_100"
  path: "templates/currency/gold_100.png"
  threshold: 0.95  # Very strict to avoid similar icons
```

## Error Messages

### Build-Time Errors

**Missing template in registry**:
```
routine 'DailyRewards' step 3 validation failed:
  template 'NonExistent' not found in registry
```

**Both template and template_name specified**:
```
routine 'DailyRewards' step 5 validation failed:
  cannot specify both 'template' and 'template_name', use one or the other
```

**Neither specified**:
```
routine 'DailyRewards' step 7 validation failed:
  must specify either 'template' or 'template_name'
```

### Runtime Errors

**Template removed from registry**:
```
Error: template 'ButtonName' not found in registry
```

**Template file not found** (inline):
```
Error: failed to open template file: templates/ui/missing.png: no such file or directory
```

**Template not found in frame**:
```
Error: template main_menu not found (confidence: 0.65, threshold: 0.90)
```

## Migration Guide

### From Hardcoded to Registry

**Before** (hardcoded in code):
```go
template := cv.Template{
    Name:      "OK",
    Path:      "templates/ui/OK.png",
    Threshold: 0.8,
}
```

**After** (registry):
```yaml
# templates/ui_templates.yaml
templates:
  - name: OK
    path: ui/OK.png
    threshold: 0.8
    preload: true
```

```yaml
# routines/daily.yaml
- action: WhileTemplateExists
  template_name: "OK"
```

### From Inline to Registry

**Before** (inline in every routine):
```yaml
# routine1.yaml
- action: WhileTemplateExists
  template:
    name: "OK"
    path: "templates/ui/OK.png"
    threshold: 0.8

# routine2.yaml
- action: WhileTemplateExists
  template:
    name: "OK"
    path: "templates/ui/OK.png"
    threshold: 0.8
```

**After** (centralized in registry):
```yaml
# templates/ui_templates.yaml
templates:
  - name: OK
    path: ui/OK.png
    threshold: 0.8
    preload: true

# routine1.yaml and routine2.yaml
- action: WhileTemplateExists
  template_name: "OK"
```

**Benefits**:
- Change threshold once instead of in every routine
- Preload image at startup for better performance
- Build-time validation ensures template exists
- Easier to maintain and update

## Summary

**Template Names** (Recommended):
- ✅ Build-time validation
- ✅ Image caching (preload/on-demand/unload-after)
- ✅ Centralized configuration
- ✅ Performance optimized
- ❌ Requires template registry setup

**Inline Templates**:
- ✅ Self-contained routines
- ✅ Quick prototyping
- ✅ One-off special cases
- ❌ No build-time validation
- ❌ Limited caching (service-level only)
- ❌ Duplicated configuration

Choose **template names** for production routines and **inline templates** for prototyping or special cases.
