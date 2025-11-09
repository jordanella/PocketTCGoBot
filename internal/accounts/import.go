package accounts

import (
	"database/sql"
	"fmt"
	"path/filepath"
)

// ImportResult tracks the results of an import operation
type ImportResult struct {
	TotalFiles    int
	Imported      int
	Skipped       int
	Failed        int
	Errors        []string
	ImportedIDs   []int64
}

// ImportFromDirectory imports all XML account files from a directory into the database
// Returns an ImportResult with statistics about the operation
func ImportFromDirectory(db *sql.DB, directory string) (*ImportResult, error) {
	result := &ImportResult{
		Errors:      make([]string, 0),
		ImportedIDs: make([]int64, 0),
	}

	// Load all XML files from directory
	accountFiles, err := LoadAccountsFromXML(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to load accounts from directory: %w", err)
	}

	result.TotalFiles = len(accountFiles)

	// Import each account
	for _, accountFile := range accountFiles {
		// Validate account has required fields
		if accountFile.DeviceAccount == "" || accountFile.DevicePassword == "" {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: missing credentials", accountFile.Filename))
			continue
		}

		// Check if account already exists
		var exists bool
		err := db.QueryRow(`
			SELECT COUNT(*) > 0
			FROM accounts
			WHERE device_account = ?
		`, accountFile.DeviceAccount).Scan(&exists)

		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: database query failed: %v", accountFile.Filename, err))
			continue
		}

		if exists {
			result.Skipped++
			continue
		}

		// Insert into database
		res, err := db.Exec(`
			INSERT INTO accounts (
				device_account,
				device_password,
				pool_status,
				failure_count,
				packs_opened,
				created_at,
				last_used_at
			) VALUES (?, ?, 'available', 0, 0, datetime('now'), NULL)
		`, accountFile.DeviceAccount, accountFile.DevicePassword)

		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: insert failed: %v", accountFile.Filename, err))
			continue
		}

		// Get the inserted ID
		id, err := res.LastInsertId()
		if err == nil {
			result.ImportedIDs = append(result.ImportedIDs, id)
		}

		result.Imported++
	}

	return result, nil
}

// ImportSingleFile imports a single XML account file into the database
// Returns the inserted account ID or error
func ImportSingleFile(db *sql.DB, filePath string) (int64, error) {
	// Parse the XML file
	dir := filepath.Dir(filePath)
	accountFiles, err := LoadAccountsFromXML(dir)
	if err != nil {
		return 0, fmt.Errorf("failed to load account: %w", err)
	}

	// Find the specific file
	filename := filepath.Base(filePath)
	var account *AccountFile
	for _, af := range accountFiles {
		if af.Filename == filename {
			account = af
			break
		}
	}

	if account == nil {
		return 0, fmt.Errorf("account file not found or invalid: %s", filename)
	}

	// Validate credentials
	if account.DeviceAccount == "" || account.DevicePassword == "" {
		return 0, fmt.Errorf("missing credentials in file")
	}

	// Check if already exists
	var exists bool
	err = db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM accounts
		WHERE device_account = ?
	`, account.DeviceAccount).Scan(&exists)

	if err != nil {
		return 0, fmt.Errorf("database query failed: %w", err)
	}

	if exists {
		return 0, fmt.Errorf("account already exists in database")
	}

	// Insert into database
	res, err := db.Exec(`
		INSERT INTO accounts (
			device_account,
			device_password,
			pool_status,
			failure_count,
			packs_opened,
			created_at,
			last_used_at
		) VALUES (?, ?, 'available', 0, 0, datetime('now'), NULL)
	`, account.DeviceAccount, account.DevicePassword)

	if err != nil {
		return 0, fmt.Errorf("insert failed: %w", err)
	}

	return res.LastInsertId()
}

// ExportToDirectory exports accounts from the database to XML files
// If accountIDs is nil, exports all accounts. Otherwise exports only specified IDs.
func ExportToDirectory(db *sql.DB, directory string, accountIDs []int64) (*ImportResult, error) {
	result := &ImportResult{
		Errors:      make([]string, 0),
		ImportedIDs: make([]int64, 0),
	}

	// Build query
	var rows *sql.Rows
	var err error

	if accountIDs == nil || len(accountIDs) == 0 {
		// Export all accounts
		rows, err = db.Query(`
			SELECT id, device_account, device_password
			FROM accounts
			WHERE device_account IS NOT NULL
			  AND device_password IS NOT NULL
			ORDER BY id
		`)
	} else {
		// Build placeholders for IN clause
		placeholders := ""
		args := make([]interface{}, len(accountIDs))
		for i, id := range accountIDs {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
			args[i] = id
		}

		query := fmt.Sprintf(`
			SELECT id, device_account, device_password
			FROM accounts
			WHERE id IN (%s)
			  AND device_account IS NOT NULL
			  AND device_password IS NOT NULL
			ORDER BY id
		`, placeholders)

		rows, err = db.Query(query, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}
	defer rows.Close()

	// Process each account
	for rows.Next() {
		var id int64
		var deviceAccount, devicePassword string

		if err := rows.Scan(&id, &deviceAccount, &devicePassword); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("ID %d: scan failed: %v", id, err))
			continue
		}

		result.TotalFiles++

		// Generate filename from account ID
		filename := fmt.Sprintf("account_%d.xml", id)

		// Save to XML
		err := SaveAccountToXML(directory, filename, deviceAccount, devicePassword)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("ID %d: export failed: %v", id, err))
			continue
		}

		result.Imported++
		result.ImportedIDs = append(result.ImportedIDs, id)
	}

	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}
