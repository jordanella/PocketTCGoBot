# Actions Package - YAML-Based Routine System

This package provides a robust, type-safe system for building executable bot routines from YAML files with support for recursive action composition.

## Architecture Overview

### Core Components

1. **ActionStep Interface** ([yaml.go:3-8](yaml.go#L3-L8))
   - `Validate(ab *ActionBuilder) error` - Validates the action configuration
   - `Build(ab *ActionBuilder) *ActionBuilder` - Builds the executable Step and appends it to the builder

2. **ActionBuilder** ([builder.go:11-30](builder.go#L11-L30))
   - Holds the bot reference and accumulated executable steps
   - Provides fluent API for configuration (timeout, retries, error handling)
   - Executes steps sequentially with context cancellation support

3. **Step Struct** ([builder.go:32-38](builder.go#L32-L38))
   - Internal executable representation of an action
   - Contains the execute function, recovery logic, and validation errors

4. **Action Registry** ([registry.go:11-18](registry.go#L11-L18))
   - Maps YAML action names to Go types via reflection
   - Enables polymorphic unmarshaling of ActionStep interfaces

## How Recursive Building Works

### Build Once, Execute Many

**Key Design Principle**: Routines are built once (without a bot) and can be executed on multiple bots.

This separation allows you to:
- Build routines at startup
- Execute the same routine across a pool of bots
- Avoid rebuilding the action tree for each bot

### 1. YAML Parsing

When you load a routine from YAML using `RoutineLoader.LoadFromFile()`:

```go
// Build the routine once (no bot required at build time)
loader := NewRoutineLoader()
routine, err := loader.LoadFromFile("routines/my_routine.yaml")
if err != nil {
    // Error includes step numbers and validation context
    log.Fatal(err)
}

// Execute on multiple bots
for _, bot := range botPool {
    err = routine.Execute(bot)
    if err != nil {
        log.Printf("Bot %s failed: %v", bot.ID(), err)
    }
}
```

The unmarshaling process ([routine.go:20-67](routine.go#L20-L67)):
1. Reads raw YAML into `map[string]interface{}`
2. Looks up each action's type in the registry
3. Creates concrete instances using reflection
4. Unmarshals YAML fields into the concrete struct

### 2. Validation Phase

After unmarshaling, each action is validated **recursively** ([routine_loader.go:41-46](routine_loader.go#L41-L46)):

```go
for i, action := range routine.Steps {
    if err := action.Validate(ab); err != nil {
        return nil, fmt.Errorf("routine step %d validation failed: %w", i+1, err)
    }
    ab = action.Build(ab)
}
```

For composite actions like `WhileTemplateExists`, validation descends into nested actions ([while_template_exists.go:27-32](while_template_exists.go#L27-L32)):

```go
for i, action := range a.Actions {
    if err := action.Validate(ab); err != nil {
        return fmt.Errorf("WhileTemplateExists(%s) -> nested action %d: %w",
            a.Template.Name, i+1, err)
    }
}
```

This creates error chains like:
```
routine 'MyRoutine' step 3 validation failed:
  WhileTemplateExists(popup_close) -> nested action 2:
    Click: coordinates (x=-10, y=20) must be non-negative
```

### 3. Build Phase

The `Build()` method transforms validated ActionSteps into executable Steps.

**For simple actions** like [Click](click.go#L17-L27):
```go
func (a *Click) Build(ab *ActionBuilder) *ActionBuilder {
    step := Step{
        name: "Click",
        execute: func(bot BotInterface) error {
            // Bot is provided at execution time
            return bot.ADB().Click(a.X, a.Y)
        },
        issue: a.Validate(ab), // Captures validation error if any
    }
    ab.steps = append(ab.steps, step)
    return ab
}
```

Note how the `execute` function captures `a.X` and `a.Y` from the build phase, but receives the `bot` at execution time.

**For composite actions** like [WhileTemplateExists](while_template_exists.go#L37-L76):
```go
func (a *WhileTemplateExists) Build(ab *ActionBuilder) *ActionBuilder {
    // 1. Recursively build nested actions into executable steps (at build time)
    nestedSteps := ab.buildSteps(a.Actions)

    step := Step{
        name: fmt.Sprintf("WhileTemplateExists(%s)", a.Template.Name),
        execute: func(bot BotInterface) error {
            attempt := 0
            for {
                // Check loop condition
                if a.MaxAttempts > 0 && attempt >= a.MaxAttempts {
                    return fmt.Errorf("template still exists after %d attempts", a.MaxAttempts)
                }

                // Exit if template no longer exists (uses bot at runtime)
                result, err := bot.CV().FindTemplate(a.Template.Path, &cv.MatchConfig{
                    Threshold: a.Template.Threshold,
                })
                if err != nil {
                    return fmt.Errorf("error checking template existence: %w", err)
                }
                if !result.Found {
                    return nil
                }

                // 2. Execute the pre-built nested steps (passes bot at runtime)
                subBuilder := &ActionBuilder{
                    steps: nestedSteps,
                }
                if err := subBuilder.executeSteps(bot.Context(), bot); err != nil {
                    return fmt.Errorf("loop iteration %d failed: %w", attempt+1, err)
                }

                attempt++
                time.Sleep(100 * time.Millisecond)
            }
        },
        issue: a.Validate(ab),
    }
    ab.steps = append(ab.steps, step)
    return ab
}
```

The key helper method ([builder.go:223-234](builder.go#L223-L234)):
```go
func (ab *ActionBuilder) buildSteps(actions []ActionStep) []Step {
    // Create a temporary ActionBuilder (no bot needed)
    tempBuilder := NewActionBuilder()

    for _, action := range actions {
        action.Build(tempBuilder)
    }

    return tempBuilder.steps
}
```

This shows the power of the build/execute separation:
- `nestedSteps` are built once during the Build phase
- The execute function captures `nestedSteps` in its closure
- Each loop iteration executes those same steps with the runtime bot

### 4. Execution Phase

When `ab.Execute(bot)` is called ([builder.go:120-135](builder.go#L120-L135)):
1. Receives the bot to execute on
2. Creates a context from the bot with optional timeout
3. Optionally starts error monitoring goroutine
4. Executes steps sequentially ([builder.go:145-163](builder.go#L145-L163))
5. Checks for build-time validation errors before executing each step
6. Passes the bot to each step's execute function
7. Respects context cancellation between steps

**Key advantages of this model**:
- **Reusability**: Build once, execute on any bot
- **Performance**: No need to rebuild the action tree for each execution
- **Concurrency**: Same routine can execute on multiple bots simultaneously
- **Testing**: Easy to test routines with mock bots

## Creating New Actions

### 1. Define the struct with YAML tags

```go
// internal/actions/my_action.go
package actions

type MyAction struct {
    SomeField  string `yaml:"some_field"`
    SomeNumber int    `yaml:"some_number"`
}
```

### 2. Implement the ActionStep interface

```go
func (a *MyAction) Validate(ab *ActionBuilder) error {
    if a.SomeNumber < 0 {
        return fmt.Errorf("some_number must be non-negative")
    }
    if a.SomeField == "" {
        return fmt.Errorf("some_field cannot be empty")
    }
    return nil
}

func (a *MyAction) Build(ab *ActionBuilder) *ActionBuilder {
    step := Step{
        name: "MyAction",
        execute: func(bot BotInterface) error {
            // Bot is provided at execution time
            // a.SomeField and a.SomeNumber are captured from build time
            return bot.DoSomething(a.SomeField, a.SomeNumber)
        },
        issue: a.Validate(ab),
    }
    ab.steps = append(ab.steps, step)
    return ab
}
```

### 3. Register the action

Add it to [registry.go](registry.go):
```go
var actionRegistry = map[string]reflect.Type{
    "Click":               reflect.TypeOf(Click{}),
    "WhileTemplateExists": reflect.TypeOf(WhileTemplateExists{}),
    "MyAction":            reflect.TypeOf(MyAction{}), // Add this line
}
```

### 4. Use it in YAML

```yaml
routine_name: "Test My Action"
steps:
  - action: MyAction
    some_field: "test"
    some_number: 42
```

## Creating Composite Actions (with nested actions)

For actions that contain other actions (like loops, conditionals, etc.):

```go
type MyCompositeAction struct {
    MaxIterations int          `yaml:"max_iterations"`
    NestedActions []ActionStep `yaml:"actions"`
}

func (a *MyCompositeAction) Validate(ab *ActionBuilder) error {
    if a.MaxIterations <= 0 {
        return fmt.Errorf("max_iterations must be positive")
    }

    // Validate nested actions with error path tracking
    for i, action := range a.NestedActions {
        if err := action.Validate(ab); err != nil {
            return fmt.Errorf("MyCompositeAction -> nested action %d: %w", i+1, err)
        }
    }

    return nil
}

func (a *MyCompositeAction) Build(ab *ActionBuilder) *ActionBuilder {
    // Build nested actions into executable steps (at build time)
    nestedSteps := ab.buildSteps(a.NestedActions)

    step := Step{
        name: "MyCompositeAction",
        execute: func(bot BotInterface) error {
            for i := 0; i < a.MaxIterations; i++ {
                // Execute nested steps with the runtime bot
                subBuilder := &ActionBuilder{
                    steps: nestedSteps,
                }
                if err := subBuilder.executeSteps(bot.Context(), bot); err != nil {
                    return fmt.Errorf("iteration %d failed: %w", i+1, err)
                }
            }
            return nil
        },
        issue: a.Validate(ab),
    }
    ab.steps = append(ab.steps, step)
    return ab
}
```

## Error Handling and Validation

The system provides detailed error messages that show the full path through nested actions:

```
Error: failed to load routine 'CollectDailyRewards':
  routine 'CollectDailyRewards' step 5 validation failed:
    WhileTemplateExists(claim_button) -> nested action 3:
      WhileTemplateExists(popup) -> nested action 1:
        Click: coordinates (x=-100, y=50) must be non-negative
```

This makes it easy to:
1. Identify which YAML step failed (step 5)
2. See the nesting hierarchy (outer loop -> inner loop)
3. Understand the specific validation error

## Example YAML Routine

See [example_routine.yaml](example_routine.yaml) for a complete example showing:
- Simple actions
- Nested loops
- Multiple levels of nesting
- Proper YAML structure

## Testing

When testing actions:
1. Write unit tests for `Validate()` to ensure it catches all error cases
2. Write integration tests for `Build()` to ensure steps execute correctly
3. Test nested structures to ensure recursive building works
4. Verify error messages provide clear context

## Migration from Fluent API

If you have existing fluent-style routines:

```go
// Old fluent style (bot required at build time)
ab := NewActionBuilder(bot)
ab.Click(100, 200).
   Sleep(1 * time.Second).
   Click(300, 400).
   Execute()
```

With the new architecture:

```go
// New style - build once, execute many
routine := NewActionBuilder()
routine.Click(100, 200).
   Sleep(1 * time.Second).
   Click(300, 400)

// Execute on multiple bots
for _, bot := range botPool {
    err := routine.Execute(bot)
    if err != nil {
        log.Printf("Bot %s failed: %v", bot.ID(), err)
    }
}
```

Or convert to YAML for external configuration:

```yaml
routine_name: "MyRoutine"
steps:
  - action: Click
    x: 100
    y: 200
  - action: Sleep
    duration: 1s
  - action: Click
    x: 300
    y: 400
```

Then load and execute:

```go
loader := NewRoutineLoader()
routine, _ := loader.LoadFromFile("routines/my_routine.yaml")

// Execute on multiple bots
for _, bot := range botPool {
    routine.Execute(bot)
}
```

Both approaches work - use YAML for external configuration and fluent API for programmatic construction.
