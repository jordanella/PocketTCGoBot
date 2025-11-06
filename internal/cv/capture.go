package cv

import (
	"image"
)

// Capturer interface for different capture methods
type Capturer interface {
	CaptureFrame() (*image.RGBA, error)
	GetDimensions() (width, height int)
}

// CaptureMethod defines how frames are captured
type CaptureMethod int

const (
	// CaptureMethodWindow captures directly from window handle (fastest)
	CaptureMethodWindow CaptureMethod = iota
	// CaptureMethodADB captures via ADB screenshot (slower, fallback)
	CaptureMethodADB
)

// CaptureConfig holds configuration for frame capture
type CaptureConfig struct {
	Method       CaptureMethod
	WindowHandle uintptr       // For window capture
	UseCache     bool          // Cache last frame for template matching
	MaxCacheDuration int       // Max milliseconds to cache frame
}

// DefaultCaptureConfig returns recommended capture configuration
func DefaultCaptureConfig() *CaptureConfig {
	return &CaptureConfig{
		Method:           CaptureMethodWindow,
		UseCache:         true,
		MaxCacheDuration: 100, // 100ms cache for rapid template checks
	}
}
