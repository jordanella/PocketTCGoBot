package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"jordanella.com/pocket-tcg-go/internal/accountpool"

	_ "github.com/mattn/go-sqlite3"
)

// Test program to validate YAML pool system
func main() {
	fmt.Println("=== Account Pool System Test ===\n")

	// Get project root directory
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Find project root by looking for go.mod
	projectRoot := findProjectRoot(workDir)
	if projectRoot == "" {
		log.Fatalf("Could not find project root (no go.mod found)")
	}

	fmt.Printf("Project root: %s\n\n", projectRoot)

	poolsDir := filepath.Join(projectRoot, "pools")
	dbPath := filepath.Join(projectRoot, "accounts.db")

	// Check if pools directory exists
	if _, err := os.Stat(poolsDir); os.IsNotExist(err) {
		log.Fatalf("Pools directory does not exist: %s", poolsDir)
	}

	// Check if database exists
	dbExists := true
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		dbExists = false
		fmt.Printf("⚠ Database not found at %s\n", dbPath)
		fmt.Println("  SQL pools will fail, but file pools should work\n")
	}

	// Open database (create if doesn't exist)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if !dbExists {
		fmt.Println("Creating test database schema...")
		if err := createTestSchema(db); err != nil {
			log.Fatalf("Failed to create schema: %v", err)
		}
		fmt.Println("✓ Schema created\n")
	}

	// Test 1: Create PoolManager
	fmt.Println("=== Test 1: Create PoolManager ===")
	poolManager := accountpool.NewPoolManager(poolsDir, db, "account_xmls")
	fmt.Println("✓ PoolManager created\n")

	// Test 2: Discover Pools
	fmt.Println("=== Test 2: Discover Pools ===")
	if err := poolManager.DiscoverPools(); err != nil {
		log.Fatalf("Failed to discover pools: %v", err)
	}

	pools := poolManager.ListPools()
	fmt.Printf("✓ Discovered %d pools:\n", len(pools))
	for i, poolName := range pools {
		fmt.Printf("  %d. %s\n", i+1, poolName)
	}
	fmt.Println()

	// Test 3: Inspect Pool Definitions
	fmt.Println("=== Test 3: Inspect Pool Definitions ===")
	for _, poolName := range pools {
		poolDef, err := poolManager.GetPoolDefinition(poolName)
		if err != nil {
			fmt.Printf("✗ Failed to get definition for '%s': %v\n", poolName, err)
			continue
		}

		fmt.Printf("\nPool: %s\n", poolDef.Name)
		fmt.Printf("  Type: unified\n")
		fmt.Printf("  File: %s\n", poolDef.FilePath)
		fmt.Printf("  Description: %s\n", poolDef.Config.Description)
		fmt.Printf("  Queries: %d\n", len(poolDef.Config.Queries))
		fmt.Printf("  Includes: %d\n", len(poolDef.Config.Include))
		fmt.Printf("  Excludes: %d\n", len(poolDef.Config.Exclude))
		fmt.Printf("  Watched Paths: %d\n", len(poolDef.Config.WatchedPaths))
	}
	fmt.Println()

	// Test 4: Test Pools (without creating instances)
	fmt.Println("=== Test 4: Test Pools ===")
	for _, poolName := range pools {
		fmt.Printf("\nTesting pool: %s\n", poolName)

		testResult, err := poolManager.TestPool(poolName)
		if err != nil {
			fmt.Printf("  ✗ Test failed: %v\n", err)
			continue
		}

		if testResult.Success {
			fmt.Printf("  ✓ Test passed\n")
			fmt.Printf("  Accounts found: %d\n", testResult.AccountsFound)
		} else {
			fmt.Printf("  ✗ Test failed: %s\n", testResult.Error)
		}
	}
	fmt.Println()

	// Test 5: Create Pool Instance and Get Accounts
	if len(pools) > 0 {
		fmt.Println("=== Test 5: Create Pool Instance ===")
		testPoolName := pools[0]
		fmt.Printf("\nCreating instance of pool: %s\n", testPoolName)

		pool, err := poolManager.GetPool(testPoolName)
		if err != nil {
			fmt.Printf("✗ Failed to create pool instance: %v\n", err)
		} else {
			fmt.Println("✓ Pool instance created")

			// Get pool stats
			stats := pool.GetStats()
			fmt.Printf("\nPool Statistics:\n")
			fmt.Printf("  Total: %d\n", stats.Total)
			fmt.Printf("  Available: %d\n", stats.Available)
			fmt.Printf("  In Use: %d\n", stats.InUse)
			fmt.Printf("  Completed: %d\n", stats.Completed)
			fmt.Printf("  Failed: %d\n", stats.Failed)
			fmt.Printf("  Skipped: %d\n", stats.Skipped)

			// Close pool
			if err := pool.Close(); err != nil {
				fmt.Printf("⚠ Failed to close pool: %v\n", err)
			}
		}
		fmt.Println()
	}

	// Cleanup
	fmt.Println("=== Cleanup ===")
	if err := poolManager.CloseAll(); err != nil {
		fmt.Printf("⚠ Failed to close all pools: %v\n", err)
	} else {
		fmt.Println("✓ All pools closed")
	}

	fmt.Println("\n=== Test Complete ===")
}

// findProjectRoot walks up the directory tree looking for go.mod
func findProjectRoot(start string) string {
	current := start
	for {
		goModPath := filepath.Join(current, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root
			return ""
		}
		current = parent
	}
}

// createTestSchema creates a minimal accounts table for testing
func createTestSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS accounts (
		id TEXT PRIMARY KEY,
		xml_path TEXT NOT NULL,
		pack_count INTEGER DEFAULT 0,
		last_modified TEXT,
		status TEXT DEFAULT 'available',
		failure_count INTEGER DEFAULT 0,
		last_error TEXT,
		completed_at TEXT
	);

	-- Insert some test data
	INSERT OR IGNORE INTO accounts (id, xml_path, pack_count, status, failure_count, last_modified) VALUES
		('test001', './accounts/test001.xml', 15, 'available', 0, datetime('now')),
		('test002', './accounts/test002.xml', 22, 'available', 0, datetime('now')),
		('test003', './accounts/test003.xml', 8, 'available', 0, datetime('now')),
		('test004', './accounts/test004.xml', 31, 'available', 0, datetime('now')),
		('test005', './accounts/test005.xml', 5, 'available', 0, datetime('now')),
		('test006', './accounts/test006.xml', 12, 'failed', 1, datetime('now')),
		('test007', './accounts/test007.xml', 19, 'failed', 2, datetime('now')),
		('test008', './accounts/test008.xml', 3, 'available', 0, datetime('now')),
		('test009', './accounts/test009.xml', 27, 'skipped', 1, datetime('now')),
		('test010', './accounts/test010.xml', 14, 'available', 0, datetime('now'));
	`

	_, err := db.Exec(schema)
	return err
}
