package adb

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// SwipeParams defines parameters for swipe gestures
type SwipeParams struct {
	X1, Y1, X2, Y2 int
	Duration       int // milliseconds
}

func translateX(x int) int {
	return int((float64(540) / float64(277)) * float64(x))
}

func translateY(y int) int {
	return int((float64(960) / float64(489)) * float64(y-44))
}

// Click performs a tap at the specified coordinates
func (c *Controller) Click(x, y int) error {
	cmd := fmt.Sprintf("input tap %d %d", translateX(x), translateY(y))
	fmt.Println(x, y)
	fmt.Println(translateX(x), translateY(y))
	_, err := c.Shell(cmd)
	return err
}

// Swipe performs a swipe gesture
func (c *Controller) Swipe(X1, Y1, X2, Y2 int, duration int) error {
	cmd := fmt.Sprintf("input swipe %d %d %d %d %d",
		translateX(X1), translateY(Y1), translateX(X2), translateX(Y2), duration)
	_, err := c.Shell(cmd)
	return err
}

// SendKey sends a key event (e.g., "KEYCODE_BACK", "KEYCODE_HOME")
func (c *Controller) SendKey(key string) error {
	cmd := fmt.Sprintf("input keyevent %s", key)
	_, err := c.Shell(cmd)
	return err
}

// Input sends text input
func (c *Controller) Input(text string) error {
	// Escape spaces and special characters
	escapedText := strings.ReplaceAll(text, " ", "%s")
	cmd := fmt.Sprintf("input text %s", escapedText)
	_, err := c.Shell(cmd)
	return err
}

// Push copies a file from local to device
func (c *Controller) Push(localPath, remotePath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cmd := exec.Command(c.path, "-s", c.device, "push", localPath, remotePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("push failed: %w, output: %s", err, output)
	}
	return nil
}

// Pull copies a file from device to local
func (c *Controller) Pull(remotePath, localPath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cmd := exec.Command(c.path, "-s", c.device, "pull", remotePath, localPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pull failed: %w, output: %s", err, output)
	}
	return nil
}

// ForceStop stops an application
func (c *Controller) ForceStop(packageName string) error {
	cmd := fmt.Sprintf("am force-stop %s", packageName)
	_, err := c.Shell(cmd)
	return err
}

// StartApp starts an application
func (c *Controller) StartApp(packageName, activity string) error {
	cmd := fmt.Sprintf("am start -n %s/%s", packageName, activity)
	_, err := c.Shell(cmd)
	return err
}

// ClearAppData clears application data
func (c *Controller) ClearAppData(packageName string) error {
	cmd := fmt.Sprintf("pm clear %s", packageName)
	output, err := c.Shell(cmd)
	if err != nil {
		return err
	}
	if !strings.Contains(output, "Success") {
		return fmt.Errorf("failed to clear app data: %s", output)
	}
	return nil
}

// Shell executes a shell command and returns output
func (c *Controller) Shell(command string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// For commands that need immediate execution (not using persistent shell)
	cmd := exec.Command(c.path, "-s", c.device, "shell", command)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("shell command failed: %w, output: %s", err, output)
	}

	return strings.TrimSpace(string(output)), nil
}

// ShellWithTimeout executes a shell command with a timeout
func (c *Controller) ShellWithTimeout(command string, timeout time.Duration) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cmd := exec.Command(c.path, "-s", c.device, "shell", command)

	// Set up timeout
	done := make(chan error, 1)
	var output []byte
	var err error

	go func() {
		output, err = cmd.CombinedOutput()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("shell command failed: %w, output: %s", err, output)
		}
		return strings.TrimSpace(string(output)), nil
	case <-time.After(timeout):
		cmd.Process.Kill()
		return "", fmt.Errorf("shell command timed out after %v", timeout)
	}
}

// WaitADB waits for ADB to be ready (mimics AHK's waitadb)
func (c *Controller) WaitADB() error {
	// Small delay to ensure ADB command completion
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Screenshot takes a screenshot and saves it to the specified path
func (c *Controller) Screenshot(localPath string) error {
	tempPath := "/sdcard/screenshot.png"

	// Capture screenshot on device
	if _, err := c.Shell(fmt.Sprintf("screencap -p %s", tempPath)); err != nil {
		return fmt.Errorf("failed to capture screenshot: %w", err)
	}

	// Pull to local
	if err := c.Pull(tempPath, localPath); err != nil {
		return fmt.Errorf("failed to pull screenshot: %w", err)
	}

	// Clean up
	c.Shell(fmt.Sprintf("rm %s", tempPath))

	return nil
}

// GetWindowSize returns the current window/screen size
func (c *Controller) GetWindowSize() (width, height int, err error) {
	output, err := c.Shell("wm size")
	if err != nil {
		return 0, 0, err
	}

	// Parse output like "Physical size: 1080x1920"
	var w, h int
	_, err = fmt.Sscanf(output, "Physical size: %dx%d", &w, &h)
	if err != nil {
		// Try override format
		_, err = fmt.Sscanf(output, "Override size: %dx%d", &w, &h)
		if err != nil {
			return 0, 0, fmt.Errorf("failed to parse window size: %s", output)
		}
	}

	return w, h, nil
}

// GetCurrentActivity returns the current foreground activity
func (c *Controller) GetCurrentActivity() (string, error) {
	output, err := c.Shell("dumpsys window windows | grep -E 'mCurrentFocus'")
	if err != nil {
		return "", err
	}
	return output, nil
}

// IsAppRunning checks if an app is currently running
func (c *Controller) IsAppRunning(packageName string) (bool, error) {
	output, err := c.Shell(fmt.Sprintf("pidof %s", packageName))
	if err != nil {
		return false, nil // pidof returns error if not found
	}
	return len(strings.TrimSpace(output)) > 0, nil
}
