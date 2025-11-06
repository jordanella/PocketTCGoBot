package cv

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"sync"
	"time"
)

// TemplateRegistryInterface defines interface for template registry access
type TemplateRegistryInterface interface {
	Get(name string) (Template, bool)
	ImageCache() ImageCacheInterface
}

// ImageCacheInterface defines interface for image cache access
type ImageCacheInterface interface {
	Get(name string) (*image.RGBA, Template, error)
	Release(name string) error
}

// Service handles all computer vision operations
type Service struct {
	capturer         Capturer
	templateCache    map[string]*image.RGBA
	templateRegistry TemplateRegistryInterface // Optional: for cached template images

	// Frame caching for performance
	cachedFrame     *image.RGBA
	cachedFrameTime time.Time
	cacheDuration   time.Duration

	// Title bar exclusion
	titleBarHeight int // Pixels to exclude from top of window

	mu sync.RWMutex
}

// NewService creates a new CV service
func NewService(capturer Capturer) *Service {
	return &Service{
		capturer:       capturer,
		templateCache:  make(map[string]*image.RGBA),
		cacheDuration:  100 * time.Millisecond,
		titleBarHeight: 0, // No exclusion by default
	}
}

// NewServiceWithCache creates a CV service with custom cache duration
func NewServiceWithCache(capturer Capturer, cacheDuration time.Duration) *Service {
	return &Service{
		capturer:       capturer,
		templateCache:  make(map[string]*image.RGBA),
		cacheDuration:  cacheDuration,
		titleBarHeight: 0,
	}
}

// NewServiceWithTitleBar creates a CV service with title bar exclusion
func NewServiceWithTitleBar(capturer Capturer, titleBarHeight int) *Service {
	return &Service{
		capturer:       capturer,
		templateCache:  make(map[string]*image.RGBA),
		cacheDuration:  100 * time.Millisecond,
		titleBarHeight: titleBarHeight,
	}
}

// WithTemplateRegistry sets the template registry for image caching
func (s *Service) WithTemplateRegistry(registry TemplateRegistryInterface) *Service {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templateRegistry = registry
	return s
}

// SetTitleBarHeight updates the title bar exclusion height
func (s *Service) SetTitleBarHeight(height int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.titleBarHeight = height
}

// GetTitleBarHeight returns the current title bar exclusion height
func (s *Service) GetTitleBarHeight() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.titleBarHeight
}

// CaptureFrame captures current window frame with optional caching
func (s *Service) CaptureFrame(useCache bool) (*image.RGBA, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if cached frame is still valid
	if useCache && s.cachedFrame != nil {
		elapsed := time.Since(s.cachedFrameTime)
		if elapsed < s.cacheDuration {
			return s.cachedFrame, nil
		}
	}

	// Capture new frame
	frame, err := s.capturer.CaptureFrame()
	if err != nil {
		return nil, err
	}

	// Update cache
	if useCache {
		s.cachedFrame = frame
		s.cachedFrameTime = time.Now()
	}

	return frame, nil
}

// InvalidateCache forces next capture to get fresh frame
func (s *Service) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cachedFrame = nil
}

// GetDimensions returns the capture dimensions
func (s *Service) GetDimensions() (width, height int) {
	return s.capturer.GetDimensions()
}

// FindTemplate finds a template by name in the current frame
func (s *Service) FindTemplate(templateName string, config *MatchConfig) (*MatchResult, error) {
	// Get cached frame
	frame, err := s.CaptureFrame(true)
	if err != nil {
		return nil, fmt.Errorf("failed to capture frame: %w", err)
	}

	// Load template (with caching)
	template, err := s.loadTemplate(templateName)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	// Perform matching
	if config == nil {
		config = DefaultMatchConfig()
	}

	// Apply title bar exclusion if not already set
	s.applyTitleBarExclusion(config, frame.Bounds())

	result := FindTemplate(frame, template, config)
	return result, nil
}

// FindTemplateInFrame finds template in a specific frame
func (s *Service) FindTemplateInFrame(frame *image.RGBA, templatePath string, config *MatchConfig) (*MatchResult, error) {
	template, err := s.loadTemplate(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %w", err)
	}

	if config == nil {
		config = DefaultMatchConfig()
	}

	// Apply title bar exclusion if not already set
	s.applyTitleBarExclusion(config, frame.Bounds())

	result := FindTemplate(frame, template, config)
	return result, nil
}

// FindMultipleTemplates finds multiple templates in current frame
func (s *Service) FindMultipleTemplates(templatePaths []string, config *MatchConfig) (map[string]*MatchResult, error) {
	frame, err := s.CaptureFrame(true)
	if err != nil {
		return nil, fmt.Errorf("failed to capture frame: %w", err)
	}

	results := make(map[string]*MatchResult)
	for _, path := range templatePaths {
		result, err := s.FindTemplateInFrame(frame, path, config)
		if err != nil {
			continue // Skip failed templates
		}
		results[path] = result
	}

	return results, nil
}

// WaitForTemplate waits until template appears (or timeout)
func (s *Service) WaitForTemplate(templateName string, config *MatchConfig, timeout time.Duration) (*MatchResult, error) {
	start := time.Now()

	for {
		// Always get fresh frame when waiting
		s.InvalidateCache()
		result, err := s.FindTemplate(templateName, config)
		if err != nil {
			return nil, err
		}

		if result.Found {
			return result, nil
		}

		if time.Since(start) > timeout {
			return nil, fmt.Errorf("template not found within timeout")
		}

		time.Sleep(50 * time.Millisecond)
	}
}

// CheckColor checks if a specific pixel has expected color
func (s *Service) CheckColor(x, y int, expected color.Color, tolerance uint8) (bool, error) {
	frame, err := s.CaptureFrame(true)
	if err != nil {
		return false, err
	}

	bounds := frame.Bounds()
	if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
		return false, fmt.Errorf("coordinates out of bounds")
	}

	actual := frame.At(x, y)
	r1, g1, b1, _ := actual.RGBA()
	r2, g2, b2, _ := expected.RGBA()

	// Convert to 8-bit
	r1, g1, b1 = r1>>8, g1>>8, b1>>8
	r2, g2, b2 = r2>>8, g2>>8, b2>>8

	distance := colorDistance(uint8(r1), uint8(g1), uint8(b1), uint8(r2), uint8(g2), uint8(b2))
	return distance <= tolerance, nil
}

// GetPixelColor returns color at specific pixel
func (s *Service) GetPixelColor(x, y int) (color.Color, error) {
	frame, err := s.CaptureFrame(true)
	if err != nil {
		return nil, err
	}

	bounds := frame.Bounds()
	if x < bounds.Min.X || x >= bounds.Max.X || y < bounds.Min.Y || y >= bounds.Max.Y {
		return nil, fmt.Errorf("coordinates out of bounds")
	}

	return frame.At(x, y), nil
}

// Template management

func (s *Service) loadTemplate(templateName string) (*image.RGBA, error) {
	// First check the service's local cache
	s.mu.RLock()
	if cached, ok := s.templateCache[templateName]; ok {
		s.mu.RUnlock()
		return cached, nil
	}

	// Try to get from template registry's image cache if available
	registry := s.templateRegistry
	s.mu.RUnlock()

	if registry != nil {
		imageCache := registry.ImageCache()
		if imageCache != nil {
			// Try to get by template name (path might be the template name)
			img, _, err := imageCache.Get(templateName)
			if err == nil {
				// Cache it locally too for consistency
				s.mu.Lock()
				s.templateCache[templateName] = img
				s.mu.Unlock()
				return img, nil
			}
		}
	}

	// Fallback: template not in image cache but might be in registry
	// Get template from registry to access its path
	if registry != nil {
		if template, ok := registry.Get(templateName); ok {
			// Load from template's actual path
			file, err := os.Open(string(template.Path))
			if err != nil {
				return nil, fmt.Errorf("failed to open template file %s: %w", template.Path, err)
			}
			defer file.Close()

			img, err := png.Decode(file)
			if err != nil {
				return nil, fmt.Errorf("failed to decode template: %w", err)
			}

			// Convert to RGBA
			var rgba *image.RGBA
			if r, ok := img.(*image.RGBA); ok {
				rgba = r
			} else {
				bounds := img.Bounds()
				rgba = image.NewRGBA(bounds)
				for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
					for x := bounds.Min.X; x < bounds.Max.X; x++ {
						rgba.Set(x, y, img.At(x, y))
					}
				}
			}

			// Cache it locally
			s.mu.Lock()
			s.templateCache[templateName] = rgba
			s.mu.Unlock()

			return rgba, nil
		}
	}

	// Template not found in registry
	return nil, fmt.Errorf("template '%s' not found in registry", templateName)
}

// ClearTemplateCache clears template cache (useful if templates change)
func (s *Service) ClearTemplateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templateCache = make(map[string]*image.RGBA)
}

// applyTitleBarExclusion applies title bar exclusion to match config if not already set
func (s *Service) applyTitleBarExclusion(config *MatchConfig, bounds image.Rectangle) {
	s.mu.RLock()
	titleBarHeight := s.titleBarHeight
	s.mu.RUnlock()

	// Only apply if title bar height is set and no search region is already defined
	if titleBarHeight > 0 && config.SearchRegion == nil {
		config.SearchRegion = &image.Rectangle{
			Min: image.Point{X: bounds.Min.X, Y: bounds.Min.Y + titleBarHeight},
			Max: bounds.Max,
		}
	}
}
