package accounts

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	AppPackage      = "jp.pokemon.pokemontcgp"
	AppActivity     = "com.unity3d.player.UnityPlayerActivity"
	SharedPrefsPath = "/data/data/jp.pokemon.pokemontcgp/shared_prefs/deviceAccount:.xml"
	TempPath        = "/sdcard/deviceAccount.xml"
)

// InjectAccount injects an account XML file into a specific instance
// Uses the same proven methods as ADBTestTab for reliability
func InjectAccount(adbPath string, adbPort int, xmlFilePath string) error {
	// Verify the XML file exists
	if _, err := os.Stat(xmlFilePath); os.IsNotExist(err) {
		return fmt.Errorf("account file does not exist: %s", xmlFilePath)
	}

	adbAddress := fmt.Sprintf("127.0.0.1:%d", adbPort)

	// Step 0: Connect to device
	if err := connectToDevice(adbPath, adbAddress); err != nil {
		return fmt.Errorf("failed to connect to device: %w", err)
	}

	// Step 1: Force stop the app
	if err := forceStopApp(adbPath, adbAddress); err != nil {
		return fmt.Errorf("failed to force stop app: %w", err)
	}

	// Step 2: Push XML file to device
	if err := pushXMLToDevice(adbPath, adbAddress, xmlFilePath); err != nil {
		return fmt.Errorf("failed to push XML to device: %w", err)
	}

	// Step 3: Copy to shared preferences location
	if err := copyToSharedPrefs(adbPath, adbAddress); err != nil {
		return fmt.Errorf("failed to copy to shared prefs: %w", err)
	}

	// Step 4: Clean up temporary file
	if err := cleanupTempFile(adbPath, adbAddress); err != nil {
		return fmt.Errorf("failed to cleanup temp file: %w", err)
	}

	// Step 5: Launch the app
	if err := launchApp(adbPath, adbAddress); err != nil {
		return fmt.Errorf("failed to launch app: %w", err)
	}

	return nil
}

func ExtractAccount(adbPath string, adbPort int, xmlFilePath string) error {
	adbAddress := fmt.Sprintf("127.0.0.1:%d", adbPort)

	// Step 0: Connect to device
	if err := connectToDevice(adbPath, adbAddress); err != nil {
		return fmt.Errorf("failed to connect to device: %w", err)
	}

	// Step 1: Copy to shared preferences location
	if err := copyFromSharedPrefs(adbPath, adbAddress); err != nil {
		return fmt.Errorf("failed to copy to shared prefs: %w", err)
	}

	// Step 2: Pull XML file from device
	if err := pullXMLFromDevice(adbPath, adbAddress, xmlFilePath); err != nil {
		return fmt.Errorf("failed to push XML to device: %w", err)
	}

	// Step 3: Clean up temporary file
	if err := cleanupTempFile(adbPath, adbAddress); err != nil {
		return fmt.Errorf("failed to cleanup temp file: %w", err)
	}

	return nil
}

// connectToDevice connects ADB to the device
func connectToDevice(adbPath, adbAddress string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, adbPath, "connect", adbAddress)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("connect failed: %v, output: %s", err, string(output))
	}

	// Check if connection was successful
	outputStr := string(output)
	if !strings.Contains(outputStr, "connected") && !strings.Contains(outputStr, "already connected") {
		return fmt.Errorf("unexpected connect response: %s", outputStr)
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// forceStopApp stops the Pokemon TCG Pocket app using the same method as ADBTestTab
func forceStopApp(adbPath, adbAddress string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", "am", "force-stop", AppPackage)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("force-stop failed: %v, output: %s", err, string(output))
	}
	time.Sleep(500 * time.Millisecond)
	return nil
}

// pushXMLToDevice pushes the XML file to the device
func pushXMLToDevice(adbPath, adbAddress, xmlFilePath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Convert to absolute path
	absPath, err := filepath.Abs(xmlFilePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "push", absPath, TempPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("push failed: %v, output: %s", err, string(output))
	}
	time.Sleep(500 * time.Millisecond)
	return nil
}

// PullXMLFromDevice pulls the XML file to the root directory
func pullXMLFromDevice(adbPath, adbAddress, xmlFilePath string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "pull", TempPath, xmlFilePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pull failed: %v, output: %s", err, string(output))
	}
	time.Sleep(500 * time.Millisecond)
	return nil
}

// copyToSharedPrefs copies the XML from temp location to shared preferences
func copyToSharedPrefs(adbPath, adbAddress string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Command format: adb shell "su -c 'cp /sdcard/deviceAccount.xml /data/data/jp.pokemon.pokemontcgp/shared_prefs/deviceAccount:.xml'"
	suCmd := fmt.Sprintf("su -c 'cp %s %s'", TempPath, SharedPrefsPath)
	cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", suCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If su -c fails, return detailed error
		return fmt.Errorf("failed to copy file: %v, output: %s", err, string(output))
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

func copyFromSharedPrefs(adbPath, adbAddress string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Command format: adb shell "su -c 'cp /sdcard/deviceAccount.xml /data/data/jp.pokemon.pokemontcgp/shared_prefs/deviceAccount:.xml'"
	suCmd := fmt.Sprintf("su -c 'cp %s %s'", SharedPrefsPath, TempPath)

	cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", suCmd)
	output, err := cmd.CombinedOutput()

	if err != nil {
		// If su -c fails, return detailed error
		return fmt.Errorf("failed to copy file with su -c: %v, output: %s", err, string(output))
	}

	time.Sleep(500 * time.Millisecond)
	return nil
}

// cleanupTempFile removes the temporary XML file from the device
func cleanupTempFile(adbPath, adbAddress string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", "rm", TempPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cleanup failed: %v, output: %s", err, string(output))
	}
	time.Sleep(200 * time.Millisecond)
	return nil
}

// launchApp launches the Pokemon TCG Pocket app using the same method as ADBTestTab
func launchApp(adbPath, adbAddress string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Launch with flags: FLAG_ACTIVITY_NEW_TASK (0x10000000) | FLAG_ACTIVITY_CLEAR_TOP (0x04000000) | FLAG_ACTIVITY_NO_ANIMATION (0x00010000)
	// Combined: 0x10018000
	// Using the same format as ADBTestTab
	cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", "am", "start", "-W",
		"-n", fmt.Sprintf("%s/%s", AppPackage, AppActivity),
		"-f", "0x10018000")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("launch failed: %v, output: %s", err, string(output))
	}
	time.Sleep(500 * time.Millisecond)
	return nil
}

// parseXMLData is a helper to parse XML data and extract device account
func parseXMLData(data []byte) ([]*AccountFile, error) {
	// Create a temporary file
	tempDir := filepath.Join(os.TempDir(), "pokemontcg_parse")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, err
	}

	tempFile := filepath.Join(tempDir, "temp_parse.xml")
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return nil, err
	}
	defer os.Remove(tempFile)

	// Use existing load function
	return LoadAccountsFromXML(tempDir)
}

// ExtractAppData extracts the app data directory from device to local folder
func ExtractAppData(adbPath string, adbPort int, outputDir string) error {
	adbAddress := fmt.Sprintf("127.0.0.1:%d", adbPort)

	// Step 0: Connect to device
	if err := connectToDevice(adbPath, adbAddress); err != nil {
		return fmt.Errorf("failed to connect to device: %w", err)
	}

	// Based on storage crawl: /data/data/jp.pokemon.pokemontcgp is readable
	// But we'll use copy-to-temp workflow for safety
	appDataPath := "/data/data/jp.pokemon.pokemontcgp"
	tempDataPath := "/sdcard/temp_app_data"

	// Remove any existing temp directory first
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", "rm", "-rf", tempDataPath).Run()
	cancel()

	// Copy with su (preserving directory structure)
	// Note: Must match the pattern from copyToSharedPrefs/copyFromSharedPrefs
	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	suCmd := fmt.Sprintf("su -c 'cp -r %s %s'", appDataPath, tempDataPath)
	cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", suCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy app data with su: %v, output: %s", err, string(output))
	}
	time.Sleep(500 * time.Millisecond)

	// Step 2: Pull from temp location to local
	ctx, cancel = context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	cmd = exec.CommandContext(ctx, adbPath, "-s", adbAddress, "pull", tempDataPath, outputDir)
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull app data: %v, output: %s", err, string(output))
	}
	time.Sleep(500 * time.Millisecond)

	// Step 3: Cleanup temp directory
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd = exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", "rm", "-rf", tempDataPath)
	if err := cmd.Run(); err != nil {
		// Log but don't fail on cleanup
		fmt.Printf("Warning: failed to cleanup temp directory: %v\n", err)
	}

	return nil
}

// ExtractOBBData extracts OBB files from device to local folder
func ExtractOBBData(adbPath string, adbPort int, outputDir string) error {
	adbAddress := fmt.Sprintf("127.0.0.1:%d", adbPort)

	// Step 0: Connect to device
	if err := connectToDevice(adbPath, adbAddress); err != nil {
		return fmt.Errorf("failed to connect to device: %w", err)
	}

	// Try multiple possible OBB locations (based on storage crawl results)
	possiblePaths := []string{
		"/sdcard/Android/data/jp.pokemon.pokemontcgp",
		"/storage/emulated/0/Android/data/jp.pokemon.pokemontcgp",
		"/data/media/0/Android/data/jp.pokemon.pokemontcgp",
	}

	var obbPath string
	var foundPath bool

	for _, path := range possiblePaths {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", "ls", "-la", path)
		output, err := cmd.CombinedOutput()
		cancel()

		if err == nil && !strings.Contains(string(output), "No such file or directory") {
			obbPath = path
			foundPath = true
			break
		}
	}

	if !foundPath {
		return fmt.Errorf("OBB directory not found at any known location: %v", possiblePaths)
	}

	// Step 1: Pull OBB directory directly (readable without root based on crawl results)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "pull", obbPath, outputDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to pull OBB data: %v, output: %s", err, string(output))
	}

	return nil
}

// CrawlStorage crawls the device storage and outputs directory structure to a file
func CrawlStorage(adbPath string, adbPort int, outputFile string) error {
	adbAddress := fmt.Sprintf("127.0.0.1:%d", adbPort)

	// Step 0: Connect to device
	if err := connectToDevice(adbPath, adbAddress); err != nil {
		return fmt.Errorf("failed to connect to device: %w", err)
	}

	var allOutput strings.Builder
	allOutput.WriteString("=== Device Storage Crawl ===\n\n")

	// First, let's check what storage locations are actually mounted
	allOutput.WriteString("=== Mounted Storage Locations ===\n")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", "mount")
	output, err := cmd.CombinedOutput()
	cancel()
	if err == nil {
		allOutput.WriteString(string(output))
	} else {
		allOutput.WriteString(fmt.Sprintf("Error running mount: %v\n", err))
	}
	allOutput.WriteString("\n")

	// Check basic storage locations with simple ls -l (no recursion)
	basicLocations := []string{
		"/sdcard",
		"/storage/emulated/0",
		"/data/media/0",
		"/mnt/sdcard",
	}

	for _, location := range basicLocations {
		allOutput.WriteString(fmt.Sprintf("\n=== Checking: %s ===\n", location))

		// First, just see if the directory exists and is readable
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", "ls", "-la", location)
		output, err := cmd.CombinedOutput()
		cancel()

		if err != nil {
			allOutput.WriteString(fmt.Sprintf("Error: %v\n", err))
			allOutput.WriteString(fmt.Sprintf("Output: %s\n", string(output)))
		} else {
			allOutput.WriteString(string(output))
			allOutput.WriteString("\n")
		}
	}

	// Also check for specific game-related paths
	allOutput.WriteString("\n=== Checking Specific Game Paths ===\n")
	specificPaths := []string{
		"/sdcard/Android",
		"/sdcard/Android/obb",
		"/sdcard/Android/data",
		"/storage/emulated/0/Android",
		"/storage/emulated/0/Android/obb",
		"/data/data/jp.pokemon.pokemontcgp",
	}

	for _, path := range specificPaths {
		allOutput.WriteString(fmt.Sprintf("\nPath: %s\n", path))

		// Try without su first
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", "ls", "-la", path)
		output, err := cmd.CombinedOutput()
		cancel()

		if err != nil {
			allOutput.WriteString(fmt.Sprintf("Without su - Error: %v\nOutput: %s\n", err, string(output)))

			// Try with su
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			cmd := exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", "su", "-c", fmt.Sprintf("ls -la %s", path))
			output, err := cmd.CombinedOutput()
			cancel()

			if err != nil {
				allOutput.WriteString(fmt.Sprintf("With su - Error: %v\nOutput: %s\n", err, string(output)))
			} else {
				allOutput.WriteString("With su - Success:\n")
				allOutput.WriteString(string(output))
			}
		} else {
			allOutput.WriteString("Without su - Success:\n")
			allOutput.WriteString(string(output))
		}
		allOutput.WriteString("\n")
	}

	// Check if /sdcard is a symlink and where it points to
	allOutput.WriteString("\n=== Checking /sdcard symlink ===\n")
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	cmd = exec.CommandContext(ctx, adbPath, "-s", adbAddress, "shell", "ls", "-l", "/sdcard")
	output, err = cmd.CombinedOutput()
	cancel()
	if err == nil {
		allOutput.WriteString(string(output))
	} else {
		allOutput.WriteString(fmt.Sprintf("Error: %v\nOutput: %s\n", err, string(output)))
	}

	// Write to file
	if err := os.WriteFile(outputFile, []byte(allOutput.String()), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
