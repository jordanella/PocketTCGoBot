package accounts

import (
	"fmt"
	"time"
)

// Manager manages the account pool (stub for GUI compatibility)
type Manager struct {
	accountFile string
	accounts    []*Account
}

// Account represents a single account (stub for GUI compatibility)
type Account struct {
	ID         string
	InUse      bool
	Created    time.Time
	LastUsed   time.Time
	UseCount   int
	Metadata   map[string]interface{}
}

// NewManager creates a new account manager
func NewManager(accountFile string) *Manager {
	return &Manager{
		accountFile: accountFile,
		accounts:    make([]*Account, 0),
	}
}

// NewAccount creates a new account with the given ID
func NewAccount(id string) *Account {
	return &Account{
		ID:       id,
		InUse:    false,
		Created:  time.Now(),
		Metadata: make(map[string]interface{}),
	}
}

// AddAccount adds an account to the manager
func (m *Manager) AddAccount(account *Account) error {
	// Check for duplicates
	for _, acc := range m.accounts {
		if acc.ID == account.ID {
			return fmt.Errorf("account %s already exists", account.ID)
		}
	}

	m.accounts = append(m.accounts, account)
	return nil
}

// RemoveAccount removes an account from the manager
func (m *Manager) RemoveAccount(id string) error {
	for i, acc := range m.accounts {
		if acc.ID == id {
			// Remove by slicing
			m.accounts = append(m.accounts[:i], m.accounts[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("account %s not found", id)
}

// GetAllAccounts returns all accounts
func (m *Manager) GetAllAccounts() []*Account {
	return m.accounts
}

// SaveAccounts saves accounts to disk (stub)
func (m *Manager) SaveAccounts() error {
	// TODO: Implement persistence
	return nil
}

// LoadAccounts loads accounts from disk (stub)
func (m *Manager) LoadAccounts() error {
	// TODO: Implement loading
	return nil
}
