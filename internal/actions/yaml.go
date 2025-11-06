package actions

type ActionStep interface {
	// Run chains the action onto the ActionBuilder and returns the updated builder.
	// It requires the BotInterface and the executor function for recursion in loops.
	Validate(ab *ActionBuilder) error
	Build(ab *ActionBuilder) *ActionBuilder
}

type YAMLAction struct {
}

type YAMLStep struct {
}

type YAMLRoutine struct {
	RoutineName string     `yaml:"routine_name"`
	Steps       []YAMLStep `yaml:"steps"` // Steps now holds the wrapper struct
}

type YAMLTemplate string
