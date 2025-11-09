package actions

import (
	"database/sql"
	"fmt"
	"strconv"

	"jordanella.com/pocket-tcg-go/internal/database"
)

// UpdateAccountField updates a specific field in the accounts table
// Requires device_account_id variable to be set (automatically set by InjectNextAccount)
type UpdateAccountField struct {
	Field string `yaml:"field"` // Field name: packs_opened, shinedust, hourglasses, etc.
	Value string `yaml:"value"` // Value to set (supports variable interpolation)
}

func (a *UpdateAccountField) Validate(ab *ActionBuilder) error {
	if a.Field == "" {
		return fmt.Errorf("UpdateAccountField: field is required")
	}

	// Validate field is an allowed field (security measure)
	allowedFields := map[string]bool{
		"packs_opened":   true,
		"shinedust":      true,
		"hourglasses":    true,
		"wonder_picks":   true,
		"last_used_at":   true,
		"completed_at":   true,
		"pool_status":    true,
		"failure_count":  true,
		"last_error":     true,
	}

	if !allowedFields[a.Field] {
		return fmt.Errorf("UpdateAccountField: field '%s' is not allowed for updates", a.Field)
	}

	if a.Value == "" {
		return fmt.Errorf("UpdateAccountField: value is required")
	}

	return nil
}

func (a *UpdateAccountField) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("UpdateAccountField (%s = %s)", a.Field, a.Value),
		execute: func(botIf BotInterface) error {
			// Get database from manager
			managerIf := botIf.Manager()
			if managerIf == nil {
				return fmt.Errorf("bot has no manager - cannot access database")
			}

			dbProvider, ok := managerIf.(interface{ Database() *sql.DB })
			if !ok {
				return fmt.Errorf("bot manager does not provide Database method")
			}

			db := dbProvider.Database()
			if db == nil {
				return fmt.Errorf("no database configured in manager")
			}

			// Get device_account_id variable
			deviceAccountIDStr, exists := botIf.Variables().Get("device_account_id")
			if !exists || deviceAccountIDStr == "" {
				return fmt.Errorf("device_account_id variable not set - account must be injected first")
			}

			accountID, err := strconv.ParseInt(deviceAccountIDStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid device_account_id: %w", err)
			}

			// Interpolate the value
			value, err := InterpolateString(a.Value, botIf)
			if err != nil {
				return fmt.Errorf("failed to interpolate value: %w", err)
			}

			// Update the field
			query := fmt.Sprintf("UPDATE accounts SET %s = ? WHERE id = ?", a.Field)
			result, err := db.Exec(query, value, accountID)
			if err != nil {
				return fmt.Errorf("failed to update %s: %w", a.Field, err)
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("failed to get rows affected: %w", err)
			}

			if rowsAffected == 0 {
				return fmt.Errorf("no account found with id %d", accountID)
			}

			fmt.Printf("Bot %d: Updated account %d field '%s' to '%s'\n", botIf.Instance(), accountID, a.Field, value)
			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// IncrementAccountField increments a numeric field in the accounts table
// Requires device_account_id variable to be set
type IncrementAccountField struct {
	Field  string `yaml:"field"`            // Field name: packs_opened, shinedust, hourglasses, etc.
	Amount string `yaml:"amount,omitempty"` // Amount to increment (default: "1", supports variable interpolation)
}

func (a *IncrementAccountField) Validate(ab *ActionBuilder) error {
	if a.Field == "" {
		return fmt.Errorf("IncrementAccountField: field is required")
	}

	// Validate field is a numeric field
	numericFields := map[string]bool{
		"packs_opened":  true,
		"shinedust":     true,
		"hourglasses":   true,
		"wonder_picks":  true,
		"failure_count": true,
	}

	if !numericFields[a.Field] {
		return fmt.Errorf("IncrementAccountField: field '%s' is not a numeric field", a.Field)
	}

	return nil
}

func (a *IncrementAccountField) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("IncrementAccountField (%s)", a.Field),
		execute: func(botIf BotInterface) error {
			// Get database from manager
			managerIf := botIf.Manager()
			if managerIf == nil {
				return fmt.Errorf("bot has no manager - cannot access database")
			}

			dbProvider, ok := managerIf.(interface{ Database() *sql.DB })
			if !ok {
				return fmt.Errorf("bot manager does not provide Database method")
			}

			db := dbProvider.Database()
			if db == nil {
				return fmt.Errorf("no database configured in manager")
			}

			// Get device_account_id variable
			deviceAccountIDStr, exists := botIf.Variables().Get("device_account_id")
			if !exists || deviceAccountIDStr == "" {
				return fmt.Errorf("device_account_id variable not set - account must be injected first")
			}

			accountID, err := strconv.ParseInt(deviceAccountIDStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid device_account_id: %w", err)
			}

			// Get increment amount
			amount := "1"
			if a.Amount != "" {
				amount, err = InterpolateString(a.Amount, botIf)
				if err != nil {
					return fmt.Errorf("failed to interpolate amount: %w", err)
				}
			}

			// Validate amount is numeric
			incrementValue, err := strconv.ParseInt(amount, 10, 64)
			if err != nil {
				return fmt.Errorf("amount must be a valid integer: %w", err)
			}

			// Increment the field
			query := fmt.Sprintf("UPDATE accounts SET %s = %s + ? WHERE id = ?", a.Field, a.Field)
			result, err := db.Exec(query, incrementValue, accountID)
			if err != nil {
				return fmt.Errorf("failed to increment %s: %w", a.Field, err)
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("failed to get rows affected: %w", err)
			}

			if rowsAffected == 0 {
				return fmt.Errorf("no account found with id %d", accountID)
			}

			fmt.Printf("Bot %d: Incremented account %d field '%s' by %d\n", botIf.Instance(), accountID, a.Field, incrementValue)
			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// UpdateRoutineMetrics updates metrics for the current routine execution
// Requires a routine execution to be in progress (tracked by ExecuteWithRestart)
type UpdateRoutineMetrics struct {
	PacksOpened     string `yaml:"packs_opened,omitempty"`      // Number of packs opened (supports variable interpolation)
	WonderPicksDone string `yaml:"wonder_picks_done,omitempty"` // Number of wonder picks done (supports variable interpolation)
}

func (a *UpdateRoutineMetrics) Validate(ab *ActionBuilder) error {
	if a.PacksOpened == "" && a.WonderPicksDone == "" {
		return fmt.Errorf("UpdateRoutineMetrics: at least one metric must be specified")
	}
	return nil
}

func (a *UpdateRoutineMetrics) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: "UpdateRoutineMetrics",
		execute: func(botIf BotInterface) error {
			// Get database from manager
			managerIf := botIf.Manager()
			if managerIf == nil {
				return fmt.Errorf("bot has no manager - cannot access database")
			}

			dbProvider, ok := managerIf.(interface{ Database() *sql.DB })
			if !ok {
				return fmt.Errorf("bot manager does not provide Database method")
			}

			db := dbProvider.Database()
			if db == nil {
				return fmt.Errorf("no database configured in manager")
			}

			// Get execution_id variable (set by ExecuteWithRestart)
			executionIDStr, exists := botIf.Variables().Get("execution_id")
			if !exists || executionIDStr == "" {
				fmt.Printf("Bot %d: Warning - execution_id not set, metrics update skipped\n", botIf.Instance())
				return nil // Non-fatal - routine might not be tracked
			}

			executionID, err := strconv.ParseInt(executionIDStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid execution_id: %w", err)
			}

			// Get metric values
			var packsOpened, wonderPicksDone int64

			if a.PacksOpened != "" {
				packsStr, err := InterpolateString(a.PacksOpened, botIf)
				if err != nil {
					return fmt.Errorf("failed to interpolate packs_opened: %w", err)
				}
				packsOpened, err = strconv.ParseInt(packsStr, 10, 64)
				if err != nil {
					return fmt.Errorf("packs_opened must be a valid integer: %w", err)
				}
			}

			if a.WonderPicksDone != "" {
				picksStr, err := InterpolateString(a.WonderPicksDone, botIf)
				if err != nil {
					return fmt.Errorf("failed to interpolate wonder_picks_done: %w", err)
				}
				wonderPicksDone, err = strconv.ParseInt(picksStr, 10, 64)
				if err != nil {
					return fmt.Errorf("wonder_picks_done must be a valid integer: %w", err)
				}
			}

			// Update metrics
			err = database.UpdateRoutineExecutionMetrics(db, executionID, int(packsOpened), int(wonderPicksDone))
			if err != nil {
				return fmt.Errorf("failed to update routine metrics: %w", err)
			}

			fmt.Printf("Bot %d: Updated routine execution %d metrics (packs: %d, picks: %d)\n",
				botIf.Instance(), executionID, packsOpened, wonderPicksDone)
			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}

// GetAccountField retrieves a field value from the accounts table and stores it in a variable
// Requires device_account_id variable to be set
type GetAccountField struct {
	Field      string `yaml:"field"`       // Field name to retrieve
	SaveTo     string `yaml:"save_to"`     // Variable name to store the value
	DefaultVal string `yaml:"default,omitempty"` // Default value if field is NULL
}

func (a *GetAccountField) Validate(ab *ActionBuilder) error {
	if a.Field == "" {
		return fmt.Errorf("GetAccountField: field is required")
	}
	if a.SaveTo == "" {
		return fmt.Errorf("GetAccountField: save_to is required")
	}

	// Validate field is an allowed field
	allowedFields := map[string]bool{
		"packs_opened":   true,
		"shinedust":      true,
		"hourglasses":    true,
		"wonder_picks":   true,
		"last_used_at":   true,
		"completed_at":   true,
		"pool_status":    true,
		"failure_count":  true,
		"last_error":     true,
		"device_account": true,
	}

	if !allowedFields[a.Field] {
		return fmt.Errorf("GetAccountField: field '%s' is not allowed", a.Field)
	}

	return nil
}

func (a *GetAccountField) Build(ab *ActionBuilder) *ActionBuilder {
	step := Step{
		name: fmt.Sprintf("GetAccountField (%s -> %s)", a.Field, a.SaveTo),
		execute: func(botIf BotInterface) error {
			// Get database from manager
			managerIf := botIf.Manager()
			if managerIf == nil {
				return fmt.Errorf("bot has no manager - cannot access database")
			}

			dbProvider, ok := managerIf.(interface{ Database() *sql.DB })
			if !ok {
				return fmt.Errorf("bot manager does not provide Database method")
			}

			db := dbProvider.Database()
			if db == nil {
				return fmt.Errorf("no database configured in manager")
			}

			// Get device_account_id variable
			deviceAccountIDStr, exists := botIf.Variables().Get("device_account_id")
			if !exists || deviceAccountIDStr == "" {
				return fmt.Errorf("device_account_id variable not set - account must be injected first")
			}

			accountID, err := strconv.ParseInt(deviceAccountIDStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid device_account_id: %w", err)
			}

			// Query the field
			query := fmt.Sprintf("SELECT %s FROM accounts WHERE id = ?", a.Field)
			var value sql.NullString
			err = db.QueryRow(query, accountID).Scan(&value)
			if err == sql.ErrNoRows {
				return fmt.Errorf("no account found with id %d", accountID)
			}
			if err != nil {
				return fmt.Errorf("failed to query %s: %w", a.Field, err)
			}

			// Use default if NULL
			resultValue := a.DefaultVal
			if value.Valid {
				resultValue = value.String
			}

			// Store in variable
			botIf.Variables().Set(a.SaveTo, resultValue)
			fmt.Printf("Bot %d: Retrieved account %d field '%s' = '%s' (stored in %s)\n",
				botIf.Instance(), accountID, a.Field, resultValue, a.SaveTo)

			return nil
		},
		issue: a.Validate(ab),
	}
	ab.steps = append(ab.steps, step)
	return ab
}
