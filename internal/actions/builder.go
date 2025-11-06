package actions

import (
	"context"
	"time"
)

// ActionBuilder type and core methods
type ActionBuilder struct {
	bot          BotInterface // Reference to bot
	steps        []Step
	timeout      time.Duration
	retries      int
	ignoreErrors bool
	ctx          context.Context
}

type Step struct {
	name         string
	execute      func() error
	recover      func(error) error
	timeout      time.Duration
	canInterrupt bool
}

// Builder configuration methods

func (ab *ActionBuilder) WithTimeout(d time.Duration) *ActionBuilder {
	ab.timeout = d
	return ab
}

func (ab *ActionBuilder) WithRetries(n int) *ActionBuilder {
	ab.retries = n
	return ab
}

func (ab *ActionBuilder) IgnoreErrors() *ActionBuilder {
	if ab.steps[len(ab.steps)-1].recover == nil {
		ab.steps[len(ab.steps)-1].recover = func(error) error { return nil }
	}
	ab.ignoreErrors = true
	return ab
}

func (ab *ActionBuilder) Interruptible() *ActionBuilder {
	// This would be a step-level property, not builder-level
	// For now, just return ab
	ab.steps[len(ab.steps)-1].canInterrupt = true
	return ab
}

// Execution

func (ab *ActionBuilder) Execute() error {
	ctx := context.Background()
	if ab.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, ab.timeout)
		defer cancel()
	}
	return ab.executeSteps(ctx)
}

func (ab *ActionBuilder) ExecuteOnce() error {
	ctx := context.Background()
	return ab.executeSteps(ctx)
}

// Internal

func (ab *ActionBuilder) executeSteps(ctx context.Context) error {
	for _, step := range ab.steps {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := step.execute(); err != nil {
			if !ab.ignoreErrors {
				return err
			}
		}
	}
	return nil
}

func (ab *ActionBuilder) shouldRetry(err error) bool {
	err = ab.steps[len(ab.steps)-1].recover(err)
	return err != nil
}
