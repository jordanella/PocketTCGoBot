package coordinator

import (
	"sync"
	"time"

	"jordanella.com/pocket-tcg-go/internal/bot"
)

// AccountManager handles injection account queue
type AccountManager struct {
	saveDir      string
	sortMethod   string
	minPacks     int
	maxPacks     int
	usedAccounts map[string]time.Time
	mu           sync.RWMutex
}

func NewAccountManager(saveDir string, cfg *bot.Config) *AccountManager {
	return &AccountManager{
		saveDir:      saveDir,
		usedAccounts: make(map[string]time.Time),
	}
}

func (am *AccountManager) LoadNextEligibleAccount() (*Account, error) {
	// TODO: Implement
	return nil, nil
}

func (am *AccountManager) MarkAccountAsUsed(account *Account) {
	// TODO: Implement
}

func (am *AccountManager) RefreshLists() error {
	// TODO: Implement
	return nil
}

type Account struct {
	FileName     string
	PackCount    int
	ModifiedTime time.Time
	Metadata     Metadata
}

type Metadata struct {
	BeginnerDone bool
	// ... other flags
}
