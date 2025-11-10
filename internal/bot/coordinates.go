package bot

import (
	"fmt"
)

// CoordinateTranslator handles translation between different coordinate systems
type CoordinateTranslator struct {
	config CoordinateConfig
}

// NewCoordinateTranslator creates a new coordinator translator with the given configuration
func NewCoordinateTranslator(config CoordinateConfig) *CoordinateTranslator {
	return &CoordinateTranslator{
		config: config,
	}
}

// TranslateX translates an X coordinate from source to target coordinate system
func (ct *CoordinateTranslator) TranslateX(x int) int {
	if ct.config.SourceWidth == 0 || ct.config.TargetWidth == 0 {
		// No translation if source or target is invalid
		return x
	}

	// Scale from source coordinate system to target device screen
	scaleFactorX := float64(ct.config.TargetWidth) / float64(ct.config.SourceWidth)
	translated := int(float64(x) * scaleFactorX)

	return translated
}

// TranslateY translates a Y coordinate from source to target coordinate system
// This accounts for the title bar offset
func (ct *CoordinateTranslator) TranslateY(y int) int {
	if ct.config.SourceHeight == 0 || ct.config.TargetHeight == 0 {
		// No translation if source or target is invalid
		return y
	}

	// First subtract the title bar offset from the source coordinate
	// (template coordinates are relative to game board, not window)
	yRelative := y - ct.config.TitleBarHeight

	// Scale from source coordinate system to target device screen
	scaleFactorY := float64(ct.config.TargetHeight) / float64(ct.config.SourceHeight)
	translated := int(float64(yRelative) * scaleFactorY)

	return translated
}

// TranslatePoint translates a point (x, y) from source to target coordinate system
func (ct *CoordinateTranslator) TranslatePoint(x, y int) (int, int) {
	return ct.TranslateX(x), ct.TranslateY(y)
}

// TranslateRegion translates a rectangular region from source to target coordinate system
func (ct *CoordinateTranslator) TranslateRegion(x1, y1, x2, y2 int) (int, int, int, int) {
	tx1, ty1 := ct.TranslatePoint(x1, y1)
	tx2, ty2 := ct.TranslatePoint(x2, y2)
	return tx1, ty1, tx2, ty2
}

// GetScaleFactors returns the X and Y scale factors
func (ct *CoordinateTranslator) GetScaleFactors() (float64, float64) {
	scaleX := 1.0
	scaleY := 1.0

	if ct.config.SourceWidth != 0 && ct.config.TargetWidth != 0 {
		scaleX = float64(ct.config.TargetWidth) / float64(ct.config.SourceWidth)
	}

	if ct.config.SourceHeight != 0 && ct.config.TargetHeight != 0 {
		scaleY = float64(ct.config.TargetHeight) / float64(ct.config.SourceHeight)
	}

	return scaleX, scaleY
}

// GetConfig returns the current coordinate configuration
func (ct *CoordinateTranslator) GetConfig() CoordinateConfig {
	return ct.config
}

// Validate ensures the coordinate configuration is valid
func (ct *CoordinateTranslator) Validate() error {
	if ct.config.SourceWidth <= 0 {
		return fmt.Errorf("invalid SourceWidth: %d (must be > 0)", ct.config.SourceWidth)
	}
	if ct.config.SourceHeight <= 0 {
		return fmt.Errorf("invalid SourceHeight: %d (must be > 0)", ct.config.SourceHeight)
	}
	if ct.config.TargetWidth <= 0 {
		return fmt.Errorf("invalid TargetWidth: %d (must be > 0)", ct.config.TargetWidth)
	}
	if ct.config.TargetHeight <= 0 {
		return fmt.Errorf("invalid TargetHeight: %d (must be > 0)", ct.config.TargetHeight)
	}
	if ct.config.TitleBarHeight < 0 {
		return fmt.Errorf("invalid TitleBarHeight: %d (must be >= 0)", ct.config.TitleBarHeight)
	}
	if ct.config.ScaleFactor <= 0 {
		return fmt.Errorf("invalid ScaleFactor: %f (must be > 0)", ct.config.ScaleFactor)
	}

	return nil
}

// String returns a string representation of the translator configuration
func (ct *CoordinateTranslator) String() string {
	scaleX, scaleY := ct.GetScaleFactors()
	return fmt.Sprintf("CoordinateTranslator{Source: %dx%d, Target: %dx%d, TitleBar: %dpx, ScaleX: %.3f, ScaleY: %.3f}",
		ct.config.SourceWidth, ct.config.SourceHeight,
		ct.config.TargetWidth, ct.config.TargetHeight,
		ct.config.TitleBarHeight,
		scaleX, scaleY)
}
