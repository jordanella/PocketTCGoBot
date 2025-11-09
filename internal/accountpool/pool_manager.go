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
	poolsDir  string
	db        *sql.DB
	pools     map[string]*PoolDefinition
	instances map[string]AccountPool
	mu        sync.RWMutex
}

// PoolDefinition describes a pool configuration
type PoolDefinition struct {
	Name     string      `yaml:"name"`
	Type     string      `yaml:"type"` // "file" or "sql"
	FilePath string      `yaml:"-"`    // Path to YAML file (not stored in YAML)
	Config   interface{} `yaml:"-"`    // FilePoolConfig or QueryDefinition
}

// FilePoolConfig holds file-based pool configuration
type FilePoolConfig struct {
	Name       string     `yaml:"name"`
	Type       string     `yaml:"type"` // "file"
	Directory  string     `yaml:"directory"`
	PoolConfig PoolConfig `yaml:"pool_config"`
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
func NewPoolManager(poolsDir string, db *sql.DB) *PoolManager {
	return &PoolManager{
		poolsDir:  poolsDir,
		db:        db,
		pools:     make(map[string]*PoolDefinition),
		instances: make(map[string]AccountPool),
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

	// First, peek at the type to determine which struct to use
	var typeCheck struct {
		Type string `yaml:"type"`
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal(data, &typeCheck); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	poolDef := &PoolDefinition{
		Name: typeCheck.Name,
		Type: typeCheck.Type,
	}

	// Load the appropriate config based on type
	switch typeCheck.Type {
	case "file":
		var config FilePoolConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse file pool config: %w", err)
		}
		poolDef.Config = &config

	case "sql":
		var config QueryDefinition
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse SQL pool config: %w", err)
		}
		poolDef.Config = &config

	default:
		return nil, fmt.Errorf("unknown pool type: %s", typeCheck.Type)
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

	// Create instance based on type
	var pool AccountPool
	var err error

	switch poolDef.Type {
	case "file":
		config := poolDef.Config.(*FilePoolConfig)
		pool, err = NewFileAccountPool(config.Directory, config.PoolConfig)

	case "sql":
		if pm.db == nil {
			return nil, fmt.Errorf("database not configured for SQL pools")
		}
		// Use the file path directly since it contains the full config
		pool, err = NewSQLAccountPool(pm.db, poolDef.FilePath, PoolConfig{})

	default:
		return nil, fmt.Errorf("unsupported pool type: %s", poolDef.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
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

	// Create temporary instance
	var pool AccountPool
	switch poolDef.Type {
	case "file":
		config := poolDef.Config.(*FilePoolConfig)
		pool, err = NewFileAccountPool(config.Directory, config.PoolConfig)

	case "sql":
		if pm.db == nil {
			result.Success = false
			result.Error = "database not configured"
			return result, nil
		}
		pool, err = NewSQLAccountPool(pm.db, poolDef.FilePath, PoolConfig{})

	default:
		result.Success = false
		result.Error = fmt.Sprintf("unsupported pool type: %s", poolDef.Type)
		return result, nil
	}

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
	// This is a bit hacky - we're peeking at internal state
	// In production, might want to add a ListAccounts() method to AccountPool
	// For now, just report the count
	result.SampleAccounts = make([]AccountSummary, 0)

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
