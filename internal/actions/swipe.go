package actions

import (
	"fmt"
)

type Swipe struct {
	X1       int `yaml:"x1"`
	Y1       int `yaml:"y1"`
	X2       int `yaml:"x2"`
	Y2       int `yaml:"y2"`
	Duration int `yaml:"duration"`
}

func (a *Swipe) Validate(ab *ActionBuilder) error {
	if a.X1 < 0 || a.Y1 < 0 || a.X2 < 0 || a.Y2 < 0 {
		return fmt.Errorf("coordinates (x1=%d, y1=%d, x2=%d, y2=%d) must be non-negative", a.X1, a.Y1, a.X2, a.Y2)
	}
	if a.Duration <= 0 {
		return fmt.Errorf("duration (%d) must be greater than 0", a.Duration)
	}
	return nil
}

func (a *Swipe) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "Swipe",
		execute: func(bot BotInterface) error {
			return bot.ADB().Swipe(a.X1, a.Y1, a.X2, a.Y2, a.Duration)
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
