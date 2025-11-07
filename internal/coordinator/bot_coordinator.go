package coordinator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"jordanella.com/pocket-tcg-go/internal/bot"
)

// BotCoordinator manages bot execution with account injection
type BotCoordinator struct {
	mu              sync.RWMutex
	accountManager  *AccountManager
	activeBots      map[int]*BotExecution
	requestQueue    chan *BotRequest
	stopChan        chan bool
	config          *bot.Config
}

// BotRequest represents a request to run a bot with specific configuration
type BotRequest struct {
	Instance    int
	RoutineName string
	Bot         *bot.Bot
	Account     *Account // Injected by coordinator
}

// BotExecution tracks a running bot
type BotExecution struct {
	Request   *BotRequest
	Context   context.Context
	Cancel    context.CancelFunc
	StartTime time.Time
	Status    string
}

// NewBotCoordinator creates a new bot coordinator
func NewBotCoordinator(config *bot.Config) *BotCoordinator {
	accountManager := NewAccountManager(config.FolderPath, config)

	coordinator := &BotCoordinator{
		accountManager: accountManager,
		activeBots:     make(map[int]*BotExecution),
		requestQueue:   make(chan *BotRequest, 100),
		stopChan:       make(chan bool),
		config:         config,
	}

	// Start processing requests
	go coordinator.processRequests()

	return coordinator
}

// SubmitBotRequest submits a bot request for execution
func (c *BotCoordinator) SubmitBotRequest(request *BotRequest) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if instance is already running
	if _, exists := c.activeBots[request.Instance]; exists {
		return fmt.Errorf("bot instance %d is already running", request.Instance)
	}

	// Queue the request
	select {
	case c.requestQueue <- request:
		return nil
	default:
		return fmt.Errorf("request queue is full")
	}
}

// processRequests processes bot requests from the queue
func (c *BotCoordinator) processRequests() {
	for {
		select {
		case <-c.stopChan:
			return

		case request := <-c.requestQueue:
			// Process request in goroutine
			go c.executeBot(request)
		}
	}
}

// executeBot executes a bot with account injection
func (c *BotCoordinator) executeBot(request *BotRequest) {
	// Inject account
	if err := c.injectAccount(request); err != nil {
		// Log error but continue - bot can run without account injection
		fmt.Printf("Warning: Failed to inject account for bot %d: %v\n", request.Instance, err)
	}

	// Create execution context
	ctx, cancel := context.WithCancel(context.Background())

	execution := &BotExecution{
		Request:   request,
		Context:   ctx,
		Cancel:    cancel,
		StartTime: time.Now(),
		Status:    "running",
	}

	// Register execution
	c.mu.Lock()
	c.activeBots[request.Instance] = execution
	c.mu.Unlock()

	// Execute routine if specified
	if request.RoutineName != "" {
		if err := c.executeRoutine(request); err != nil {
			fmt.Printf("Error: Bot %d routine '%s' failed: %v\n", request.Instance, request.RoutineName, err)
			execution.Status = fmt.Sprintf("error: %v", err)
		} else {
			execution.Status = "completed"
		}
	} else {
		// Run default bot logic
		if err := request.Bot.Run(); err != nil {
			fmt.Printf("Error: Bot %d run failed: %v\n", request.Instance, err)
			execution.Status = fmt.Sprintf("error: %v", err)
		} else {
			execution.Status = "completed"
		}
	}

	// Cleanup
	c.mu.Lock()
	delete(c.activeBots, request.Instance)
	c.mu.Unlock()
}

// injectAccount injects an account into the bot
func (c *BotCoordinator) injectAccount(request *BotRequest) error {
	// Load next eligible account
	account, err := c.accountManager.LoadNextEligibleAccount()
	if err != nil {
		return fmt.Errorf("failed to load account: %w", err)
	}

	if account == nil {
		return fmt.Errorf("no eligible accounts available")
	}

	// Attach account to request
	request.Account = account

	// Mark account as used
	c.accountManager.MarkAccountAsUsed(account)

	fmt.Printf("Bot %d: Injected account: %s\n", request.Instance, account.FileName)

	// TODO: Implement actual account injection via ADB
	// request.Bot.ADB().Push(account.FilePath, "/sdcard/...")

	return nil
}

// executeRoutine executes a specific routine on the bot
func (c *BotCoordinator) executeRoutine(request *BotRequest) error {
	// Get routine from bot's registry
	routineBuilder, err := request.Bot.Routines().Get(request.RoutineName)
	if err != nil {
		return fmt.Errorf("failed to get routine: %w", err)
	}

	// Execute routine
	if err := routineBuilder.Execute(request.Bot); err != nil {
		return fmt.Errorf("routine execution failed: %w", err)
	}

	fmt.Printf("Bot %d: Successfully completed routine '%s'\n", request.Instance, request.RoutineName)

	return nil
}

// StopBot stops a specific bot instance
func (c *BotCoordinator) StopBot(instance int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	execution, exists := c.activeBots[instance]
	if !exists {
		return fmt.Errorf("bot instance %d is not running", instance)
	}

	// Cancel the bot's context
	execution.Cancel()
	execution.Status = "stopped"

	return nil
}

// StopAll stops all running bots
func (c *BotCoordinator) StopAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Cancel all active bots
	for _, execution := range c.activeBots {
		execution.Cancel()
		execution.Status = "stopped"
	}

	// Clear active bots
	c.activeBots = make(map[int]*BotExecution)

	// Stop the request processor
	select {
	case c.stopChan <- true:
	default:
	}
}

// GetBotStatus returns the status of a bot instance
func (c *BotCoordinator) GetBotStatus(instance int) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	execution, exists := c.activeBots[instance]
	if !exists {
		return "not running", false
	}

	return execution.Status, true
}

// GetActiveBotCount returns the number of active bots
func (c *BotCoordinator) GetActiveBotCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.activeBots)
}

// GetActiveBots returns a list of active bot instances
func (c *BotCoordinator) GetActiveBots() []int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	instances := make([]int, 0, len(c.activeBots))
	for instance := range c.activeBots {
		instances = append(instances, instance)
	}

	return instances
}
