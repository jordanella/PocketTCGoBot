package accountpool

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// SQLAccountPool implements AccountPool using SQL queries to select accounts
type SQLAccountPool struct {
	mu           sync.RWMutex
	db           *sql.DB
	queryDef     *QueryDefinition
	accounts     map[string]*Account
	available    chan *Account
	config       PoolConfig
	closed       bool
	stopRefresh  chan struct{}
	lastRefresh  time.Time
	stats        PoolStats
}

// QueryDefinition defines a SQL query for account selection
type QueryDefinition struct {
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Type        string          `yaml:"type"`
	Version     string          `yaml:"version"`
	Query       QueryConfig     `yaml:"query"`
	PoolConfig  PoolConfig      `yaml:"pool_config"`
	GUIConfig   *GUIQueryConfig `yaml:"gui_config,omitempty"`
}

// QueryConfig defines the SQL query and parameters
type QueryConfig struct {
	Select     string      `yaml:"select"`
	Parameters []Parameter `yaml:"parameters"`
}

// Parameter represents a query parameter
type Parameter struct {
	Name  string      `yaml:"name"`
	Value interface{} `yaml:"value"`
	Type  string      `yaml:"type"` // "string", "int", "float", "bool"
}

// GUIQueryConfig stores visual query builder configuration
type GUIQueryConfig struct {
	Filters []FilterConfig `yaml:"filters"`
	Sort    []SortConfig   `yaml:"sort"`
	Limit   int            `yaml:"limit"`
}

// FilterConfig represents a single filter condition
type FilterConfig struct {
	Field    string        `yaml:"field"`
	Operator string        `yaml:"operator"` // "=", "!=", "<", ">", "<=", ">=", "in", "like"
	Value    interface{}   `yaml:"value,omitempty"`
	Values   []interface{} `yaml:"values,omitempty"`
	Display  string        `yaml:"display"`
}

// SortConfig represents a sort order
type SortConfig struct {
	Field     string `yaml:"field"`
	Direction string `yaml:"direction"` // "ASC", "DESC"
	Display   string `yaml:"display"`
}

// NewSQLAccountPool creates a pool from a SQL query definition file
func NewSQLAccountPool(db *sql.DB, queryDefPath string, config PoolConfig) (*SQLAccountPool, error) {
	// Load query definition from YAML
	queryDef, err := loadQueryDefinition(queryDefPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load query definition: %w", err)
	}

	// Merge configs (file config takes precedence, then passed config)
	finalConfig := mergePoolConfigs(queryDef.PoolConfig, config)

	pool := &SQLAccountPool{
		db:          db,
		queryDef:    queryDef,
		accounts:    make(map[string]*Account),
		available:   make(chan *Account, finalConfig.BufferSize),
		config:      finalConfig,
		stopRefresh: make(chan struct{}),
	}

	// Initial query execution
	if err := pool.refresh(); err != nil {
		return nil, fmt.Errorf("initial query execution failed: %w", err)
	}

	// Start auto-refresh if configured
	if finalConfig.RefreshInterval > 0 {
		go pool.autoRefresh()
	}

	return pool, nil
}

// loadQueryDefinition loads a query definition from a YAML file
func loadQueryDefinition(path string) (*QueryDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read query definition file: %w", err)
	}

	var queryDef QueryDefinition
	if err := yaml.Unmarshal(data, &queryDef); err != nil {
		return nil, fmt.Errorf("failed to parse query definition YAML: %w", err)
	}

	// Validate query
	if err := validateQueryDefinition(&queryDef); err != nil {
		return nil, fmt.Errorf("invalid query definition: %w", err)
	}

	return &queryDef, nil
}

// validateQueryDefinition ensures the query definition is safe and valid
func validateQueryDefinition(queryDef *QueryDefinition) error {
	if queryDef.Name == "" {
		return fmt.Errorf("query name is required")
	}

	if queryDef.Query.Select == "" {
		return fmt.Errorf("query SELECT statement is required")
	}

	// Validate query is safe (SELECT only)
	if err := validateQuerySafety(queryDef.Query.Select); err != nil {
		return err
	}

	return nil
}

// validateQuerySafety ensures the query is safe to execute
func validateQuerySafety(query string) error {
	// Simple safety checks - only allow SELECT
	// In production, could use a SQL parser for more thorough validation
	upper := ""
	for _, c := range query {
		if c >= 'a' && c <= 'z' {
			upper += string(c - 32)
		} else if c >= 'A' && c <= 'Z' {
			upper += string(c)
		} else if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			upper += " "
		}
	}

	// Must start with SELECT
	if len(upper) < 6 || upper[:6] != "SELECT" {
		return fmt.Errorf("only SELECT queries are allowed")
	}

	// Check for dangerous keywords
	dangerous := []string{"DROP", "DELETE", "UPDATE", "INSERT", "ALTER", "CREATE", "EXEC", "EXECUTE"}
	for _, keyword := range dangerous {
		// Simple check - look for keyword surrounded by spaces
		if contains(upper, " "+keyword+" ") || contains(upper, " "+keyword) {
			return fmt.Errorf("query contains forbidden keyword: %s", keyword)
		}
	}

	return nil
}

// contains checks if s contains substr (case-insensitive for SQL keywords)
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// mergePoolConfigs merges two pool configs (priority config overrides base)
func mergePoolConfigs(base, priority PoolConfig) PoolConfig {
	result := base

	// Override with priority values if set
	if priority.MinPacks != 0 {
		result.MinPacks = priority.MinPacks
	}
	if priority.MaxPacks != 0 {
		result.MaxPacks = priority.MaxPacks
	}
	if priority.MaxFailures != 0 {
		result.MaxFailures = priority.MaxFailures
	}
	if priority.BufferSize != 0 {
		result.BufferSize = priority.BufferSize
	}
	if priority.MaxWaitTime != 0 {
		result.MaxWaitTime = priority.MaxWaitTime
	}
	if priority.RefreshInterval != 0 {
		result.RefreshInterval = priority.RefreshInterval
	}

	// Boolean fields - use priority if explicitly set
	result.RetryFailed = priority.RetryFailed || base.RetryFailed
	result.WaitForAccounts = priority.WaitForAccounts || base.WaitForAccounts

	return result
}

// refresh executes the SQL query and updates the account pool
func (p *SQLAccountPool) refresh() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Build parameter array
	params := make([]interface{}, len(p.queryDef.Query.Parameters))
	for i, param := range p.queryDef.Query.Parameters {
		params[i] = param.Value
	}

	// Execute query with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := p.db.QueryContext(ctx, p.queryDef.Query.Select, params...)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Parse results into accounts
	newAccounts := make(map[string]*Account)
	for rows.Next() {
		account := &Account{
			Metadata: make(map[string]string),
		}

		// Scan result - expects columns: id, xml_path, pack_count, last_modified, status, failure_count, last_error
		var lastModifiedStr string
		var lastError sql.NullString
		err := rows.Scan(
			&account.ID,
			&account.XMLPath,
			&account.PackCount,
			&lastModifiedStr,
			&account.Status,
			&account.FailureCount,
			&lastError,
		)
		if err != nil {
			return fmt.Errorf("failed to scan account row: %w", err)
		}

		// Parse timestamp
		if lastModifiedStr != "" {
			if t, err := time.Parse(time.RFC3339, lastModifiedStr); err == nil {
				account.LastModified = t
			}
		}

		// Handle nullable last error
		if lastError.Valid {
			account.LastError = lastError.String
		}

		// Validate XML path exists
		if _, err := os.Stat(account.XMLPath); err != nil {
			// File doesn't exist, skip this account
			continue
		}

		newAccounts[account.ID] = account
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	// Update accounts map
	oldAccounts := p.accounts
	p.accounts = newAccounts

	// Preserve runtime state for accounts that still exist
	for id, newAccount := range p.accounts {
		if oldAccount, exists := oldAccounts[id]; exists {
			// Preserve runtime fields
			newAccount.AssignedAt = oldAccount.AssignedAt
			newAccount.AssignedTo = oldAccount.AssignedTo
			newAccount.ProcessedAt = oldAccount.ProcessedAt
			newAccount.Result = oldAccount.Result
		}
	}

	// Refill available channel
	p.refillAvailableChannel()

	// Update stats
	p.updateStats()

	p.lastRefresh = time.Now()
	return nil
}

// refillAvailableChannel repopulates the buffered channel
func (p *SQLAccountPool) refillAvailableChannel() {
	// Drain existing channel
	for len(p.available) > 0 {
		<-p.available
	}

	// Refill with current available accounts
	for _, account := range p.accounts {
		if account.Status == AccountStatusAvailable {
			select {
			case p.available <- account:
			default:
				// Channel full, remaining accounts will be fetched on demand
				return
			}
		}
	}
}

// updateStats recalculates pool statistics
func (p *SQLAccountPool) updateStats() {
	stats := PoolStats{}

	for _, account := range p.accounts {
		stats.Total++

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
			}
		case AccountStatusFailed:
			stats.Failed++
		case AccountStatusSkipped:
			stats.Skipped++
		}
	}

	p.stats = stats
}

// autoRefresh periodically refreshes the pool
func (p *SQLAccountPool) autoRefresh() {
	if p.config.RefreshInterval == 0 {
		return
	}

	ticker := time.NewTicker(p.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopRefresh:
			return
		case <-ticker.C:
			if err := p.refresh(); err != nil {
				// Log error but continue
				fmt.Printf("SQL pool auto-refresh failed: %v\n", err)
			}
		}
	}
}

// GetNext implements AccountPool.GetNext
func (p *SQLAccountPool) GetNext(ctx context.Context) (*Account, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, fmt.Errorf("account pool is closed")
	}
	p.mu.RUnlock()

	// Try to get from channel
	select {
	case account := <-p.available:
		// Mark as in use
		p.mu.Lock()
		account.Status = AccountStatusInUse
		now := time.Now()
		account.AssignedAt = &now
		p.mu.Unlock()
		return account, nil

	case <-ctx.Done():
		if p.config.WaitForAccounts {
			return nil, fmt.Errorf("timeout waiting for available accounts")
		}
		return nil, fmt.Errorf("no accounts available")

	default:
		if !p.config.WaitForAccounts {
			return nil, fmt.Errorf("no accounts available")
		}

		// Wait for account with timeout
		timeout := p.config.MaxWaitTime
		if timeout == 0 {
			timeout = 5 * time.Minute
		}

		select {
		case account := <-p.available:
			p.mu.Lock()
			account.Status = AccountStatusInUse
			now := time.Now()
			account.AssignedAt = &now
			p.mu.Unlock()
			return account, nil

		case <-time.After(timeout):
			return nil, fmt.Errorf("timeout waiting for available accounts")

		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled while waiting for accounts")
		}
	}
}

// Return implements AccountPool.Return
func (p *SQLAccountPool) Return(account *Account) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("account pool is closed")
	}

	// Update status
	account.Status = AccountStatusAvailable
	account.AssignedAt = nil
	account.AssignedTo = 0

	// Add back to available channel if there's room
	select {
	case p.available <- account:
	default:
		// Channel full, account will be available on next GetNext scan
	}

	return nil
}

// MarkUsed implements AccountPool.MarkUsed
func (p *SQLAccountPool) MarkUsed(account *Account, result AccountResult) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("account pool is closed")
	}

	// Update account
	account.Result = &result
	now := time.Now()
	account.ProcessedAt = &now

	if result.Success {
		account.Status = AccountStatusCompleted
	} else {
		account.FailureCount++
		account.LastError = result.Error

		if p.config.RetryFailed && account.FailureCount < p.config.MaxFailures {
			account.Status = AccountStatusAvailable
			// Add back to channel
			select {
			case p.available <- account:
			default:
			}
		} else {
			account.Status = AccountStatusFailed
		}
	}

	// Update stats
	p.updateStats()

	return nil
}

// MarkFailed implements AccountPool.MarkFailed
func (p *SQLAccountPool) MarkFailed(account *Account, reason string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return fmt.Errorf("account pool is closed")
	}

	account.FailureCount++
	account.LastError = reason
	account.Status = AccountStatusFailed

	// Update stats
	p.updateStats()

	return nil
}

// GetByID implements AccountPool.GetByID
func (p *SQLAccountPool) GetByID(id string) (*Account, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	account, exists := p.accounts[id]
	if !exists {
		return nil, fmt.Errorf("account '%s' not found", id)
	}

	return account, nil
}

// GetStats implements AccountPool.GetStats
func (p *SQLAccountPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.stats
}

// Refresh implements AccountPool.Refresh
func (p *SQLAccountPool) Refresh() error {
	return p.refresh()
}

// Close implements AccountPool.Close
func (p *SQLAccountPool) Close() error {
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

// GetQueryDefinition returns the query definition for this pool
func (p *SQLAccountPool) GetQueryDefinition() *QueryDefinition {
	return p.queryDef
}

// GetLastRefreshTime returns when the pool was last refreshed
func (p *SQLAccountPool) GetLastRefreshTime() time.Time {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.lastRefresh
}
