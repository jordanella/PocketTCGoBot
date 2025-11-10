package accountpool

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// UnifiedAccountPool implements a flexible account pool with queries, inclusions, exclusions, and watched paths
type UnifiedAccountPool struct {
	mu           sync.RWMutex
	db           *sql.DB
	definition   *UnifiedPoolDefinition
	accounts     map[string]*Account // Resolved account list by device_account
	available    chan *Account
	config       PoolConfig
	closed       bool
	stopRefresh  chan struct{}
	lastRefresh  time.Time
	stats        PoolStats
	xmlStorageDir string // Global XML storage directory
}

// UnifiedPoolDefinition defines a unified pool configuration
type UnifiedPoolDefinition struct {
	PoolName    string             `yaml:"pool_name"`
	Description string             `yaml:"description"`
	Queries     []QuerySource      `yaml:"queries,omitempty"`      // Query sources (optional)
	Include     []string           `yaml:"include,omitempty"`      // Manual inclusions (optional)
	Exclude     []string           `yaml:"exclude,omitempty"`      // Manual exclusions (optional)
	WatchedPaths []string          `yaml:"watched_paths,omitempty"` // Folders to import from (optional)
	Config      UnifiedPoolConfig  `yaml:"config"`                 // Pool configuration
}

// QuerySource represents a single SQL query for populating accounts
type QuerySource struct {
	Name       string                 `yaml:"name"`
	SQL        string                 `yaml:"sql"`
	Parameters map[string]interface{} `yaml:"parameters,omitempty"`
}

// UnifiedPoolConfig holds pool behavior configuration
type UnifiedPoolConfig struct {
	SortMethod      string `yaml:"sort_method"`       // "packs_asc", "packs_desc", "modified_asc", "modified_desc"
	RetryFailed     bool   `yaml:"retry_failed"`      // Whether to retry failed accounts
	MaxFailures     int    `yaml:"max_failures"`      // Max times to retry
	RefreshInterval int    `yaml:"refresh_interval"` // Seconds between auto-refresh (0 = disabled)
}

// NewUnifiedAccountPool creates a new unified account pool
func NewUnifiedAccountPool(db *sql.DB, definitionPath string, xmlStorageDir string) (*UnifiedAccountPool, error) {
	// Load pool definition from YAML
	def, err := loadUnifiedPoolDefinition(definitionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load pool definition: %w", err)
	}

	// Validate definition
	if err := validateUnifiedPoolDefinition(def); err != nil {
		return nil, fmt.Errorf("invalid pool definition: %w", err)
	}

	// Ensure XML storage directory exists
	if err := os.MkdirAll(xmlStorageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create XML storage directory: %w", err)
	}

	pool := &UnifiedAccountPool{
		db:            db,
		definition:    def,
		accounts:      make(map[string]*Account),
		available:     make(chan *Account, 100),
		xmlStorageDir: xmlStorageDir,
		stopRefresh:   make(chan struct{}),
		config: PoolConfig{
			RetryFailed: def.Config.RetryFailed,
			MaxFailures: def.Config.MaxFailures,
			BufferSize:  100,
		},
	}

	// Initial refresh to populate accounts
	if err := pool.refresh(); err != nil {
		return nil, fmt.Errorf("initial refresh failed: %w", err)
	}

	// Start auto-refresh if configured
	if def.Config.RefreshInterval > 0 {
		go pool.autoRefresh()
	}

	return pool, nil
}

// loadUnifiedPoolDefinition loads a pool definition from YAML
func loadUnifiedPoolDefinition(path string) (*UnifiedPoolDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read pool definition file: %w", err)
	}

	var def UnifiedPoolDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &def, nil
}

// validateUnifiedPoolDefinition validates the pool definition
func validateUnifiedPoolDefinition(def *UnifiedPoolDefinition) error {
	if def.PoolName == "" {
		return fmt.Errorf("pool_name is required")
	}

	// Validate queries (ensure they're SELECT only)
	for i, query := range def.Queries {
		if query.SQL == "" {
			return fmt.Errorf("query %d (%s) has empty SQL", i, query.Name)
		}
		if err := validateQuerySafety(query.SQL); err != nil {
			return fmt.Errorf("query %d (%s): %w", i, query.Name, err)
		}
	}

	// Check for conflicts between include and exclude
	if len(def.Include) > 0 && len(def.Exclude) > 0 {
		conflicts := findConflicts(def.Include, def.Exclude)
		if len(conflicts) > 0 {
			// Log warning but allow (exclusions will be applied last)
			fmt.Printf("Warning: Pool '%s' has accounts in both include and exclude: %v (exclusions will be applied)\n",
				def.PoolName, conflicts)
		}
	}

	return nil
}

// findConflicts finds accounts that appear in both lists
func findConflicts(include, exclude []string) []string {
	conflicts := []string{}
	excludeSet := make(map[string]bool)

	for _, e := range exclude {
		excludeSet[e] = true
	}

	for _, i := range include {
		if excludeSet[i] {
			conflicts = append(conflicts, i)
		}
	}

	return conflicts
}

// refresh executes account resolution: queries → include → exclude → watched paths
func (p *UnifiedAccountPool) refresh() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	resolvedAccounts := make(map[string]*Account)

	// Step 1: Execute all queries
	for _, query := range p.definition.Queries {
		accounts, err := p.executeQuery(query)
		if err != nil {
			return fmt.Errorf("query '%s' failed: %w", query.Name, err)
		}

		// Add query results to resolved set
		for _, account := range accounts {
			resolvedAccounts[account.DeviceAccount] = account
		}
	}

	// Step 2: Add manual inclusions
	for _, deviceAccount := range p.definition.Include {
		// Fetch from database
		account, err := p.fetchAccountFromDB(deviceAccount)
		if err != nil {
			fmt.Printf("Warning: Failed to fetch included account '%s': %v\n", deviceAccount, err)
			continue
		}
		resolvedAccounts[deviceAccount] = account
	}

	// Step 3: Sync watched paths (adds to DB and aggregated list)
	if len(p.definition.WatchedPaths) > 0 {
		watchedAccounts, err := p.syncWatchedPaths()
		if err != nil {
			fmt.Printf("Warning: Failed to sync watched paths: %v\n", err)
		} else {
			// Add watched path accounts to resolved set
			for _, account := range watchedAccounts {
				resolvedAccounts[account.DeviceAccount] = account
			}
		}
	}

	// Step 4: Apply exclusions (remove from resolved set)
	for _, deviceAccount := range p.definition.Exclude {
		delete(resolvedAccounts, deviceAccount)
	}

	// Preserve runtime state for accounts that still exist
	oldAccounts := p.accounts
	p.accounts = resolvedAccounts

	for deviceAccount, newAccount := range p.accounts {
		if oldAccount, exists := oldAccounts[deviceAccount]; exists {
			// Preserve runtime fields
			newAccount.Status = oldAccount.Status
			newAccount.AssignedAt = oldAccount.AssignedAt
			newAccount.AssignedTo = oldAccount.AssignedTo
			newAccount.ProcessedAt = oldAccount.ProcessedAt
			newAccount.Result = oldAccount.Result
			newAccount.FailureCount = oldAccount.FailureCount
			newAccount.LastError = oldAccount.LastError
		}
	}

	// Sort accounts
	p.sortAccounts()

	// Refill available channel
	p.refillAvailableChannel()

	// Update stats
	p.updateStats()

	p.lastRefresh = time.Now()
	return nil
}

// executeQuery executes a single query and returns accounts
func (p *UnifiedAccountPool) executeQuery(query QuerySource) ([]*Account, error) {
	// Build parameter array
	params := make([]interface{}, 0)
	// Note: For now, we don't use named parameters - SQL should be complete
	// In future, could add parameter substitution

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	rows, err := p.db.QueryContext(ctx, query.SQL, params...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	accounts := make([]*Account, 0)

	for rows.Next() {
		account := &Account{
			Metadata: make(map[string]string),
		}

		// Scan result - expects: device_account, device_password, shinedust, packs_opened, last_used_at
		var lastUsedStr sql.NullString
		var shinedust, packsOpened int

		err := rows.Scan(
			&account.DeviceAccount,
			&account.DevicePassword,
			&shinedust,
			&packsOpened,
			&lastUsedStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		account.ID = account.DeviceAccount
		account.PackCount = packsOpened
		account.Status = AccountStatusAvailable

		// Parse timestamp
		if lastUsedStr.Valid && lastUsedStr.String != "" {
			if t, err := time.Parse(time.RFC3339, lastUsedStr.String); err == nil {
				account.LastModified = t
			}
		}

		accounts = append(accounts, account)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return accounts, nil
}

// fetchAccountFromDB retrieves a single account by device_account
func (p *UnifiedAccountPool) fetchAccountFromDB(deviceAccount string) (*Account, error) {
	query := `
		SELECT device_account, device_password, shinedust, packs_opened, last_used_at
		FROM accounts
		WHERE device_account = ?
	`

	account := &Account{
		Metadata: make(map[string]string),
	}

	var lastUsedStr sql.NullString
	var shinedust, packsOpened int

	err := p.db.QueryRow(query, deviceAccount).Scan(
		&account.DeviceAccount,
		&account.DevicePassword,
		&shinedust,
		&packsOpened,
		&lastUsedStr,
	)

	if err != nil {
		return nil, fmt.Errorf("account not found in database: %w", err)
	}

	account.ID = account.DeviceAccount
	account.PackCount = packsOpened
	account.Status = AccountStatusAvailable

	// Parse timestamp
	if lastUsedStr.Valid && lastUsedStr.String != "" {
		if t, err := time.Parse(time.RFC3339, lastUsedStr.String); err == nil {
			account.LastModified = t
		}
	}

	return account, nil
}

// syncWatchedPaths scans watched folders, imports to DB, copies to global storage, and returns accounts
func (p *UnifiedAccountPool) syncWatchedPaths() ([]*Account, error) {
	accounts := make([]*Account, 0)

	for _, watchedPath := range p.definition.WatchedPaths {
		// Check if path exists
		if _, err := os.Stat(watchedPath); os.IsNotExist(err) {
			fmt.Printf("Warning: Watched path does not exist: %s\n", watchedPath)
			continue
		}

		// Scan for XML files
		files, err := os.ReadDir(watchedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read watched path '%s': %w", watchedPath, err)
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			// Only process XML files
			if !strings.HasSuffix(strings.ToLower(file.Name()), ".xml") {
				continue
			}

			xmlPath := filepath.Join(watchedPath, file.Name())

			// Parse account from XML
			account, err := p.parseAccountXML(xmlPath)
			if err != nil {
				fmt.Printf("Warning: Failed to parse XML '%s': %v\n", xmlPath, err)
				continue
			}

			// Import to database (upsert)
			if err := p.importAccountToDB(account); err != nil {
				fmt.Printf("Warning: Failed to import account '%s' to database: %v\n", account.DeviceAccount, err)
				continue
			}

			// Copy to global storage
			if err := p.copyToGlobalStorage(xmlPath, account.DeviceAccount); err != nil {
				fmt.Printf("Warning: Failed to copy XML to global storage: %v\n", err)
				// Continue anyway - account is in DB
			}

			accounts = append(accounts, account)
		}
	}

	return accounts, nil
}

// parseAccountXML parses an account XML file
func (p *UnifiedAccountPool) parseAccountXML(xmlPath string) (*Account, error) {
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read XML: %w", err)
	}

	content := string(data)

	// Extract device_account
	deviceAccount := extractXMLTagContent(content, "account")
	if deviceAccount == "" {
		return nil, fmt.Errorf("missing <account> tag")
	}

	// Extract device_password
	devicePassword := extractXMLTagContent(content, "password")
	if devicePassword == "" {
		return nil, fmt.Errorf("missing <password> tag")
	}

	account := &Account{
		ID:             deviceAccount,
		DeviceAccount:  deviceAccount,
		DevicePassword: devicePassword,
		XMLPath:        xmlPath,
		Metadata:       make(map[string]string),
		Status:         AccountStatusAvailable,
	}

	return account, nil
}

// extractXMLTag extracts content from <tag>content</tag>
// Note: This is duplicated in pool_manager.go - should be refactored to shared utility
func extractXMLTagContent(xml, tag string) string {
	openTag := "<" + tag + ">"
	closeTag := "</" + tag + ">"

	startIdx := strings.Index(xml, openTag)
	if startIdx == -1 {
		return ""
	}
	startIdx += len(openTag)

	endIdx := strings.Index(xml[startIdx:], closeTag)
	if endIdx == -1 {
		return ""
	}

	return xml[startIdx : startIdx+endIdx]
}

// importAccountToDB inserts or updates an account in the database
func (p *UnifiedAccountPool) importAccountToDB(account *Account) error {
	query := `
		INSERT INTO accounts (device_account, device_password, created_at, last_used_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, NULL)
		ON CONFLICT(device_account) DO UPDATE SET
			device_password = excluded.device_password
	`

	_, err := p.db.Exec(query, account.DeviceAccount, account.DevicePassword)
	return err
}

// copyToGlobalStorage copies an XML file to global storage
func (p *UnifiedAccountPool) copyToGlobalStorage(sourcePath, deviceAccount string) error {
	destPath := filepath.Join(p.xmlStorageDir, deviceAccount+".xml")

	// Read source
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination: %w", err)
	}

	return nil
}

// sortAccounts sorts the account list based on configuration
func (p *UnifiedAccountPool) sortAccounts() {
	// Convert map to slice for sorting
	accountList := make([]*Account, 0, len(p.accounts))
	for _, account := range p.accounts {
		accountList = append(accountList, account)
	}

	// Sort based on config
	// (Implementation would use sort.Slice with appropriate comparator)
	// For now, we'll keep them unsorted

	// Rebuild map (order doesn't matter for map, but this keeps consistency)
	p.accounts = make(map[string]*Account)
	for _, account := range accountList {
		p.accounts[account.DeviceAccount] = account
	}
}

// refillAvailableChannel repopulates the buffered channel
func (p *UnifiedAccountPool) refillAvailableChannel() {
	// Drain existing channel
	for len(p.available) > 0 {
		<-p.available
	}

	// Refill with available accounts
	for _, account := range p.accounts {
		if account.Status == AccountStatusAvailable {
			select {
			case p.available <- account:
			default:
				// Channel full
				return
			}
		}
	}
}

// updateStats recalculates pool statistics
func (p *UnifiedAccountPool) updateStats() {
	stats := PoolStats{
		LastRefresh: p.lastRefresh,
	}

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
func (p *UnifiedAccountPool) autoRefresh() {
	if p.definition.Config.RefreshInterval == 0 {
		return
	}

	interval := time.Duration(p.definition.Config.RefreshInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopRefresh:
			return
		case <-ticker.C:
			if err := p.refresh(); err != nil {
				fmt.Printf("Auto-refresh failed for pool '%s': %v\n", p.definition.PoolName, err)
			}
		}
	}
}

// GetNext implements AccountPool.GetNext
func (p *UnifiedAccountPool) GetNext(ctx context.Context) (*Account, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, ErrPoolClosed
	}
	p.mu.RUnlock()

	select {
	case account := <-p.available:
		// Mark as in use
		p.mu.Lock()
		account.Status = AccountStatusInUse
		now := time.Now()
		account.AssignedAt = &now
		p.mu.Unlock()

		// Ensure XML exists
		if err := p.ensureXMLExists(account); err != nil {
			return nil, fmt.Errorf("failed to ensure XML exists: %w", err)
		}

		return account, nil

	case <-ctx.Done():
		return nil, ctx.Err()

	default:
		return nil, ErrNoAccountsAvailable
	}
}

// ensureXMLExists ensures the account has an XML file in global storage
func (p *UnifiedAccountPool) ensureXMLExists(account *Account) error {
	xmlPath := filepath.Join(p.xmlStorageDir, account.DeviceAccount+".xml")

	// Check if file exists
	if _, err := os.Stat(xmlPath); err == nil {
		// File exists, use it
		account.XMLPath = xmlPath
		return nil
	}

	// File doesn't exist, generate it
	xmlContent := fmt.Sprintf(`<account>%s</account>
<password>%s</password>`, account.DeviceAccount, account.DevicePassword)

	if err := os.WriteFile(xmlPath, []byte(xmlContent), 0644); err != nil {
		return fmt.Errorf("failed to generate XML: %w", err)
	}

	account.XMLPath = xmlPath
	return nil
}

// Return implements AccountPool.Return
func (p *UnifiedAccountPool) Return(account *Account) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	account.Status = AccountStatusAvailable
	account.AssignedAt = nil
	account.AssignedTo = 0

	// Add back to channel
	select {
	case p.available <- account:
	default:
		// Channel full
	}

	return nil
}

// MarkUsed implements AccountPool.MarkUsed
func (p *UnifiedAccountPool) MarkUsed(account *Account, result AccountResult) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

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
			select {
			case p.available <- account:
			default:
			}
		} else {
			account.Status = AccountStatusFailed
		}
	}

	p.updateStats()
	return nil
}

// MarkFailed implements AccountPool.MarkFailed
func (p *UnifiedAccountPool) MarkFailed(account *Account, reason string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return ErrPoolClosed
	}

	account.FailureCount++
	account.LastError = reason
	account.Status = AccountStatusFailed

	p.updateStats()
	return nil
}

// GetByID implements AccountPool.GetByID
func (p *UnifiedAccountPool) GetByID(id string) (*Account, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	account, exists := p.accounts[id]
	if !exists {
		return nil, ErrAccountNotFound
	}

	return account.Clone(), nil
}

// GetStats implements AccountPool.GetStats
func (p *UnifiedAccountPool) GetStats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// Refresh implements AccountPool.Refresh
func (p *UnifiedAccountPool) Refresh() error {
	return p.refresh()
}

// ListAccounts implements AccountPool.ListAccounts
func (p *UnifiedAccountPool) ListAccounts() []*Account {
	p.mu.RLock()
	defer p.mu.RUnlock()

	accounts := make([]*Account, 0, len(p.accounts))
	for _, account := range p.accounts {
		accounts = append(accounts, account.Clone())
	}

	return accounts
}

// Close implements AccountPool.Close
func (p *UnifiedAccountPool) Close() error {
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

// GetDefinition returns the pool definition
func (p *UnifiedAccountPool) GetDefinition() *UnifiedPoolDefinition {
	return p.definition
}

// validateQuerySafety validates that a SQL query is safe to execute
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

// contains checks if s contains substr
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
