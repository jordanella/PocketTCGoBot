package actions

import (
	"fmt"
	"time"
)

type Sleep struct {
	Duration int `yaml:"duration"`
}

func (a *Sleep) Validate(ab *ActionBuilder) error {
	if a.Duration <= 0 {
		return fmt.Errorf("duration (%d) must be greater than 0", a.Duration)
	}
	return nil
}

func (a *Sleep) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "Sleep",
		execute: func(bot BotInterface) error {
			time.Sleep(time.Duration(a.Duration) * time.Millisecond)
			return nil
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}
