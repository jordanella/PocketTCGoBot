package bot

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"jordanella.com/pocket-tcg-go/internal/accountpool"
	"jordanella.com/pocket-tcg-go/internal/actions"
	"jordanella.com/pocket-tcg-go/internal/adb"
	"jordanella.com/pocket-tcg-go/internal/cv"
	"jordanella.com/pocket-tcg-go/internal/database"
	"jordanella.com/pocket-tcg-go/internal/emulator"
	"jordanella.com/pocket-tcg-go/internal/monitor"
	"jordanella.com/pocket-tcg-go/pkg/templates"
)

// ActionLibrary is an interface to avoid circular dependency with actions package
type ActionLibrary interface {
	GoHome() error
	GoToMissions() error
	LevelUp() error
	EraseInput() error
	// Add other action methods as needed
}

// RecoveryAction defines the type of recovery action to take
type RecoveryAction string

const (
	RecoveryActionNone         RecoveryAction = "none"          // Do nothing
	RecoveryActionLog          RecoveryAction = "log"           // Log only
	RecoveryActionPause        RecoveryAction = "pause"         // Pause the bot
	RecoveryActionRestart      RecoveryAction = "restart"       // Restart the routine
	RecoveryActionReconnectADB RecoveryAction = "reconnect_adb" // Attempt to reconnect ADB
	RecoveryActionRestartApp   RecoveryAction = "restart_app"   // Restart the target app
	RecoveryActionStop         RecoveryAction = "stop"          // Stop the bot
)

// RecoveryConfig defines recovery behavior for different health issues
type RecoveryConfig struct {
	ADBConnectionLost     RecoveryAction
	InstanceWindowMissing RecoveryAction
	DeviceUnresponsive    RecoveryAction
	ScreenFrozen          RecoveryAction
	BotStuck              RecoveryAction
	MaxRecoveryAttempts   int
}

// DefaultRecoveryConfig returns sensible defaults
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		ADBConnectionLost:     RecoveryActionReconnectADB,
		InstanceWindowMissing: RecoveryActionStop,
		DeviceUnresponsive:    RecoveryActionRestartApp,
		ScreenFrozen:          RecoveryActionRestartApp,
		BotStuck:              RecoveryActionRestart,
		MaxRecoveryAttempts:   3,
	}
}

// Core Bot struct definition
type Bot struct {
	instance          int
	adb               *adb.Controller
	cv                *cv.Service
	config            *Config
	state             *State
	actions           ActionLibrary
	emulatorManager   *emulator.Manager
	screenHistory     *ScreenHistory
	errorMonitor      *monitor.ErrorMonitor
	healthCheck       *monitor.HealthChecker
	db                *database.DB
	templateRegistry  actions.TemplateRegistryInterface
	routineRegistry   actions.RoutineRegistryInterface
	routineController *RoutineController
	variableStore     actions.VariableStoreInterface
	sentryManager     *actions.SentryManager // Global sentry lifecycle manager
	orchestrationID   string
	lastRoutineName   string // Track last executed routine for restart
	restartPolicy     *RestartPolicy
	recoveryConfig    RecoveryConfig       // Recovery behavior configuration
	recoveryAttempts  map[string]int       // Track recovery attempts per issue type
	onUnhealthyAction func()               // Callback when unhealthy event occurs
	manager           *Manager             // Reference to parent manager (optional)
	currentAccount    *accountpool.Account // Currently assigned account (nil if none)
	ctx               context.Context
	cancel            context.CancelFunc
}

// Lifecycle methods
func New(instance int, config *Config) (*Bot, error) {
	ctx, cancel := context.WithCancel(context.Background())

	return &Bot{
		instance:          instance,
		config:            config,
		state:             &State{},
		screenHistory:     NewScreenHistory(50), // Track last 50 screen states
		routineController: NewRoutineController(),
		variableStore:     actions.NewVariableStore(),
		recoveryConfig:    DefaultRecoveryConfig(),
		recoveryAttempts:  make(map[string]int),
		ctx:               ctx,
		cancel:            cancel,
	}, nil
}

func (b *Bot) Initialize() error {
	return b.initializeInternal(false)
}

// InitializeWithSharedRegistries initializes the bot assuming registries are already injected
func (b *Bot) InitializeWithSharedRegistries() error {
	return b.initializeInternal(true)
}

func (b *Bot) initializeInternal(sharedRegistries bool) error {
	// Find ADB - use explicit path if set, otherwise search
	var adbPath string
	var err error

	if b.config.ADBPath != "" {
		// Use explicit ADB path from config
		adbPath = b.config.ADBPath
	} else {
		// Search for ADB in MuMu folder
		adbPath, err = adb.FindADB(b.config.FolderPath)
		if err != nil {
			return fmt.Errorf("failed to find ADB (set ADBPath in config or ensure MuMu is installed): %w", err)
		}
	}

	// Create emulator manager
	b.emulatorManager = emulator.NewManager(b.config.FolderPath, adbPath)

	// Discover instances
	if err := b.emulatorManager.DiscoverInstances(); err != nil {
		return fmt.Errorf("failed to discover instances: %w", err)
	}

	// Position windows if configured
	if b.config.Columns > 0 {
		windowConfig := emulator.NewWindowConfig(
			b.config.Columns,
			b.config.RowGap,
			getScaleParam(b.config.DefaultLanguage),
			b.config.SelectedMonitor,
		)
		if err := b.emulatorManager.PositionAllInstances(windowConfig); err != nil {
			// Non-fatal: just log warning
			fmt.Printf("Warning: Failed to position windows: %v\n", err)
		}
	}

	// Connect ADB to this bot's instance
	if err := b.emulatorManager.ConnectInstance(b.instance); err != nil {
		return fmt.Errorf("failed to connect ADB to instance %d: %w", b.instance, err)
	}

	// Get the connected instance
	inst, err := b.emulatorManager.GetInstance(b.instance)
	if err != nil {
		return fmt.Errorf("failed to get instance %d: %w", b.instance, err)
	}
	b.adb = inst.ADB

	// Apply configuration defaults
	b.config.ApplyDefaults()

	// Set up coordinate translator for ADB controller
	coordConfig := b.config.GetCoordinateTranslationConfig()
	translator := NewCoordinateTranslator(coordConfig)
	if err := translator.Validate(); err != nil {
		fmt.Printf("Warning: Coordinate translator validation failed: %v (using defaults)\n", err)
	} else {
		b.adb.SetCoordinateTranslator(translator)
		fmt.Printf("Bot %d: %s\n", b.instance, translator.String())
	}

	// Initialize CV service with window capture
	windowCapture, err := cv.NewWindowCapture(inst.MuMu.WindowHandle)
	if err != nil {
		return fmt.Errorf("failed to create window capture: %w", err)
	}

	// Use title bar height from config
	titleBarHeight := b.config.TitleBarHeight

	b.cv = cv.NewServiceWithTitleBar(windowCapture, titleBarHeight)

	// Initialize database
	dbPath := filepath.Join(b.config.FolderPath, "bot.db")
	db, err := database.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	b.db = db

	// Run database migrations
	if err := b.db.RunMigrations(); err != nil {
		b.db.Close()
		return fmt.Errorf("failed to run database migrations: %w", err)
	}

	// Initialize error monitor
	b.errorMonitor = monitor.NewErrorMonitor(b)
	b.errorMonitor.Start()

	// Initialize health checker with callback
	b.healthCheck = monitor.NewHealthChecker(b).
		WithCheckInterval(10 * time.Second).
		WithUnhealthyCallback(func(reason string, err error) {
			fmt.Printf("Bot %d: Health check failed - %s: %v\n", b.instance, reason, err)

			// Execute recovery action based on reason
			b.executeRecoveryAction(reason, err)

			// Trigger custom unhealthy action if configured
			if b.onUnhealthyAction != nil {
				b.onUnhealthyAction()
			}
		})
	b.healthCheck.Start()

	// Initialize registries only if not using shared ones
	if !sharedRegistries {
		// Initialize template registry (from current directory)
		templatesPath := "templates"
		b.templateRegistry = templates.NewTemplateRegistry(templatesPath)
		// Load templates from YAML files if directory exists
		templatesConfigPath := filepath.Join("config", "templates")
		if err := b.templateRegistry.(*templates.TemplateRegistry).LoadFromDirectory(templatesConfigPath); err != nil {
			// Non-fatal: templates directory might not exist or be empty
			fmt.Printf("Info: Template directory not loaded: %v\n", err)
		}

		// Initialize routine registry (from current directory)
		routinesPath := "routines"
		b.routineRegistry = actions.NewRoutineRegistry(routinesPath)
		b.routineRegistry.(*actions.RoutineRegistry).WithTemplateRegistry(b.templateRegistry)
	}

	// Initialize global sentry manager (always initialized, regardless of registry source)
	// Note: This must be done after all other initialization since SentryManager needs access to bot services
	// Create a temporary interface-compatible wrapper if needed
	var botInterface actions.BotInterface = b
	b.sentryManager = actions.NewSentryManager(botInterface)
	fmt.Printf("Bot %d: Sentry manager initialized\n", b.instance)

	return nil
}

// getScaleParam returns the window width based on UI scale setting
func getScaleParam(language string) int {
	// Scale125 uses 287px, Scale100 uses 277px
	if language == "Scale125" {
		return 287
	}
	return 277
}

func (b *Bot) Run() error {
	// Main bot loop
	// TODO: Implement runCycle
	return nil
}

func (b *Bot) Shutdown() {
	b.shutdownInternal(false)
}

// ShutdownWithSharedRegistries shuts down the bot without clearing shared registries
func (b *Bot) ShutdownWithSharedRegistries() {
	b.shutdownInternal(true)
}

func (b *Bot) shutdownInternal(sharedRegistries bool) {
	// Stop all sentries first
	if b.sentryManager != nil {
		b.sentryManager.StopAll()
	}

	// Stop error monitor
	if b.errorMonitor != nil {
		b.errorMonitor.Stop()
	}

	// Stop health checker
	if b.healthCheck != nil {
		b.healthCheck.Stop()
	}

	// Only clean up registries if not using shared ones
	if !sharedRegistries {
		// Unload all cached template images
		if b.templateRegistry != nil {
			if registry, ok := b.templateRegistry.(*templates.TemplateRegistry); ok {
				registry.UnloadAll()
			}
		}
		// Note: Routines are eagerly loaded and don't need per-bot cleanup
	}

	// Close database connection
	if b.db != nil {
		b.db.Close()
		b.db = nil
	}

	// Disconnect this instance's ADB (emulator manager handles the actual disconnection)
	if b.emulatorManager != nil {
		b.emulatorManager.DisconnectInstance(b.instance)
	}

	if b.cancel != nil {
		b.cancel()
	}
}

// Pause/Resume/Stop - now delegated to RoutineController
func (b *Bot) Pause() {
	if b.routineController != nil {
		b.routineController.Pause()
	}
}

func (b *Bot) Resume() {
	if b.routineController != nil {
		b.routineController.Resume()
	}
}

func (b *Bot) Stop() {
	if b.routineController != nil {
		b.routineController.ForceStop()
	}
}

// RoutineController returns the routine execution controller
func (b *Bot) RoutineController() actions.RoutineControllerInterface {
	return b.routineController
}

// State returns the bot's current state
func (b *Bot) State() *State {
	return b.state
}

// SetActions allows dependency injection of the actions library
// This breaks the circular dependency by allowing actions to be set after bot creation
func (b *Bot) SetActions(actions ActionLibrary) {
	b.actions = actions
}

// BotInterface implementation - these methods allow actions package to access bot capabilities
func (b *Bot) ADB() *adb.Controller {
	return b.adb
}

func (b *Bot) CV() *cv.Service {
	return b.cv
}

func (b *Bot) Context() context.Context {
	return b.ctx
}

func (b *Bot) IsPaused() bool {
	if b.routineController == nil {
		return false
	}
	return b.routineController.IsPaused()
}

func (b *Bot) IsStopped() bool {
	if b.routineController == nil {
		return false
	}
	return b.routineController.IsStopped()
}

// EmulatorManager returns the emulator manager for multi-instance coordination
func (b *Bot) EmulatorManager() *emulator.Manager {
	return b.emulatorManager
}

// ScreenHistory returns the screen history tracker
func (b *Bot) ScreenHistory() *ScreenHistory {
	return b.screenHistory
}

// DB returns the database connection
func (b *Bot) DB() *database.DB {
	return b.db
}

// UpdateScreenHistory detects current screen and adds to history
func (b *Bot) UpdateScreenHistory() ScreenState {
	result := b.DetectCurrentScreenWithConfidence()
	b.screenHistory.Add(result)
	return result.Screen
}

// ErrorMonitor returns the error monitor for registering handlers or getting the error channel
func (b *Bot) ErrorMonitor() *monitor.ErrorMonitor {
	return b.errorMonitor
}

// Config returns the bot configuration (implements actions.ConfigInterface)
func (b *Bot) Config() actions.ConfigInterface {
	return configAdapter{b.config}
}

// Templates returns the template registry (implements actions.BotInterface)
func (b *Bot) Templates() actions.TemplateRegistryInterface {
	return b.templateRegistry
}

// Routines returns the routine registry (implements actions.BotInterface)
func (b *Bot) Routines() actions.RoutineRegistryInterface {
	return b.routineRegistry
}

// Variables returns the variable store (implements actions.BotInterface)
func (b *Bot) Variables() actions.VariableStoreInterface {
	return b.variableStore
}

// GetAllVariables returns a snapshot of all variables (thread-safe)
func (b *Bot) GetAllVariables() map[string]string {
	return b.variableStore.GetAll()
}

// SentryManager returns the global sentry manager (implements actions.BotInterface)
func (b *Bot) SentryManager() *actions.SentryManager {
	return b.sentryManager
}

// SetLastRoutine sets the name of the last executed routine
func (b *Bot) SetLastRoutine(routineName string) {
	b.lastRoutineName = routineName
}

// GetLastRoutine returns the name of the last executed routine
func (b *Bot) GetLastRoutine() string {
	return b.lastRoutineName
}

// Instance returns the bot instance number
func (b *Bot) Instance() int {
	return b.instance
}

// SetRestartPolicy configures the restart policy for this bot
func (b *Bot) SetRestartPolicy(policy *RestartPolicy) {
	b.restartPolicy = policy
}

// GetRestartPolicy returns the configured restart policy
func (b *Bot) GetRestartPolicy() *RestartPolicy {
	if b.restartPolicy == nil {
		// Return default policy if not configured
		defaultPolicy := DefaultRestartPolicy()
		return &defaultPolicy
	}
	return b.restartPolicy
}

// SetUnhealthyAction sets the callback to execute when bot becomes unhealthy
func (b *Bot) SetUnhealthyAction(action func()) {
	b.onUnhealthyAction = action
}

// Manager returns the parent manager (may be nil if bot created standalone)
// Manager returns the bot's manager (nil if none)
// Returns interface{} to avoid circular dependency with actions package
func (b *Bot) Manager() interface{} {
	return b.manager
}

// GetCurrentAccount returns the currently assigned account (nil if none)
// Returns interface{} to avoid circular dependency with accountpool package
func (b *Bot) GetCurrentAccount() interface{} {
	return b.currentAccount
}

// InjectAccount performs account injection via ADB
// This is called by the InjectNextAccount action
// Takes interface{} to avoid circular dependency, expects *accountpool.Account
func (b *Bot) InjectAccount(accountIf interface{}) error {
	// Type assert to concrete Account type
	account, ok := accountIf.(*accountpool.Account)
	if !ok {
		return fmt.Errorf("InjectAccount expects *accountpool.Account, got %T", accountIf)
	}

	if account == nil {
		return fmt.Errorf("cannot inject nil account")
	}

	if b.adb == nil {
		return fmt.Errorf("ADB not initialized")
	}

	// Get package data directory
	packageName := "jp.pokemon.pokemontcgp"
	dataPath := fmt.Sprintf("/data/data/%s/shared_prefs", packageName)

	// Target file path on device
	targetFile := fmt.Sprintf("%s/account.xml", dataPath)

	fmt.Printf("Bot %d: Injecting account '%s' from %s\n", b.instance, account.ID, account.XMLPath)

	// Push XML file to device
	if err := b.adb.Push(account.XMLPath, targetFile); err != nil {
		return fmt.Errorf("failed to push account XML: %w", err)
	}

	// Set permissions (readable by app)
	if _, err := b.adb.Shell(fmt.Sprintf("chmod 660 %s", targetFile)); err != nil {
		fmt.Printf("Warning: Failed to set permissions on %s: %v\n", targetFile, err)
	}

	// Store current account reference
	b.currentAccount = account

	fmt.Printf("Bot %d: Account '%s' injected successfully\n", b.instance, account.ID)
	return nil
}

// ClearCurrentAccount clears the current account assignment
func (b *Bot) ClearCurrentAccount() {
	b.currentAccount = nil
}

// configAdapter wraps *Config to implement actions.ConfigInterface
type configAdapter struct {
	*Config
}

// Actions returns the actions configuration as the interface type
func (ca configAdapter) Actions() actions.ActionsConfig {
	return ca.Config.Actions()
}

// Ensure configAdapter implements ConfigInterface at compile time
var _ actions.ConfigInterface = configAdapter{}

// executeRecoveryAction handles automatic recovery based on health check failures
func (b *Bot) executeRecoveryAction(reason string, _ error) {
	// Map reason to recovery action
	var action RecoveryAction
	switch reason {
	case "adb_connection_lost":
		action = b.recoveryConfig.ADBConnectionLost
	case "instance_window_missing":
		action = b.recoveryConfig.InstanceWindowMissing
	case "device_unresponsive":
		action = b.recoveryConfig.DeviceUnresponsive
	case "screen_frozen":
		action = b.recoveryConfig.ScreenFrozen
	case "bot_stuck":
		action = b.recoveryConfig.BotStuck
	default:
		fmt.Printf("Bot %d: Unknown health issue '%s', defaulting to log\n", b.instance, reason)
		action = RecoveryActionLog
	}

	// Track recovery attempts
	b.recoveryAttempts[reason]++
	attemptCount := b.recoveryAttempts[reason]

	// Check if max attempts exceeded
	if attemptCount > b.recoveryConfig.MaxRecoveryAttempts {
		fmt.Printf("Bot %d: Max recovery attempts (%d) exceeded for '%s', stopping bot\n",
			b.instance, b.recoveryConfig.MaxRecoveryAttempts, reason)
		b.Stop()
		return
	}

	fmt.Printf("Bot %d: Executing recovery action '%s' for '%s' (attempt %d/%d)\n",
		b.instance, action, reason, attemptCount, b.recoveryConfig.MaxRecoveryAttempts)

	// Execute the recovery action
	switch action {
	case RecoveryActionNone:
		// Do nothing
		return

	case RecoveryActionLog:
		// Already logged above
		return

	case RecoveryActionPause:
		b.Pause()

	case RecoveryActionRestart:
		// Restart the last executed routine
		if b.lastRoutineName != "" {
			fmt.Printf("Bot %d: Restarting routine '%s'\n", b.instance, b.lastRoutineName)
			// Stop current routine
			b.Stop()
			// The manager should handle restart via RestartBot()
		} else {
			fmt.Printf("Bot %d: Cannot restart - no last routine recorded\n", b.instance)
		}

	case RecoveryActionReconnectADB:
		// Attempt to reconnect ADB
		if b.emulatorManager != nil {
			fmt.Printf("Bot %d: Attempting to reconnect ADB\n", b.instance)
			// Disconnect and reconnect
			b.emulatorManager.DisconnectInstance(b.instance)
			if err := b.emulatorManager.ConnectInstance(b.instance); err != nil {
				fmt.Printf("Bot %d: Failed to reconnect ADB: %v\n", b.instance, err)
				b.Stop()
			} else {
				fmt.Printf("Bot %d: ADB reconnected successfully\n", b.instance)
				// Reset recovery attempts on success
				b.recoveryAttempts[reason] = 0
			}
		}

	case RecoveryActionRestartApp:
		// Restart the target app (Pokemon TCG Pocket)
		if b.adb != nil {
			packageName := "jp.pokemon.pokemontcgp" // Pokemon TCG Pocket package name
			fmt.Printf("Bot %d: Restarting app '%s'\n", b.instance, packageName)

			// Force stop the app
			if _, err := b.adb.Shell(fmt.Sprintf("am force-stop %s", packageName)); err != nil {
				fmt.Printf("Bot %d: Failed to stop app: %v\n", b.instance, err)
			}

			// Wait a moment
			time.Sleep(2 * time.Second)

			// Restart the app
			if _, err := b.adb.Shell(fmt.Sprintf("monkey -p %s -c android.intent.category.LAUNCHER 1", packageName)); err != nil {
				fmt.Printf("Bot %d: Failed to restart app: %v\n", b.instance, err)
				b.Stop()
			} else {
				fmt.Printf("Bot %d: App restarted successfully\n", b.instance)
				// Reset recovery attempts on success
				b.recoveryAttempts[reason] = 0
			}
		}

	case RecoveryActionStop:
		fmt.Printf("Bot %d: Stopping bot due to '%s'\n", b.instance, reason)
		b.Stop()

	default:
		fmt.Printf("Bot %d: Unknown recovery action '%s'\n", b.instance, action)
	}
}

func (b *Bot) OrchestrationID() string {
	return b.orchestrationID
}

// SetOrchestrationID sets the UUID of the bot group this bot belongs to
func (b *Bot) SetOrchestrationID(id string) {
	b.orchestrationID = id
}
