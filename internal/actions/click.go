package actions

import "fmt"

type Click struct {
	X int `yaml:"x"`
	Y int `yaml:"y"`
}

func (a *Click) Validate(ab *ActionBuilder) error {
	if a.X < 0 || a.Y < 0 {
		return fmt.Errorf("coordinates (x=%d, y=%d) must be non-negative", a.X, a.Y)
	}
	return nil
}

func (a *Click) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "Click",
		execute: func(bot BotInterface) error {
			return bot.ADB().Click(a.X, a.Y)
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
