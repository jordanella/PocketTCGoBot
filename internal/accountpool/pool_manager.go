package accountpool

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// PoolManager manages account pool definitions and instances
type PoolManager struct {
	poolsDir      string
	db            *sql.DB
	xmlStorageDir string // Global XML storage directory (./account_xmls/)
	pools         map[string]*PoolDefinition
	instances     map[string]AccountPool
	mu            sync.RWMutex
	eventBus      interface{} // events.EventBus - interface{} to avoid circular import
}

// PoolDefinition describes a pool configuration
// All pools are now unified - the Type field has been removed
type PoolDefinition struct {
	Name     string                   `yaml:"name"`
	FilePath string                   `yaml:"-"` // Path to YAML file (not stored in YAML)
	Config   *UnifiedPoolDefinition   `yaml:"-"` // Pool configuration
}

// TestResult contains results from testing a pool
type TestResult struct {
	Success       bool
	AccountsFound int
	SampleAccounts []AccountSummary
	Error         string
}

// AccountSummary provides a brief account overview
type AccountSummary struct {
	ID        string
	PackCount int
	Status    AccountStatus
	XMLPath   string
}

// NewPoolManager creates a new pool manager
func NewPoolManager(poolsDir string, db *sql.DB, xmlStorageDir string) *PoolManager {
	// Ensure XML storage directory exists
	os.MkdirAll(xmlStorageDir, 0755)

	return &PoolManager{
		poolsDir:      poolsDir,
		db:            db,
		xmlStorageDir: xmlStorageDir,
		pools:         make(map[string]*PoolDefinition),
		instances:     make(map[string]AccountPool),
		eventBus:      nil,
	}
}

// SetEventBus sets the event bus for publishing pool events
func (pm *PoolManager) SetEventBus(eventBus interface{}) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.eventBus = eventBus

	// Also set on existing pool instances
	for _, instance := range pm.instances {
		if unifiedPool, ok := instance.(*UnifiedAccountPool); ok {
			unifiedPool.SetEventBus(eventBus)
		}
	}
}

// DiscoverPools scans the pools directory for pool definitions
func (pm *PoolManager) DiscoverPools() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Ensure pools directory exists
	if err := os.MkdirAll(pm.poolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create pools directory: %w", err)
	}

	// Scan for YAML files
	entries, err := os.ReadDir(pm.poolsDir)
	if err != nil {
		return fmt.Errorf("failed to read pools directory: %w", err)
	}

	newPools := make(map[string]*PoolDefinition)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .yaml files (skip .example files)
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".example") {
			continue
		}

		filePath := filepath.Join(pm.poolsDir, name)

		// Load pool definition
		poolDef, err := pm.loadPoolDefinition(filePath)
		if err != nil {
			// Log error but continue with other pools
			fmt.Printf("Warning: Failed to load pool from %s: %v\n", filePath, err)
			continue
		}

		poolDef.FilePath = filePath
		newPools[poolDef.Name] = poolDef
	}

	pm.pools = newPools
	return nil
}

// loadPoolDefinition loads a pool definition from a YAML file
func (pm *PoolManager) loadPoolDefinition(filePath string) (*PoolDefinition, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// All pools are unified pools now
	var config UnifiedPoolDefinition
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	poolDef := &PoolDefinition{
		Name:   config.PoolName,
		Config: &config,
	}

	return poolDef, nil
}

// ListPools returns all discovered pool names
func (pm *PoolManager) ListPools() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.pools))
	for name := range pm.pools {
		names = append(names, name)
	}
	return names
}

// GetPoolDefinition retrieves a pool definition by name
func (pm *PoolManager) GetPoolDefinition(name string) (*PoolDefinition, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	poolDef, exists := pm.pools[name]
	if !exists {
		return nil, fmt.Errorf("pool '%s' not found", name)
	}

	return poolDef, nil
}

// GetPool retrieves or creates a pool instance
func (pm *PoolManager) GetPool(name string) (AccountPool, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if instance already exists
	if instance, exists := pm.instances[name]; exists {
		return instance, nil
	}

	// Get pool definition
	poolDef, exists := pm.pools[name]
	if !exists {
		return nil, fmt.Errorf("pool '%s' not found", name)
	}

	// Create unified pool instance
	if pm.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	pool, err := NewUnifiedAccountPool(pm.db, poolDef.FilePath, pm.xmlStorageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// Set event bus if available
	if pm.eventBus != nil {
		pool.SetEventBus(pm.eventBus)
	}

	// Cache instance
	pm.instances[name] = pool
	return pool, nil
}

// CreatePool saves a new pool definition
func (pm *PoolManager) CreatePool(poolDef *PoolDefinition) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if pool already exists
	if _, exists := pm.pools[poolDef.Name]; exists {
		return fmt.Errorf("pool '%s' already exists", poolDef.Name)
	}

	// Generate filename
	filename := sanitizeFilename(poolDef.Name) + ".yaml"
	filePath := filepath.Join(pm.poolsDir, filename)

	// Save to disk
	if err := pm.savePoolDefinition(filePath, poolDef); err != nil {
		return err
	}

	poolDef.FilePath = filePath
	pm.pools[poolDef.Name] = poolDef

	return nil
}

// UpdatePool modifies an existing pool definition
func (pm *PoolManager) UpdatePool(name string, poolDef *PoolDefinition) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if pool exists
	existing, exists := pm.pools[name]
	if !exists {
		return fmt.Errorf("pool '%s' not found", name)
	}

	// If name changed, remove old instance
	if name != poolDef.Name {
		delete(pm.instances, name)
	}

	// Save to disk (use existing file path if name didn't change)
	filePath := existing.FilePath
	if name != poolDef.Name {
		// Name changed, create new file
		filename := sanitizeFilename(poolDef.Name) + ".yaml"
		filePath = filepath.Join(pm.poolsDir, filename)
	}

	if err := pm.savePoolDefinition(filePath, poolDef); err != nil {
		return err
	}

	// If name changed, delete old file
	if name != poolDef.Name && existing.FilePath != filePath {
		os.Remove(existing.FilePath)
		delete(pm.pools, name)
	}

	poolDef.FilePath = filePath
	pm.pools[poolDef.Name] = poolDef

	// Invalidate cached instance
	delete(pm.instances, poolDef.Name)

	return nil
}

// DeletePool removes a pool definition
func (pm *PoolManager) DeletePool(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	poolDef, exists := pm.pools[name]
	if !exists {
		return fmt.Errorf("pool '%s' not found", name)
	}

	// Close instance if active
	if instance, exists := pm.instances[name]; exists {
		instance.Close()
		delete(pm.instances, name)
	}

	// Delete file
	if err := os.Remove(poolDef.FilePath); err != nil {
		return fmt.Errorf("failed to delete pool file: %w", err)
	}

	delete(pm.pools, name)
	return nil
}

// TestPool executes a pool query/scan without creating a persistent instance
func (pm *PoolManager) TestPool(name string) (*TestResult, error) {
	poolDef, err := pm.GetPoolDefinition(name)
	if err != nil {
		return nil, err
	}

	result := &TestResult{
		SampleAccounts: make([]AccountSummary, 0),
	}

	// Create temporary unified pool instance
	if pm.db == nil {
		result.Success = false
		result.Error = "database not configured"
		return result, nil
	}

	pool, err := NewUnifiedAccountPool(pm.db, poolDef.FilePath, pm.xmlStorageDir)
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		return result, nil
	}
	defer pool.Close()

	// Get stats
	stats := pool.GetStats()
	result.Success = true
	result.AccountsFound = stats.Total

	// Get sample accounts (up to 10)
	accounts := pool.ListAccounts()
	sampleLimit := 10
	if len(accounts) < sampleLimit {
		sampleLimit = len(accounts)
	}

	result.SampleAccounts = make([]AccountSummary, 0, sampleLimit)
	for i := 0; i < sampleLimit; i++ {
		acc := accounts[i]
		summary := AccountSummary{
			ID:        acc.ID,
			PackCount: acc.PackCount,
			Status:    acc.Status,
		}
		result.SampleAccounts = append(result.SampleAccounts, summary)
	}

	return result, nil
}

// savePoolDefinition saves a pool definition to a YAML file
func (pm *PoolManager) savePoolDefinition(filePath string, poolDef *PoolDefinition) error {
	// Marshal the config
	data, err := yaml.Marshal(poolDef.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal pool config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write pool file: %w", err)
	}

	return nil
}

// sanitizeFilename converts a pool name to a safe filename
func sanitizeFilename(name string) string {
	// Replace spaces with underscores
	filename := strings.ReplaceAll(name, " ", "_")

	// Remove unsafe characters
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range unsafe {
		filename = strings.ReplaceAll(filename, char, "")
	}

	// Convert to lowercase
	filename = strings.ToLower(filename)

	return filename
}

// RefreshPool manually refreshes a pool instance
func (pm *PoolManager) RefreshPool(name string) error {
	pool, err := pm.GetPool(name)
	if err != nil {
		return err
	}

	return pool.Refresh()
}

// ClosePool closes a pool instance (removes from cache)
func (pm *PoolManager) ClosePool(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	instance, exists := pm.instances[name]
	if !exists {
		return nil // Already closed
	}

	if err := instance.Close(); err != nil {
		return err
	}

	delete(pm.instances, name)
	return nil
}

// CloseAll closes all active pool instances
func (pm *PoolManager) CloseAll() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, instance := range pm.instances {
		instance.Close()
		delete(pm.instances, name)
	}

	return nil
}

// ImportFolder imports account XMLs from an arbitrary folder into the database and global storage
func (pm *PoolManager) ImportFolder(folderPath string) (imported []string, err error) {
	// Check if folder exists
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("folder does not exist: %s", folderPath)
	}

	// Read directory
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	imported = make([]string, 0)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// Only process XML files
		if !strings.HasSuffix(strings.ToLower(file.Name()), ".xml") {
			continue
		}

		xmlPath := filepath.Join(folderPath, file.Name())

		// Parse account
		account, err := parseAccountXMLFile(xmlPath)
		if err != nil {
			fmt.Printf("Warning: Failed to parse '%s': %v\n", xmlPath, err)
			continue
		}

		// Import to database
		if err := importAccountToDB(pm.db, account); err != nil {
			fmt.Printf("Warning: Failed to import account '%s': %v\n", account.DeviceAccount, err)
			continue
		}

		// Copy to global storage
		if err := copyToGlobalStorage(xmlPath, pm.xmlStorageDir, account.DeviceAccount); err != nil {
			fmt.Printf("Warning: Failed to copy to global storage: %v\n", err)
			// Continue anyway - account is in DB
		}

		imported = append(imported, account.DeviceAccount)
	}

	return imported, nil
}

// ExportPoolXMLs exports all XMLs from a pool to a destination folder
func (pm *PoolManager) ExportPoolXMLs(poolName, destFolder string) error {
	// Get pool instance
	pool, err := pm.GetPool(poolName)
	if err != nil {
		return fmt.Errorf("failed to get pool: %w", err)
	}

	// Get all accounts from pool
	accounts := pool.ListAccounts()
	if len(accounts) == 0 {
		return fmt.Errorf("pool has no accounts to export")
	}

	// Ensure destination folder exists
	if err := os.MkdirAll(destFolder, 0755); err != nil {
		return fmt.Errorf("failed to create destination folder: %w", err)
	}

	// Export each account
	exported := 0
	failed := 0

	for _, account := range accounts {
		// Get or generate XML
		xmlContent, err := pm.GetAccountXML(account.DeviceAccount)
		if err != nil {
			fmt.Printf("Warning: Failed to get XML for account '%s': %v\n", account.DeviceAccount, err)
			failed++
			continue
		}

		// Write to destination
		destPath := filepath.Join(destFolder, account.DeviceAccount+".xml")
		if err := os.WriteFile(destPath, xmlContent, 0644); err != nil {
			fmt.Printf("Warning: Failed to write XML for account '%s': %v\n", account.DeviceAccount, err)
			failed++
			continue
		}

		exported++
	}

	if exported == 0 {
		return fmt.Errorf("failed to export any accounts (all %d failed)", failed)
	}

	fmt.Printf("Successfully exported %d accounts (%d failed) from pool '%s' to '%s'\n",
		exported, failed, poolName, destFolder)

	return nil
}

// ExportAccountXML exports a single account XML by device_account
func (pm *PoolManager) ExportAccountXML(deviceAccount, destFolder string) error {
	// Ensure destination folder exists
	if err := os.MkdirAll(destFolder, 0755); err != nil {
		return fmt.Errorf("failed to create destination folder: %w", err)
	}

	// Check if XML exists in global storage
	sourcePath := filepath.Join(pm.xmlStorageDir, deviceAccount+".xml")
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		// XML doesn't exist in global storage, need to generate from DB
		account, err := pm.fetchAccountFromDB(deviceAccount)
		if err != nil {
			return fmt.Errorf("account not found in database: %w", err)
		}

		// Generate XML
		xmlContent := fmt.Sprintf(`<account>%s</account>
<password>%s</password>`, account.DeviceAccount, account.DevicePassword)

		// Save to global storage first
		if err := os.WriteFile(sourcePath, []byte(xmlContent), 0644); err != nil {
			return fmt.Errorf("failed to save to global storage: %w", err)
		}
	}

	// Copy to destination
	destPath := filepath.Join(destFolder, deviceAccount+".xml")
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read source XML: %w", err)
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write destination XML: %w", err)
	}

	return nil
}

// GetAccountXML retrieves the XML content for an account (from global storage or generates it)
func (pm *PoolManager) GetAccountXML(deviceAccount string) ([]byte, error) {
	// Check global storage first
	xmlPath := filepath.Join(pm.xmlStorageDir, deviceAccount+".xml")
	if data, err := os.ReadFile(xmlPath); err == nil {
		return data, nil
	}

	// Not in global storage, generate from database
	account, err := pm.fetchAccountFromDB(deviceAccount)
	if err != nil {
		return nil, fmt.Errorf("account not found: %w", err)
	}

	// Generate XML content
	xmlContent := fmt.Sprintf(`<account>%s</account>
<password>%s</password>`, account.DeviceAccount, account.DevicePassword)

	// Save to global storage for future use
	if err := os.WriteFile(xmlPath, []byte(xmlContent), 0644); err != nil {
		// Log warning but return content anyway
		fmt.Printf("Warning: Failed to cache XML to global storage: %v\n", err)
	}

	return []byte(xmlContent), nil
}

// EnsureXMLExists ensures an account has an XML file in global storage
func (pm *PoolManager) EnsureXMLExists(deviceAccount string) error {
	xmlPath := filepath.Join(pm.xmlStorageDir, deviceAccount+".xml")

	// Check if exists
	if _, err := os.Stat(xmlPath); err == nil {
		return nil // Already exists
	}

	// Generate from database
	_, err := pm.GetAccountXML(deviceAccount)
	return err
}

// Helper methods

// parseAccountXMLFile parses an account XML file
func parseAccountXMLFile(xmlPath string) (*Account, error) {
	data, err := os.ReadFile(xmlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read XML: %w", err)
	}

	content := string(data)

	// Extract device_account
	deviceAccount := extractXMLTag(content, "account")
	if deviceAccount == "" {
		return nil, fmt.Errorf("missing <account> tag")
	}

	// Extract device_password
	devicePassword := extractXMLTag(content, "password")
	if devicePassword == "" {
		return nil, fmt.Errorf("missing <password> tag")
	}

	return &Account{
		ID:             deviceAccount,
		DeviceAccount:  deviceAccount,
		DevicePassword: devicePassword,
		XMLPath:        xmlPath,
		Metadata:       make(map[string]string),
		Status:         AccountStatusAvailable,
	}, nil
}

// Note: extractXMLTag, importAccountToDB, and copyToGlobalStorage have been
// moved to utils.go to eliminate code duplication

// fetchAccountFromDB retrieves a single account from the database
func (pm *PoolManager) fetchAccountFromDB(deviceAccount string) (*Account, error) {
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

	err := pm.db.QueryRow(query, deviceAccount).Scan(
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

	return account, nil
}
