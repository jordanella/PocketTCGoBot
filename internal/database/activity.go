package database

import (
	"database/sql"
	"fmt"
	"time"
)

// Activity logging operations

// StartActivity creates a new activity log entry and returns its ID
func (db *DB) StartActivity(accountID int, activityType, routineName, botVersion string) (int64, error) {
	var activityID int64
	err := db.ExecTx(func(tx *sql.Tx) error {
		result, err := tx.Exec(`
			INSERT INTO activity_log (
				account_id, activity_type, routine_name, bot_version,
				started_at, status
			) VALUES (?, ?, ?, ?, ?, 'running')
		`, accountID, activityType, routineName, botVersion, time.Now())

		if err != nil {
			return fmt.Errorf("failed to insert activity log: %w", err)
		}

		activityID, err = result.LastInsertId()
		return err
	})

	if err != nil {
		return 0, err
	}

	return activityID, nil
}

// CompleteActivity marks an activity as completed successfully
func (db *DB) CompleteActivity(activityID int64) error {
	return db.updateActivityStatus(activityID, "completed", nil)
}

// FailActivity marks an activity as failed with an error message
func (db *DB) FailActivity(activityID int64, errorMessage string) error {
	return db.updateActivityStatus(activityID, "failed", &errorMessage)
}

// AbortActivity marks an activity as aborted
func (db *DB) AbortActivity(activityID int64, reason string) error {
	return db.updateActivityStatus(activityID, "aborted", &reason)
}

// updateActivityStatus updates the status and completion time of an activity
func (db *DB) updateActivityStatus(activityID int64, status string, errorMessage *string) error {
	return db.ExecTx(func(tx *sql.Tx) error {
		completedAt := time.Now()

		// Get start time to calculate duration
		var startedAt time.Time
		err := tx.QueryRow(`SELECT started_at FROM activity_log WHERE id = ?`, activityID).Scan(&startedAt)
		if err != nil {
			return fmt.Errorf("failed to get activity start time: %w", err)
		}

		duration := int(completedAt.Sub(startedAt).Seconds())

		_, err = tx.Exec(`
			UPDATE activity_log
			SET completed_at = ?,
				duration_seconds = ?,
				status = ?,
				error_message = ?
			WHERE id = ?
		`, completedAt, duration, status, errorMessage, activityID)

		return err
	})
}

// GetActivityByID retrieves an activity log by ID
func (db *DB) GetActivityByID(activityID int64) (*ActivityLog, error) {
	activity := &ActivityLog{}
	err := db.conn.QueryRow(`
		SELECT
			id, account_id, activity_type, started_at, completed_at,
			duration_seconds, status, error_message, bot_version, routine_name
		FROM activity_log
		WHERE id = ?
	`, activityID).Scan(
		&activity.ID, &activity.AccountID, &activity.ActivityType,
		&activity.StartedAt, &activity.CompletedAt, &activity.DurationSeconds,
		&activity.Status, &activity.ErrorMessage, &activity.BotVersion, &activity.RoutineName,
	)

	if err != nil {
		return nil, err
	}

	return activity, nil
}

// GetRecentActivityForAccount returns recent activities for an account
func (db *DB) GetRecentActivityForAccount(accountID int, limit int) ([]*ActivityLog, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.conn.Query(`
		SELECT
			id, account_id, activity_type, started_at, completed_at,
			duration_seconds, status, error_message, bot_version, routine_name
		FROM activity_log
		WHERE account_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`, accountID, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := []*ActivityLog{}
	for rows.Next() {
		activity := &ActivityLog{}
		err := rows.Scan(
			&activity.ID, &activity.AccountID, &activity.ActivityType,
			&activity.StartedAt, &activity.CompletedAt, &activity.DurationSeconds,
			&activity.Status, &activity.ErrorMessage, &activity.BotVersion, &activity.RoutineName,
		)
		if err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}

	return activities, rows.Err()
}

// GetRecentActivity returns the most recent activities across all accounts
func (db *DB) GetRecentActivity(limit int) ([]*RecentActivity, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := db.conn.Query(`
		SELECT
			id, username, activity_type, started_at, completed_at,
			duration_seconds, status, error_message
		FROM v_recent_activity
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := []*RecentActivity{}
	for rows.Next() {
		activity := &RecentActivity{}
		err := rows.Scan(
			&activity.ID, &activity.Username, &activity.ActivityType,
			&activity.StartedAt, &activity.CompletedAt, &activity.DurationSeconds,
			&activity.Status, &activity.ErrorMessage,
		)
		if err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}

	return activities, rows.Err()
}

// GetRunningActivities returns all activities currently in 'running' status
func (db *DB) GetRunningActivities() ([]*ActivityLog, error) {
	rows, err := db.conn.Query(`
		SELECT
			id, account_id, activity_type, started_at, completed_at,
			duration_seconds, status, error_message, bot_version, routine_name
		FROM activity_log
		WHERE status = 'running'
		ORDER BY started_at ASC
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := []*ActivityLog{}
	for rows.Next() {
		activity := &ActivityLog{}
		err := rows.Scan(
			&activity.ID, &activity.AccountID, &activity.ActivityType,
			&activity.StartedAt, &activity.CompletedAt, &activity.DurationSeconds,
			&activity.Status, &activity.ErrorMessage, &activity.BotVersion, &activity.RoutineName,
		)
		if err != nil {
			return nil, err
		}
		activities = append(activities, activity)
	}

	return activities, rows.Err()
}

// GetActivityStats returns activity statistics for a time range
func (db *DB) GetActivityStats(accountID int, startDate, endDate time.Time) (map[string]int, error) {
	rows, err := db.conn.Query(`
		SELECT activity_type, COUNT(*) as count
		FROM activity_log
		WHERE account_id = ?
			AND started_at BETWEEN ? AND ?
		GROUP BY activity_type
	`, accountID, startDate, endDate)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var activityType string
		var count int
		if err := rows.Scan(&activityType, &count); err != nil {
			return nil, err
		}
		stats[activityType] = count
	}

	return stats, rows.Err()
}

// DeleteOldActivities deletes activity logs older than the specified date
func (db *DB) DeleteOldActivities(olderThan time.Time) (int64, error) {
	var deleted int64
	err := db.ExecTx(func(tx *sql.Tx) error {
		result, err := tx.Exec(`
			DELETE FROM activity_log
			WHERE started_at < ?
		`, olderThan)

		if err != nil {
			return err
		}

		deleted, err = result.RowsAffected()
		return err
	})

	return deleted, err
}
