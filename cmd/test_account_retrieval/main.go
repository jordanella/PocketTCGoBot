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
			id TEXT PRIMARY KEY,
			xml_path TEXT,
			pack_count INTEGER,
			last_modified TEXT,
			status TEXT,
			failure_count INTEGER,
			last_error TEXT,
			completed_at TEXT
		);
		DELETE FROM accounts;
	`)
	if err != nil {
		log.Fatal(err)
	}

	// Insert test accounts and create XML files
	testAccounts := []struct {
		id     string
		packs  int
		status string
	}{
		{"test001", 15, "available"},
		{"test002", 22, "available"},
		{"test003", 18, "available"},
	}

	stmt, _ := db.Prepare("INSERT INTO accounts (id, xml_path, pack_count, status, failure_count, last_modified) VALUES (?, ?, ?, ?, 0, datetime('now'))")
	for _, acc := range testAccounts {
		xmlPath := filepath.Join(testAccountsDir, acc.id+".xml")
		os.WriteFile(xmlPath, []byte("<account/>"), 0644)
		stmt.Exec(acc.id, xmlPath, acc.packs, acc.status)
	}
	stmt.Close()

	fmt.Println("✓ Created 3 test accounts with XML files\n")

	// Create pool manager
	poolManager := accountpool.NewPoolManager(poolsDir, db)
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

	for i := 0; i < min(3, stats.Available); i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		account, err := pool.GetNext(ctx)
		cancel()

		if err != nil {
			fmt.Printf("  ✗ Account %d: Failed - %v\n", i+1, err)
			break
		}

		fmt.Printf("  ✓ Account %d: %s (packs: %d, status: %s)\n", i+1, account.ID, account.PackCount, account.Status)
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
