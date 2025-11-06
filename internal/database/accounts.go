package database

import (
	"database/sql"
	"fmt"
	"time"
)

// Account operations

// GetOrCreateAccount retrieves an existing account or creates a new one
func (db *DB) GetOrCreateAccount(deviceAccount, devicePassword string) (*Account, error) {
	// Try to find existing account
	account, err := db.GetAccountByDeviceAccount(deviceAccount)
	if err == nil {
		return account, nil
	}

	// If not found, create new account
	if err == sql.ErrNoRows {
		return db.CreateAccount(deviceAccount, devicePassword, "")
	}

	return nil, err
}

// CreateAccount creates a new account with default values
func (db *DB) CreateAccount(deviceAccount, devicePassword, filePath string) (*Account, error) {
	now := time.Now()

	var accountID int64
	err := db.ExecTx(func(tx *sql.Tx) error {
		result, err := tx.Exec(`
			INSERT INTO accounts (
				device_account, device_password, file_path,
				created_at, is_active, is_banned
			) VALUES (?, ?, ?, ?, 1, 0)
		`, deviceAccount, devicePassword, filePath, now)

		if err != nil {
			return fmt.Errorf("failed to insert account: %w", err)
		}

		accountID, err = result.LastInsertId()
		return err
	})

	if err != nil {
		return nil, err
	}

	return db.GetAccountByID(int(accountID))
}

// GetAccountByID retrieves an account by its ID
func (db *DB) GetAccountByID(id int) (*Account, error) {
	account := &Account{}
	err := db.conn.QueryRow(`
		SELECT
			id, device_account, device_password, username, friend_code,
			shinedust, hourglasses, pokegold, pack_points,
			packs_opened, wonder_picks_done, account_level,
			created_at, last_used_at, stamina_recovery_time,
			file_path, is_active, is_banned, notes
		FROM accounts
		WHERE id = ?
	`, id).Scan(
		&account.ID, &account.DeviceAccount, &account.DevicePassword,
		&account.Username, &account.FriendCode,
		&account.Shinedust, &account.Hourglasses, &account.Pokegold, &account.PackPoints,
		&account.PacksOpened, &account.WonderPicksDone, &account.AccountLevel,
		&account.CreatedAt, &account.LastUsedAt, &account.StaminaRecoveryTime,
		&account.FilePath, &account.IsActive, &account.IsBanned, &account.Notes,
	)

	if err != nil {
		return nil, err
	}

	return account, nil
}

// GetAccountByDeviceAccount retrieves an account by its device account string
func (db *DB) GetAccountByDeviceAccount(deviceAccount string) (*Account, error) {
	account := &Account{}
	err := db.conn.QueryRow(`
		SELECT
			id, device_account, device_password, username, friend_code,
			shinedust, hourglasses, pokegold, pack_points,
			packs_opened, wonder_picks_done, account_level,
			created_at, last_used_at, stamina_recovery_time,
			file_path, is_active, is_banned, notes
		FROM accounts
		WHERE device_account = ?
	`, deviceAccount).Scan(
		&account.ID, &account.DeviceAccount, &account.DevicePassword,
		&account.Username, &account.FriendCode,
		&account.Shinedust, &account.Hourglasses, &account.Pokegold, &account.PackPoints,
		&account.PacksOpened, &account.WonderPicksDone, &account.AccountLevel,
		&account.CreatedAt, &account.LastUsedAt, &account.StaminaRecoveryTime,
		&account.FilePath, &account.IsActive, &account.IsBanned, &account.Notes,
	)

	if err != nil {
		return nil, err
	}

	return account, nil
}

// ListActiveAccounts returns all active (not banned) accounts
func (db *DB) ListActiveAccounts() ([]*Account, error) {
	rows, err := db.conn.Query(`
		SELECT
			id, device_account, device_password, username, friend_code,
			shinedust, hourglasses, pokegold, pack_points,
			packs_opened, wonder_picks_done, account_level,
			created_at, last_used_at, stamina_recovery_time,
			file_path, is_active, is_banned, notes
		FROM accounts
		WHERE is_active = 1 AND is_banned = 0
		ORDER BY last_used_at ASC
	`)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []*Account{}
	for rows.Next() {
		account := &Account{}
		err := rows.Scan(
			&account.ID, &account.DeviceAccount, &account.DevicePassword,
			&account.Username, &account.FriendCode,
			&account.Shinedust, &account.Hourglasses, &account.Pokegold, &account.PackPoints,
			&account.PacksOpened, &account.WonderPicksDone, &account.AccountLevel,
			&account.CreatedAt, &account.LastUsedAt, &account.StaminaRecoveryTime,
			&account.FilePath, &account.IsActive, &account.IsBanned, &account.Notes,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	return accounts, rows.Err()
}

// UpdateAccountResources updates the currency and resource values for an account
func (db *DB) UpdateAccountResources(accountID int, shinedust, hourglasses, pokegold, packPoints int) error {
	return db.ExecTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			UPDATE accounts
			SET shinedust = ?, hourglasses = ?, pokegold = ?, pack_points = ?
			WHERE id = ?
		`, shinedust, hourglasses, pokegold, packPoints, accountID)
		return err
	})
}

// UpdateAccountStats updates pack and wonder pick counts
func (db *DB) UpdateAccountStats(accountID int, packsOpened, wonderPicksDone, accountLevel int) error {
	return db.ExecTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			UPDATE accounts
			SET packs_opened = ?, wonder_picks_done = ?, account_level = ?
			WHERE id = ?
		`, packsOpened, wonderPicksDone, accountLevel, accountID)
		return err
	})
}

// UpdateAccountLastUsed updates the last_used_at timestamp for an account
func (db *DB) UpdateAccountLastUsed(accountID int) error {
	return db.ExecTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			UPDATE accounts
			SET last_used_at = ?
			WHERE id = ?
		`, time.Now(), accountID)
		return err
	})
}

// UpdateStaminaRecoveryTime updates when stamina/packs will be available
func (db *DB) UpdateStaminaRecoveryTime(accountID int, recoveryTime time.Time) error {
	return db.ExecTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			UPDATE accounts
			SET stamina_recovery_time = ?
			WHERE id = ?
		`, recoveryTime, accountID)
		return err
	})
}

// MarkAccountBanned marks an account as banned
func (db *DB) MarkAccountBanned(accountID int) error {
	return db.ExecTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			UPDATE accounts
			SET is_banned = 1, is_active = 0
			WHERE id = ?
		`, accountID)
		return err
	})
}

// SetAccountActive sets the is_active flag for an account
func (db *DB) SetAccountActive(accountID int, active bool) error {
	return db.ExecTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			UPDATE accounts
			SET is_active = ?
			WHERE id = ?
		`, active, accountID)
		return err
	})
}

// UpdateAccountUsername updates the in-game username for an account
func (db *DB) UpdateAccountUsername(accountID int, username string) error {
	return db.ExecTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			UPDATE accounts
			SET username = ?
			WHERE id = ?
		`, username, accountID)
		return err
	})
}

// UpdateAccountNotes updates the notes field for an account
func (db *DB) UpdateAccountNotes(accountID int, notes string) error {
	return db.ExecTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			UPDATE accounts
			SET notes = ?
			WHERE id = ?
		`, notes, accountID)
		return err
	})
}

// DeleteAccount deletes an account (cascades to related records)
func (db *DB) DeleteAccount(accountID int) error {
	return db.ExecTx(func(tx *sql.Tx) error {
		_, err := tx.Exec(`DELETE FROM accounts WHERE id = ?`, accountID)
		return err
	})
}

// GetAccountsReadyForStamina returns accounts whose stamina has recovered
func (db *DB) GetAccountsReadyForStamina() ([]*Account, error) {
	now := time.Now()
	rows, err := db.conn.Query(`
		SELECT
			id, device_account, device_password, username, friend_code,
			shinedust, hourglasses, pokegold, pack_points,
			packs_opened, wonder_picks_done, account_level,
			created_at, last_used_at, stamina_recovery_time,
			file_path, is_active, is_banned, notes
		FROM accounts
		WHERE is_active = 1
			AND is_banned = 0
			AND stamina_recovery_time IS NOT NULL
			AND stamina_recovery_time <= ?
		ORDER BY stamina_recovery_time ASC
	`, now)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []*Account{}
	for rows.Next() {
		account := &Account{}
		err := rows.Scan(
			&account.ID, &account.DeviceAccount, &account.DevicePassword,
			&account.Username, &account.FriendCode,
			&account.Shinedust, &account.Hourglasses, &account.Pokegold, &account.PackPoints,
			&account.PacksOpened, &account.WonderPicksDone, &account.AccountLevel,
			&account.CreatedAt, &account.LastUsedAt, &account.StaminaRecoveryTime,
			&account.FilePath, &account.IsActive, &account.IsBanned, &account.Notes,
		)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}

	return accounts, rows.Err()
}
