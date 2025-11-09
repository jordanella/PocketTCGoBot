package database

import (
	"database/sql"
	"fmt"
	"time"
)

// GetAccountIDByDeviceAccount retrieves the database account ID by device_account
func GetAccountIDByDeviceAccount(db *sql.DB, deviceAccount string) (int64, error) {
	var id int64
	err := db.QueryRow(`
		SELECT id
		FROM accounts
		WHERE device_account = ?
	`, deviceAccount).Scan(&id)

	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("account not found for device_account: %s", deviceAccount)
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get account ID: %w", err)
	}

	return id, nil
}

// RoutineExecution represents a tracked routine execution
type RoutineExecution struct {
	ID               int64
	AccountID        int64
	RoutineName      string
	ExecutionStatus  string // 'started', 'completed', 'failed'
	StartedAt        time.Time
	CompletedAt      *time.Time
	DurationSeconds  *int
	ErrorMessage     *string
	PacksOpened      int
	WonderPicksDone  int
	BotInstance      int
}

// StartRoutineExecution records the start of a routine execution
func StartRoutineExecution(db *sql.DB, accountID int64, routineName string, botInstance int) (int64, error) {
	result, err := db.Exec(`
		INSERT INTO routine_executions (
			account_id,
			routine_name,
			execution_status,
			started_at,
			bot_instance
		) VALUES (?, ?, 'started', datetime('now'), ?)
	`, accountID, routineName, botInstance)

	if err != nil {
		return 0, fmt.Errorf("failed to start routine execution: %w", err)
	}

	return result.LastInsertId()
}

// CompleteRoutineExecution marks a routine execution as completed
func CompleteRoutineExecution(db *sql.DB, executionID int64, packsOpened, wonderPicksDone int) error {
	_, err := db.Exec(`
		UPDATE routine_executions
		SET execution_status = 'completed',
		    completed_at = datetime('now'),
		    duration_seconds = CAST((julianday('now') - julianday(started_at)) * 86400 AS INTEGER),
		    packs_opened = ?,
		    wonder_picks_done = ?
		WHERE id = ?
	`, packsOpened, wonderPicksDone, executionID)

	if err != nil {
		return fmt.Errorf("failed to complete routine execution: %w", err)
	}

	return nil
}

// FailRoutineExecution marks a routine execution as failed with an error message
func FailRoutineExecution(db *sql.DB, executionID int64, errorMessage string) error {
	_, err := db.Exec(`
		UPDATE routine_executions
		SET execution_status = 'failed',
		    completed_at = datetime('now'),
		    duration_seconds = CAST((julianday('now') - julianday(started_at)) * 86400 AS INTEGER),
		    error_message = ?
		WHERE id = ?
	`, errorMessage, executionID)

	if err != nil {
		return fmt.Errorf("failed to mark routine as failed: %w", err)
	}

	return nil
}

// GetRoutineExecution retrieves a routine execution by ID
func GetRoutineExecution(db *sql.DB, executionID int64) (*RoutineExecution, error) {
	var exec RoutineExecution
	var completedAt sql.NullTime
	var durationSeconds sql.NullInt64
	var errorMessage sql.NullString

	err := db.QueryRow(`
		SELECT
			id,
			account_id,
			routine_name,
			execution_status,
			started_at,
			completed_at,
			duration_seconds,
			error_message,
			packs_opened,
			wonder_picks_done,
			bot_instance
		FROM routine_executions
		WHERE id = ?
	`, executionID).Scan(
		&exec.ID,
		&exec.AccountID,
		&exec.RoutineName,
		&exec.ExecutionStatus,
		&exec.StartedAt,
		&completedAt,
		&durationSeconds,
		&errorMessage,
		&exec.PacksOpened,
		&exec.WonderPicksDone,
		&exec.BotInstance,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get routine execution: %w", err)
	}

	// Handle nullable fields
	if completedAt.Valid {
		exec.CompletedAt = &completedAt.Time
	}
	if durationSeconds.Valid {
		duration := int(durationSeconds.Int64)
		exec.DurationSeconds = &duration
	}
	if errorMessage.Valid {
		exec.ErrorMessage = &errorMessage.String
	}

	return &exec, nil
}

// GetLastRoutineExecution retrieves the most recent execution for an account and routine
func GetLastRoutineExecution(db *sql.DB, accountID int64, routineName string) (*RoutineExecution, error) {
	var exec RoutineExecution
	var completedAt sql.NullTime
	var durationSeconds sql.NullInt64
	var errorMessage sql.NullString

	err := db.QueryRow(`
		SELECT
			id,
			account_id,
			routine_name,
			execution_status,
			started_at,
			completed_at,
			duration_seconds,
			error_message,
			packs_opened,
			wonder_picks_done,
			bot_instance
		FROM routine_executions
		WHERE account_id = ? AND routine_name = ?
		ORDER BY started_at DESC
		LIMIT 1
	`, accountID, routineName).Scan(
		&exec.ID,
		&exec.AccountID,
		&exec.RoutineName,
		&exec.ExecutionStatus,
		&exec.StartedAt,
		&completedAt,
		&durationSeconds,
		&errorMessage,
		&exec.PacksOpened,
		&exec.WonderPicksDone,
		&exec.BotInstance,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No previous execution
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get last routine execution: %w", err)
	}

	// Handle nullable fields
	if completedAt.Valid {
		exec.CompletedAt = &completedAt.Time
	}
	if durationSeconds.Valid {
		duration := int(durationSeconds.Int64)
		exec.DurationSeconds = &duration
	}
	if errorMessage.Valid {
		exec.ErrorMessage = &errorMessage.String
	}

	return &exec, nil
}

// UpdateRoutineExecutionMetrics updates metrics for an ongoing routine execution
func UpdateRoutineExecutionMetrics(db *sql.DB, executionID int64, packsOpened, wonderPicksDone int) error {
	_, err := db.Exec(`
		UPDATE routine_executions
		SET packs_opened = ?,
		    wonder_picks_done = ?
		WHERE id = ?
	`, packsOpened, wonderPicksDone, executionID)

	if err != nil {
		return fmt.Errorf("failed to update routine metrics: %w", err)
	}

	return nil
}

// GetAccountRoutineHistory retrieves all executions for an account and routine
func GetAccountRoutineHistory(db *sql.DB, accountID int64, routineName string, limit int) ([]*RoutineExecution, error) {
	query := `
		SELECT
			id,
			account_id,
			routine_name,
			execution_status,
			started_at,
			completed_at,
			duration_seconds,
			error_message,
			packs_opened,
			wonder_picks_done,
			bot_instance
		FROM routine_executions
		WHERE account_id = ? AND routine_name = ?
		ORDER BY started_at DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := db.Query(query, accountID, routineName)
	if err != nil {
		return nil, fmt.Errorf("failed to get routine history: %w", err)
	}
	defer rows.Close()

	var executions []*RoutineExecution
	for rows.Next() {
		var exec RoutineExecution
		var completedAt sql.NullTime
		var durationSeconds sql.NullInt64
		var errorMessage sql.NullString

		err := rows.Scan(
			&exec.ID,
			&exec.AccountID,
			&exec.RoutineName,
			&exec.ExecutionStatus,
			&exec.StartedAt,
			&completedAt,
			&durationSeconds,
			&errorMessage,
			&exec.PacksOpened,
			&exec.WonderPicksDone,
			&exec.BotInstance,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan routine execution: %w", err)
		}

		// Handle nullable fields
		if completedAt.Valid {
			exec.CompletedAt = &completedAt.Time
		}
		if durationSeconds.Valid {
			duration := int(durationSeconds.Int64)
			exec.DurationSeconds = &duration
		}
		if errorMessage.Valid {
			exec.ErrorMessage = &errorMessage.String
		}

		executions = append(executions, &exec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating routine executions: %w", err)
	}

	return executions, nil
}
