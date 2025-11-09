package database

import (
	"database/sql"
	"fmt"
	"time"
)

// Migration represents a database schema migration
type Migration struct {
	Version     int
	Description string
	Up          func(*sql.Tx) error
	Down        func(*sql.Tx) error
}

// migrations is the ordered list of all database migrations
var migrations = []Migration{
	{
		Version:     1,
		Description: "Create schema_version table",
		Up:          migration001Up,
		Down:        migration001Down,
	},
	{
		Version:     2,
		Description: "Create accounts table",
		Up:          migration002Up,
		Down:        migration002Down,
	},
	{
		Version:     3,
		Description: "Create activity_log and error_log tables",
		Up:          migration003Up,
		Down:        migration003Down,
	},
	{
		Version:     4,
		Description: "Create pack_results and cards_pulled tables",
		Up:          migration004Up,
		Down:        migration004Down,
	},
	{
		Version:     5,
		Description: "Create account_collection and wonder_pick_results tables",
		Up:          migration005Up,
		Down:        migration005Down,
	},
	{
		Version:     6,
		Description: "Create mission_completion and bot_statistics tables",
		Up:          migration006Up,
		Down:        migration006Down,
	},
	{
		Version:     7,
		Description: "Create views",
		Up:          migration007Up,
		Down:        migration007Down,
	},
	{
		Version:     8,
		Description: "Create pool_accounts table for account pool system",
		Up:          migration008Up,
		Down:        migration008Down,
	},
}

// RunMigrations runs all pending database migrations
func (db *DB) RunMigrations() error {
	// Get current version
	currentVersion, err := db.getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	fmt.Printf("Current database version: %d\n", currentVersion)

	// Run pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		fmt.Printf("Running migration %d: %s\n", migration.Version, migration.Description)

		err := db.ExecTx(func(tx *sql.Tx) error {
			// Run migration
			if err := migration.Up(tx); err != nil {
				return fmt.Errorf("migration %d failed: %w", migration.Version, err)
			}

			// Record migration
			_, err := tx.Exec(`
				INSERT INTO schema_version (version, description, applied_at)
				VALUES (?, ?, ?)
			`, migration.Version, migration.Description, time.Now())

			return err
		})

		if err != nil {
			return err
		}

		fmt.Printf("Migration %d completed successfully\n", migration.Version)
	}

	fmt.Println("All migrations completed")
	return nil
}

// getCurrentVersion returns the current schema version
func (db *DB) getCurrentVersion() (int, error) {
	// Check if schema_version table exists
	var tableExists bool
	err := db.conn.QueryRow(`
		SELECT COUNT(*) > 0
		FROM sqlite_master
		WHERE type='table' AND name='schema_version'
	`).Scan(&tableExists)

	if err != nil {
		return 0, err
	}

	if !tableExists {
		return 0, nil
	}

	// Get latest version
	var version int
	err = db.conn.QueryRow(`
		SELECT COALESCE(MAX(version), 0)
		FROM schema_version
	`).Scan(&version)

	if err != nil {
		return 0, err
	}

	return version, nil
}

// Migration 001: Schema version tracking table
func migration001Up(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS schema_version (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version INTEGER NOT NULL UNIQUE,
			description TEXT NOT NULL,
			applied_at DATETIME NOT NULL
		)
	`)
	return err
}

func migration001Down(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE IF EXISTS schema_version`)
	return err
}

// Migration 002: Accounts table
func migration002Up(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_account TEXT NOT NULL UNIQUE,
			device_password TEXT NOT NULL,
			username TEXT,
			friend_code TEXT,

			-- Resources
			shinedust INTEGER DEFAULT 0,
			hourglasses INTEGER DEFAULT 0,
			pokegold INTEGER DEFAULT 0,
			pack_points INTEGER DEFAULT 0,

			-- Statistics
			packs_opened INTEGER DEFAULT 0,
			wonder_picks_done INTEGER DEFAULT 0,
			account_level INTEGER DEFAULT 1,

			-- Timestamps
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_used_at DATETIME,
			stamina_recovery_time DATETIME,

			-- Metadata
			file_path TEXT,
			is_active BOOLEAN DEFAULT 1,
			is_banned BOOLEAN DEFAULT 0,
			notes TEXT
		);

		CREATE INDEX idx_accounts_device_account ON accounts(device_account);
		CREATE INDEX idx_accounts_last_used ON accounts(last_used_at);
		CREATE INDEX idx_accounts_active ON accounts(is_active);
	`)
	return err
}

func migration002Down(tx *sql.Tx) error {
	_, err := tx.Exec(`
		DROP INDEX IF EXISTS idx_accounts_active;
		DROP INDEX IF EXISTS idx_accounts_last_used;
		DROP INDEX IF EXISTS idx_accounts_device_account;
		DROP TABLE IF EXISTS accounts;
	`)
	return err
}

// Migration 003: Activity and Error logging
func migration003Up(tx *sql.Tx) error {
	// Activity log
	_, err := tx.Exec(`
		CREATE TABLE activity_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL,
			activity_type TEXT NOT NULL,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			duration_seconds INTEGER,
			status TEXT DEFAULT 'running',
			error_message TEXT,
			bot_version TEXT,
			routine_name TEXT,
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
		);

		CREATE INDEX idx_activity_account ON activity_log(account_id);
		CREATE INDEX idx_activity_type ON activity_log(activity_type);
		CREATE INDEX idx_activity_started ON activity_log(started_at);
	`)
	if err != nil {
		return err
	}

	// Error log
	_, err = tx.Exec(`
		CREATE TABLE error_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER,
			activity_log_id INTEGER,
			error_type TEXT NOT NULL,
			error_severity TEXT NOT NULL,
			error_message TEXT NOT NULL,
			stack_trace TEXT,
			screen_state TEXT,
			template_name TEXT,
			action_name TEXT,
			was_recovered BOOLEAN DEFAULT 0,
			recovery_action TEXT,
			recovery_time_ms INTEGER,
			occurred_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE SET NULL,
			FOREIGN KEY (activity_log_id) REFERENCES activity_log(id) ON DELETE SET NULL
		);

		CREATE INDEX idx_error_account ON error_log(account_id);
		CREATE INDEX idx_error_type ON error_log(error_type);
		CREATE INDEX idx_error_occurred ON error_log(occurred_at);
		CREATE INDEX idx_error_severity ON error_log(error_severity);
	`)
	return err
}

func migration003Down(tx *sql.Tx) error {
	_, err := tx.Exec(`
		DROP INDEX IF EXISTS idx_error_severity;
		DROP INDEX IF EXISTS idx_error_occurred;
		DROP INDEX IF EXISTS idx_error_type;
		DROP INDEX IF EXISTS idx_error_account;
		DROP TABLE IF EXISTS error_log;

		DROP INDEX IF EXISTS idx_activity_started;
		DROP INDEX IF EXISTS idx_activity_type;
		DROP INDEX IF EXISTS idx_activity_account;
		DROP TABLE IF EXISTS activity_log;
	`)
	return err
}

// Migration 004: Pack results and cards
func migration004Up(tx *sql.Tx) error {
	// Pack results
	_, err := tx.Exec(`
		CREATE TABLE pack_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL,
			activity_log_id INTEGER,
			pack_type TEXT NOT NULL,
			pack_name TEXT,
			is_god_pack BOOLEAN DEFAULT 0,
			card_count INTEGER DEFAULT 5,
			rarity_breakdown TEXT,
			pack_points_earned INTEGER DEFAULT 0,
			opened_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
			FOREIGN KEY (activity_log_id) REFERENCES activity_log(id) ON DELETE SET NULL
		);

		CREATE INDEX idx_pack_account ON pack_results(account_id);
		CREATE INDEX idx_pack_opened ON pack_results(opened_at);
		CREATE INDEX idx_pack_type ON pack_results(pack_type);
		CREATE INDEX idx_pack_god_pack ON pack_results(is_god_pack) WHERE is_god_pack = 1;
	`)
	if err != nil {
		return err
	}

	// Cards pulled
	_, err = tx.Exec(`
		CREATE TABLE cards_pulled (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pack_result_id INTEGER NOT NULL,
			account_id INTEGER NOT NULL,
			card_id TEXT NOT NULL,
			card_name TEXT,
			card_number TEXT,
			rarity TEXT NOT NULL,
			card_type TEXT,
			is_full_art BOOLEAN DEFAULT 0,
			is_ex BOOLEAN DEFAULT 0,
			detection_confidence REAL,
			detected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (pack_result_id) REFERENCES pack_results(id) ON DELETE CASCADE,
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
		);

		CREATE INDEX idx_cards_pack ON cards_pulled(pack_result_id);
		CREATE INDEX idx_cards_account ON cards_pulled(account_id);
		CREATE INDEX idx_cards_rarity ON cards_pulled(rarity);
		CREATE INDEX idx_cards_name ON cards_pulled(card_name);
	`)
	return err
}

func migration004Down(tx *sql.Tx) error {
	_, err := tx.Exec(`
		DROP INDEX IF EXISTS idx_cards_name;
		DROP INDEX IF EXISTS idx_cards_rarity;
		DROP INDEX IF EXISTS idx_cards_account;
		DROP INDEX IF EXISTS idx_cards_pack;
		DROP TABLE IF EXISTS cards_pulled;

		DROP INDEX IF EXISTS idx_pack_god_pack;
		DROP INDEX IF EXISTS idx_pack_type;
		DROP INDEX IF EXISTS idx_pack_opened;
		DROP INDEX IF EXISTS idx_pack_account;
		DROP TABLE IF EXISTS pack_results;
	`)
	return err
}

// Migration 005: Collections and wonder picks
func migration005Up(tx *sql.Tx) error {
	// Account collection
	_, err := tx.Exec(`
		CREATE TABLE account_collection (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL,
			card_id TEXT NOT NULL,
			card_name TEXT NOT NULL,
			card_number TEXT,
			rarity TEXT NOT NULL,
			quantity INTEGER DEFAULT 1,
			first_obtained_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_obtained_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
			UNIQUE(account_id, card_id)
		);

		CREATE INDEX idx_collection_account ON account_collection(account_id);
		CREATE INDEX idx_collection_card_id ON account_collection(card_id);
		CREATE INDEX idx_collection_rarity ON account_collection(rarity);
	`)
	if err != nil {
		return err
	}

	// Wonder pick results
	_, err = tx.Exec(`
		CREATE TABLE wonder_pick_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL,
			activity_log_id INTEGER,
			card_selected TEXT,
			card_rarity TEXT,
			success BOOLEAN DEFAULT 1,
			energy_cost INTEGER DEFAULT 1,
			was_free BOOLEAN DEFAULT 0,
			picked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE,
			FOREIGN KEY (activity_log_id) REFERENCES activity_log(id) ON DELETE SET NULL
		);

		CREATE INDEX idx_wonder_account ON wonder_pick_results(account_id);
		CREATE INDEX idx_wonder_picked ON wonder_pick_results(picked_at);
	`)
	return err
}

func migration005Down(tx *sql.Tx) error {
	_, err := tx.Exec(`
		DROP INDEX IF EXISTS idx_wonder_picked;
		DROP INDEX IF EXISTS idx_wonder_account;
		DROP TABLE IF EXISTS wonder_pick_results;

		DROP INDEX IF EXISTS idx_collection_rarity;
		DROP INDEX IF EXISTS idx_collection_card_id;
		DROP INDEX IF EXISTS idx_collection_account;
		DROP TABLE IF EXISTS account_collection;
	`)
	return err
}

// Migration 006: Missions and statistics
func migration006Up(tx *sql.Tx) error {
	// Mission completion
	_, err := tx.Exec(`
		CREATE TABLE mission_completion (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id INTEGER NOT NULL,
			mission_type TEXT NOT NULL,
			mission_name TEXT,
			shinedust_reward INTEGER DEFAULT 0,
			hourglasses_reward INTEGER DEFAULT 0,
			pokegold_reward INTEGER DEFAULT 0,
			pack_points_reward INTEGER DEFAULT 0,
			completed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
		);

		CREATE INDEX idx_mission_account ON mission_completion(account_id);
		CREATE INDEX idx_mission_type ON mission_completion(mission_type);
		CREATE INDEX idx_mission_completed ON mission_completion(completed_at);
	`)
	if err != nil {
		return err
	}

	// Bot statistics
	_, err = tx.Exec(`
		CREATE TABLE bot_statistics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			total_accounts INTEGER DEFAULT 0,
			active_accounts INTEGER DEFAULT 0,
			banned_accounts INTEGER DEFAULT 0,
			total_packs_opened INTEGER DEFAULT 0,
			total_wonder_picks INTEGER DEFAULT 0,
			total_god_packs INTEGER DEFAULT 0,
			total_runtime_hours REAL DEFAULT 0,
			total_errors INTEGER DEFAULT 0,
			total_recoveries INTEGER DEFAULT 0,
			stats_date DATE DEFAULT CURRENT_DATE,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(stats_date)
		);

		CREATE INDEX idx_stats_date ON bot_statistics(stats_date);
	`)
	return err
}

func migration006Down(tx *sql.Tx) error {
	_, err := tx.Exec(`
		DROP INDEX IF EXISTS idx_stats_date;
		DROP TABLE IF EXISTS bot_statistics;

		DROP INDEX IF EXISTS idx_mission_completed;
		DROP INDEX IF EXISTS idx_mission_type;
		DROP INDEX IF EXISTS idx_mission_account;
		DROP TABLE IF EXISTS mission_completion;
	`)
	return err
}

// Migration 007: Views
func migration007Up(tx *sql.Tx) error {
	// Active accounts view
	_, err := tx.Exec(`
		CREATE VIEW v_active_accounts AS
		SELECT
			a.id,
			a.username,
			a.device_account,
			a.account_level,
			a.packs_opened,
			a.shinedust,
			a.hourglasses,
			a.pokegold,
			a.last_used_at,
			COUNT(DISTINCT pr.id) as total_packs,
			COUNT(DISTINCT cp.id) as total_cards_pulled,
			COUNT(DISTINCT ac.id) as unique_cards_owned
		FROM accounts a
		LEFT JOIN pack_results pr ON a.id = pr.account_id
		LEFT JOIN cards_pulled cp ON a.id = cp.account_id
		LEFT JOIN account_collection ac ON a.id = ac.account_id
		WHERE a.is_active = 1 AND a.is_banned = 0
		GROUP BY a.id;
	`)
	if err != nil {
		return err
	}

	// Recent activity view
	_, err = tx.Exec(`
		CREATE VIEW v_recent_activity AS
		SELECT
			al.id,
			a.username,
			al.activity_type,
			al.started_at,
			al.completed_at,
			al.duration_seconds,
			al.status,
			al.error_message
		FROM activity_log al
		JOIN accounts a ON al.account_id = a.id
		ORDER BY al.started_at DESC
		LIMIT 100;
	`)
	if err != nil {
		return err
	}

	// Pack statistics view
	_, err = tx.Exec(`
		CREATE VIEW v_pack_statistics AS
		SELECT
			a.id as account_id,
			a.username,
			COUNT(pr.id) as total_packs_opened,
			SUM(CASE WHEN pr.is_god_pack = 1 THEN 1 ELSE 0 END) as god_packs,
			COUNT(DISTINCT pr.pack_type) as pack_types_opened,
			MAX(pr.opened_at) as last_pack_opened
		FROM accounts a
		LEFT JOIN pack_results pr ON a.id = pr.account_id
		GROUP BY a.id;
	`)
	return err
}

func migration007Down(tx *sql.Tx) error {
	_, err := tx.Exec(`
		DROP VIEW IF EXISTS v_pack_statistics;
		DROP VIEW IF EXISTS v_recent_activity;
		DROP VIEW IF EXISTS v_active_accounts;
	`)
	return err
}

// Migration 008: Add pool system columns to accounts table
func migration008Up(tx *sql.Tx) error {
	_, err := tx.Exec(`
		-- Add pool lifecycle tracking columns
		ALTER TABLE accounts ADD COLUMN pool_status TEXT DEFAULT 'available';
		ALTER TABLE accounts ADD COLUMN failure_count INTEGER DEFAULT 0;
		ALTER TABLE accounts ADD COLUMN last_error TEXT;
		ALTER TABLE accounts ADD COLUMN completed_at DATETIME;

		-- Create indexes for pool queries
		CREATE INDEX idx_accounts_pool_status ON accounts(pool_status);
		CREATE INDEX idx_accounts_failure_count ON accounts(failure_count);
		CREATE INDEX idx_accounts_completed ON accounts(completed_at);
	`)
	return err
}

func migration008Down(tx *sql.Tx) error {
	// Note: SQLite doesn't support DROP COLUMN, so we'd need to recreate the table
	// For now, just drop the indexes
	_, err := tx.Exec(`
		DROP INDEX IF EXISTS idx_accounts_completed;
		DROP INDEX IF EXISTS idx_accounts_failure_count;
		DROP INDEX IF EXISTS idx_accounts_pool_status;
	`)
	return err
}
