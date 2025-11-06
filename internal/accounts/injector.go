package accounts

import (
	"fmt"
	"os"
	"path/filepath"

	"jordanella.com/pocket-tcg-go/internal/adb"
)

// Injector handles account XML injection into the game
type Injector struct {
	adb      *adb.Controller
	instance int
}

// NewInjector creates a new account injector
func NewInjector(adbController *adb.Controller, instance int) *Injector {
	return &Injector{
		adb:      adbController,
		instance: instance,
	}
}

// InjectAccount pushes an account XML to the device and copies it to game data
func (i *Injector) InjectAccount(xmlPath string) error {
	// 1. Force stop the game first
	if err := i.adb.ForceStop("jp.pokemon.pokemontcgp"); err != nil {
		return fmt.Errorf("failed to force stop game: %w", err)
	}

	// 2. Push XML to sdcard (temporary location)
	tempPath := "/sdcard/deviceAccount.xml"
	if err := i.adb.Push(xmlPath, tempPath); err != nil {
		return fmt.Errorf("failed to push account XML: %w", err)
	}

	// 3. Copy to game's shared_prefs directory
	gamePath := "/data/data/jp.pokemon.pokemontcgp/shared_prefs/deviceAccount:.xml"
	if _, err := i.adb.Shell(fmt.Sprintf("cp %s %s", tempPath, gamePath)); err != nil {
		return fmt.Errorf("failed to copy to game directory: %w", err)
	}

	// 4. Clean up temporary file
	if _, err := i.adb.Shell(fmt.Sprintf("rm %s", tempPath)); err != nil {
		// Non-fatal, just log
		fmt.Printf("Warning: failed to remove temp file: %v\n", err)
	}

	return nil
}

// ExtractAccount pulls the current account XML from the device
func (i *Injector) ExtractAccount(destPath string) error {
	// Path in game data
	gamePath := "/data/data/jp.pokemon.pokemontcgp/shared_prefs/deviceAccount:.xml"
	tempPath := "/sdcard/deviceAccount.xml"

	// 1. Copy from game directory to sdcard
	if _, err := i.adb.Shell(fmt.Sprintf("cp %s %s", gamePath, tempPath)); err != nil {
		return fmt.Errorf("failed to copy from game directory: %w", err)
	}

	// 2. Pull from sdcard to local
	if err := i.adb.Pull(tempPath, destPath); err != nil {
		return fmt.Errorf("failed to pull account XML: %w", err)
	}

	// 3. Clean up temporary file
	if _, err := i.adb.Shell(fmt.Sprintf("rm %s", tempPath)); err != nil {
		fmt.Printf("Warning: failed to remove temp file: %v\n", err)
	}

	return nil
}

// BackupCurrentAccount extracts and saves current account to backup directory
func (i *Injector) BackupCurrentAccount(saveDir string) (string, error) {
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	backupFile := filepath.Join(saveDir, fmt.Sprintf("backup_%d.xml", os.Getpid()))

	if err := i.ExtractAccount(backupFile); err != nil {
		return "", err
	}

	return backupFile, nil
}

// DeleteCurrentAccount removes the account from device (for account reset)
func (i *Injector) DeleteCurrentAccount() error {
	gamePath := "/data/data/jp.pokemon.pokemontcgp/shared_prefs/deviceAccount:.xml"

	// Force stop first
	if err := i.adb.ForceStop("jp.pokemon.pokemontcgp"); err != nil {
		return fmt.Errorf("failed to force stop game: %w", err)
	}

	// Delete the account file
	if _, err := i.adb.Shell(fmt.Sprintf("rm %s", gamePath)); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	return nil
}

// CheckAccountExists checks if an account file exists on device
func (i *Injector) CheckAccountExists() (bool, error) {
	gamePath := "/data/data/jp.pokemon.pokemontcgp/shared_prefs/deviceAccount:.xml"

	output, err := i.adb.Shell(fmt.Sprintf("test -f %s && echo 'exists' || echo 'notfound'", gamePath))
	if err != nil {
		return false, err
	}

	return output == "exists", nil
}
