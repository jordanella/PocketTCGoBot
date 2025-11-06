package actions

import (
	"context"

	"jordanella.com/pocket-tcg-go/internal/adb"
	"jordanella.com/pocket-tcg-go/internal/cv"
)

// BotInterface defines the capabilities that actions need from the bot
// This breaks the circular dependency by allowing actions to depend on an interface
// instead of the concrete bot.Bot type
type BotInterface interface {
	// Access to core services
	ADB() *adb.Controller
	CV() *cv.Service

	// Context management
	Context() context.Context

	// State queries (add as needed)
	IsPaused() bool
	IsStopped() bool

	// Add other methods that actions need to call on the bot
}
