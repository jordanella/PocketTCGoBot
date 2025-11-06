package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDatabaseInitialization(t *testing.T) {
	// Create temporary test directory
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	// Open database
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	err = db.RunMigrations()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify schema version
	version, err := db.GetVersion()
	if err != nil {
		t.Fatalf("Failed to get version: %v", err)
	}

	if version != 7 {
		t.Errorf("Expected version 7, got %d", version)
	}

	// Verify file exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}
}

func TestAccountOperations(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = db.RunMigrations()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test CreateAccount
	account, err := db.CreateAccount("test_device_account", "test_password", "/path/to/file")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	if account.DeviceAccount != "test_device_account" {
		t.Errorf("Expected device_account 'test_device_account', got '%s'", account.DeviceAccount)
	}

	// Test GetAccountByID
	retrieved, err := db.GetAccountByID(account.ID)
	if err != nil {
		t.Fatalf("Failed to get account by ID: %v", err)
	}

	if retrieved.ID != account.ID {
		t.Errorf("Expected ID %d, got %d", account.ID, retrieved.ID)
	}

	// Test GetOrCreateAccount (should retrieve existing)
	existing, err := db.GetOrCreateAccount("test_device_account", "test_password")
	if err != nil {
		t.Fatalf("Failed to get or create account: %v", err)
	}

	if existing.ID != account.ID {
		t.Errorf("GetOrCreateAccount should have returned existing account")
	}

	// Test UpdateAccountResources
	err = db.UpdateAccountResources(account.ID, 1000, 500, 100, 50)
	if err != nil {
		t.Fatalf("Failed to update account resources: %v", err)
	}

	updated, err := db.GetAccountByID(account.ID)
	if err != nil {
		t.Fatalf("Failed to get updated account: %v", err)
	}

	if updated.Shinedust != 1000 || updated.Hourglasses != 500 {
		t.Errorf("Resources not updated correctly")
	}

	// Test ListActiveAccounts
	accounts, err := db.ListActiveAccounts()
	if err != nil {
		t.Fatalf("Failed to list active accounts: %v", err)
	}

	if len(accounts) != 1 {
		t.Errorf("Expected 1 active account, got %d", len(accounts))
	}
}

func TestActivityLogging(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = db.RunMigrations()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create test account
	account, err := db.CreateAccount("test_account", "password", "")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Test StartActivity
	activityID, err := db.StartActivity(account.ID, "wonder_pick", "DoWonderPick", "v1.0.0")
	if err != nil {
		t.Fatalf("Failed to start activity: %v", err)
	}

	if activityID == 0 {
		t.Error("Activity ID should not be 0")
	}

	// Test CompleteActivity
	time.Sleep(1100 * time.Millisecond) // Ensure at least 1 second duration
	err = db.CompleteActivity(activityID)
	if err != nil {
		t.Fatalf("Failed to complete activity: %v", err)
	}

	// Verify activity was completed
	activity, err := db.GetActivityByID(activityID)
	if err != nil {
		t.Fatalf("Failed to get activity: %v", err)
	}

	if activity.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", activity.Status)
	}

	if activity.CompletedAt == nil {
		t.Error("CompletedAt should not be nil")
	}

	if activity.DurationSeconds == nil {
		t.Error("DurationSeconds should not be nil")
	} else if *activity.DurationSeconds < 1 {
		t.Errorf("Expected duration >= 1 second, got %d", *activity.DurationSeconds)
	}

	// Test GetRunningActivities
	activityID2, err := db.StartActivity(account.ID, "pack_opening", "OpenPack", "v1.0.0")
	if err != nil {
		t.Fatalf("Failed to start second activity: %v", err)
	}

	running, err := db.GetRunningActivities()
	if err != nil {
		t.Fatalf("Failed to get running activities: %v", err)
	}

	if len(running) != 1 {
		t.Errorf("Expected 1 running activity, got %d", len(running))
	}

	if running[0].ID != int(activityID2) {
		t.Errorf("Expected running activity ID %d, got %d", activityID2, running[0].ID)
	}
}

func TestErrorLogging(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = db.RunMigrations()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create test account
	account, err := db.CreateAccount("test_account", "password", "")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Test LogError
	stackTrace := "stack trace here"
	screenState := "HomeScreen"
	templateName := "error_popup"
	actionName := "ClickButton"

	errorID, err := db.LogError(
		&account.ID,
		nil,
		"ErrorPopup",
		"high",
		"Unexpected popup appeared",
		&stackTrace,
		&screenState,
		&templateName,
		&actionName,
	)

	if err != nil {
		t.Fatalf("Failed to log error: %v", err)
	}

	if errorID == 0 {
		t.Error("Error ID should not be 0")
	}

	// Test GetErrorByID
	errorLog, err := db.GetErrorByID(errorID)
	if err != nil {
		t.Fatalf("Failed to get error: %v", err)
	}

	if errorLog.ErrorType != "ErrorPopup" {
		t.Errorf("Expected error type 'ErrorPopup', got '%s'", errorLog.ErrorType)
	}

	if errorLog.WasRecovered {
		t.Error("Error should not be marked as recovered yet")
	}

	// Test MarkErrorRecovered
	err = db.MarkErrorRecovered(errorID, "Dismissed popup", 1500)
	if err != nil {
		t.Fatalf("Failed to mark error as recovered: %v", err)
	}

	// Verify recovery
	recoveredError, err := db.GetErrorByID(errorID)
	if err != nil {
		t.Fatalf("Failed to get recovered error: %v", err)
	}

	if !recoveredError.WasRecovered {
		t.Error("Error should be marked as recovered")
	}

	if recoveredError.RecoveryAction == nil || *recoveredError.RecoveryAction != "Dismissed popup" {
		t.Error("Recovery action not set correctly")
	}

	// Test GetUnrecoveredErrors
	db.LogError(&account.ID, nil, "ErrorStuck", "critical", "Bot is stuck", nil, nil, nil, nil)
	unrecovered, err := db.GetUnrecoveredErrors(100)
	if err != nil {
		t.Fatalf("Failed to get unrecovered errors: %v", err)
	}

	if len(unrecovered) != 1 {
		t.Errorf("Expected 1 unrecovered error, got %d", len(unrecovered))
	}
}

func TestPackTracking(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = db.RunMigrations()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create test account
	account, err := db.CreateAccount("test_account", "password", "")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	// Test LogPackOpening
	rarityBreakdown := map[string]int{
		"1_diamond": 3,
		"2_diamond": 1,
		"3_diamond": 1,
	}

	packName := "Genetic Apex"
	packID, err := db.LogPackOpening(
		account.ID,
		nil,
		"genetic_apex",
		&packName,
		false,
		5,
		rarityBreakdown,
		5,
	)

	if err != nil {
		t.Fatalf("Failed to log pack opening: %v", err)
	}

	// Test LogCardPulled
	cardName := "Pikachu"
	cardNumber := "001/165"
	cardType := "pokemon"
	confidence := 0.95

	cardID, err := db.LogCardPulled(
		packID,
		account.ID,
		"pikachu_001",
		&cardName,
		&cardNumber,
		"3_diamond",
		&cardType,
		false,
		false,
		&confidence,
	)

	if err != nil {
		t.Fatalf("Failed to log card pulled: %v", err)
	}

	if cardID == 0 {
		t.Error("Card ID should not be 0")
	}

	// Test GetCardsFromPack
	cards, err := db.GetCardsFromPack(packID)
	if err != nil {
		t.Fatalf("Failed to get cards from pack: %v", err)
	}

	if len(cards) != 1 {
		t.Errorf("Expected 1 card, got %d", len(cards))
	}

	// Test GetAccountCollection
	collection, err := db.GetAccountCollection(account.ID)
	if err != nil {
		t.Fatalf("Failed to get account collection: %v", err)
	}

	if len(collection) != 1 {
		t.Errorf("Expected 1 card in collection, got %d", len(collection))
	}

	if collection[0].Quantity != 1 {
		t.Errorf("Expected quantity 1, got %d", collection[0].Quantity)
	}

	// Pull the same card again
	_, err = db.LogCardPulled(
		packID,
		account.ID,
		"pikachu_001",
		&cardName,
		&cardNumber,
		"3_diamond",
		&cardType,
		false,
		false,
		&confidence,
	)

	if err != nil {
		t.Fatalf("Failed to log second card: %v", err)
	}

	// Verify quantity increased
	collection, err = db.GetAccountCollection(account.ID)
	if err != nil {
		t.Fatalf("Failed to get updated collection: %v", err)
	}

	if collection[0].Quantity != 2 {
		t.Errorf("Expected quantity 2, got %d", collection[0].Quantity)
	}
}

func TestTransactions(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	err = db.RunMigrations()
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test transaction rollback on error
	err = db.ExecTx(func(tx *sql.Tx) error {
		_, err := tx.Exec("INSERT INTO accounts (device_account, device_password) VALUES (?, ?)", "test", "pass")
		if err != nil {
			return err
		}

		// Force an error to trigger rollback
		_, err = tx.Exec("INVALID SQL QUERY")
		return err
	})

	// Since transaction should rollback, account should not exist
	accounts, err := db.ListActiveAccounts()
	if err != nil {
		t.Fatalf("Failed to list accounts: %v", err)
	}

	if len(accounts) != 0 {
		t.Error("Transaction did not rollback correctly")
	}
}
