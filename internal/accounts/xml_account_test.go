package accounts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAccountsFromXML(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()

	// Create a test XML file in Android SharedPreferences format
	testXML := `<?xml version='1.0' encoding='utf-8' standalone='yes' ?>
<map>
    <string name="deviceAccount">test_account</string>
    <string name="devicePassword">test_password</string>
</map>`

	testFile := filepath.Join(tempDir, "test_account.xml")
	if err := os.WriteFile(testFile, []byte(testXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load accounts
	accounts, err := LoadAccountsFromXML(tempDir)
	if err != nil {
		t.Fatalf("Failed to load accounts: %v", err)
	}

	// Verify
	if len(accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(accounts))
	}

	if accounts[0].Filename != "test_account.xml" {
		t.Errorf("Expected filename 'test_account.xml', got '%s'", accounts[0].Filename)
	}

	if accounts[0].DeviceAccount != "test_account" {
		t.Errorf("Expected device account 'test_account', got '%s'", accounts[0].DeviceAccount)
	}

	if accounts[0].DevicePassword != "test_password" {
		t.Errorf("Expected device password 'test_password', got '%s'", accounts[0].DevicePassword)
	}
}

func TestLoadAccountsFromXMLLegacyFormat(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()

	// Create a test XML file in legacy format
	testXML := `<?xml version="1.0" encoding="UTF-8"?>
<XMLAccount>
  <deviceAccount>legacy_account</deviceAccount>
  <devicePassword>legacy_password</devicePassword>
</XMLAccount>`

	testFile := filepath.Join(tempDir, "legacy_account.xml")
	if err := os.WriteFile(testFile, []byte(testXML), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load accounts
	accounts, err := LoadAccountsFromXML(tempDir)
	if err != nil {
		t.Fatalf("Failed to load accounts: %v", err)
	}

	// Verify
	if len(accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(accounts))
	}

	if accounts[0].DeviceAccount != "legacy_account" {
		t.Errorf("Expected device account 'legacy_account', got '%s'", accounts[0].DeviceAccount)
	}

	if accounts[0].DevicePassword != "legacy_password" {
		t.Errorf("Expected device password 'legacy_password', got '%s'", accounts[0].DevicePassword)
	}
}

func TestSaveAccountToXML(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()

	// Save account
	err := SaveAccountToXML(tempDir, "new_account.xml", "new_device", "new_password")
	if err != nil {
		t.Fatalf("Failed to save account: %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(tempDir, "new_account.xml")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Account file was not created")
	}

	// Load and verify
	accounts, err := LoadAccountsFromXML(tempDir)
	if err != nil {
		t.Fatalf("Failed to load accounts: %v", err)
	}

	if len(accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(accounts))
	}

	if accounts[0].DeviceAccount != "new_device" {
		t.Errorf("Expected device account 'new_device', got '%s'", accounts[0].DeviceAccount)
	}
}

func TestDeleteAccountXML(t *testing.T) {
	// Create temp directory for testing
	tempDir := t.TempDir()

	// Save account
	err := SaveAccountToXML(tempDir, "delete_test.xml", "test_device", "test_password")
	if err != nil {
		t.Fatalf("Failed to save account: %v", err)
	}

	filePath := filepath.Join(tempDir, "delete_test.xml")

	// Delete account
	err = DeleteAccountXML(filePath)
	if err != nil {
		t.Fatalf("Failed to delete account: %v", err)
	}

	// Verify file doesn't exist
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Errorf("Account file still exists after deletion")
	}
}
