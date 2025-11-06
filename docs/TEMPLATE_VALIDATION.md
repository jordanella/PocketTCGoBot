# Build-Time Template Validation

The ActionBuilder now supports **build-time template validation** through an optional template registry reference. This catches missing templates during routine loading instead of at runtime.

## How It Works

### 1. Template Registry in ActionBuilder

The `ActionBuilder` has an optional `templateRegistry` field:

```go
type ActionBuilder struct {
    steps              []Step
    templateRegistry   TemplateRegistryInterface // Optional: for validating template names
    // ... other fields
}
```

### 2. Setting the Registry

When loading routines, provide the template registry for validation:

```go
// Load templates first
registry := templates.InitializeGlobalRegistry("templates")
registry.LoadFromDirectory("templates")

// Create loader with registry
loader := NewRoutineLoader().WithTemplateRegistry(registry)

// Load routine - validates templates exist
routine, err := loader.LoadFromFile("routines/daily_tasks.yaml")
if err != nil {
    // Error will include "template 'X' not found in registry" if template missing
    log.Fatal(err)
}
```

### 3. Automatic Propagation

The template registry automatically propagates to nested actions:

```go
func (ab *ActionBuilder) buildSteps(actions []ActionStep) []Step {
    tempBuilder := NewActionBuilder()
    tempBuilder.templateRegistry = ab.templateRegistry  // Propagate to nested builders

    for _, action := range actions {
        action.Build(tempBuilder)
    }

    return tempBuilder.steps
}
```

This ensures nested loops and composite actions can also validate templates.

## Validation in Actions

### WhileTemplateExists Example

```go
func (a *WhileTemplateExists) Validate(ab *ActionBuilder) error {
    // ... other validation

    // Validate template exists in registry (if using template_name and registry is available)
    if a.TemplateName != "" && ab.templateRegistry != nil {
        if !ab.templateRegistry.Has(a.TemplateName) {
            return fmt.Errorf("template '%s' not found in registry", a.TemplateName)
        }
    }

    // ... rest of validation
}
```

### Custom Action Example

```go
type FindImage struct {
    TemplateName string `yaml:"template_name"`
    Count        int    `yaml:"count"`
}

func (a *FindImage) Validate(ab *ActionBuilder) error {
    if a.Count <= 0 {
        return fmt.Errorf("count (%d) must be greater than 0", a.Count)
    }

    if a.TemplateName == "" {
        return fmt.Errorf("template_name cannot be empty")
    }

    // Validate template exists (if registry is available)
    if ab.templateRegistry != nil {
        if !ab.templateRegistry.Has(a.TemplateName) {
            return fmt.Errorf("template '%s' not found in registry", a.TemplateName)
        }
    }

    return nil
}
```

## Benefits

### 1. Early Error Detection

**Without build-time validation:**
```
Loading routine... OK
Starting bot 1... OK
Starting bot 2... OK
...
[5 minutes later]
Bot 3 ERROR: template 'ClaimButton' not found in registry
```

**With build-time validation:**
```
Loading routine... ERROR
  routine 'DailyRewards' step 3 validation failed:
    template 'ClaimButton' not found in registry
[Fix template before any bots start]
```

### 2. Better Error Messages

Validation provides the full error path:

```
routine 'CollectDailyRewards' step 5 validation failed:
  WhileTemplateExists(ClaimButton) -> nested action 2:
    template 'PopupClose' not found in registry
```

This tells you:
- Which routine failed
- Which step (step 5)
- The nesting structure (WhileTemplateExists -> nested action 2)
- The specific problem (template not found)

### 3. Fail Fast

Routines are validated completely before any execution begins:
- All template references are checked
- All nested actions are validated
- Configuration errors are caught immediately

## Usage Patterns

### Pattern 1: Global Registry (Recommended)

```go
func main() {
    // Initialize templates once at startup
    registry := templates.InitializeGlobalRegistry("templates")
    if err := registry.LoadFromDirectory("templates"); err != nil {
        log.Fatal(err)
    }

    // Create loader with registry
    loader := NewRoutineLoader().WithTemplateRegistry(registry)

    // Load routines - all templates validated
    dailyRoutine, err := loader.LoadFromFile("routines/daily.yaml")
    if err != nil {
        log.Fatal(err)
    }

    weeklyRoutine, err := loader.LoadFromFile("routines/weekly.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // Execute on bots - templates guaranteed to exist
    for _, bot := range botPool {
        go dailyRoutine.Execute(bot)
    }
}
```

### Pattern 2: Per-Bot Registry

```go
func createBot(id string) *Bot {
    // Each bot has its own template registry
    registry := templates.NewTemplateRegistry("templates")
    registry.LoadFromDirectory("templates")

    loader := NewRoutineLoader().WithTemplateRegistry(registry)
    routine, _ := loader.LoadFromFile("routines/bot.yaml")

    bot := &Bot{
        id:               id,
        templateRegistry: registry,
        routine:          routine,
    }

    return bot
}
```

### Pattern 3: Without Registry (Optional Validation)

```go
// No template registry - validation skipped
loader := NewRoutineLoader()  // No WithTemplateRegistry() call

// Templates will be looked up at runtime instead
routine, err := loader.LoadFromFile("routines/daily.yaml")

// Runtime error if template missing
err = routine.Execute(bot)
// ERROR: template 'X' not found in registry
```

## Implementation Checklist

When adding template validation to a custom action:

- [ ] Add `TemplateName` field with `yaml:"template_name"` tag
- [ ] Check `ab.templateRegistry != nil` before validation
- [ ] Use `ab.templateRegistry.Has(name)` to check existence
- [ ] Return clear error message with template name
- [ ] Document that template_name is preferred over inline templates

### Template Action Pattern

```go
type YourAction struct {
    TemplateName string       `yaml:"template_name,omitempty"` // Template lookup
    Template     cv.Template  `yaml:"template,omitempty"`      // Or inline spec
    // ... other fields
}

func (a *YourAction) Validate(ab *ActionBuilder) error {
    // Require one or the other
    if a.TemplateName == "" && a.Template.Name == "" {
        return fmt.Errorf("must specify either 'template_name' or 'template'")
    }

    // Validate registry lookup
    if a.TemplateName != "" && ab.templateRegistry != nil {
        if !ab.templateRegistry.Has(a.TemplateName) {
            return fmt.Errorf("template '%s' not found in registry", a.TemplateName)
        }
    }

    // ... other validation
    return nil
}

func (a *YourAction) Build(ab *ActionBuilder) *ActionBuilder {
    step := Step{
        name: "YourAction",
        execute: func(bot BotInterface) error {
            // Resolve template at runtime
            var template cv.Template
            if a.TemplateName != "" {
                var ok bool
                template, ok = bot.Templates().Get(a.TemplateName)
                if !ok {
                    return fmt.Errorf("template '%s' not found", a.TemplateName)
                }
            } else {
                template = a.Template
            }

            // Use template...
            return nil
        },
        issue: a.Validate(ab),
    }
    ab.steps = append(ab.steps, step)
    return ab
}
```

## Testing

### Unit Test with Mock Registry

```go
func TestActionValidation(t *testing.T) {
    // Create mock registry
    registry := templates.NewTemplateRegistry("testdata")
    registry.Register(cv.Template{
        Name:      "TestTemplate",
        Path:      "test.png",
        Threshold: 0.8,
    })

    // Create builder with registry
    ab := NewActionBuilder().WithTemplateRegistry(registry)

    // Test valid template
    action := &YourAction{TemplateName: "TestTemplate"}
    err := action.Validate(ab)
    assert.NoError(t, err)

    // Test invalid template
    action = &YourAction{TemplateName: "NonExistent"}
    err = action.Validate(ab)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "not found in registry")
}
```

### Integration Test

```go
func TestRoutineLoading(t *testing.T) {
    // Setup registry
    registry := templates.NewTemplateRegistry("templates")
    registry.LoadFromDirectory("templates")

    // Load routine with validation
    loader := NewRoutineLoader().WithTemplateRegistry(registry)
    routine, err := loader.LoadFromFile("routines/test.yaml")

    assert.NoError(t, err)
    assert.NotNil(t, routine)
}
```

## Summary

Build-time template validation provides:
- **Early error detection** - Find problems before execution
- **Better error messages** - Full context with nesting paths
- **Fail fast** - Validate entire routine tree at load time
- **Optional** - Works with or without registry
- **Automatic propagation** - Nested actions validated too

This significantly improves the debugging experience and prevents runtime failures due to missing templates.
