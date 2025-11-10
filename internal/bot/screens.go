package bot

import (
	"fmt"
	"time"

	"jordanella.com/pocket-tcg-go/internal/cv"
)

// ScreenState represents the current game screen
type ScreenState int

const (
	// ScreenUnknown - cannot determine current screen
	ScreenUnknown ScreenState = iota

	// Main screens
	ScreenHome     // Home/main menu
	ScreenPack     // Pack selection screen
	ScreenMission  // Mission/quest screen
	ScreenShop     // Shop screen
	ScreenSocial   // Friends/social screen
	ScreenBattle   // Battle screen
	ScreenDeck     // Deck builder screen
	ScreenGacha    // Wonder pick screen

	// Pack opening states
	ScreenPackOpening   // Pack animation playing
	ScreenCardsRevealed // Cards shown after opening

	// Loading/transition states
	ScreenLoading // Generic loading screen
	ScreenError   // Error popup/screen
	ScreenMaintenance // Maintenance notification

	// Account states
	ScreenLogin      // Login screen
	ScreenTutorial   // Tutorial/onboarding
	ScreenNewAccount // New account setup
)

// String returns human-readable screen name
func (s ScreenState) String() string {
	switch s {
	case ScreenHome:
		return "Home"
	case ScreenPack:
		return "Pack"
	case ScreenMission:
		return "Mission"
	case ScreenShop:
		return "Shop"
	case ScreenSocial:
		return "Social"
	case ScreenBattle:
		return "Battle"
	case ScreenDeck:
		return "Deck"
	case ScreenGacha:
		return "Gacha"
	case ScreenPackOpening:
		return "PackOpening"
	case ScreenCardsRevealed:
		return "CardsRevealed"
	case ScreenLoading:
		return "Loading"
	case ScreenError:
		return "Error"
	case ScreenMaintenance:
		return "Maintenance"
	case ScreenLogin:
		return "Login"
	case ScreenTutorial:
		return "Tutorial"
	case ScreenNewAccount:
		return "NewAccount"
	default:
		return "Unknown"
	}
}

// Template paths for each screen
var screenTemplates = map[ScreenState]string{
	ScreenHome:          "templates/home_screen.png",
	ScreenPack:          "templates/pack_screen.png",
	ScreenMission:       "templates/mission_screen.png",
	ScreenShop:          "templates/shop_screen.png",
	ScreenSocial:        "templates/social_screen.png",
	ScreenBattle:        "templates/battle_screen.png",
	ScreenDeck:          "templates/deck_screen.png",
	ScreenGacha:         "templates/gacha_screen.png",
	ScreenPackOpening:   "templates/pack_opening.png",
	ScreenCardsRevealed: "templates/cards_revealed.png",
	ScreenLoading:       "templates/loading.png",
	ScreenError:         "templates/error_popup.png",
	ScreenMaintenance:   "templates/maintenance.png",
	ScreenLogin:         "templates/login_screen.png",
	ScreenTutorial:      "templates/tutorial.png",
	ScreenNewAccount:    "templates/new_account.png",
}

// ScreenDetectionResult contains detection details
type ScreenDetectionResult struct {
	Screen     ScreenState
	Confidence float64
	Detected   time.Time
}

// DetectCurrentScreen identifies which screen the bot is currently on
func (b *Bot) DetectCurrentScreen() ScreenState {
	result := b.DetectCurrentScreenWithConfidence()
	return result.Screen
}

// DetectCurrentScreenWithConfidence returns screen with confidence score
func (b *Bot) DetectCurrentScreenWithConfidence() *ScreenDetectionResult {
	// Get current frame (uses frame cache for performance)
	frame, err := b.cv.CaptureFrame(true)
	if err != nil {
		return &ScreenDetectionResult{
			Screen:     ScreenUnknown,
			Confidence: 0.0,
			Detected:   time.Now(),
		}
	}

	config := &cv.MatchConfig{
		Method:    cv.MatchMethodSSD,
		Threshold: 0.75, // Lower threshold for screen detection
	}

	bestScreen := ScreenUnknown
	bestConfidence := 0.0

	// Check each screen template
	for screen, templatePath := range screenTemplates {
		result, err := b.cv.FindTemplateInFrame(frame, templatePath, config)
		if err != nil {
			continue // Skip on error (template not found, etc.)
		}

		if result.Found && result.Confidence > bestConfidence {
			bestConfidence = result.Confidence
			bestScreen = screen
		}
	}

	return &ScreenDetectionResult{
		Screen:     bestScreen,
		Confidence: bestConfidence,
		Detected:   time.Now(),
	}
}

// IsOnScreen checks if currently on a specific screen
func (b *Bot) IsOnScreen(expected ScreenState) bool {
	detected := b.DetectCurrentScreen()
	return detected == expected
}

// WaitForScreen waits until a specific screen appears
func (b *Bot) WaitForScreen(expected ScreenState, timeout time.Duration) error {
	start := time.Now()

	for {
		// Force fresh capture each check
		b.cv.InvalidateCache()

		detected := b.DetectCurrentScreen()
		if detected == expected {
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting for screen %s (detected: %s)", expected, detected)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// WaitForAnyScreen waits for any of the specified screens
func (b *Bot) WaitForAnyScreen(screens []ScreenState, timeout time.Duration) (ScreenState, error) {
	start := time.Now()

	for {
		b.cv.InvalidateCache()

		detected := b.DetectCurrentScreen()

		// Check if detected screen is in our list
		for _, screen := range screens {
			if detected == screen {
				return detected, nil
			}
		}

		if time.Since(start) > timeout {
			return ScreenUnknown, fmt.Errorf("timeout waiting for screens (detected: %s)", detected)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// WaitForScreenChange waits for screen to change from current state
func (b *Bot) WaitForScreenChange(timeout time.Duration) (ScreenState, error) {
	initial := b.DetectCurrentScreen()
	start := time.Now()

	for {
		b.cv.InvalidateCache()
		time.Sleep(100 * time.Millisecond)

		detected := b.DetectCurrentScreen()
		if detected != initial && detected != ScreenUnknown {
			return detected, nil
		}

		if time.Since(start) > timeout {
			return ScreenUnknown, fmt.Errorf("timeout waiting for screen change from %s", initial)
		}
	}
}

// IsNavigableScreen returns true if screen can be navigated from
func (s ScreenState) IsNavigableScreen() bool {
	switch s {
	case ScreenHome, ScreenPack, ScreenMission, ScreenShop, ScreenSocial,
		ScreenBattle, ScreenDeck, ScreenGacha:
		return true
	default:
		return false
	}
}

// IsTransitionScreen returns true if screen is a transition/loading state
func (s ScreenState) IsTransitionScreen() bool {
	switch s {
	case ScreenLoading, ScreenPackOpening:
		return true
	default:
		return false
	}
}

// IsErrorScreen returns true if screen represents an error state
func (s ScreenState) IsErrorScreen() bool {
	switch s {
	case ScreenError, ScreenMaintenance:
		return true
	default:
		return false
	}
}

// GetTemplatePath returns the template file path for a screen
func (s ScreenState) GetTemplatePath() string {
	if path, ok := screenTemplates[s]; ok {
		return path
	}
	return ""
}

// DetectMultipleScreens checks for multiple screens simultaneously
func (b *Bot) DetectMultipleScreens(screens []ScreenState) map[ScreenState]*cv.MatchResult {
	frame, err := b.cv.CaptureFrame(true)
	if err != nil {
		return nil
	}

	config := &cv.MatchConfig{
		Method:    cv.MatchMethodSSD,
		Threshold: 0.75,
	}

	results := make(map[ScreenState]*cv.MatchResult)

	for _, screen := range screens {
		templatePath := screen.GetTemplatePath()
		if templatePath == "" {
			continue
		}

		result, err := b.cv.FindTemplateInFrame(frame, templatePath, config)
		if err != nil {
			continue
		}

		results[screen] = result
	}

	return results
}

// ScreenHistory tracks recent screen states for debugging
type ScreenHistory struct {
	States   []ScreenDetectionResult
	MaxSize  int
	Position int
}

// NewScreenHistory creates a new screen history tracker
func NewScreenHistory(maxSize int) *ScreenHistory {
	return &ScreenHistory{
		States:  make([]ScreenDetectionResult, 0, maxSize),
		MaxSize: maxSize,
	}
}

// Add records a screen detection
func (sh *ScreenHistory) Add(result *ScreenDetectionResult) {
	if len(sh.States) < sh.MaxSize {
		sh.States = append(sh.States, *result)
	} else {
		sh.States[sh.Position] = *result
		sh.Position = (sh.Position + 1) % sh.MaxSize
	}
}

// GetRecent returns the last N screen detections
func (sh *ScreenHistory) GetRecent(n int) []ScreenDetectionResult {
	size := len(sh.States)
	if n > size {
		n = size
	}

	recent := make([]ScreenDetectionResult, 0, n)
	for i := 0; i < n; i++ {
		idx := (sh.Position - 1 - i + size) % size
		if idx < 0 {
			idx += size
		}
		recent = append(recent, sh.States[idx])
	}

	return recent
}

// GetLastScreen returns the most recent screen detection
func (sh *ScreenHistory) GetLastScreen() ScreenState {
	if len(sh.States) == 0 {
		return ScreenUnknown
	}

	idx := sh.Position - 1
	if idx < 0 {
		idx = len(sh.States) - 1
	}

	return sh.States[idx].Screen
}
