package actions

import (
	"context"

	"jordanella.com/pocket-tcg-go/internal/adb"
	"jordanella.com/pocket-tcg-go/internal/cv"
	"jordanella.com/pocket-tcg-go/internal/monitor"
)

// ActionsConfig contains timing configuration for actions
type ActionsConfig interface {
	GetDelayBetweenActions() int // milliseconds
	GetScreenshotDelay() int     // milliseconds
}

// ConfigInterface provides access to timing configuration
type ConfigInterface interface {
	Actions() ActionsConfig
}

// BotInterface defines the capabilities that actions need from the bot
// This breaks the circular dependency by allowing actions to depend on an interface
// instead of the concrete bot.Bot type
type BotInterface interface {
	// Access to core services
	ADB() *adb.Controller
	CV() *cv.Service
	ErrorMonitor() *monitor.ErrorMonitor
	Config() ConfigInterface
	Templates() TemplateRegistryInterface
	Routines() RoutineRegistryInterface
	RoutineController() RoutineControllerInterface
	Variables() VariableStoreInterface

	// Context management
	Context() context.Context

	// State queries (add as needed)
	IsPaused() bool
	IsStopped() bool
	Instance() int

	// Account management
	Manager() interface{} // Returns bot.ManagerInterface
	GetCurrentAccount() interface{} // Returns *bot.Account
	InjectAccount(account interface{}) error // Takes *bot.Account
	ClearCurrentAccount()

	// Add other methods that actions need to call on the bot
}

// TemplateRegistryInterface defines the interface for template lookup
// This allows actions to reference templates by name from YAML scripts
type TemplateRegistryInterface interface {
	Get(name string) (cv.Template, bool)
	MustGet(name string) cv.Template
	Has(name string) bool
}

// RoutineRegistryInterface defines the interface for routine lookup
type RoutineRegistryInterface interface {
	Get(name string) (*ActionBuilder, error)
	GetWithSentries(name string) (*ActionBuilder, []Sentry, error)
	Has(name string) bool
	Reload() error
	ListAvailable() []string
	GetMetadata(filename string) interface{}
	GetValidationError(filename string) error
}

// RoutineControllerInterface defines the interface for routine state control
// This allows sentries to pause/resume the main routine execution
type RoutineControllerInterface interface {
	IsRunning() bool
	IsPaused() bool
	IsStopped() bool
	Pause() bool
	Resume() bool
	ForceStop() bool
	CheckPauseOrStop() bool
	Reset()
	SetRunning()
	SetCompleted()
	SetIdle()
	GetState() interface{} // Returns the current state (RoutineExecutionState)
}

// VariableStoreInterface defines the interface for runtime variable storage
// Variables are stored as strings and can be used in conditions and actions
type VariableStoreInterface interface {
	Set(name string, value string)
	Get(name string) (string, bool)
	Has(name string) bool
	Delete(name string)
	Clear()
	GetAll() map[string]string
}
