package accountpool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// FileAccountPool implements AccountPool using XML files in a directory
type FileAccountPool struct {
	mu           sync.RWMutex
	accounts     map[string]*Account // All accounts by ID
	available    chan *Account       // Available accounts buffer
	config       PoolConfig
	basePath     string
	closed       bool
	stopRefresh  chan struct{}
	refreshTimer *time.Timer

	// Statistics
	stats PoolStats
}

// NewFileAccountPool creates a new file-based account pool
func NewFileAccountPool(basePath string, config PoolConfig) (*FileAccountPool, error) {
	// Validate base path
	if basePath == "" {
		return nil, fmt.Errorf("base path cannot be empty")
	}

	// Check if path exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("base path does not exist: %s", basePath)
	}

	// Apply defaults
	if config.BufferSize == 0 {
		config.BufferSize = 100
	}

	pool := &FileAccountPool{
		accounts:    make(map[string]*Account),
		available:   make(chan *Account, config.BufferSize),
		config:      config,
		basePath:    basePath,
		stopRefresh: make(chan struct{}),
		stats:       PoolStats{LastRefresh: time.Now()},
	}

	// Initial load
	if err := pool.Refresh(); err != nil {
		return nil, fmt.Errorf("failed to load accounts: %w", err)
	}

	// Start auto-refresh if configured
	if config.AutoRefresh && config.RefreshInterval > 0 {
		go pool.autoRefreshLoop()
	}

	return pool, nil
}

// GetNext returns the next available account
func (p *FileAccountPool) GetNext(ctx context.Context) (*Account, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, ErrPoolClosed
	}
	p.mu.RUnlock()

	// Try to get an account from the buffer
	select {
	case account := <-p.available:
		// Mark as in use
		p.mu.Lock()
		account.Status = AccountStatusInUse
		now := time.Now()
		account.AssignedAt = &now
		p.mu.Unlock()

		p.updateStats()
		return account, nil

	case <-ctx.Done():
		return nil, ctx.Err()

	default:
		// No accounts immediately available
		if p.config.WaitForAccounts {
			return p.waitForAccount(ctx)
		}
		return nil, ErrNoAccountsAvailable
	}
}

// waitForAccount waits for an account to become available
func (p *FileAccountPool) waitForAccount(ctx context.Context) (*Account, error) {
	// Create a timeout context if MaxWaitTime is set
	var cancel context.CancelFunc
	if p.config.MaxWaitTime > 0 {
		ctx, cancel = context.WithTimeout(ctx, p.config.MaxWaitTime)
		defer cancel()
	}

	// Try refresh if auto-refresh is enabled
	if p.config.AutoRefresh {
		if err := p.Refresh(); err != nil {
			fmt.Printf("Warning: Failed to refresh account pool: %v\n", err)
		}
	}

	// Wait for account or timeout
	select {
	case account := <-p.available:
		p.mu.Lock()
		account.Status = AccountStatusInUse
		now := time.Now()
		account.AssignedAt = &now
		p.mu.Unlock()

		p.updateStats()
		return account, nil

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Return puts an account back into the pool
func (p *FileAccountPool) Return(account *Account) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	// Find the account
	existing, exists := p.accounts[account.ID]
	if !exists {
		return ErrAccountNotFound
	}

	// Reset state
	existing.Status = AccountStatusAvailable
	existing.AssignedAt = nil
	existing.AssignedTo = 0

	// Put back in available buffer (non-blocking)
	select {
	case p.available <- existing:
		// Successfully returned
	default:
		// Buffer full, will be picked up on next refresh
	}

	p.updateStats()
	return nil
}

// MarkUsed marks an account as successfully processed
func (p *FileAccountPool) MarkUsed(account *Account, result AccountResult) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	existing, exists := p.accounts[account.ID]
	if !exists {
		return ErrAccountNotFound
	}

	// Update account
	existing.Status = AccountStatusCompleted
	now := time.Time{}
	existing.ProcessedAt = &now
	existing.Result = &result
	existing.LastError = ""

	p.updateStats()
	return nil
}

// MarkFailed marks an account as failed
func (p *FileAccountPool) MarkFailed(account *Account, reason string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	existing, exists := p.accounts[account.ID]
	if !exists {
		return ErrAccountNotFound
	}

	existing.FailureCount++
	existing.LastError = reason

	// Check if we should retry
	if p.config.RetryFailed && existing.FailureCount < p.config.MaxFailures {
		// Return to pool for retry
		existing.Status = AccountStatusAvailable
		existing.AssignedAt = nil
		existing.AssignedTo = 0

		select {
		case p.available <- existing:
			// Requeued for retry
		default:
			// Buffer full
		}
	} else {
		// Mark as permanently failed
		existing.Status = AccountStatusFailed
	}

	p.updateStats()
	return nil
}

// GetByID retrieves an account by ID
func (p *FileAccountPool) GetByID(id string) (*Account, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	account, exists := p.accounts[id]
	if !exists {
		return nil, ErrAccountNotFound
	}

	return account.Clone(), nil
}

// GetStats returns current pool statistics
func (p *FileAccountPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// Refresh reloads accounts from the file system
func (p *FileAccountPool) Refresh() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	// Scan directory for XML files
	accounts, err := p.scanDirectory()
	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	// Update existing accounts and add new ones
	for _, newAccount := range accounts {
		if existing, exists := p.accounts[newAccount.ID]; exists {
			// Update existing account (preserve state)
			existing.XMLPath = newAccount.XMLPath
			existing.PackCount = newAccount.PackCount
			existing.LastModified = newAccount.LastModified
			// Don't update status or result
		} else {
			// Add new account
			p.accounts[newAccount.ID] = newAccount
		}
	}

	// Rebuild available buffer
	p.rebuildAvailableBuffer()

	p.stats.LastRefresh = time.Now()
	p.updateStats()

	return nil
}

// scanDirectory scans the base path for XML files
func (p *FileAccountPool) scanDirectory() ([]*Account, error) {
	var accounts []*Account

	// Walk directory
	err := filepath.Walk(p.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process XML files
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".xml") {
			return nil
		}

		// Create account from file
		account := &Account{
			ID:           strings.TrimSuffix(info.Name(), filepath.Ext(info.Name())),
			XMLPath:      path,
			PackCount:    0, // TODO: Parse XML to get pack count
			LastModified: info.ModTime(),
			Metadata:     make(map[string]string),
			Status:       AccountStatusAvailable,
		}

		// Apply filters
		if p.config.MinPacks > 0 && account.PackCount < p.config.MinPacks {
			return nil // Skip - not enough packs
		}
		if p.config.MaxPacks > 0 && account.PackCount > p.config.MaxPacks {
			return nil // Skip - too many packs
		}

		accounts = append(accounts, account)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort accounts
	p.sortAccounts(accounts)

	return accounts, nil
}

// sortAccounts sorts accounts based on configured sort method
func (p *FileAccountPool) sortAccounts(accounts []*Account) {
	switch p.config.SortMethod {
	case SortMethodModifiedAsc:
		sort.Slice(accounts, func(i, j int) bool {
			return accounts[i].LastModified.Before(accounts[j].LastModified)
		})
	case SortMethodModifiedDesc:
		sort.Slice(accounts, func(i, j int) bool {
			return accounts[i].LastModified.After(accounts[j].LastModified)
		})
	case SortMethodPacksAsc:
		sort.Slice(accounts, func(i, j int) bool {
			return accounts[i].PackCount < accounts[j].PackCount
		})
	case SortMethodPacksDesc:
		sort.Slice(accounts, func(i, j int) bool {
			return accounts[i].PackCount > accounts[j].PackCount
		})
	}
}

// rebuildAvailableBuffer rebuilds the available account buffer
func (p *FileAccountPool) rebuildAvailableBuffer() {
	// Drain existing buffer
	for len(p.available) > 0 {
		<-p.available
	}

	// Add available accounts to buffer
	for _, account := range p.accounts {
		if account.Status == AccountStatusAvailable {
			select {
			case p.available <- account:
				// Added to buffer
			default:
				// Buffer full, will be available on next GetNext
				break
			}
		}
	}
}

// updateStats recalculates pool statistics
func (p *FileAccountPool) updateStats() {
	stats := PoolStats{
		Total:       len(p.accounts),
		LastRefresh: p.stats.LastRefresh,
	}

	var totalDuration time.Duration
	var durationCount int

	for _, account := range p.accounts {
		switch account.Status {
		case AccountStatusAvailable:
			stats.Available++
		case AccountStatusInUse:
			stats.InUse++
		case AccountStatusCompleted:
			stats.Completed++
			if account.Result != nil {
				stats.TotalPacksOpened += account.Result.PacksOpened
				stats.TotalCardsFound += account.Result.CardsFound
				stats.TotalStars += account.Result.StarsTotal
				stats.TotalKeeps += account.Result.KeepCount
				totalDuration += account.Result.Duration
				durationCount++
			}
		case AccountStatusFailed:
			stats.Failed++
		case AccountStatusSkipped:
			stats.Skipped++
		}
	}

	if durationCount > 0 {
		stats.AverageDuration = totalDuration / time.Duration(durationCount)
	}

	p.stats = stats
}

// autoRefreshLoop periodically refreshes the pool
func (p *FileAccountPool) autoRefreshLoop() {
	ticker := time.NewTicker(p.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := p.Refresh(); err != nil {
				fmt.Printf("Auto-refresh failed: %v\n", err)
			}
		case <-p.stopRefresh:
			return
		}
	}
}

// Close closes the pool and releases resources
func (p *FileAccountPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	close(p.stopRefresh)
	close(p.available)

	return nil
}
