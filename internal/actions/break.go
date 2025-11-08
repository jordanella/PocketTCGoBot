package actions

// BreakLoop is a special error type that signals a loop should terminate early
type BreakLoop struct {
	Message string
}

func (e *BreakLoop) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "break loop"
}

// Break action terminates the innermost loop early
// This can be used within While, Until, or Repeat loops
type Break struct {
	// No configuration needed - just signals to break
}

func (a *Break) Validate(ab *ActionBuilder) error {
	// Break has no configuration to validate
	return nil
}

func (a *Break) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "Break",
		execute: func(bot BotInterface) error {
			// Return the special BreakLoop error to signal loop termination
			return &BreakLoop{Message: "loop terminated by Break action"}
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
