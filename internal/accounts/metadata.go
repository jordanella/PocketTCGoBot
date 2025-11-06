package accounts

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"jordanella.com/pocket-tcg-go/internal/bot"
)

// Metadata extraction from XML filenames
// Legacy format from AHK: deviceAccount[flags][username][friendcode][packcount]P.xml
// Example: deviceAccount_W_JohnDoe_1234567890_25P.xml
// Eventual methods for importing from AHK bot

var (
	// Regex to parse metadata from filename
	metadataRegex = regexp.MustCompile(`deviceAccount(?:_([A-Z]+))?(?:_([^_]+))?(?:_(\d+))?(?:_(\d+)P)?\.xml`)
)

// ExtractMetadata parses metadata from XML filename
func ExtractMetadata(fileName string) (*bot.AccountState, error) {
	account := &bot.AccountState{
		FileName: fileName,
		Flags:    []string{},
	}

	// Get file modification time
	// Note: caller should provide full path if needed

	baseName := filepath.Base(fileName)
	matches := metadataRegex.FindStringSubmatch(baseName)

	if matches == nil {
		// No metadata in filename, just basic account
		return account, nil
	}

	// Parse flags (e.g., "W" for wonderpick, "T" for testing)
	if matches[1] != "" {
		flagStr := matches[1]
		for _, flag := range strings.Split(flagStr, "") {
			account.Flags = append(account.Flags, flag)
		}
	}

	// Parse username
	if matches[2] != "" {
		account.Username = matches[2]
	}

	// Parse friend code
	if matches[3] != "" {
		account.FriendCode = matches[3]
	}

	// Parse pack count
	if matches[4] != "" {
		packCount, err := strconv.Atoi(matches[4])
		if err == nil {
			account.OpenPacks = packCount
			account.HasPackInfo = true
		}
	}

	return account, nil
}

// HasFlag checks if account has a specific flag
func HasFlag(account *bot.AccountState, flag string) bool {
	for _, f := range account.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

// AddFlag adds a flag to account if not already present
func AddFlag(account *bot.AccountState, flag string) {
	if !HasFlag(account, flag) {
		account.Flags = append(account.Flags, flag)
	}
}

// RemoveFlag removes a flag from account
func RemoveFlag(account *bot.AccountState, flag string) {
	newFlags := []string{}
	for _, f := range account.Flags {
		if f != flag {
			newFlags = append(newFlags, f)
		}
	}
	account.Flags = newFlags
}

// BuildFileName constructs filename with embedded metadata
func BuildFileName(account *bot.AccountState) string {
	parts := []string{"deviceAccount"}

	// Add flags if present
	if len(account.Flags) > 0 {
		flagStr := strings.Join(account.Flags, "")
		parts = append(parts, flagStr)
	}

	// Add username if present
	if account.Username != "" {
		parts = append(parts, account.Username)
	}

	// Add friend code if present
	if account.FriendCode != "" {
		parts = append(parts, account.FriendCode)
	}

	// Add pack count if available
	if account.HasPackInfo {
		parts = append(parts, fmt.Sprintf("%dP", account.OpenPacks))
	}

	return strings.Join(parts, "_") + ".xml"
}

// UpdateMetadata updates the account filename with new metadata
func UpdateMetadata(account *bot.AccountState, saveDir string) error {
	oldPath := filepath.Join(saveDir, account.FileName)
	newFileName := BuildFileName(account)
	newPath := filepath.Join(saveDir, newFileName)

	// Skip if filename hasn't changed
	if oldPath == newPath {
		return nil
	}

	// Rename file
	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename %s to %s: %w", oldPath, newPath, err)
	}

	account.FileName = newFileName
	return nil
}

// GetFileModTime gets modification time of account file
func GetFileModTime(filePath string) (time.Time, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}

// ParsePackCountFromFilename extracts pack count from filename
func ParsePackCountFromFilename(fileName string) (int, bool) {
	account, err := ExtractMetadata(fileName)
	if err != nil {
		return 0, false
	}
	return account.OpenPacks, account.HasPackInfo
}
