package adb

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
)

// ADB controller type and lifecycle
type Controller struct {
	path     string
	port     string
	shell    *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	mu       sync.Mutex
	device   string // Device ID: "127.0.0.1:port"
	connected bool
}

// NewController creates a new ADB controller
func NewController(adbPath, port string) *Controller {
	return &Controller{
		path:   adbPath,
		port:   port,
		device: fmt.Sprintf("127.0.0.1:%s", port),
	}
}

// Connect establishes connection to the ADB device
func (c *Controller) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Connect to device
	cmd := exec.Command(c.path, "connect", c.device)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to connect to device %s: %w, output: %s", c.device, err, output)
	}

	// Verify connection
	if !strings.Contains(string(output), "connected") && !strings.Contains(string(output), "already connected") {
		return fmt.Errorf("unexpected connect output: %s", output)
	}

	c.connected = true

	// Start persistent shell for faster commands
	if err := c.startShell(); err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	return nil
}

// startShell starts a persistent ADB shell session
func (c *Controller) startShell() error {
	c.shell = exec.Command(c.path, "-s", c.device, "shell")

	var err error
	c.stdin, err = c.shell.StdinPipe()
	if err != nil {
		return err
	}

	c.stdout, err = c.shell.StdoutPipe()
	if err != nil {
		return err
	}

	c.stderr, err = c.shell.StderrPipe()
	if err != nil {
		return err
	}

	if err := c.shell.Start(); err != nil {
		return err
	}

	return nil
}

// Disconnect closes the ADB connection
func (c *Controller) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.shell != nil && c.shell.Process != nil {
		c.stdin.Close()
		c.shell.Process.Kill()
		c.shell.Wait()
	}

	c.connected = false
	return nil
}

// IsConnected returns whether the controller is connected
func (c *Controller) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}
