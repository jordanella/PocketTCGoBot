package emulator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

// MuMu Player constants
const (
	MuMuClassName     = "Qt5150QWindowIcon"
	MuMuProcessName   = "MuMuPlayer.exe"
	MuMuBasePort      = 16384
	MuMuPortIncrement = 32
)

// MuMuVersion represents MuMu Player version
type MuMuVersion int

const (
	MuMuUnknown MuMuVersion = iota
	MuMuV5                  // MuMu Player 5 (older)
	MuMuV12                 // MuMu Player 12 (newer)
)

// MuMuInstance represents a MuMu Player instance
type MuMuInstance struct {
	Index        int
	WindowTitle  string
	WindowHandle uintptr
	ADBPort      int
	Version      MuMuVersion
	PlayerName   string // Custom player name from config
	X, Y         int    // Window position
	Width        int    // Window width
	Height       int    // Window height
}

// MuMuExtraConfig represents the extra_config.json structure
type MuMuExtraConfig struct {
	RelateId       string `json:"relateId"`
	PlayerName     string `json:"playerName"`
	Status         int    `json:"status"`
	ErrorCode      int    `json:"errorCode"`
	CreateTime     int64  `json:"createTime"`
	ImportFilePath string `json:"importFilePath"`
}

// MuMuManager manages MuMu Player instances
type MuMuManager struct {
	folderPath string
	version    MuMuVersion
	instances  []*MuMuInstance
}

// NewMuMuManager creates a new MuMu manager
func NewMuMuManager(folderPath string) *MuMuManager {
	mgr := &MuMuManager{
		folderPath: folderPath,
		instances:  make([]*MuMuInstance, 0),
	}
	mgr.detectVersion()
	return mgr
}

// detectVersion detects MuMu Player version
func (m *MuMuManager) detectVersion() {
	// Check for MuMu 12
	paths := []string{
		filepath.Join(m.folderPath, "MuMuPlayerGlobal-12.0"),
		filepath.Join(m.folderPath, "MuMu Player 12"),
	}

	for _, path := range paths {
		nxMainPath := filepath.Join(path, "nx_main")
		if _, err := os.Stat(nxMainPath); err == nil {
			m.version = MuMuV12
			return
		}
	}

	m.version = MuMuV5
}

// GetVersion returns detected MuMu version
func (m *MuMuManager) GetVersion() MuMuVersion {
	return m.version
}

// GetTitleHeight returns title bar height for the detected version
func (m *MuMuManager) GetTitleHeight() int {
	if m.version == MuMuV12 {
		return 50
	}
	return 45
}

// FindInstances discovers all running MuMu instances
// Uses config files as source of truth and matches windows by player name
func (m *MuMuManager) FindInstances() ([]*MuMuInstance, error) {
	m.instances = make([]*MuMuInstance, 0)

	// First, load all instance configs (source of truth)
	configs, err := m.GetAllInstanceConfigs()
	if err != nil {
		return nil, fmt.Errorf("failed to load instance configs: %w", err)
	}

	// Build a map of playerName -> instanceIndex from configs
	nameToIndex := make(map[string]int)
	for index, config := range configs {
		if config.PlayerName != "" {
			nameToIndex[config.PlayerName] = index
		}
	}

	// Enumerate all windows
	var enumCallback = syscall.NewCallback(func(hwnd syscall.Handle, lparam uintptr) uintptr {
		// Get window title
		titleLen := sendMessage(hwnd, WM_GETTEXTLENGTH, 0, 0) + 1
		if titleLen <= 1 {
			return 1 // Continue enumeration
		}

		title := make([]uint16, titleLen)
		sendMessage(hwnd, WM_GETTEXT, uintptr(titleLen), uintptr(unsafe.Pointer(&title[0])))
		titleStr := syscall.UTF16ToString(title)

		// Verify it's actually MuMu by checking class name
		className := make([]uint16, 256)
		getClassName(hwnd, &className[0], 256)
		classNameStr := syscall.UTF16ToString(className)

		if strings.Contains(classNameStr, "Qt") {
			// Try to match window title to a player name from configs
			instanceIndex, found := nameToIndex[titleStr]
			if !found {
				// If no player name match, skip this window
				return 1
			}

			// Found a match! Use the config-based instance index
			instance := &MuMuInstance{
				Index:        instanceIndex,
				WindowTitle:  titleStr,
				WindowHandle: uintptr(hwnd),
				ADBPort:      MuMuBasePort + (instanceIndex * MuMuPortIncrement),
				Version:      m.version,
				PlayerName:   titleStr,
			}

			// Get window position
			var rect RECT
			getWindowRect(hwnd, &rect)
			instance.X = int(rect.Left)
			instance.Y = int(rect.Top)
			instance.Width = int(rect.Right - rect.Left)
			instance.Height = int(rect.Bottom - rect.Top)

			m.instances = append(m.instances, instance)
		}

		return 1 // Continue enumeration
	})

	enumWindows(enumCallback, 0)

	return m.instances, nil
}

// GetInstance returns a specific instance by index
func (m *MuMuManager) GetInstance(index int) (*MuMuInstance, error) {
	for _, inst := range m.instances {
		if inst.Index == index {
			return inst, nil
		}
	}
	return nil, fmt.Errorf("instance %d not found", index)
}

// PositionWindow positions a window based on grid layout
func (m *MuMuManager) PositionWindow(instance *MuMuInstance, config *WindowConfig) error {
	if instance.WindowHandle == 0 {
		return fmt.Errorf("invalid window handle")
	}

	// Calculate position
	x, y := config.CalculatePosition(instance.Index, m.GetTitleHeight())
	width := config.ScaleParam
	height := m.GetTitleHeight() + 489 + 4 // titleHeight + game height + border

	// Remove title bar
	hwnd := syscall.Handle(instance.WindowHandle)
	style := getWindowLong(hwnd, GWL_STYLE)
	setWindowLong(hwnd, GWL_STYLE, style&^WS_CAPTION)

	// Move and resize window
	setWindowPos(hwnd, 0, int32(x), int32(y), int32(width), int32(height), SWP_NOZORDER|SWP_FRAMECHANGED)

	// Restore title bar
	setWindowLong(hwnd, GWL_STYLE, style)

	// Redraw window
	invalidateRect(hwnd, nil, true)

	// Update instance position
	instance.X = x
	instance.Y = y
	instance.Width = width
	instance.Height = height

	return nil
}

// WindowConfig holds window positioning configuration
type WindowConfig struct {
	Columns       int
	RowGap        int
	ScaleParam    int
	MonitorIndex  int
	MonitorLeft   int
	MonitorTop    int
	MonitorRight  int
	MonitorBottom int
}

// NewWindowConfig creates window config from bot config
func NewWindowConfig(columns, rowGap, scaleParam, monitorIndex int) *WindowConfig {
	config := &WindowConfig{
		Columns:      columns,
		RowGap:       rowGap,
		ScaleParam:   scaleParam,
		MonitorIndex: monitorIndex,
	}

	// Get monitor info
	config.getMonitorInfo()

	return config
}

// getMonitorInfo retrieves monitor bounds
func (c *WindowConfig) getMonitorInfo() {
	// For now, use primary monitor
	// TODO: Support multiple monitors
	c.MonitorLeft = 0
	c.MonitorTop = 0

	// Get screen dimensions
	c.MonitorRight = int(getSystemMetrics(SM_CXSCREEN))
	c.MonitorBottom = int(getSystemMetrics(SM_CYSCREEN))
}

// CalculatePosition calculates window position based on grid layout
func (c *WindowConfig) CalculatePosition(instanceIndex, titleHeight int) (x, y int) {
	rowHeight := titleHeight + 489 + 4

	currentRow := (instanceIndex - 1) / c.Columns
	col := (instanceIndex - 1) % c.Columns

	y = c.MonitorTop + (currentRow * rowHeight) + (currentRow * c.RowGap)
	x = c.MonitorLeft + (col * c.ScaleParam)

	return x, y
}

// Windows API constants and functions
const (
	WM_GETTEXT       = 0x000D
	WM_GETTEXTLENGTH = 0x000E
	GWL_STYLE        = -16
	WS_CAPTION       = 0x00C00000
	SWP_NOZORDER     = 0x0004
	SWP_FRAMECHANGED = 0x0020
	SM_CXSCREEN      = 0
	SM_CYSCREEN      = 1
)

type RECT struct {
	Left, Top, Right, Bottom int32
}

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procGetWindowTextLength = user32.NewProc("GetWindowTextLengthW")
	procGetClassName        = user32.NewProc("GetClassNameW")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procGetWindowLong       = user32.NewProc("GetWindowLongW")
	procSetWindowLong       = user32.NewProc("SetWindowLongW")
	procInvalidateRect      = user32.NewProc("InvalidateRect")
	procSendMessage         = user32.NewProc("SendMessageW")
	procGetSystemMetrics    = user32.NewProc("GetSystemMetrics")
)

func enumWindows(callback uintptr, lparam uintptr) {
	procEnumWindows.Call(callback, lparam)
}

func getClassName(hwnd syscall.Handle, className *uint16, maxCount int) {
	procGetClassName.Call(uintptr(hwnd), uintptr(unsafe.Pointer(className)), uintptr(maxCount))
}

func getWindowRect(hwnd syscall.Handle, rect *RECT) {
	procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)))
}

func setWindowPos(hwnd syscall.Handle, hWndInsertAfter uintptr, x, y, cx, cy int32, flags uint32) {
	procSetWindowPos.Call(
		uintptr(hwnd),
		hWndInsertAfter,
		uintptr(x),
		uintptr(y),
		uintptr(cx),
		uintptr(cy),
		uintptr(flags),
	)
}

func getWindowLong(hwnd syscall.Handle, index int) uint32 {
	ret, _, _ := procGetWindowLong.Call(uintptr(hwnd), uintptr(index))
	return uint32(ret)
}

func setWindowLong(hwnd syscall.Handle, index int, newLong uint32) {
	procSetWindowLong.Call(uintptr(hwnd), uintptr(index), uintptr(newLong))
}

func invalidateRect(hwnd syscall.Handle, rect *RECT, erase bool) {
	var eraseVal uintptr
	if erase {
		eraseVal = 1
	}
	procInvalidateRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)), eraseVal)
}

func sendMessage(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	ret, _, _ := procSendMessage.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
	return ret
}

func getSystemMetrics(index int) int32 {
	ret, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int32(ret)
}

// LaunchInstance launches a MuMu instance by index
func (m *MuMuManager) LaunchInstance(index int) error {
	fmt.Printf("[LaunchInstance] Starting launch for instance %d\n", index)
	fmt.Printf("[LaunchInstance] Folder path: %s\n", m.folderPath)

	// Find MuMuPlayer executable
	var mumuExePath string

	// Try different possible locations based on version
	possiblePaths := []string{
		filepath.Join(m.folderPath, "MuMuPlayerGlobal-12.0", "shell", "MuMuPlayer.exe"),
		filepath.Join(m.folderPath, "MuMu Player 12", "shell", "MuMuPlayer.exe"),
		filepath.Join(m.folderPath, "shell", "MuMuPlayer.exe"),
		filepath.Join(m.folderPath, "MuMuPlayer.exe"),
	}

	fmt.Printf("[LaunchInstance] Searching for MuMuPlayer.exe...\n")
	for _, path := range possiblePaths {
		fmt.Printf("[LaunchInstance] Trying: %s\n", path)
		if _, err := os.Stat(path); err == nil {
			mumuExePath = path
			fmt.Printf("[LaunchInstance] Found at: %s\n", mumuExePath)
			break
		}
	}

	if mumuExePath == "" {
		return fmt.Errorf("MuMuPlayer.exe not found in %s", m.folderPath)
	}

	// Launch with instance index as parameter
	// Use ShellExecute to launch without elevated privileges (MuMu has issues when run as admin)
	args := fmt.Sprintf("-v %d", index)
	fmt.Printf("[LaunchInstance] Launching: %s %s\n", mumuExePath, args)

	// Use Windows ShellExecute via COM to launch without elevation
	fmt.Printf("[LaunchInstance] Calling shellExecuteNonElevated...\n")
	if err := shellExecuteNonElevated(mumuExePath, args); err != nil {
		fmt.Printf("[LaunchInstance] ERROR: %v\n", err)
		return fmt.Errorf("failed to launch MuMu instance %d: %w", index, err)
	}

	fmt.Printf("[LaunchInstance] Launch successful!\n")
	return nil
}

// IsInstanceRunning checks if an instance is currently running
func (m *MuMuManager) IsInstanceRunning(index int) bool {
	for _, inst := range m.instances {
		if inst.Index == index {
			return true
		}
	}
	return false
}

// shellExecuteNonElevated launches a program without elevated privileges using ShellExecute
// This is necessary because MuMu Player has issues when run as administrator
func shellExecuteNonElevated(file, args string) error {
	// Convert strings to UTF16
	filePtr, err := syscall.UTF16PtrFromString(file)
	if err != nil {
		return err
	}

	var argsPtr *uint16
	if args != "" {
		argsPtr, err = syscall.UTF16PtrFromString(args)
		if err != nil {
			return err
		}
	}

	verbPtr, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}

	// Get working directory from file path
	dir := filepath.Dir(file)
	dirPtr, err := syscall.UTF16PtrFromString(dir)
	if err != nil {
		return err
	}

	// Load shell32.dll and get ShellExecuteW
	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecute := shell32.NewProc("ShellExecuteW")

	// Call ShellExecuteW
	// HINSTANCE ShellExecuteW(
	//   HWND    hwnd,
	//   LPCWSTR lpOperation,
	//   LPCWSTR lpFile,
	//   LPCWSTR lpParameters,
	//   LPCWSTR lpDirectory,
	//   INT     nShowCmd
	// )
	ret, _, err := shellExecute.Call(
		0,                                // hwnd
		uintptr(unsafe.Pointer(verbPtr)), // lpOperation = "open"
		uintptr(unsafe.Pointer(filePtr)), // lpFile
		uintptr(unsafe.Pointer(argsPtr)), // lpParameters
		uintptr(unsafe.Pointer(dirPtr)),  // lpDirectory
		uintptr(1),                       // nShowCmd = SW_SHOWNORMAL
	)

	// ShellExecute returns a value > 32 on success
	if ret <= 32 {
		if ret == 0 {
			return fmt.Errorf("ShellExecute failed: out of memory or resources")
		}
		return fmt.Errorf("ShellExecute failed with error code: %d", ret)
	}

	return nil
}

// ReadInstanceConfig reads the extra_config.json for a specific instance
func (m *MuMuManager) ReadInstanceConfig(instanceIndex int) (*MuMuExtraConfig, error) {
	// Construct path to vms folder
	vmsPath := filepath.Join(m.folderPath, "vms")

	// Look for the instance folder
	instanceFolder := filepath.Join(vmsPath, fmt.Sprintf("MuMuPlayerGlobal-12.0-%d", instanceIndex))
	configPath := filepath.Join(instanceFolder, "configs", "extra_config.json")

	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config for instance %d: %w", instanceIndex, err)
	}

	// Parse JSON
	var config MuMuExtraConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config for instance %d: %w", instanceIndex, err)
	}

	return &config, nil
}

// GetAllInstanceConfigs reads all available instance configurations from the vms folder
func (m *MuMuManager) GetAllInstanceConfigs() (map[int]*MuMuExtraConfig, error) {
	configs := make(map[int]*MuMuExtraConfig)

	// Construct path to vms folder
	vmsPath := filepath.Join(m.folderPath, "vms")

	// Check if vms folder exists
	if _, err := os.Stat(vmsPath); os.IsNotExist(err) {
		return configs, fmt.Errorf("vms folder not found at %s", vmsPath)
	}

	// Read all directories in vms folder
	entries, err := os.ReadDir(vmsPath)
	if err != nil {
		return configs, fmt.Errorf("failed to read vms folder: %w", err)
	}

	// Parse each instance folder
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Extract instance number from folder name (e.g., "MuMuPlayerGlobal-12.0-1" -> 1)
		folderName := entry.Name()
		if !strings.HasPrefix(folderName, "MuMuPlayerGlobal-12.0-") {
			continue
		}

		instanceStr := strings.TrimPrefix(folderName, "MuMuPlayerGlobal-12.0-")
		if instanceStr == "base" {
			continue // Skip base folder
		}

		var instanceIndex int
		if _, err := fmt.Sscanf(instanceStr, "%d", &instanceIndex); err != nil {
			continue // Skip if not a number
		}

		// Try to read config for this instance
		config, err := m.ReadInstanceConfig(instanceIndex)
		if err != nil {
			// Skip instances without valid config
			continue
		}

		configs[instanceIndex] = config
	}

	return configs, nil
}
