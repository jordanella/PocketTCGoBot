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

func main() {
	fmt.Println("=== Account Retrieval Debug Test ===\n")

	// Setup
	projectRoot := findProjectRoot(".")
	poolsDir := filepath.Join(projectRoot, "pools")
	dbPath := filepath.Join(projectRoot, "accounts.db")
	testAccountsDir := filepath.Join(projectRoot, "test_accounts")

	// Ensure test data exists
	os.MkdirAll(testAccountsDir, 0755)

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create simple test data
	fmt.Println("Creating test data...")
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_account TEXT NOT NULL UNIQUE,
			device_password TEXT NOT NULL,
			packs_opened INTEGER DEFAULT 0,
			last_used_at DATETIME,
			pool_status TEXT DEFAULT 'available',
			failure_count INTEGER DEFAULT 0,
			last_error TEXT,
			completed_at DATETIME
		);
		DELETE FROM accounts;
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Insert test accounts
	testAccounts := []struct {
		account  string
		password string
		packs    int
	}{
		{"test001@example.com", "password001", 15},
		{"test002@example.com", "password002", 22},
		{"test003@example.com", "password003", 18},
	}

	for _, acc := range testAccounts {
		_, err := db.Exec("INSERT INTO accounts (device_account, device_password, packs_opened, pool_status, failure_count, last_used_at) VALUES (?, ?, ?, 'available', 0, datetime('now'))",
			acc.account, acc.password, acc.packs)
		if err != nil {
			log.Fatalf("Failed to insert account %s: %v", acc.account, err)
		}
	}

	fmt.Println("✓ Created 3 test accounts in database\n")

	// Create pool manager
	poolManager := accountpool.NewPoolManager(poolsDir, db, "account_xmls")
	poolManager.DiscoverPools()

	// Get Premium Farmers Pool
	fmt.Println("Getting pool instance...")
	pool, err := poolManager.GetPool("Premium Farmers Pool")
	if err != nil {
		log.Fatalf("Failed to get pool: %v", err)
	}

	// Check stats
	stats := pool.GetStats()
	fmt.Printf("\nInitial Pool Stats:\n")
	fmt.Printf("  Total: %d\n", stats.Total)
	fmt.Printf("  Available: %d\n", stats.Available)
	fmt.Printf("  In Use: %d\n", stats.InUse)

	if stats.Available == 0 {
		fmt.Println("\n⚠ No accounts showing as available!")
		fmt.Println("This is the problem we're debugging.\n")

		// Try to manually inspect the pool
		// We can't directly access internal fields, but let's try GetNext with a very short timeout
		fmt.Println("Attempting GetNext with 100ms timeout...")
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		account, err := pool.GetNext(ctx)
		if err != nil {
			fmt.Printf("✗ GetNext failed: %v\n", err)
			fmt.Println("\nThe pool has accounts (Total > 0) but GetNext can't retrieve them.")
			fmt.Println("This suggests the available channel isn't being filled.\n")
		} else {
			fmt.Printf("✓ Got account: %s\n", account.ID)
		}

		return
	}

	// Try to get accounts
	fmt.Println("\nAttempting to retrieve accounts...")

	retrievedAccounts := []*accountpool.Account{}
	for i := 0; i < min(3, stats.Available); i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		account, err := pool.GetNext(ctx)
		cancel()

		if err != nil {
			fmt.Printf("  ✗ Account %d: Failed - %v\n", i+1, err)
			break
		}

		fmt.Printf("  ✓ Account %d: %s (packs: %d, status: %s)\n", i+1, account.ID, account.PackCount, account.Status)
		retrievedAccounts = append(retrievedAccounts, account)
	}

	// Return accounts to pool
	fmt.Println("\nReturning accounts to pool...")
	for i, account := range retrievedAccounts {
		if err := pool.Return(account); err != nil {
			fmt.Printf("  ✗ Failed to return account %d: %v\n", i+1, err)
		} else {
			fmt.Printf("  ✓ Returned account %s\n", account.ID)
		}
	}

	// Final stats
	fmt.Println("\nFinal Pool Stats:")
	finalStats := pool.GetStats()
	fmt.Printf("  Total: %d\n", finalStats.Total)
	fmt.Printf("  Available: %d\n", finalStats.Available)
	fmt.Printf("  In Use: %d\n", finalStats.InUse)

	pool.Close()
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
