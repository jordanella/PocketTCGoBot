package accountpool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

var (
	// ErrNoAccountsAvailable is returned when the pool has no available accounts
	ErrNoAccountsAvailable = errors.New("no accounts available")

	// ErrAccountNotFound is returned when an account ID is not found
	ErrAccountNotFound = errors.New("account not found")

	// ErrPoolClosed is returned when attempting operations on a closed pool
	ErrPoolClosed = errors.New("account pool is closed")
)

// AccountPool manages a pool of accounts for bot processing
type AccountPool interface {
	// GetNext returns the next available account from the pool
	// Blocks until an account is available or context is cancelled
	GetNext(ctx context.Context) (*Account, error)

	// Return puts an account back into the pool (e.g., if not used due to error)
	Return(account *Account) error

	// MarkUsed marks an account as successfully processed with results
	MarkUsed(account *Account, result AccountResult) error

	// MarkFailed marks an account as failed with a reason
	MarkFailed(account *Account, reason string) error

	// GetByID retrieves an account by its ID
	GetByID(id string) (*Account, error)

	// GetStats returns current pool statistics
	GetStats() PoolStats

	// Refresh reloads accounts from the source
	Refresh() error

	// ListAccounts returns all accounts in the pool (for export/inspection)
	ListAccounts() []*Account

	// Close closes the pool and releases resources
	Close() error
}

// Account represents a single account in the pool
type Account struct {
	ID           string            // Unique identifier (typically device_account)
	XMLPath      string            // Full path to the account XML file (generated on-demand or cached)
	DeviceAccount string           // Device account credential
	DevicePassword string          // Device password credential
	PackCount    int               // Number of packs available
	LastModified time.Time         // Last modification time
	Metadata     map[string]string // Additional metadata (tags, notes, etc.)

	// State tracking
	Status       AccountStatus      // Current status
	AssignedAt   *time.Time         // When account was assigned to a bot
	AssignedTo   int                // Bot instance number (0 if not assigned)
	ProcessedAt  *time.Time         // When account was processed
	Result       *AccountResult     // Processing result
	FailureCount int                // Number of times this account has failed
	LastError    string             // Last error message
}

// AccountStatus represents the current state of an account
type AccountStatus string

const (
	AccountStatusAvailable AccountStatus = "available" // Ready to be assigned
	AccountStatusInUse     AccountStatus = "in_use"    // Currently assigned to a bot
	AccountStatusCompleted AccountStatus = "completed" // Successfully processed
	AccountStatusFailed    AccountStatus = "failed"    // Failed processing
	AccountStatusSkipped   AccountStatus = "skipped"   // Manually skipped
)

// AccountResult holds the results of processing an account
type AccountResult struct {
	Success      bool          // Whether processing was successful
	PacksOpened  int           // Number of packs opened
	CardsFound   int           // Number of cards found
	StarsTotal   int           // Total stars across all cards
	KeepCount    int           // Number of cards kept/saved
	Error        string        // Error message if failed
	Duration     time.Duration // How long processing took
	Timestamp    time.Time     // When processing completed
	BotInstance  int           // Which bot processed this account
}

// PoolStats provides statistics about the account pool
type PoolStats struct {
	Total       int       // Total accounts in pool
	Available   int       // Accounts ready to be assigned
	InUse       int       // Accounts currently assigned
	Completed   int       // Successfully processed accounts
	Failed      int       // Failed accounts
	Skipped     int       // Manually skipped accounts
	LastRefresh time.Time // Last time pool was refreshed

	// Aggregated results
	TotalPacksOpened int           // Total packs opened across all accounts
	TotalCardsFound  int           // Total cards found
	TotalStars       int           // Total stars collected
	TotalKeeps       int           // Total cards kept
	AverageDuration  time.Duration // Average processing duration
}

// SortMethod defines how accounts should be sorted
type SortMethod int

const (
	SortMethodModifiedAsc  SortMethod = iota // Oldest first
	SortMethodModifiedDesc                   // Newest first
	SortMethodPacksAsc                       // Fewest packs first
	SortMethodPacksDesc                      // Most packs first
)

func (s SortMethod) String() string {
	switch s {
	case SortMethodModifiedAsc:
		return "ModifiedAsc"
	case SortMethodModifiedDesc:
		return "ModifiedDesc"
	case SortMethodPacksAsc:
		return "PacksAsc"
	case SortMethodPacksDesc:
		return "PacksDesc"
	default:
		return "ModifiedAsc"
	}
}

// PoolConfig configures how the account pool behaves
type PoolConfig struct {
	// Filtering
	MinPacks     int        // Minimum packs required (0 = no minimum)
	MaxPacks     int        // Maximum packs allowed (0 = no maximum)
	SortMethod   SortMethod // How to sort accounts

	// Retry behavior
	MaxFailures  int  // Max times to retry a failed account (0 = no retry)
	RetryFailed  bool // Whether to retry failed accounts

	// Refresh behavior
	AutoRefresh       bool          // Automatically refresh when pool is empty
	RefreshInterval   time.Duration // How often to auto-refresh (0 = disabled)
	WaitForAccounts   bool          // Wait for accounts if pool is empty
	MaxWaitTime       time.Duration // Max time to wait for accounts (0 = infinite)

	// Concurrency
	BufferSize int // Size of the available account buffer (default: 100)
}

// DefaultPoolConfig returns sensible defaults for pool configuration
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MinPacks:          0,
		MaxPacks:          0,
		SortMethod:        SortMethodModifiedAsc, // Process oldest first
		MaxFailures:       3,
		RetryFailed:       false, // Don't retry by default
		AutoRefresh:       false, // Don't auto-refresh by default
		RefreshInterval:   0,
		WaitForAccounts:   false,
		MaxWaitTime:       0,
		BufferSize:        100,
	}
}

// Clone creates a deep copy of the account
func (a *Account) Clone() *Account {
	clone := &Account{
		ID:             a.ID,
		XMLPath:        a.XMLPath,
		DeviceAccount:  a.DeviceAccount,
		DevicePassword: a.DevicePassword,
		PackCount:      a.PackCount,
		LastModified:   a.LastModified,
		Metadata:       make(map[string]string),
		Status:         a.Status,
		AssignedTo:     a.AssignedTo,
		FailureCount:   a.FailureCount,
		LastError:      a.LastError,
	}

	// Copy metadata
	for k, v := range a.Metadata {
		clone.Metadata[k] = v
	}

	// Copy time pointers
	if a.AssignedAt != nil {
		t := *a.AssignedAt
		clone.AssignedAt = &t
	}
	if a.ProcessedAt != nil {
		t := *a.ProcessedAt
		clone.ProcessedAt = &t
	}

	// Copy result
	if a.Result != nil {
		r := *a.Result
		clone.Result = &r
	}

	return clone
}

// GenerateXML creates an XML file with the account credentials
// Returns the path to the generated XML file
func (a *Account) GenerateXML(tempDir string) (string, error) {
	// XML template matching the expected format
	xmlContent := fmt.Sprintf(`<account>%s</account>
<password>%s</password>`, a.DeviceAccount, a.DevicePassword)

	// Create temp file
	filePath := fmt.Sprintf("%s/%s.xml", tempDir, a.ID)
	if err := os.WriteFile(filePath, []byte(xmlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write XML file: %w", err)
	}

	return filePath, nil
}

// IsAvailable returns whether the account can be assigned
func (a *Account) IsAvailable() bool {
	return a.Status == AccountStatusAvailable
}

// IsInUse returns whether the account is currently assigned
func (a *Account) IsInUse() bool {
	return a.Status == AccountStatusInUse
}

// IsCompleted returns whether the account has been successfully processed
func (a *Account) IsCompleted() bool {
	return a.Status == AccountStatusCompleted
}

// IsFailed returns whether the account has failed processing
func (a *Account) IsFailed() bool {
	return a.Status == AccountStatusFailed
}
