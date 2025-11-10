package database

import (
	"database/sql"
	"fmt"
	"time"
)

// CheckoutAccount atomically checks out an account to an emulator instance
// Returns error if account is already checked out to a different active orchestration
func CheckoutAccount(db *sql.DB, deviceAccount string, orchestrationID string, emulatorInstance int) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// First, check if account is already checked out
	var existingOrchestration sql.NullString
	var existingInstance sql.NullInt64
	var checkedOutAt sql.NullTime

	err = tx.QueryRow(`
		SELECT checked_out_to_orchestration, checked_out_to_instance, checked_out_at
		FROM accounts
		WHERE device_account = ?
	`, deviceAccount).Scan(&existingOrchestration, &existingInstance, &checkedOutAt)

	if err != nil {
		return fmt.Errorf("failed to query account checkout status: %w", err)
	}

	// If account is checked out to a different orchestration, check if it's still active
	if existingOrchestration.Valid && existingOrchestration.String != orchestrationID {
		// TODO: Add orchestration health check here
		// For now, if checkout is older than 10 minutes, we can reclaim it
		if checkedOutAt.Valid && time.Since(checkedOutAt.Time) < 10*time.Minute {
			return fmt.Errorf("account %s is already checked out to orchestration %s (instance %d)",
				deviceAccount, existingOrchestration.String, existingInstance.Int64)
		}
		// Stale checkout, we can reclaim it
	}

	// Check out the account
	_, err = tx.Exec(`
		UPDATE accounts
		SET checked_out_to_orchestration = ?,
		    checked_out_to_instance = ?,
		    checked_out_at = datetime('now')
		WHERE device_account = ?
	`, orchestrationID, emulatorInstance, deviceAccount)

	if err != nil {
		return fmt.Errorf("failed to checkout account: %w", err)
	}

	return tx.Commit()
}

// ReleaseAccount clears the checkout information for an account
func ReleaseAccount(db *sql.DB, deviceAccount string, orchestrationID string) error {
	result, err := db.Exec(`
		UPDATE accounts
		SET checked_out_to_orchestration = NULL,
		    checked_out_to_instance = NULL,
		    checked_out_at = NULL
		WHERE device_account = ?
		AND checked_out_to_orchestration = ?
	`, deviceAccount, orchestrationID)

	if err != nil {
		return fmt.Errorf("failed to release account: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("account %s was not checked out to orchestration %s", deviceAccount, orchestrationID)
	}

	return nil
}

// IsAccountCheckedOut checks if an account is currently checked out
func IsAccountCheckedOut(db *sql.DB, deviceAccount string) (bool, string, int, error) {
	var orchestrationID sql.NullString
	var instance sql.NullInt64
	var checkedOutAt sql.NullTime

	err := db.QueryRow(`
		SELECT checked_out_to_orchestration, checked_out_to_instance, checked_out_at
		FROM accounts
		WHERE device_account = ?
	`, deviceAccount).Scan(&orchestrationID, &instance, &checkedOutAt)

	if err != nil {
		return false, "", 0, fmt.Errorf("failed to query checkout status: %w", err)
	}

	if !orchestrationID.Valid {
		return false, "", 0, nil
	}

	// Check if checkout is stale (older than 10 minutes)
	if checkedOutAt.Valid && time.Since(checkedOutAt.Time) > 10*time.Minute {
		return false, "", 0, nil // Consider stale checkouts as not checked out
	}

	return true, orchestrationID.String, int(instance.Int64), nil
}

// GetAccountsCheckedOutByOrchestration returns all accounts checked out by a specific orchestration
func GetAccountsCheckedOutByOrchestration(db *sql.DB, orchestrationID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT device_account
		FROM accounts
		WHERE checked_out_to_orchestration = ?
		ORDER BY checked_out_at ASC
	`, orchestrationID)

	if err != nil {
		return nil, fmt.Errorf("failed to query checked out accounts: %w", err)
	}
	defer rows.Close()

	var accounts []string
	for rows.Next() {
		var deviceAccount string
		if err := rows.Scan(&deviceAccount); err != nil {
			return nil, fmt.Errorf("failed to scan device account: %w", err)
		}
		accounts = append(accounts, deviceAccount)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return accounts, nil
}

// ReleaseAllAccountsForOrchestration releases all accounts for a given orchestration (cleanup on shutdown)
func ReleaseAllAccountsForOrchestration(db *sql.DB, orchestrationID string) (int64, error) {
	result, err := db.Exec(`
		UPDATE accounts
		SET checked_out_to_orchestration = NULL,
		    checked_out_to_instance = NULL,
		    checked_out_at = NULL
		WHERE checked_out_to_orchestration = ?
	`, orchestrationID)

	if err != nil {
		return 0, fmt.Errorf("failed to release accounts: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rows, nil
}

// GetCheckedOutAccountForInstance returns the account currently checked out to a specific instance
func GetCheckedOutAccountForInstance(db *sql.DB, orchestrationID string, emulatorInstance int) (string, error) {
	var deviceAccount sql.NullString

	err := db.QueryRow(`
		SELECT device_account
		FROM accounts
		WHERE checked_out_to_orchestration = ?
		AND checked_out_to_instance = ?
	`, orchestrationID, emulatorInstance).Scan(&deviceAccount)

	if err == sql.ErrNoRows {
		return "", nil // No account checked out to this instance
	}

	if err != nil {
		return "", fmt.Errorf("failed to query checked out account: %w", err)
	}

	if !deviceAccount.Valid {
		return "", nil
	}

	return deviceAccount.String, nil
}

// VerifyAndUpdateAccountCheckout verifies the account on emulator matches database state
// This should be called during emulator startup after ADB connection is verified
func VerifyAndUpdateAccountCheckout(db *sql.DB, orchestrationID string, emulatorInstance int, actualDeviceAccount string) error {
	// Get what the database thinks is checked out
	expectedAccount, err := GetCheckedOutAccountForInstance(db, orchestrationID, emulatorInstance)
	if err != nil {
		return fmt.Errorf("failed to get expected account: %w", err)
	}

	// If database matches reality, we're good
	if expectedAccount == actualDeviceAccount {
		return nil
	}

	// Dissonance detected - update database to match reality
	if expectedAccount != "" {
		// Database has a different account checked out - release it
		if err := ReleaseAccount(db, expectedAccount, orchestrationID); err != nil {
			return fmt.Errorf("failed to release incorrect account: %w", err)
		}
	}

	// If there's an account on the emulator, check it out in the database
	if actualDeviceAccount != "" {
		if err := CheckoutAccount(db, actualDeviceAccount, orchestrationID, emulatorInstance); err != nil {
			return fmt.Errorf("failed to checkout actual account: %w", err)
		}
	}

	return nil
}
