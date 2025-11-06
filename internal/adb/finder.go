package adb

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// FindADB attempts to locate the ADB executable
func FindADB(preferredPath string) (string, error) {
	// Try preferred path first
	if preferredPath != "" {
		adbPath := filepath.Join(preferredPath, "adb", "adb.exe")
		if runtime.GOOS != "windows" {
			adbPath = filepath.Join(preferredPath, "adb", "adb")
		}

		if _, err := os.Stat(adbPath); err == nil {
			return adbPath, nil
		}
	}

	// Try common paths
	commonPaths := []string{
		// MuMu Player
		`C:\Program Files\Netease\MuMuPlayer-12.0\shell\adb.exe`,
		`C:\Program Files (x86)\Netease\MuMuPlayer-12.0\shell\adb.exe`,

		// Android SDK
		`C:\Android\sdk\platform-tools\adb.exe`,
		`C:\Users\%USERNAME%\AppData\Local\Android\Sdk\platform-tools\adb.exe`,

		// PATH
		"adb.exe",
	}

	if runtime.GOOS != "windows" {
		commonPaths = []string{
			"/usr/bin/adb",
			"/usr/local/bin/adb",
			"~/Android/Sdk/platform-tools/adb",
			"adb",
		}
	}

	for _, path := range commonPaths {
		// Expand environment variables
		expandedPath := os.ExpandEnv(path)

		if _, err := os.Stat(expandedPath); err == nil {
			return expandedPath, nil
		}

		// Try exec.LookPath for PATH entries
		if !strings.Contains(path, string(filepath.Separator)) {
			if adbPath, err := exec.LookPath(path); err == nil {
				return adbPath, nil
			}
		}
	}

	return "", fmt.Errorf("adb not found, please specify path in config")
}

// DetectMuMuPort attempts to detect the MuMu emulator port
func DetectMuMuPort() (string, error) {
	// Common MuMu ports
	commonPorts := []string{
		"16416", // MuMu instance 1 (most common)
		"16448", // MuMu instance 2
		"16480", // MuMu instance 3
		"5555",  // Generic Android emulator
	}

	// Try to list devices
	adbPath, err := FindADB("")
	if err != nil {
		return "", err
	}

	cmd := exec.Command(adbPath, "devices")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	// Parse output for connected devices
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "127.0.0.1:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				device := parts[0]
				port := strings.TrimPrefix(device, "127.0.0.1:")
				return port, nil
			}
		}
	}

	// If no devices found, try common ports
	for _, port := range commonPorts {
		// Try to connect
		cmd := exec.Command(adbPath, "connect", fmt.Sprintf("127.0.0.1:%s", port))
		output, err := cmd.CombinedOutput()
		if err == nil && strings.Contains(string(output), "connected") {
			return port, nil
		}
	}

	return "", fmt.Errorf("could not detect MuMu port")
}

// ConnectADB is a helper function to find and connect to ADB
func ConnectADB(folderPath string) (*Controller, error) {
	// Find ADB
	adbPath, err := FindADB(folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find ADB: %w", err)
	}

	// Detect port
	port, err := DetectMuMuPort()
	if err != nil {
		// Default to 16416 (MuMu instance 1)
		port = "16416"
	}

	// Create controller
	ctrl := NewController(adbPath, port)

	// Connect
	if err := ctrl.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to device: %w", err)
	}

	return ctrl, nil
}
