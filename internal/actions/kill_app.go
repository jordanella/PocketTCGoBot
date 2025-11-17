package actions

import "fmt"

// KillApp force-stops the Pokemon TCG Pocket app
type KillApp struct {
	// Optional custom package name (defaults to Pokemon TCG Pocket)
	Package string `yaml:"package,omitempty"`
}

func (a *KillApp) Validate(ab *ActionBuilder) error {
	// Validation is optional - defaults will be used if not specified
	return nil
}

func (a *KillApp) Build(ab *ActionBuilder) *ActionBuilder {
	// Set default if not provided
	packageName := a.Package
	if packageName == "" {
		packageName = defaultPocketTCGPackage
	}

	step := Step{
		name: fmt.Sprintf("KillApp (%s)", packageName),
		execute: func(bot BotInterface) error {
			return bot.ADB().ForceStop(packageName)
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
