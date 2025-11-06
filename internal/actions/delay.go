package actions

import (
	"fmt"
	"time"
)

type Delay struct {
	Count int `yaml:"count"`
}

func (a *Delay) Validate(ab *ActionBuilder) error {
	if a.Count <= 0 {
		return fmt.Errorf("count (%d) must be greater than 0", a.Count)
	}
	return nil
}

func (a *Delay) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "Delay",
		execute: func(bot BotInterface) error {
			delayMs := bot.Config().Actions().GetDelayBetweenActions()
			duration := time.Duration(delayMs*a.Count) * time.Millisecond
			time.Sleep(duration)
			return nil
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}
