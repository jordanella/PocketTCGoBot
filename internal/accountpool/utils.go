package accountpool

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// extractXMLTag extracts content from <tag>content</tag>
// This is a shared utility used by both pool_manager.go and unified_pool.go
func extractXMLTag(xml, tag string) string {
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
// This is a shared utility used by both pool_manager.go and unified_pool.go
func importAccountToDB(db *sql.DB, account *Account) error {
	query := `
		INSERT INTO accounts (device_account, device_password, created_at, last_used_at)
		VALUES (?, ?, CURRENT_TIMESTAMP, NULL)
		ON CONFLICT(device_account) DO UPDATE SET
			device_password = excluded.device_password
	`

	_, err := db.Exec(query, account.DeviceAccount, account.DevicePassword)
	return err
}

// copyToGlobalStorage copies an XML file to global storage directory
// This is a shared utility used by both pool_manager.go and unified_pool.go
func copyToGlobalStorage(sourcePath, destDir, deviceAccount string) error {
	destPath := filepath.Join(destDir, deviceAccount+".xml")

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
