package gui

import "jordanella.com/pocket-tcg-go/internal/bot"

// GUIConfig wraps bot.Config and provides GUI-friendly accessors
// This adapter allows the GUI to work with a simplified config structure
type GUIConfig struct {
	*bot.Config
}

// ADBConfig represents ADB-related settings
type ADBConfig struct {
	Path string
}

// MuMuConfig represents MuMu emulator settings
type MuMuConfig struct {
	Path         string
	WindowWidth  int
	WindowHeight int
}

// ActionsConfig represents action timing settings
type ActionsConfig struct {
	DelayBetweenActions int
	ScreenshotDelay     int
}

// LoggingConfig represents logging settings
type LoggingConfig struct {
	Enabled bool
	Level   string
}

// NewGUIConfig creates a GUI config wrapper
func NewGUIConfig(cfg *bot.Config) *GUIConfig {
	if cfg == nil {
		cfg = &bot.Config{}
	}
	return &GUIConfig{Config: cfg}
}

// ADB returns ADB configuration (adapter method)
func (g *GUIConfig) ADB() ADBConfig {
	// In the actual bot.Config, there's no ADB path field
	// We can use FolderPath as a proxy or add it later
	return ADBConfig{
		Path: g.FolderPath, // Placeholder - actual ADB path would need to be added to bot.Config
	}
}

// MuMu returns MuMu configuration (adapter method)
func (g *GUIConfig) MuMu() MuMuConfig {
	// bot.Config doesn't have window dimensions either
	// These would need to be added or derived from DefaultLanguage
	width := 540  // Default for Scale100
	height := 960 // Default for Scale100

	if g.DefaultLanguage == "Scale125" {
		width = 675
		height = 1200
	}

	return MuMuConfig{
		Path:         g.FolderPath,
		WindowWidth:  width,
		WindowHeight: height,
	}
}

// Actions returns actions configuration (adapter method)
func (g *GUIConfig) Actions() ActionsConfig {
	return ActionsConfig{
		DelayBetweenActions: g.Delay,
		ScreenshotDelay:     g.WaitTime * 1000, // Convert seconds to ms
	}
}

// Logging returns logging configuration (adapter method)
func (g *GUIConfig) Logging() LoggingConfig {
	level := "INFO"
	if g.VerboseLogging {
		level = "DEBUG"
	}

	return LoggingConfig{
		Enabled: true, // Always enabled for now
		Level:   level,
	}
}

// SetADB updates ADB configuration
func (g *GUIConfig) SetADB(adb ADBConfig) {
	g.FolderPath = adb.Path
}

// SetMuMu updates MuMu configuration
func (g *GUIConfig) SetMuMu(mumu MuMuConfig) {
	g.FolderPath = mumu.Path
	// Window dimensions would need to be stored if we add fields to bot.Config
}

// SetActions updates actions configuration
func (g *GUIConfig) SetActions(actions ActionsConfig) {
	g.Delay = actions.DelayBetweenActions
	g.WaitTime = actions.ScreenshotDelay / 1000 // Convert ms to seconds
}

// SetLogging updates logging configuration
func (g *GUIConfig) SetLogging(logging LoggingConfig) {
	g.VerboseLogging = (logging.Level == "DEBUG")
}
