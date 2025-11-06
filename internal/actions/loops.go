package actions

import (
	"fmt"
	"time"

	"jordanella.com/pocket-tcg-go/internal/cv"
)

// Loop control structures for ActionBuilder

// ============================================================================
// Boolean Logic Helpers
// ============================================================================

// Any returns true if any of the conditions are true (OR logic)
func Any(conditions ...func() bool) func() bool {
	return func() bool {
		for _, cond := range conditions {
			if cond() {
				return true
			}
		}
		return false
	}
}

// All returns true if all conditions are true (AND logic)
func All(conditions ...func() bool) func() bool {
	return func() bool {
		for _, cond := range conditions {
			if !cond() {
				return false
			}
		}
		return true
	}
}

// Not inverts a condition
func Not(condition func() bool) func() bool {
	return func() bool {
		return !condition()
	}
}

// ============================================================================
// Conditional Execution
// ============================================================================

// If executes actions conditionally based on a boolean condition
func (ab *ActionBuilder) If(condition func() bool, then func(*ActionBuilder)) *ActionBuilder {
	step := Step{
		name: "If",
		execute: func() error {
			if condition() {
				subBuilder := &ActionBuilder{
					bot: ab.bot,
					ctx: ab.ctx,
				}
				then(subBuilder)
				return subBuilder.Execute()
			}
			return nil
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// IfElse executes one of two action branches based on condition
func (ab *ActionBuilder) IfElse(condition func() bool, then func(*ActionBuilder), otherwise func(*ActionBuilder)) *ActionBuilder {
	step := Step{
		name: "IfElse",
		execute: func() error {
			subBuilder := &ActionBuilder{
				bot: ab.bot,
				ctx: ab.ctx,
			}

			if condition() {
				then(subBuilder)
			} else {
				otherwise(subBuilder)
			}

			return subBuilder.Execute()
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// ============================================================================
// Basic Loops
// ============================================================================

// Repeat executes actions a fixed number of times
func (ab *ActionBuilder) Repeat(times int, action func(*ActionBuilder)) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("Repeat(%d)", times),
		execute: func() error {
			for i := 0; i < times; i++ {
				subBuilder := &ActionBuilder{
					bot: ab.bot,
					ctx: ab.ctx,
				}
				action(subBuilder)
				if err := subBuilder.Execute(); err != nil {
					return fmt.Errorf("repeat iteration %d failed: %w", i+1, err)
				}
			}
			return nil
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// Until loops until a condition becomes true
func (ab *ActionBuilder) Until(condition func() bool, action func(*ActionBuilder), maxAttempts int) *ActionBuilder {
	step := Step{
		name: "Until",
		execute: func() error {
			attempt := 0
			for {
				if maxAttempts > 0 && attempt >= maxAttempts {
					return fmt.Errorf("until loop exceeded %d attempts", maxAttempts)
				}

				// Check condition first
				if condition() {
					return nil
				}

				// Execute action
				subBuilder := &ActionBuilder{
					bot: ab.bot,
					ctx: ab.ctx,
				}
				action(subBuilder)
				if err := subBuilder.Execute(); err != nil {
					return fmt.Errorf("until loop iteration %d failed: %w", attempt+1, err)
				}

				attempt++
				time.Sleep(100 * time.Millisecond)
			}
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// While loops while a condition remains true
func (ab *ActionBuilder) While(condition func() bool, action func(*ActionBuilder), maxAttempts int) *ActionBuilder {
	step := Step{
		name: "While",
		execute: func() error {
			attempt := 0
			for {
				if maxAttempts > 0 && attempt >= maxAttempts {
					return fmt.Errorf("while loop exceeded %d attempts", maxAttempts)
				}

				// Check condition - exit if false
				if !condition() {
					return nil
				}

				// Execute action
				subBuilder := &ActionBuilder{
					bot: ab.bot,
					ctx: ab.ctx,
				}
				action(subBuilder)
				if err := subBuilder.Execute(); err != nil {
					return fmt.Errorf("while loop iteration %d failed: %w", attempt+1, err)
				}

				attempt++
				time.Sleep(100 * time.Millisecond)
			}
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// ============================================================================
// Template-Based Loops (Common Patterns)
// ============================================================================

// UntilAnyTemplateRun loops until any of the specified templates appears, using a pre-built ActionBuilder
func (ab *ActionBuilder) UntilAnyTemplate(templates []cv.Template, actions *ActionBuilder, maxAttempts int) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("UntilAnyTemplate(%d templates)", len(templates)),
		execute: func() error {
			attempt := 0
			for {
				if maxAttempts > 0 && attempt >= maxAttempts {
					return fmt.Errorf("template not found after %d attempts", maxAttempts)
				}

				// Check if any template exists
				ab.bot.CV().InvalidateCache()
				for _, tmpl := range templates {
					if ab.templateExists(tmpl) {
						return nil // Found one!
					}
				}

				// Re-execute the action builder's steps
				subBuilder := &ActionBuilder{
					bot:   ab.bot,
					ctx:   ab.ctx,
					steps: actions.steps,
				}
				if err := subBuilder.Execute(); err != nil {
					return fmt.Errorf("loop iteration %d failed: %w", attempt+1, err)
				}

				attempt++
				time.Sleep(100 * time.Millisecond)
			}
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// WhileTemplateExistsRun loops while a template exists, using a pre-built ActionBuilder
func (ab *ActionBuilder) WhileTemplateExists(template cv.Template, actions *ActionBuilder, maxAttempts int) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("WhileTemplateExists(%s)", template.Name),
		execute: func() error {
			attempt := 0
			for {
				if maxAttempts > 0 && attempt >= maxAttempts {
					return fmt.Errorf("template still exists after %d attempts", maxAttempts)
				}

				ab.bot.CV().InvalidateCache()

				// Exit if template no longer exists
				if !ab.templateExists(template) {
					return nil
				}

				// Re-execute the action builder's steps
				subBuilder := &ActionBuilder{
					bot:   ab.bot,
					ctx:   ab.ctx,
					steps: actions.steps,
				}
				if err := subBuilder.Execute(); err != nil {
					return fmt.Errorf("loop iteration %d failed: %w", attempt+1, err)
				}

				attempt++
				time.Sleep(100 * time.Millisecond)
			}
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// UntilTemplateDisappearsRun loops until a template disappears, using a pre-built ActionBuilder
func (ab *ActionBuilder) UntilTemplateDisappears(template cv.Template, actions *ActionBuilder, maxAttempts int) *ActionBuilder {
	return ab.WhileTemplateExists(template, actions, maxAttempts)
}

// ============================================================================
// Advanced Loop Control
// ============================================================================

// LoopUntil provides a simple loop that exits when condition returns true
// This is useful for polling scenarios where you need to check multiple conditions
func (ab *ActionBuilder) LoopUntil(condition func() bool, maxAttempts int, pollInterval time.Duration) *ActionBuilder {
	step := Step{
		name: "LoopUntil",
		execute: func() error {
			if pollInterval == 0 {
				pollInterval = 100 * time.Millisecond
			}

			attempt := 0
			for {
				if maxAttempts > 0 && attempt >= maxAttempts {
					return fmt.Errorf("loop exceeded %d attempts", maxAttempts)
				}

				if condition() {
					return nil
				}

				attempt++
				time.Sleep(pollInterval)
			}
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// ============================================================================
// Helper Methods
// ============================================================================

// TemplateExists checks if a template exists (helper for conditions)
func (ab *ActionBuilder) TemplateExists(template cv.Template) bool {
	return ab.templateExists(template)
}

// templateExists is the internal implementation
func (ab *ActionBuilder) templateExists(template cv.Template) bool {
	threshold := template.Threshold
	if threshold == 0 {
		threshold = 0.8
	}

	config := &cv.MatchConfig{
		Method:    cv.MatchMethodSSD,
		Threshold: threshold,
	}

	templatePath := buildTemplatePath(template.Name)
	result, err := ab.bot.CV().FindTemplate(templatePath, config)
	return err == nil && result.Found
}
