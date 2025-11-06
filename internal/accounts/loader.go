package accounts

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"jordanella.com/pocket-tcg-go/internal/bot"
)

// Loader manages account loading, tracking, and persistence
type Loader struct {
	instance        int
	saveDir         string
	usedAccountsLog string
	currentList     string
	masterList      string
	config          *bot.Config

	usedAccounts map[string]bool // In-memory cache of used accounts
}

// NewLoader creates a new account loader
func NewLoader(instance int, config *bot.Config) *Loader {
	baseDir := filepath.Join("..", "Accounts", "Saved", fmt.Sprintf("%d", instance))

	return &Loader{
		instance:        instance,
		saveDir:         baseDir,
		usedAccountsLog: filepath.Join(baseDir, "used_accounts.txt"),
		currentList:     filepath.Join(baseDir, "list_current.txt"),
		masterList:      filepath.Join(baseDir, "list.txt"),
		config:          config,
		usedAccounts:    make(map[string]bool),
	}
}

// Initialize sets up directories and loads used accounts cache
func (l *Loader) Initialize() error {
	// Ensure save directory exists
	if err := os.MkdirAll(l.saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save directory: %w", err)
	}

	// Load used accounts cache
	return l.loadUsedAccountsCache()
}

// LoadNextAccount loads the next available account from the list
func (l *Loader) LoadNextAccount() (*bot.AccountState, error) {
	// Read current list
	accounts, err := l.readAccountList(l.currentList)
	if err != nil {
		return nil, err
	}

	// Find first valid, unused account
	for _, account := range accounts {
		if l.isUsed(account.FileName) {
			continue
		}

		// Check if file exists
		fullPath := filepath.Join(l.saveDir, account.FileName)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			continue
		}

		// Get modification time
		modTime, err := GetFileModTime(fullPath)
		if err == nil {
			account.ModifiedTime = modTime
		}

		return account, nil
	}

	return nil, fmt.Errorf("no available accounts in list")
}

// MarkAccountAsUsed marks an account as used to avoid reloading it
func (l *Loader) MarkAccountAsUsed(account *bot.AccountState) error {
	// Add to in-memory cache
	l.usedAccounts[account.FileName] = true

	// Append to log file
	f, err := os.OpenFile(l.usedAccountsLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open used accounts log: %w", err)
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	_, err = fmt.Fprintf(f, "%s\t%s\n", timestamp, account.FileName)
	return err
}

// SaveAccount saves an account XML file to a category folder
func (l *Loader) SaveAccount(account *bot.AccountState, category string) error {
	// Update metadata in filename
	account.FileName = BuildFileName(account)

	// Determine destination directory
	destDir := filepath.Join(l.saveDir, category)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create category directory: %w", err)
	}

	// Source and destination paths
	srcPath := filepath.Join(l.saveDir, account.FileNameOrig)
	destPath := filepath.Join(destDir, account.FileName)

	// Copy or move file
	if err := os.Rename(srcPath, destPath); err != nil {
		// If rename fails, try copy
		return copyFile(srcPath, destPath)
	}

	return nil
}

// CreateAccountList generates account lists with filtering and sorting
func (l *Loader) CreateAccountList() error {
	// Cleanup stale used accounts first
	if err := l.CleanupUsedAccounts(); err != nil {
		return err
	}

	// Check if regeneration is needed
	if !l.shouldRegenerateList() {
		return nil
	}

	// Load all accounts from save directory
	accounts, err := l.scanAccountFiles()
	if err != nil {
		return err
	}

	// Filter accounts based on criteria
	accounts = l.filterAccounts(accounts)

	// Sort accounts
	accounts = l.sortAccounts(accounts)

	// Write to list files
	if err := l.writeAccountList(l.masterList, accounts); err != nil {
		return err
	}
	if err := l.writeAccountList(l.currentList, accounts); err != nil {
		return err
	}

	// Update last generated timestamp
	return l.updateLastGeneratedTimestamp()
}

// CleanupUsedAccounts removes stale entries from used accounts log
func (l *Loader) CleanupUsedAccounts() error {
	// Read used accounts log
	entries, err := l.readUsedAccountsLog()
	if err != nil {
		return err
	}

	// Filter out entries older than 48 hours
	cutoffTime := time.Now().Add(-48 * time.Hour)
	validEntries := []usedAccountEntry{}

	for _, entry := range entries {
		if entry.timestamp.After(cutoffTime) {
			validEntries = append(validEntries, entry)
		}
	}

	// Rewrite log with valid entries only
	return l.writeUsedAccountsLog(validEntries)
}

// Helper methods

func (l *Loader) isUsed(fileName string) bool {
	return l.usedAccounts[fileName]
}

func (l *Loader) loadUsedAccountsCache() error {
	entries, err := l.readUsedAccountsLog()
	if err != nil {
		// If file doesn't exist, that's okay
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		l.usedAccounts[entry.fileName] = true
	}
	return nil
}

type usedAccountEntry struct {
	timestamp time.Time
	fileName  string
}

func (l *Loader) readUsedAccountsLog() ([]usedAccountEntry, error) {
	f, err := os.Open(l.usedAccountsLog)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	entries := []usedAccountEntry{}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		timestamp, err := time.Parse("2006-01-02 15:04:05", parts[0])
		if err != nil {
			continue
		}

		entries = append(entries, usedAccountEntry{
			timestamp: timestamp,
			fileName:  parts[1],
		})
	}

	return entries, scanner.Err()
}

func (l *Loader) writeUsedAccountsLog(entries []usedAccountEntry) error {
	f, err := os.Create(l.usedAccountsLog)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, entry := range entries {
		timestamp := entry.timestamp.Format("2006-01-02 15:04:05")
		fmt.Fprintf(f, "%s\t%s\n", timestamp, entry.fileName)
	}

	return nil
}

func (l *Loader) shouldRegenerateList() bool {
	// Check if list files exist
	if _, err := os.Stat(l.currentList); os.IsNotExist(err) {
		return true
	}

	// Check if current list is empty
	accounts, err := l.readAccountList(l.currentList)
	if err != nil || len(accounts) <= 1 {
		return true
	}

	// Check time-based regeneration (hourly)
	lastGenFile := filepath.Join(l.saveDir, "list_last_generated.txt")
	data, err := os.ReadFile(lastGenFile)
	if err != nil {
		return true
	}

	lastGen, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return true
	}

	return time.Since(lastGen) > 60*time.Minute
}

func (l *Loader) updateLastGeneratedTimestamp() error {
	lastGenFile := filepath.Join(l.saveDir, "list_last_generated.txt")
	return os.WriteFile(lastGenFile, []byte(time.Now().Format(time.RFC3339)), 0644)
}

func (l *Loader) scanAccountFiles() ([]*bot.AccountState, error) {
	files, err := os.ReadDir(l.saveDir)
	if err != nil {
		return nil, err
	}

	accounts := []*bot.AccountState{}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".xml") {
			continue
		}

		account, err := ExtractMetadata(file.Name())
		if err != nil {
			continue
		}

		// Get file info
		fullPath := filepath.Join(l.saveDir, file.Name())
		modTime, err := GetFileModTime(fullPath)
		if err == nil {
			account.ModifiedTime = modTime
		}

		accounts = append(accounts, account)
	}

	return accounts, nil
}

func (l *Loader) filterAccounts(accounts []*bot.AccountState) []*bot.AccountState {
	filtered := []*bot.AccountState{}

	for _, account := range accounts {
		// Skip used accounts
		if l.isUsed(account.FileName) {
			continue
		}

		// Filter by pack count if available
		if account.HasPackInfo {
			if account.OpenPacks < l.config.InjectMinPacks {
				continue
			}
			if account.OpenPacks > l.config.InjectMaxPacks {
				continue
			}
		}

		filtered = append(filtered, account)
	}

	return filtered
}

func (l *Loader) sortAccounts(accounts []*bot.AccountState) []*bot.AccountState {
	switch l.config.InjectSortMethod {
	case bot.SortMethodModifiedAsc:
		sort.Slice(accounts, func(i, j int) bool {
			return accounts[i].ModifiedTime.Before(accounts[j].ModifiedTime)
		})
	case bot.SortMethodModifiedDesc:
		sort.Slice(accounts, func(i, j int) bool {
			return accounts[i].ModifiedTime.After(accounts[j].ModifiedTime)
		})
	case bot.SortMethodPacksAsc:
		sort.Slice(accounts, func(i, j int) bool {
			return accounts[i].OpenPacks < accounts[j].OpenPacks
		})
	case bot.SortMethodPacksDesc:
		sort.Slice(accounts, func(i, j int) bool {
			return accounts[i].OpenPacks > accounts[j].OpenPacks
		})
	}

	return accounts
}

func (l *Loader) readAccountList(path string) ([]*bot.AccountState, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	accounts := []*bot.AccountState{}
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) < 5 || !strings.HasSuffix(line, ".xml") {
			continue
		}

		account, err := ExtractMetadata(line)
		if err != nil {
			continue
		}

		accounts = append(accounts, account)
	}

	return accounts, scanner.Err()
}

func (l *Loader) writeAccountList(path string, accounts []*bot.AccountState) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, account := range accounts {
		fmt.Fprintln(f, account.FileName)
	}

	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
