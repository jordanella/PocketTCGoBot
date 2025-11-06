package actions

type SendKey struct {
	Key string `yaml:"key"`
}

func (a *SendKey) Validate(ab *ActionBuilder) error {
	return nil
}

func (a *SendKey) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "Send Key",
		execute: func(bot BotInterface) error {
			return bot.ADB().SendKey(a.Key)
		},
	}
	ab.steps = append(ab.steps, step)
	return ab
}
