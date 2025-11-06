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

	// Context management
	Context() context.Context

	// State queries (add as needed)
	IsPaused() bool
	IsStopped() bool

	// Add other methods that actions need to call on the bot
}
