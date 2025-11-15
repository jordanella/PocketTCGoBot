package bot

import (
	"time"
)

// Configuration type - comprehensive settings from AHK bot
type Config struct {
	// Instance configuration
	Instance         int
	Columns          int
	RowGap           int
	SelectedMonitor  int
	DefaultLanguage  string // "Scale100" or "Scale125"
	FolderPath       string // Path to MuMu folder

	// Delete/Injection Methods
	DeleteMethod     DeleteMethod
	InjectSortMethod SortMethod
	InjectMinPacks   int
	InjectMaxPacks   int

	// Account waiting logic
	WaitForEligibleAccounts bool
	MaxWaitHours            int

	// Pack preferences (which packs to open)
	EnabledPacks map[string]bool // Mewtwo, Charizard, Pikachu, Mew, etc.
	ShinyPacks   map[string]bool

	// Star requirements (global and per-pack)
	MinStars      int
	MinStarsShiny int
	MinStarsPerPack map[string]int // Per-pack minimums

	// Pack validation criteria
	CheckShinyPackOnly bool
	TrainerCheck       bool
	FullArtCheck       bool
	RainbowCheck       bool
	ShinyCheck         bool
	CrownCheck         bool
	ImmersiveCheck     bool
	InvalidCheck       bool
	PseudoGodPack      bool

	// Mission settings
	SkipMissionsInjectMissions  bool
	ClaimSpecialMissions        bool
	ClaimDailyMission          bool
	WonderpickForEventMissions bool

	// Resource management
	SpendHourGlass bool
	OpenExtraPack  bool

	// Social features
	FriendID      string
	FriendIDs     []string
	CheckWPThanks bool
	ShowcaseEnabled bool

	// Save for Trade (S4T) integration
	S4TEnabled         bool
	S4TSilent          bool
	S4T3Diamond        bool
	S4T4Diamond        bool
	S4T1Star           bool
	S4TGholdengo       bool
	S4TTrainer         bool
	S4TRainbow         bool
	S4TFullArt         bool
	S4TCrown           bool
	S4TImmersive       bool
	S4TShiny1Star      bool
	S4TShiny2Star      bool
	S4TWonderPick      bool
	S4TWPMinCards      int
	S4TDiscordWebhook  string
	S4TDiscordUserID   string
	S4TSendAccountXml  bool

	// OCR settings
	OCRLanguage  string
	OCRShinedust bool

	// Bot behavior
	GodPackAction GodPackAction
	PackMethod    int
	NukeAccount   bool
	RunMain       bool
	Mains         int

	// Performance tuning
	Delay      int // milliseconds
	SwipeSpeed int // milliseconds
	SlowMotion bool
	WaitTime   int // seconds

	// Display/UI
	ShowStatus bool

	// Debug
	VerboseLogging bool
	DeadCheck      bool

	// Extended configuration for GUI and advanced features
	ADBPath          string // Path to ADB executable
	MuMuWindowWidth  int    // MuMu window width
	MuMuWindowHeight int    // MuMu window height
	TitleBarHeight   int    // Height of window title bar to exclude from searches (pixels)
	LogLevel         string // "DEBUG", "INFO", "WARN", "ERROR"
	LoggingEnabled   bool   // Whether logging is enabled

	// Coordinate Translation Settings
	SourceScreenWidth  int // Source coordinate system width (default: 277 for template coordinates)
	SourceScreenHeight int // Source coordinate system height (default: 489 for game board)
	GameBoardHeight    int // Actual game board height in pixels (default: 489)
	WindowBorderHeight int // Border/padding height in pixels (default: 4)

	// Multi-Instance Settings
	InstanceStartDelay  int // Delay in seconds between instance starts (default: 10)
	InstanceLaunchDelay int // Delay in seconds when launching emulator instances (default: 2)

	// Global Action Timing (defaults for actions that don't specify their own timing)
	GlobalClickDelay      int // Delay after click actions in milliseconds (default: uses Delay)
	GlobalSwipeDelay      int // Delay after swipe actions in milliseconds (default: uses SwipeSpeed)
	GlobalTemplateTimeout int // Default timeout for template matching in milliseconds (default: 5000)
	GlobalRetryAttempts   int // Default number of retry attempts for actions (default: 3)
	GlobalRetryDelay      int // Delay between retry attempts in milliseconds (default: 1000)

	// Monitor and Display Settings
	MonitorScaleFactor float64 // DPI scaling factor for monitor (default: 1.0 for 100%, 1.25 for 125%)
	MonitorOffsetX     int     // X offset for selected monitor (pixels)
	MonitorOffsetY     int     // Y offset for selected monitor (pixels)
}

type DeleteMethod int

const (
	DeleteMethodCreateBots DeleteMethod = iota
	DeleteMethodInject13P
	DeleteMethodInjectWonderPick96P
	DeleteMethodInjectMissions
)

func (d DeleteMethod) String() string {
	switch d {
	case DeleteMethodCreateBots:
		return "Create Bots (13P)"
	case DeleteMethodInject13P:
		return "Inject 13P+"
	case DeleteMethodInjectWonderPick96P:
		return "Inject Wonderpick 96P+"
	case DeleteMethodInjectMissions:
		return "Inject Missions"
	default:
		return "Unknown"
	}
}

type SortMethod int

const (
	SortMethodModifiedAsc SortMethod = iota
	SortMethodModifiedDesc
	SortMethodPacksAsc
	SortMethodPacksDesc
)

func (s SortMethod) String() string {
	switch s {
	case SortMethodModifiedAsc:
		return "ModifiedAsc"
	case SortMethodModifiedDesc:
		return "ModifiedDesc"
	case SortMethodPacksAsc:
		return "PacksAsc"
	case SortMethodPacksDesc:
		return "PacksDesc"
	default:
		return "ModifiedAsc"
	}
}

type GodPackAction int

const (
	GodPackClose GodPackAction = iota
	GodPackPause
	GodPackContinue
)

func (g GodPackAction) String() string {
	switch g {
	case GodPackClose:
		return "Close"
	case GodPackPause:
		return "Pause"
	case GodPackContinue:
		return "Continue"
	default:
		return "Continue"
	}
}

// Config loading and validation
func LoadConfig(path string) (*Config, error) {
	// Stub - use internal/config package instead
	return nil, nil
}

func (c *Config) Validate() error {
	// TODO: Implement validation
	return nil
}

func (c *Config) IsInjectMode() bool {
	return c.DeleteMethod != DeleteMethodCreateBots
}

// GUI-friendly accessor types
type ADBConfig struct {
	Path string
}

type MuMuConfig struct {
	Path         string
	WindowWidth  int
	WindowHeight int
}

type ActionsConfig struct {
	DelayBetweenActions int
	ScreenshotDelay     int
}

// GetDelayBetweenActions returns the delay between actions in milliseconds
func (ac ActionsConfig) GetDelayBetweenActions() int {
	return ac.DelayBetweenActions
}

// GetScreenshotDelay returns the screenshot delay in milliseconds
func (ac ActionsConfig) GetScreenshotDelay() int {
	return ac.ScreenshotDelay
}

type LoggingConfig struct {
	Enabled bool
	Level   string
}

// ADB returns ADB configuration
func (c *Config) ADB() ADBConfig {
	path := c.ADBPath
	if path == "" {
		// Default ADB path
		path = c.FolderPath + "\\vmonitor\\bin\\adb_server.exe"
	}
	return ADBConfig{Path: path}
}

// MuMu returns MuMu emulator configuration
func (c *Config) MuMu() MuMuConfig {
	width := c.MuMuWindowWidth
	height := c.MuMuWindowHeight

	// Default values based on scale if not set
	if width == 0 || height == 0 {
		if c.DefaultLanguage == "Scale125" {
			width = 675
			height = 1200
		} else {
			width = 540
			height = 960
		}
	}

	return MuMuConfig{
		Path:         c.FolderPath,
		WindowWidth:  width,
		WindowHeight: height,
	}
}

// Actions returns action timing configuration
func (c *Config) Actions() ActionsConfig {
	return ActionsConfig{
		DelayBetweenActions: c.Delay,
		ScreenshotDelay:     c.WaitTime * 1000, // Convert seconds to ms
	}
}

// Logging returns logging configuration
func (c *Config) Logging() LoggingConfig {
	enabled := c.LoggingEnabled
	level := c.LogLevel

	// Derive from other settings if not explicitly set
	if level == "" {
		if c.VerboseLogging {
			level = "DEBUG"
		} else {
			level = "INFO"
		}
	}

	return LoggingConfig{
		Enabled: enabled,
		Level:   level,
	}
}

// SetADB updates ADB configuration
func (c *Config) SetADB(adb ADBConfig) {
	c.ADBPath = adb.Path
}

// SetMuMu updates MuMu configuration
func (c *Config) SetMuMu(mumu MuMuConfig) {
	c.FolderPath = mumu.Path
	c.MuMuWindowWidth = mumu.WindowWidth
	c.MuMuWindowHeight = mumu.WindowHeight
}

// SetActions updates action timing configuration
func (c *Config) SetActions(actions ActionsConfig) {
	c.Delay = actions.DelayBetweenActions
	c.WaitTime = actions.ScreenshotDelay / 1000 // Convert ms to seconds
}

// SetLogging updates logging configuration
func (c *Config) SetLogging(logging LoggingConfig) {
	c.LoggingEnabled = logging.Enabled
	c.LogLevel = logging.Level
	c.VerboseLogging = (logging.Level == "DEBUG")
}

// ApplyDefaults sets default values for any uninitialized configuration fields
func (c *Config) ApplyDefaults() {
	// Coordinate translation defaults
	if c.SourceScreenWidth == 0 {
		c.SourceScreenWidth = 277 // Default template coordinate width
	}
	if c.SourceScreenHeight == 0 {
		c.SourceScreenHeight = 489 // Default game board height
	}
	if c.GameBoardHeight == 0 {
		c.GameBoardHeight = 489 // Actual game board height
	}
	if c.WindowBorderHeight == 0 {
		c.WindowBorderHeight = 4 // Default border/padding
	}

	// Multi-instance defaults
	if c.InstanceStartDelay == 0 {
		c.InstanceStartDelay = 10 // 10 seconds between instance starts
	}
	if c.InstanceLaunchDelay == 0 {
		c.InstanceLaunchDelay = 2 // 2 seconds when launching emulators
	}

	// Global action timing defaults
	if c.GlobalClickDelay == 0 {
		if c.Delay > 0 {
			c.GlobalClickDelay = c.Delay
		} else {
			c.GlobalClickDelay = 250 // Default 250ms
		}
	}
	if c.GlobalSwipeDelay == 0 {
		if c.SwipeSpeed > 0 {
			c.GlobalSwipeDelay = c.SwipeSpeed
		} else {
			c.GlobalSwipeDelay = 500 // Default 500ms
		}
	}
	if c.GlobalTemplateTimeout == 0 {
		c.GlobalTemplateTimeout = 5000 // Default 5 seconds
	}
	if c.GlobalRetryAttempts == 0 {
		c.GlobalRetryAttempts = 3 // Default 3 retry attempts
	}
	if c.GlobalRetryDelay == 0 {
		c.GlobalRetryDelay = 1000 // Default 1 second between retries
	}

	// Monitor scale factor based on DefaultLanguage
	if c.MonitorScaleFactor == 0 {
		if c.DefaultLanguage == "Scale125" {
			c.MonitorScaleFactor = 1.25
		} else {
			c.MonitorScaleFactor = 1.0
		}
	}

	// Title bar height default (if not already set)
	if c.TitleBarHeight == 0 {
		c.TitleBarHeight = 45 // Default for MuMu 5 (will be overridden by emulator manager)
	}

	// Basic timing defaults
	if c.Delay == 0 {
		c.Delay = 250
	}
	if c.SwipeSpeed == 0 {
		c.SwipeSpeed = 500
	}
	if c.WaitTime == 0 {
		c.WaitTime = 5
	}
}

// GetCoordinateTranslationConfig returns coordinate translation parameters
func (c *Config) GetCoordinateTranslationConfig() CoordinateConfig {
	// Ensure defaults are applied
	c.ApplyDefaults()

	return CoordinateConfig{
		SourceWidth:  c.SourceScreenWidth,
		SourceHeight: c.SourceScreenHeight,
		TargetWidth:  c.MuMuWindowWidth,
		TargetHeight: c.MuMuWindowHeight,
		TitleBarHeight: c.TitleBarHeight,
		GameBoardHeight: c.GameBoardHeight,
		ScaleFactor: c.MonitorScaleFactor,
	}
}

// CoordinateConfig holds coordinate translation parameters
type CoordinateConfig struct {
	SourceWidth     int     // Source coordinate system width (templates)
	SourceHeight    int     // Source coordinate system height (templates)
	TargetWidth     int     // Target device screen width
	TargetHeight    int     // Target device screen height
	TitleBarHeight  int     // Title bar height to offset Y coordinates
	GameBoardHeight int     // Actual game board height
	ScaleFactor     float64 // Monitor DPI scale factor
}

// RestartPolicy defines how bots should restart on failure
type RestartPolicy struct {
	Enabled        bool          `yaml:"enabled" json:"enabled"`               // Whether auto-restart is enabled
	MaxRetries     int           `yaml:"max_retries" json:"max_retries"`       // Maximum number of restart attempts (0 = unlimited)
	InitialDelay   time.Duration `yaml:"initial_delay" json:"initial_delay"`   // Initial backoff delay
	MaxDelay       time.Duration `yaml:"max_delay" json:"max_delay"`           // Maximum backoff delay
	BackoffFactor  float64       `yaml:"backoff_factor" json:"backoff_factor"` // Exponential backoff multiplier
	ResetOnSuccess bool          `yaml:"reset_on_success" json:"reset_on_success"` // Reset retry counter on successful execution
}

// DefaultRestartPolicy returns sensible defaults
func DefaultRestartPolicy() RestartPolicy {
	return RestartPolicy{
		Enabled:        false, // Disabled by default for safety
		MaxRetries:     3,
		InitialDelay:   5 * time.Second,
		MaxDelay:       5 * time.Minute,
		BackoffFactor:  2.0,
		ResetOnSuccess: true,
	}
}
