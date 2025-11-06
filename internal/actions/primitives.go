package actions

import (
	"time"

	"jordanella.com/pocket-tcg-go/internal/adb"
	"jordanella.com/pocket-tcg-go/internal/cv"
)

// Basic building blocks

func (ab *ActionBuilder) FindAndClick(tmpl cv.Template, x, y int) *ActionBuilder {
	// TODO: Implement
	return ab
}

func (ab *ActionBuilder) WaitFor(tmpl cv.Template, timeout time.Duration) *ActionBuilder {
	// TODO: Implement
	return ab
}

func (ab *ActionBuilder) Click(x, y int) *ActionBuilder {
	step := Step{
		name: "Click",
		execute: func() error {
			return ab.bot.ADB().Click(x, y)
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

func (ab *ActionBuilder) Swipe(params adb.SwipeParams) *ActionBuilder {
	step := Step{
		name: "Swipe",
		execute: func() error {
			return ab.bot.ADB().Swipe(params)
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

func (ab *ActionBuilder) Sleep(d time.Duration) *ActionBuilder {
	step := Step{
		name: "Sleep",
		execute: func() error {
			time.Sleep(d)
			return nil
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

func (ab *ActionBuilder) SendKey(key string) *ActionBuilder {
	step := Step{
		name: "SendKey",
		execute: func() error {
			return ab.bot.ADB().SendKey(key)
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

func (ab *ActionBuilder) Input(text string) *ActionBuilder {
	step := Step{
		name: "Input",
		execute: func() error {
			return ab.bot.ADB().Input(text)
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// Custom

func (ab *ActionBuilder) Do(fn func() error) *ActionBuilder {
	step := Step{
		name:    "Custom",
		execute: fn,
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// Steps allows you to define a sequence of actions inline
// The function is called immediately and adds steps to the current builder
// This makes it easier to group related actions together
func (ab *ActionBuilder) Steps(fn func(*ActionBuilder)) *ActionBuilder {
	// Call the function with the current builder to add steps inline
	fn(ab)
	return ab
}

// WithSteps is an alias for Until that reads more naturally when you want to repeat a sequence
// Usage: l.Action().WithSteps(templates.Shop, func(ab) { ... }, 45)
// This is just syntactic sugar over Until/UntilTemplateAppears
func (ab *ActionBuilder) WithSteps(template cv.Template, steps *ActionBuilder, maxAttempts int) *ActionBuilder {
	return ab.UntilTemplateAppears(template, steps, maxAttempts)
}
