package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"jordanella.com/pocket-tcg-go/internal/accounts"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Command line flags
	importDir := flag.String("dir", "", "Directory containing XML account files to import")
	exportDir := flag.String("export", "", "Directory to export accounts to (exports all if specified)")
	dbPath := flag.String("db", "accounts.db", "Path to database file")
	flag.Parse()

	if *importDir == "" && *exportDir == "" {
		fmt.Println("Usage:")
		fmt.Println("  Import: import_accounts -dir <directory> [-db <database>]")
		fmt.Println("  Export: import_accounts -export <directory> [-db <database>]")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  import_accounts -dir ./xml_accounts")
		fmt.Println("  import_accounts -export ./exported_accounts")
		os.Exit(1)
	}

	// Find project root to locate database
	projectRoot := findProjectRoot(".")
	fullDBPath := filepath.Join(projectRoot, *dbPath)

	// Open database
	db, err := sql.Open("sqlite3", fullDBPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	if *importDir != "" {
		performImport(db, *importDir)
	}

	if *exportDir != "" {
		performExport(db, *exportDir)
	}
}

func performImport(db *sql.DB, directory string) {
	fmt.Printf("=== Importing Accounts from %s ===\n\n", directory)

	// Check if directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		log.Fatalf("Directory does not exist: %s", directory)
	}

	// Import accounts
	result, err := accounts.ImportFromDirectory(db, directory)
	if err != nil {
		log.Fatalf("Import failed: %v", err)
	}

	// Display results
	fmt.Printf("Import Summary:\n")
	fmt.Printf("  Total files:     %d\n", result.TotalFiles)
	fmt.Printf("  Imported:        %d\n", result.Imported)
	fmt.Printf("  Skipped:         %d (already in database)\n", result.Skipped)
	fmt.Printf("  Failed:          %d\n", result.Failed)
	fmt.Println()

	if len(result.Errors) > 0 {
		fmt.Println("Errors:")
		for _, errMsg := range result.Errors {
			fmt.Printf("  - %s\n", errMsg)
		}
		fmt.Println()
	}

	if result.Imported > 0 {
		fmt.Printf("✓ Successfully imported %d accounts\n", result.Imported)
		fmt.Println("\nImported account IDs:", result.ImportedIDs)
	}
}

func performExport(db *sql.DB, directory string) {
	fmt.Printf("=== Exporting Accounts to %s ===\n\n", directory)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		log.Fatalf("Failed to create export directory: %v", err)
	}

	// Export all accounts
	result, err := accounts.ExportToDirectory(db, directory, nil)
	if err != nil {
		log.Fatalf("Export failed: %v", err)
	}

	// Display results
	fmt.Printf("Export Summary:\n")
	fmt.Printf("  Total accounts:  %d\n", result.TotalFiles)
	fmt.Printf("  Exported:        %d\n", result.Imported)
	fmt.Printf("  Failed:          %d\n", result.Failed)
	fmt.Println()

	if len(result.Errors) > 0 {
		fmt.Println("Errors:")
		for _, errMsg := range result.Errors {
			fmt.Printf("  - %s\n", errMsg)
		}
		fmt.Println()
	}

	if result.Imported > 0 {
		fmt.Printf("✓ Successfully exported %d accounts to %s\n", result.Imported, directory)
	}
}

func findProjectRoot(start string) string {
	current, _ := filepath.Abs(start)
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "."
		}
		current = parent
	}
}
