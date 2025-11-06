# Template Registry System

Dynamic template loading system for PocketTCG Bot that allows templates to be defined in YAML files and referenced by name in action scripts.

## Overview

The template registry system provides:
- **YAML-based template definitions** - Define all templates in external configuration files
- **Dynamic loading** - Load templates at runtime without recompiling
- **Name-based lookup** - Reference templates by name in action scripts
- **Type safety** - Compile-time checks with runtime validation
- **Thread-safe** - Concurrent access from multiple bots
- **Backward compatibility** - Existing hardcoded templates still work

## Architecture

```
templates/
├── ui_templates.yaml       # UI element templates
├── cards_templates.yaml    # Card detection templates
└── ...other template files

pkg/templates/
├── registry.go            # Template registry implementation
├── templates.go           # (Legacy) Hardcoded templates
└── README.md             # This file

Actions use templates by:
1. Looking up by name: template_name: "OK"
2. Direct specification: template: { name: "OK", path: "...", threshold: 0.8 }
```

## Quick Start

### 1. Define Templates in YAML

Create `templates/ui_templates.yaml`:

```yaml
templates:
  - name: OK
    path: ui/OK.png
    threshold: 0.8

  - name: Confirm
    path: ui/Confirm.png
    threshold: 0.8
    region:
      x1: 110
      y1: 350
      x2: 150
      y2: 404

  - name: Main
    path: ui/Main.png
    threshold: 0.9
    region:
      x1: 120
      y1: 316
      x2: 143
      y2: 335
```

### 2. Load Templates at Startup

```go
package main

import (
    "log"
    "jordanella.com/pocket-tcg-go/pkg/templates"
)

func main() {
    // Initialize global registry with base path
    registry := templates.InitializeGlobalRegistry("templates")

    // Load all template files from directory
    if err := registry.LoadFromDirectory("templates"); err != nil {
        log.Fatal(err)
    }

    log.Printf("Loaded %d templates", registry.Count())

    // Or load specific files
    // if err := registry.LoadFromFile("templates/ui_templates.yaml"); err != nil {
    //     log.Fatal(err)
    // }
}
```

### 3. Use Templates in Action Scripts

**YAML routine using template names:**

```yaml
routine_name: "Daily Rewards"
steps:
  - action: WhileTemplateExists
    max_attempts: 10
    template_name: "Claim"  # Look up from registry
    actions:
      - action: Click
        x: 140
        y: 400

  - action: WhileTemplateExists
    max_attempts: 5
    template_name: "OK"  # Another registry lookup
    actions:
      - action: Click
        x: 140
        y: 400
```

**Or specify template directly (not recommended):**

```yaml
steps:
  - action: WhileTemplateExists
    max_attempts: 10
    template:
      name: "Claim"
      path: "ui/Claim.png"
      threshold: 0.8
    actions:
      - action: Click
        x: 140
        y: 400
```

## Template Registry API

### Creating a Registry

```go
// Create a new registry
registry := templates.NewTemplateRegistry("templates")  // base path for images

// Or use the global singleton
registry := templates.GlobalRegistry()
```

### Loading Templates

```go
// Load from a single file
err := registry.LoadFromFile("templates/ui_templates.yaml")

// Load all YAML files from a directory
err := registry.LoadFromDirectory("templates")

// Load into global registry (convenience functions)
err := templates.LoadFromDirectory("templates")
```

### Retrieving Templates

```go
// Get template (returns template, bool)
tmpl, ok := registry.Get("OK")
if !ok {
    log.Fatal("Template not found")
}

// MustGet panics if not found (use during init)
tmpl := registry.MustGet("OK")

// GetOrDefault returns a basic template if not found
tmpl := registry.GetOrDefault("OK", 0.8)

// Check if template exists
if registry.Has("OK") {
    // ...
}
```

### Managing Templates

```go
// Register template programmatically
registry.Register(cv.Template{
    Name:      "CustomTemplate",
    Path:      "custom/template.png",
    Threshold: 0.85,
})

// Register multiple
registry.RegisterBatch([]cv.Template{...})

// List all template names
names := registry.List()

// Remove a template
removed := registry.Remove("OK")

// Clear all templates
registry.Clear()
```

## YAML Template Structure

```yaml
templates:
  - name: TemplateName          # Required: unique identifier
    path: relative/path.png      # Required: path from base directory
    threshold: 0.8               # Optional: match confidence (default: 0.8)
    region:                      # Optional: search region
      x1: 100
      y1: 200
      x2: 150
      y2: 250
    scale: 1.0                   # Optional: scale factor
```

### Field Reference

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `name` | string | Yes | - | Unique identifier for template lookup |
| `path` | string | Yes | - | Path to image file (relative to base path) |
| `threshold` | float64 | No | 0.8 | Match confidence threshold (0.0-1.0) |
| `region` | object | No | nil | Search region to improve performance |
| `region.x1` | int | No | - | Top-left X coordinate |
| `region.y1` | int | No | - | Top-left Y coordinate |
| `region.x2` | int | No | - | Bottom-right X coordinate |
| `region.y2` | int | No | - | Bottom-right Y coordinate |
| `scale` | float64 | No | 0.0 | Scale factor for template matching |

## Migration from Hardcoded Templates

### Step 1: Extract Template Definitions

**Before** ([templates.go](templates.go)):
```go
Main = cv.Template{
    Name: "Main",
    Region: &cv.Region{X1: 120, Y1: 316, X2: 143, Y2: 335},
}
OK = cv.Template{Name: "OK"}
Confirm = cv.Template{
    Name: "Confirm",
    Region: &cv.Region{X1: 110, Y1: 350, X2: 150, Y2: 404},
}
```

**After** (templates/ui_templates.yaml):
```yaml
templates:
  - name: Main
    path: ui/Main.png
    threshold: 0.8
    region:
      x1: 120
      y1: 316
      x2: 143
      y2: 335

  - name: OK
    path: ui/OK.png
    threshold: 0.8

  - name: Confirm
    path: ui/Confirm.png
    threshold: 0.8
    region:
      x1: 110
      y1: 350
      x2: 150
      y2: 404
```

### Step 2: Add Path Information

Hardcoded templates don't include `Path`. You need to add this:

```yaml
templates:
  - name: Main
    path: ui/Main.png  # Add the actual image path
    # ... rest of fields
```

The path is relative to the base path specified when creating the registry.

### Step 3: Update Action Scripts

**Before**:
```go
// Programmatic usage
ab.UntilTemplateDisappears(templates.Main, ...)
```

**After** (YAML):
```yaml
steps:
  - action: WhileTemplateExists
    template_name: "Main"  # Use template registry
    actions:
      - # ...
```

### Step 4: Load Templates at Startup

Add template loading to your bot initialization:

```go
func initializeBot() *bot.Bot {
    // Initialize template registry
    registry := templates.InitializeGlobalRegistry("templates")
    if err := registry.LoadFromDirectory("templates"); err != nil {
        log.Fatalf("Failed to load templates: %v", err)
    }

    // Create bot with registry
    b := bot.New(config)
    // ... rest of initialization

    return b
}
```

### Step 5: Implement Templates() Method

Your Bot type needs to implement the `TemplateRegistryInterface`:

```go
type Bot struct {
    // ... other fields
    templateRegistry *templates.TemplateRegistry
}

func (b *Bot) Templates() actions.TemplateRegistryInterface {
    return b.templateRegistry
}
```

## Organizing Template Files

### Recommended Structure

```
templates/
├── ui/                      # UI element images
│   ├── Main.png
│   ├── OK.png
│   ├── Confirm.png
│   └── ...
├── cards/                   # Card images
│   ├── rare/
│   ├── common/
│   └── ...
├── ui_templates.yaml        # UI template definitions
├── cards_templates.yaml     # Card template definitions
└── missions_templates.yaml  # Mission template definitions
```

### Splitting Templates by Category

**ui_templates.yaml**:
```yaml
templates:
  - name: Main
    path: ui/Main.png
  # ... other UI templates
```

**cards_templates.yaml**:
```yaml
templates:
  - name: PikachuRare
    path: cards/rare/pikachu.png
    threshold: 0.9
  # ... other card templates
```

Load all at startup:
```go
registry.LoadFromDirectory("templates")  // Loads all .yaml files
```

## Best Practices

### 1. Use Registry Lookup in YAML

**Preferred:**
```yaml
- action: WhileTemplateExists
  template_name: "OK"  # Lookup by name
```

**Avoid:**
```yaml
- action: WhileTemplateExists
  template:            # Direct specification
    name: "OK"
    path: "ui/OK.png"
    threshold: 0.8
```

### 2. Set Appropriate Thresholds

- **0.95-1.0**: Exact matches only (pixel-perfect)
- **0.85-0.95**: High confidence (recommended for UI elements)
- **0.75-0.85**: Medium confidence (for variable content)
- **0.6-0.75**: Low confidence (use with caution)

### 3. Use Regions to Improve Performance

```yaml
- name: SkipButton
  path: ui/Skip.png
  threshold: 0.85
  region:  # Only search in bottom-right corner
    x1: 200
    y1: 450
    x2: 280
    y2: 530
```

### 4. Name Templates Descriptively

```yaml
# Good
- name: MissionClaimButton
- name: PackOpenConfirm
- name: FriendRequestSend

# Bad
- name: Button1
- name: Btn
- name: Template
```

### 5. Group Related Templates

```yaml
templates:
  # Navigation
  - name: Main
  - name: Menu
  - name: Home

  # Buttons
  - name: OK
  - name: Confirm
  - name: Skip

  # Missions
  - name: Claim
  - name: ClaimAll
```

## Thread Safety

The TemplateRegistry is thread-safe and can be safely accessed from multiple goroutines:

```go
// Safe to call from multiple bots concurrently
template1, _ := registry.Get("OK")
template2, _ := registry.Get("Confirm")
```

## Error Handling

### Loading Errors

```go
if err := registry.LoadFromFile("templates/ui.yaml"); err != nil {
    log.Printf("Failed to load templates: %v", err)
    // Handle error appropriately
}
```

Common errors:
- File not found
- Invalid YAML syntax
- Missing required fields (name, path)
- Duplicate template names (last one wins)

### Runtime Errors

```go
// In action execution
template, ok := bot.Templates().Get("NonExistent")
if !ok {
    return fmt.Errorf("template 'NonExistent' not found in registry")
}
```

## Testing

### Mock Registry for Tests

```go
func TestMyAction(t *testing.T) {
    registry := templates.NewTemplateRegistry("testdata/templates")
    registry.Register(cv.Template{
        Name:      "TestTemplate",
        Path:      "test.png",
        Threshold: 0.8,
    })

    // Use registry in test
    tmpl, ok := registry.Get("TestTemplate")
    assert.True(t, ok)
    assert.Equal(t, "TestTemplate", tmpl.Name)
}
```

## Performance Considerations

1. **Load Once**: Load templates at startup, not on every bot creation
2. **Use Regions**: Define search regions to reduce CV processing time
3. **Cache Results**: The registry uses a map for O(1) lookups
4. **Concurrent Safe**: Multiple bots can share one registry

## Troubleshooting

### Template Not Found

```
Error: template 'OK' not found in registry
```

Solutions:
- Verify template is defined in YAML file
- Check YAML file was loaded (`registry.LoadFromDirectory()`)
- Verify template name spelling
- Check `registry.List()` to see loaded templates

### Wrong Template Matched

Solutions:
- Increase threshold value
- Define a more specific search region
- Use a more unique template image
- Check template image quality

### Path Not Found

```
Error: failed to read template file
```

Solutions:
- Verify base path is correct
- Check relative paths in YAML
- Ensure image files exist
- Use forward slashes in paths (even on Windows)
