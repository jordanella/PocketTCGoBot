package actions

type Input struct {
	Text string `yaml:"text"`
}

func (a *Input) Validate(ab *ActionBuilder) error {
	return nil
}

func (a *Input) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "Input",
		execute: func(bot BotInterface) error {
			return bot.ADB().Input(a.Text)
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}
