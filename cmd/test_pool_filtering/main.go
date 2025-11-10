package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"jordanella.com/pocket-tcg-go/internal/accountpool"

	_ "github.com/mattn/go-sqlite3"
)

// Test program to validate pool filtering and account retrieval
func main() {
	fmt.Println("=== Pool Filtering Test ===\n")

	// Setup
	projectRoot := findProjectRoot(".")
	if projectRoot == "" {
		log.Fatal("Could not find project root")
	}

	poolsDir := filepath.Join(projectRoot, "pools")
	dbPath := filepath.Join(projectRoot, "accounts.db")
	testAccountsDir := filepath.Join(projectRoot, "test_accounts")

	// Create test accounts directory
	if err := os.MkdirAll(testAccountsDir, 0755); err != nil {
		log.Fatalf("Failed to create test_accounts directory: %v", err)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create schema and populate with test data
	fmt.Println("=== Setting up test data ===")
	if err := setupTestData(db, testAccountsDir); err != nil {
		log.Fatalf("Failed to setup test data: %v", err)
	}
	fmt.Println("✓ Test data created\n")

	// Create PoolManager
	poolManager := accountpool.NewPoolManager(poolsDir, db, "account_xmls")
	if err := poolManager.DiscoverPools(); err != nil {
		log.Fatalf("Failed to discover pools: %v", err)
	}

	pools := poolManager.ListPools()
	fmt.Printf("Discovered %d pools\n\n", len(pools))

	// Test each SQL pool's filtering
	testSQLPoolFiltering(poolManager, "Premium Farmers Pool", db)
	testSQLPoolFiltering(poolManager, "Fresh Account Pool", db)
	testSQLPoolFiltering(poolManager, "High Value Retry Pool", db)

	// Test file pool
	testFilePool(poolManager, "Test File Pool")

	// Test account retrieval
	testAccountRetrieval(poolManager, "Premium Farmers Pool")

	// Cleanup
	fmt.Println("\n=== Cleanup ===")
	poolManager.CloseAll()
	fmt.Println("✓ All pools closed")
}

func setupTestData(db *sql.DB, testAccountsDir string) error {
	// Create schema
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
	DELETE FROM accounts;
	`
	if _, err := db.Exec(schema); err != nil {
		return err
	}

	// Insert diverse test data
	testData := []struct {
		id           string
		packs        int
		status       string
		failures     int
		lastError    string
		completedAt  string
	}{
		// Premium accounts (10+ packs, <3 failures)
		{"premium001", 25, "available", 0, "", ""},
		{"premium002", 15, "available", 1, "", ""},
		{"premium003", 31, "skipped", 2, "timeout", ""},
		{"premium004", 12, "available", 0, "", ""},
		{"premium005", 40, "available", 2, "", ""},

		// Fresh accounts (available, 0 failures, never completed, 5+ packs)
		{"fresh001", 8, "available", 0, "", ""},
		{"fresh002", 12, "available", 0, "", ""},
		{"fresh003", 6, "available", 0, "", ""},
		{"fresh004", 20, "available", 0, "", ""},

		// Retry candidates (failed, 1-2 failures, 15+ packs, not banned)
		{"retry001", 18, "failed", 1, "connection timeout", ""},
		{"retry002", 22, "failed", 2, "login failed", ""},
		{"retry003", 16, "skipped", 1, "template not found", ""},

		// Should NOT match any pool
		{"excluded001", 45, "failed", 5, "too many failures", ""},              // Too many failures
		{"excluded002", 8, "available", 0, "", ""},                              // < 10 packs for premium
		{"excluded003", 20, "banned", 1, "banned", ""},                         // Banned
		{"excluded004", 15, "available", 0, "", "2025-01-01T00:00:00Z"},       // Already completed
	}

	stmt, err := db.Prepare(`
		INSERT INTO accounts (id, xml_path, pack_count, status, failure_count, last_error, completed_at, last_modified)
		VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, data := range testData {
		xmlPath := filepath.Join(testAccountsDir, data.id+".xml")

		// Create dummy XML file
		if err := os.WriteFile(xmlPath, []byte("<account/>"), 0644); err != nil {
			return err
		}

		if _, err := stmt.Exec(data.id, xmlPath, data.packs, data.status, data.failures, data.lastError, data.completedAt); err != nil {
			return err
		}
	}

	return nil
}

func testSQLPoolFiltering(poolManager *accountpool.PoolManager, poolName string, db *sql.DB) {
	fmt.Printf("=== Testing: %s ===\n", poolName)

	// Get expected count from database
	poolDef, err := poolManager.GetPoolDefinition(poolName)
	if err != nil {
		fmt.Printf("✗ Failed to get pool definition: %v\n\n", err)
		return
	}

	// Skip if no queries defined
	if poolDef.Config == nil || len(poolDef.Config.Queries) == 0 {
		fmt.Println("Skipping pool with no queries\n")
		return
	}

	// Test first query
	query := poolDef.Config.Queries[0]

	// Build parameters
	params := make([]interface{}, 0)
	for _, val := range query.Parameters {
		params = append(params, val)
	}

	// Execute query to see what we'd get
	rows, err := db.Query(query.SQL, params...)
	if err != nil {
		fmt.Printf("✗ Query failed: %v\n\n", err)
		return
	}
	defer rows.Close()

	var ids []string
	var packCounts []int
	for rows.Next() {
		var id, xmlPath, lastModified, status, lastError string
		var packCount, failureCount int
		if err := rows.Scan(&id, &xmlPath, &packCount, &lastModified, &status, &failureCount, &lastError); err != nil {
			fmt.Printf("✗ Scan failed: %v\n\n", err)
			return
		}
		ids = append(ids, id)
		packCounts = append(packCounts, packCount)
	}

	fmt.Printf("Query returned %d accounts:\n", len(ids))
	for i, id := range ids {
		fmt.Printf("  - %s (packs: %d)\n", id, packCounts[i])
	}

	// Test pool
	testResult, err := poolManager.TestPool(poolName)
	if err != nil {
		fmt.Printf("✗ Pool test failed: %v\n\n", err)
		return
	}

	if !testResult.Success {
		fmt.Printf("✗ Pool test reported failure: %s\n\n", testResult.Error)
		return
	}

	fmt.Printf("✓ Pool loaded %d accounts\n", testResult.AccountsFound)

	if testResult.AccountsFound != len(ids) {
		fmt.Printf("⚠ Mismatch: query returned %d but pool found %d\n", len(ids), testResult.AccountsFound)
		fmt.Println("  (This may be due to missing XML files)")
	}

	fmt.Println()
}

func testFilePool(poolManager *accountpool.PoolManager, poolName string) {
	fmt.Printf("=== Testing: %s ===\n", poolName)

	testResult, err := poolManager.TestPool(poolName)
	if err != nil {
		fmt.Printf("✗ Pool test failed: %v\n\n", err)
		return
	}

	if !testResult.Success {
		fmt.Printf("✗ Pool test reported failure: %s\n\n", testResult.Error)
		return
	}

	fmt.Printf("✓ File pool loaded %d accounts\n", testResult.AccountsFound)
	fmt.Println()
}

func testAccountRetrieval(poolManager *accountpool.PoolManager, poolName string) {
	fmt.Printf("=== Testing Account Retrieval: %s ===\n", poolName)

	pool, err := poolManager.GetPool(poolName)
	if err != nil {
		fmt.Printf("✗ Failed to get pool: %v\n\n", err)
		return
	}

	stats := pool.GetStats()
	fmt.Printf("Pool stats:\n")
	fmt.Printf("  Total: %d\n", stats.Total)
	fmt.Printf("  Available: %d\n", stats.Available)

	if stats.Available == 0 {
		fmt.Println("No accounts available to retrieve\n")
		return
	}

	// Try to get an account
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	account, err := pool.GetNext(ctx)
	if err != nil {
		fmt.Printf("✗ Failed to get account: %v\n\n", err)
		return
	}

	fmt.Printf("✓ Retrieved account:\n")
	fmt.Printf("  ID: %s\n", account.ID)
	fmt.Printf("  Packs: %d\n", account.PackCount)
	fmt.Printf("  Status: %s\n", account.Status)
	fmt.Printf("  XML Path: %s\n", account.XMLPath)

	// Return account
	if err := pool.Return(account); err != nil {
		fmt.Printf("⚠ Failed to return account: %v\n", err)
	} else {
		fmt.Println("✓ Account returned to pool")
	}

	fmt.Println()
}

func findProjectRoot(start string) string {
	current, _ := filepath.Abs(start)
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}
