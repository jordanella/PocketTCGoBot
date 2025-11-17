package actions

import "fmt"

// LaunchApp launches the Pokemon TCG Pocket app
type LaunchApp struct {
	// Optional custom package name (defaults to Pokemon TCG Pocket)
	Package string `yaml:"package,omitempty"`
	// Optional custom activity (defaults to main activity)
	Activity string `yaml:"activity,omitempty"`
}

const (
	// Default package name for Pokemon TCG Pocket
	defaultPocketTCGPackage = "jp.pokemon.pokemontcgp"
	// Default main activity for Pokemon TCG Pocket
	defaultPocketTCGActivity = "jp.pokemon.pokemontcgp.startup.MainActivity"
)

func (a *LaunchApp) Validate(ab *ActionBuilder) error {
	// Validation is optional - defaults will be used if not specified
	return nil
}

func (a *LaunchApp) Build(ab *ActionBuilder) *ActionBuilder {
	// Set defaults if not provided
	packageName := a.Package
	if packageName == "" {
		packageName = defaultPocketTCGPackage
	}

	activity := a.Activity
	if activity == "" {
		activity = defaultPocketTCGActivity
	}

	step := Step{
		name: fmt.Sprintf("LaunchApp (%s)", packageName),
		execute: func(bot BotInterface) error {
			return bot.ADB().StartApp(packageName, activity)
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
