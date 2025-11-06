package database

import (
	"database/sql"
	"fmt"
	"time"
)

// Error logging operations

// LogError creates a new error log entry
func (db *DB) LogError(
	accountID *int,
	activityLogID *int,
	errorType string,
	errorSeverity string,
	errorMessage string,
	stackTrace *string,
	screenState *string,
	templateName *string,
	actionName *string,
) (int64, error) {
	var errorID int64
	err := db.ExecTx(func(tx *sql.Tx) error {
		result, err := tx.Exec(`
			INSERT INTO error_log (
				account_id, activity_log_id, error_type, error_severity,
				error_message, stack_trace, screen_state, template_name,
				action_name, occurred_at, was_recovered
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0)
		`, accountID, activityLogID, errorType, errorSeverity,
			errorMessage, stackTrace, screenState, templateName,
			actionName, time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert error log: %w", err)
		}

		errorID, err = result.LastInsertId()
		return err
	})

	if err != nil {
		return 0, err
	}

	return errorID, nil
}

// MarkErrorRecovered updates an error log entry to mark it as recovered
func (db *DB) MarkErrorRecovered(errorID int64, recoveryAction string, recoveryTimeMs int) error {
	return db.ExecTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			UPDATE error_log
			SET was_recovered = 1,
				recovery_action = ?,
				recovery_time_ms = ?
			WHERE id = ?
		`, recoveryAction, recoveryTimeMs, errorID)
		return err
	})
}

// GetErrorByID retrieves an error log by ID
func (db *DB) GetErrorByID(errorID int64) (*ErrorLog, error) {
	errorLog := &ErrorLog{}
	err := db.conn.QueryRow(`
		SELECT
			id, account_id, activity_log_id, error_type, error_severity,
			error_message, stack_trace, screen_state, template_name,
			action_name, was_recovered, recovery_action, recovery_time_ms,
			occurred_at
		FROM error_log
		WHERE id = ?
	`, errorID).Scan(
		&errorLog.ID, &errorLog.AccountID, &errorLog.ActivityLogID,
		&errorLog.ErrorType, &errorLog.ErrorSeverity, &errorLog.ErrorMessage,
		&errorLog.StackTrace, &errorLog.ScreenState, &errorLog.TemplateName,
		&errorLog.ActionName, &errorLog.WasRecovered, &errorLog.RecoveryAction,
		&errorLog.RecoveryTimeMs, &errorLog.OccurredAt,
	)

	if err != nil {
		return nil, err
	}

	return errorLog, nil
}

// GetRecentErrorsForAccount returns recent errors for an account
func (db *DB) GetRecentErrorsForAccount(accountID int, limit int) ([]*ErrorLog, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.conn.Query(`
		SELECT
			id, account_id, activity_log_id, error_type, error_severity,
			error_message, stack_trace, screen_state, template_name,
			action_name, was_recovered, recovery_action, recovery_time_ms,
			occurred_at
		FROM error_log
		WHERE account_id = ?
		ORDER BY occurred_at DESC
		LIMIT ?
	`, accountID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	errors := []*ErrorLog{}
	for rows.Next() {
		errorLog := &ErrorLog{}
		err := rows.Scan(
			&errorLog.ID, &errorLog.AccountID, &errorLog.ActivityLogID,
			&errorLog.ErrorType, &errorLog.ErrorSeverity, &errorLog.ErrorMessage,
			&errorLog.StackTrace, &errorLog.ScreenState, &errorLog.TemplateName,
			&errorLog.ActionName, &errorLog.WasRecovered, &errorLog.RecoveryAction,
			&errorLog.RecoveryTimeMs, &errorLog.OccurredAt,
		)
		if err != nil {
			return nil, err
		}
		errors = append(errors, errorLog)
	}

	return errors, rows.Err()
}

// GetRecentErrors returns the most recent errors across all accounts
func (db *DB) GetRecentErrors(limit int) ([]*ErrorLog, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.conn.Query(`
		SELECT
			id, account_id, activity_log_id, error_type, error_severity,
			error_message, stack_trace, screen_state, template_name,
			action_name, was_recovered, recovery_action, recovery_time_ms,
			occurred_at
		FROM error_log
		ORDER BY occurred_at DESC
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	errors := []*ErrorLog{}
	for rows.Next() {
		errorLog := &ErrorLog{}
		err := rows.Scan(
			&errorLog.ID, &errorLog.AccountID, &errorLog.ActivityLogID,
			&errorLog.ErrorType, &errorLog.ErrorSeverity, &errorLog.ErrorMessage,
			&errorLog.StackTrace, &errorLog.ScreenState, &errorLog.TemplateName,
			&errorLog.ActionName, &errorLog.WasRecovered, &errorLog.RecoveryAction,
			&errorLog.RecoveryTimeMs, &errorLog.OccurredAt,
		)
		if err != nil {
			return nil, err
		}
		errors = append(errors, errorLog)
	}

	return errors, rows.Err()
}

// GetUnrecoveredErrors returns errors that were not recovered
func (db *DB) GetUnrecoveredErrors(limit int) ([]*ErrorLog, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.conn.Query(`
		SELECT
			id, account_id, activity_log_id, error_type, error_severity,
			error_message, stack_trace, screen_state, template_name,
			action_name, was_recovered, recovery_action, recovery_time_ms,
			occurred_at
		FROM error_log
		WHERE was_recovered = 0
		ORDER BY occurred_at DESC
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	errors := []*ErrorLog{}
	for rows.Next() {
		errorLog := &ErrorLog{}
		err := rows.Scan(
			&errorLog.ID, &errorLog.AccountID, &errorLog.ActivityLogID,
			&errorLog.ErrorType, &errorLog.ErrorSeverity, &errorLog.ErrorMessage,
			&errorLog.StackTrace, &errorLog.ScreenState, &errorLog.TemplateName,
			&errorLog.ActionName, &errorLog.WasRecovered, &errorLog.RecoveryAction,
			&errorLog.RecoveryTimeMs, &errorLog.OccurredAt,
		)
		if err != nil {
			return nil, err
		}
		errors = append(errors, errorLog)
	}

	return errors, rows.Err()
}

// GetErrorStatsByType returns error counts grouped by type
func (db *DB) GetErrorStatsByType(accountID *int, startDate, endDate time.Time) (map[string]int, error) {
	query := `
		SELECT error_type, COUNT(*) as count
		FROM error_log
		WHERE occurred_at BETWEEN ? AND ?
	`
	args := []interface{}{startDate, endDate}

	if accountID != nil {
		query += " AND account_id = ?"
		args = append(args, *accountID)
	}

	query += " GROUP BY error_type"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var errorType string
		var count int
		if err := rows.Scan(&errorType, &count); err != nil {
			return nil, err
		}
		stats[errorType] = count
	}

	return stats, rows.Err()
}

// GetErrorStatsBySeverity returns error counts grouped by severity
func (db *DB) GetErrorStatsBySeverity(accountID *int, startDate, endDate time.Time) (map[string]int, error) {
	query := `
		SELECT error_severity, COUNT(*) as count
		FROM error_log
		WHERE occurred_at BETWEEN ? AND ?
	`
	args := []interface{}{startDate, endDate}

	if accountID != nil {
		query += " AND account_id = ?"
		args = append(args, *accountID)
	}

	query += " GROUP BY error_severity"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			return nil, err
		}
		stats[severity] = count
	}

	return stats, rows.Err()
}

// GetRecoveryRate returns the percentage of errors that were recovered
func (db *DB) GetRecoveryRate(accountID *int, startDate, endDate time.Time) (float64, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN was_recovered = 1 THEN 1 ELSE 0 END) as recovered
		FROM error_log
		WHERE occurred_at BETWEEN ? AND ?
	`
	args := []interface{}{startDate, endDate}

	if accountID != nil {
		query += " AND account_id = ?"
		args = append(args, *accountID)
	}

	var total, recovered int
	err := db.conn.QueryRow(query, args...).Scan(&total, &recovered)
	if err != nil {
		return 0, err
	}

	if total == 0 {
		return 0, nil
	}

	return float64(recovered) / float64(total) * 100, nil
}

// DeleteOldErrors deletes error logs older than the specified date
func (db *DB) DeleteOldErrors(olderThan time.Time) (int64, error) {
	var deleted int64
	err := db.ExecTx(func(tx *sql.Tx) error {
		result, err := tx.Exec(`
			DELETE FROM error_log
			WHERE occurred_at < ?
		`, olderThan)

		if err != nil {
			return err
		}

		deleted, err = result.RowsAffected()
		return err
	})

	return deleted, err
}
