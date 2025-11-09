package bot

import (
	"fmt"
	"math"
	"path/filepath"
	"sync"
	"time"

	"jordanella.com/pocket-tcg-go/internal/accountpool"
	"jordanella.com/pocket-tcg-go/internal/actions"
	"jordanella.com/pocket-tcg-go/pkg/templates"
)

// Manager coordinates multiple bot instances and manages shared resources
type Manager struct {
	mu               sync.RWMutex
	bots             map[int]*Bot
	config           *Config
	basePath         string // Base path for templates and routines
	templateRegistry actions.TemplateRegistryInterface
	routineRegistry  actions.RoutineRegistryInterface
	accountPool      accountpool.AccountPool // Shared account pool (optional)
}

// NewManager creates a new bot manager with shared registries
// Uses current directory as base path for templates and routines
// DEPRECATED: Use NewManagerWithRegistries for better control
func NewManager(config *Config) (*Manager, error) {
	return NewManagerWithBasePath(config, "")
}

// NewManagerWithBasePath creates a new bot manager with shared registries using a custom base path
// If basePath is empty, uses current directory
// DEPRECATED: Use NewManagerWithRegistries to share registries across multiple managers
func NewManagerWithBasePath(config *Config, basePath string) (*Manager, error) {
	// Initialize shared template registry
	templatesPath := filepath.Join(basePath, "templates")
	templateRegistry := templates.NewTemplateRegistry(templatesPath)

	// Load templates from YAML files if directory exists
	templatesConfigPath := filepath.Join(basePath, "templates", "registry")
	if err := templateRegistry.LoadFromDirectory(templatesConfigPath); err != nil {
		// Non-fatal: templates directory might not exist or be empty
		fmt.Printf("Info: Template directory not loaded: %v\n", err)
	}

	// Initialize shared routine registry
	routinesPath := filepath.Join(basePath, "routines")
	routineRegistry := actions.NewRoutineRegistry(routinesPath).WithTemplateRegistry(templateRegistry)

	return &Manager{
		bots:             make(map[int]*Bot),
		config:           config,
		basePath:         basePath,
		templateRegistry: templateRegistry,
		routineRegistry:  routineRegistry,
	}, nil
}

// NewManagerWithRegistries creates a new bot manager with externally provided registries
// This allows multiple managers to share the same template and routine registries
func NewManagerWithRegistries(
	config *Config,
	templateRegistry actions.TemplateRegistryInterface,
	routineRegistry actions.RoutineRegistryInterface,
) *Manager {
	return &Manager{
		bots:             make(map[int]*Bot),
		config:           config,
		templateRegistry: templateRegistry,
		routineRegistry:  routineRegistry,
	}
}

// CreateBot creates a new bot instance with shared registries
func (m *Manager) CreateBot(instance int) (*Bot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if bot already exists
	if _, exists := m.bots[instance]; exists {
		return nil, fmt.Errorf("bot instance %d already exists", instance)
	}

	// Create bot with shared config
	bot, err := New(instance, m.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot %d: %w", instance, err)
	}

	// Inject shared registries BEFORE initialization
	bot.templateRegistry = m.templateRegistry
	bot.routineRegistry = m.routineRegistry
	bot.manager = m // Set manager reference

	// Initialize the bot (this will skip registry initialization since they're already set)
	if err := bot.InitializeWithSharedRegistries(); err != nil {
		return nil, fmt.Errorf("failed to initialize bot %d: %w", instance, err)
	}

	m.bots[instance] = bot
	return bot, nil
}

// GetBot retrieves a bot instance by number
func (m *Manager) GetBot(instance int) (*Bot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bot, exists := m.bots[instance]
	return bot, exists
}

// ShutdownBot shuts down a specific bot instance
func (m *Manager) ShutdownBot(instance int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	bot, exists := m.bots[instance]
	if !exists {
		return fmt.Errorf("bot instance %d not found", instance)
	}

	// Shutdown the bot (this will NOT unload shared registries)
	bot.ShutdownWithSharedRegistries()

	delete(m.bots, instance)
	return nil
}

// RestartBot restarts a bot instance with its last executed routine
// Returns the last routine name that will be restarted, or error if bot doesn't exist or has no last routine
func (m *Manager) RestartBot(instance int) (string, error) {
	m.mu.RLock()
	bot, exists := m.bots[instance]
	m.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("bot instance %d not found", instance)
	}

	// Get the last routine name
	lastRoutine := bot.GetLastRoutine()
	if lastRoutine == "" {
		return "", fmt.Errorf("bot instance %d has no routine to restart", instance)
	}

	// Reset the routine controller to prepare for new execution
	bot.RoutineController().Reset()

	// Note: The actual routine execution must be triggered by the coordinator
	// This method only prepares the bot for restart
	return lastRoutine, nil
}

// ExecuteWithRestart executes a routine with auto-restart on failure
// Uses the provided RestartPolicy to determine retry behavior
func (m *Manager) ExecuteWithRestart(instance int, routineName string, policy RestartPolicy) error {
	m.mu.RLock()
	bot, exists := m.bots[instance]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("bot instance %d not found", instance)
	}

	// Track the routine name for restart capability
	bot.SetLastRoutine(routineName)

	// Get routine from registry
	routineBuilder, err := bot.Routines().Get(routineName)
	if err != nil {
		return fmt.Errorf("failed to get routine '%s': %w", routineName, err)
	}

	// If restart is not enabled, execute once and return
	if !policy.Enabled {
		return routineBuilder.Execute(bot)
	}

	// Execute with retry logic
	retryCount := 0
	currentDelay := policy.InitialDelay

	for {
		// Execute the routine
		err := routineBuilder.Execute(bot)

		// Success - reset retry counter if configured
		if err == nil {
			if policy.ResetOnSuccess && retryCount > 0 {
				fmt.Printf("Bot %d: Routine '%s' succeeded after %d retries\n", instance, routineName, retryCount)
			}
			return nil
		}

		// Check if we've exceeded max retries
		if policy.MaxRetries > 0 && retryCount >= policy.MaxRetries {
			return fmt.Errorf("bot %d routine '%s' failed after %d retries: %w", instance, routineName, retryCount, err)
		}

		// Log retry attempt
		retryCount++
		fmt.Printf("Bot %d: Routine '%s' failed (attempt %d/%d): %v. Retrying in %v...\n",
			instance, routineName, retryCount, policy.MaxRetries, err, currentDelay)

		// Wait before retry
		time.Sleep(currentDelay)

		// Calculate next backoff delay using exponential backoff
		nextDelay := time.Duration(float64(policy.InitialDelay) * math.Pow(policy.BackoffFactor, float64(retryCount)))
		if nextDelay > policy.MaxDelay {
			nextDelay = policy.MaxDelay
		}
		currentDelay = nextDelay

		// Reset the routine controller for next attempt
		bot.RoutineController().Reset()
	}
}

// ShutdownAll shuts down all bot instances and cleans up shared resources
func (m *Manager) ShutdownAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Shutdown all bots
	for instance, bot := range m.bots {
		bot.ShutdownWithSharedRegistries()
		delete(m.bots, instance)
	}

	// Unload all template images
	if m.templateRegistry != nil {
		if registry, ok := m.templateRegistry.(*templates.TemplateRegistry); ok {
			registry.UnloadAll()
		}
	}

	// Note: Routines are eagerly loaded and don't need cleanup on shutdown
}

// GetActiveCount returns the number of active bot instances
func (m *Manager) GetActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.bots)
}

// TemplateRegistry returns the shared template registry
func (m *Manager) TemplateRegistry() actions.TemplateRegistryInterface {
	return m.templateRegistry
}

// RoutineRegistry returns the shared routine registry
func (m *Manager) RoutineRegistry() actions.RoutineRegistryInterface {
	return m.routineRegistry
}

// SetAccountPool sets the shared account pool
func (m *Manager) SetAccountPool(pool accountpool.AccountPool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.accountPool = pool
}

// AccountPool returns the shared account pool
func (m *Manager) AccountPool() accountpool.AccountPool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.accountPool
}

// ReloadRoutines clears and reloads all routines from disk
// Useful for development when routines are being modified
func (m *Manager) ReloadRoutines() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.routineRegistry != nil {
		return m.routineRegistry.Reload()
	}

	return nil
}

// ReloadTemplates clears and reloads all templates from YAML
// Useful for development when templates are being modified
func (m *Manager) ReloadTemplates() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if registry, ok := m.templateRegistry.(*templates.TemplateRegistry); ok {
		registry.Clear()
		templatesConfigPath := filepath.Join(m.basePath, "config", "templates")
		return registry.LoadFromDirectory(templatesConfigPath)
	}

	return fmt.Errorf("template registry not available")
}

// GetBotVariables returns all variables for a specific bot instance
func (m *Manager) GetBotVariables(instance int) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	bot, exists := m.bots[instance]
	if !exists {
		return nil, fmt.Errorf("bot instance %d not found", instance)
	}

	return bot.GetAllVariables(), nil
}
