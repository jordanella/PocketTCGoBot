package database

import (
	"time"
)

// Account represents a bot account with all its resources and metadata
type Account struct {
	ID             int       `db:"id"`
	DeviceAccount  string    `db:"device_account"`
	DevicePassword string    `db:"device_password"`
	Username       *string   `db:"username"`
	FriendCode     *string   `db:"friend_code"`

	// Resources
	Shinedust   int `db:"shinedust"`
	Hourglasses int `db:"hourglasses"`
	Pokegold    int `db:"pokegold"`
	PackPoints  int `db:"pack_points"`

	// Statistics
	PacksOpened     int `db:"packs_opened"`
	WonderPicksDone int `db:"wonder_picks_done"`
	AccountLevel    int `db:"account_level"`

	// Timestamps
	CreatedAt          time.Time  `db:"created_at"`
	LastUsedAt         *time.Time `db:"last_used_at"`
	StaminaRecoveryTime *time.Time `db:"stamina_recovery_time"`

	// Metadata
	FilePath string  `db:"file_path"`
	IsActive bool    `db:"is_active"`
	IsBanned bool    `db:"is_banned"`
	Notes    *string `db:"notes"`
}

// ActivityLog represents a single activity session
type ActivityLog struct {
	ID              int        `db:"id"`
	AccountID       int        `db:"account_id"`
	ActivityType    string     `db:"activity_type"`
	StartedAt       time.Time  `db:"started_at"`
	CompletedAt     *time.Time `db:"completed_at"`
	DurationSeconds *int       `db:"duration_seconds"`
	Status          string     `db:"status"`
	ErrorMessage    *string    `db:"error_message"`
	BotVersion      *string    `db:"bot_version"`
	RoutineName     *string    `db:"routine_name"`
}

// ErrorLog represents a detailed error record
type ErrorLog struct {
	ID             int        `db:"id"`
	AccountID      *int       `db:"account_id"`
	ActivityLogID  *int       `db:"activity_log_id"`
	ErrorType      string     `db:"error_type"`
	ErrorSeverity  string     `db:"error_severity"`
	ErrorMessage   string     `db:"error_message"`
	StackTrace     *string    `db:"stack_trace"`
	ScreenState    *string    `db:"screen_state"`
	TemplateName   *string    `db:"template_name"`
	ActionName     *string    `db:"action_name"`
	WasRecovered   bool       `db:"was_recovered"`
	RecoveryAction *string    `db:"recovery_action"`
	RecoveryTimeMs *int       `db:"recovery_time_ms"`
	OccurredAt     time.Time  `db:"occurred_at"`
}

// PackResult represents a single pack opening
type PackResult struct {
	ID               int        `db:"id"`
	AccountID        int        `db:"account_id"`
	ActivityLogID    *int       `db:"activity_log_id"`
	PackType         string     `db:"pack_type"`
	PackName         *string    `db:"pack_name"`
	IsGodPack        bool       `db:"is_god_pack"`
	CardCount        int        `db:"card_count"`
	RarityBreakdown  *string    `db:"rarity_breakdown"`
	PackPointsEarned int        `db:"pack_points_earned"`
	OpenedAt         time.Time  `db:"opened_at"`
}

// CardPulled represents a single card from a pack
type CardPulled struct {
	ID                   int       `db:"id"`
	PackResultID         int       `db:"pack_result_id"`
	AccountID            int       `db:"account_id"`
	CardID               string    `db:"card_id"`
	CardName             *string   `db:"card_name"`
	CardNumber           *string   `db:"card_number"`
	Rarity               string    `db:"rarity"`
	CardType             *string   `db:"card_type"`
	IsFullArt            bool      `db:"is_full_art"`
	IsEx                 bool      `db:"is_ex"`
	DetectionConfidence  *float64  `db:"detection_confidence"`
	DetectedAt           time.Time `db:"detected_at"`
}

// AccountCollection represents a card owned by an account
type AccountCollection struct {
	ID              int       `db:"id"`
	AccountID       int       `db:"account_id"`
	CardID          string    `db:"card_id"`
	CardName        string    `db:"card_name"`
	CardNumber      *string   `db:"card_number"`
	Rarity          string    `db:"rarity"`
	Quantity        int       `db:"quantity"`
	FirstObtainedAt time.Time `db:"first_obtained_at"`
	LastObtainedAt  time.Time `db:"last_obtained_at"`
}

// WonderPickResult represents a wonder pick attempt
type WonderPickResult struct {
	ID            int        `db:"id"`
	AccountID     int        `db:"account_id"`
	ActivityLogID *int       `db:"activity_log_id"`
	CardSelected  *string    `db:"card_selected"`
	CardRarity    *string    `db:"card_rarity"`
	Success       bool       `db:"success"`
	EnergyCost    int        `db:"energy_cost"`
	WasFree       bool       `db:"was_free"`
	PickedAt      time.Time  `db:"picked_at"`
}

// MissionCompletion represents a completed mission
type MissionCompletion struct {
	ID                  int       `db:"id"`
	AccountID           int       `db:"account_id"`
	MissionType         string    `db:"mission_type"`
	MissionName         *string   `db:"mission_name"`
	ShinedustReward     int       `db:"shinedust_reward"`
	HourglassesReward   int       `db:"hourglasses_reward"`
	PokegoldReward      int       `db:"pokegold_reward"`
	PackPointsReward    int       `db:"pack_points_reward"`
	CompletedAt         time.Time `db:"completed_at"`
}

// BotStatistics represents daily bot statistics
type BotStatistics struct {
	ID                  int       `db:"id"`
	TotalAccounts       int       `db:"total_accounts"`
	ActiveAccounts      int       `db:"active_accounts"`
	BannedAccounts      int       `db:"banned_accounts"`
	TotalPacksOpened    int       `db:"total_packs_opened"`
	TotalWonderPicks    int       `db:"total_wonder_picks"`
	TotalGodPacks       int       `db:"total_god_packs"`
	TotalRuntimeHours   float64   `db:"total_runtime_hours"`
	TotalErrors         int       `db:"total_errors"`
	TotalRecoveries     int       `db:"total_recoveries"`
	StatsDate           string    `db:"stats_date"`
	UpdatedAt           time.Time `db:"updated_at"`
}

// View models (for querying pre-built views)

// ActiveAccount represents the v_active_accounts view
type ActiveAccount struct {
	ID                 int        `db:"id"`
	Username           *string    `db:"username"`
	DeviceAccount      string     `db:"device_account"`
	AccountLevel       int        `db:"account_level"`
	PacksOpened        int        `db:"packs_opened"`
	Shinedust          int        `db:"shinedust"`
	Hourglasses        int        `db:"hourglasses"`
	Pokegold           int        `db:"pokegold"`
	LastUsedAt         *time.Time `db:"last_used_at"`
	TotalPacks         int        `db:"total_packs"`
	TotalCardsPulled   int        `db:"total_cards_pulled"`
	UniqueCardsOwned   int        `db:"unique_cards_owned"`
}

// RecentActivity represents the v_recent_activity view
type RecentActivity struct {
	ID              int        `db:"id"`
	Username        *string    `db:"username"`
	ActivityType    string     `db:"activity_type"`
	StartedAt       time.Time  `db:"started_at"`
	CompletedAt     *time.Time `db:"completed_at"`
	DurationSeconds *int       `db:"duration_seconds"`
	Status          string     `db:"status"`
	ErrorMessage    *string    `db:"error_message"`
}

// PackStatistics represents the v_pack_statistics view
type PackStatistics struct {
	AccountID        int        `db:"account_id"`
	Username         *string    `db:"username"`
	TotalPacksOpened int        `db:"total_packs_opened"`
	GodPacks         int        `db:"god_packs"`
	PackTypesOpened  int        `db:"pack_types_opened"`
	LastPackOpened   *time.Time `db:"last_pack_opened"`
}
