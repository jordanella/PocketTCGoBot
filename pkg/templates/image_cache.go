package templates

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"sync"

	"jordanella.com/pocket-tcg-go/internal/cv"
)

// CachedTemplate extends cv.Template with image caching capabilities
type CachedTemplate struct {
	cv.Template
	image       *image.RGBA  // Cached image data
	mu          sync.RWMutex // Protects image field
	preload     bool         // Whether to preload image at startup
	unloadAfter bool         // Whether to unload after use
	useCount    int          // Number of times loaded (for stats)
}

// ImageCache manages template image loading and caching
type ImageCache struct {
	templates map[string]*CachedTemplate
	mu        sync.RWMutex
	stats     CacheStats
}

// CacheStats tracks cache performance
type CacheStats struct {
	Hits        int64 // Cache hits
	Misses      int64 // Cache misses (had to load)
	Loads       int64 // Total load operations
	Unloads     int64 // Total unload operations
	PreloadFail int64 // Failed preloads
}

// NewImageCache creates a new image cache
func NewImageCache() *ImageCache {
	return &ImageCache{
		templates: make(map[string]*CachedTemplate),
	}
}

// Register adds a template to the cache
func (ic *ImageCache) Register(template cv.Template, preload, unloadAfter bool) error {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	cached := &CachedTemplate{
		Template:    template,
		preload:     preload,
		unloadAfter: unloadAfter,
	}

	// Preload if requested
	if preload {
		if err := cached.load(); err != nil {
			ic.stats.PreloadFail++
			return fmt.Errorf("failed to preload template %s: %w", template.Name, err)
		}
		ic.stats.Loads++
	}

	ic.templates[template.Name] = cached
	return nil
}

// Get retrieves a template and its image, loading if necessary
func (ic *ImageCache) Get(name string) (*image.RGBA, cv.Template, error) {
	ic.mu.RLock()
	cached, ok := ic.templates[name]
	ic.mu.RUnlock()

	if !ok {
		return nil, cv.Template{}, fmt.Errorf("template '%s' not found in cache", name)
	}

	// Get or load image
	img, err := cached.getOrLoad()
	if err != nil {
		return nil, cv.Template{}, err
	}

	// Update stats
	ic.mu.Lock()
	if cached.image != nil && cached.useCount > 0 {
		ic.stats.Hits++
	} else {
		ic.stats.Misses++
	}
	ic.mu.Unlock()

	return img, cached.Template, nil
}

// Release unloads a template image if unloadAfter is set
func (ic *ImageCache) Release(name string) error {
	ic.mu.RLock()
	cached, ok := ic.templates[name]
	ic.mu.RUnlock()

	if !ok {
		return fmt.Errorf("template '%s' not found in cache", name)
	}

	if cached.unloadAfter {
		if err := cached.unload(); err != nil {
			return err
		}
		ic.mu.Lock()
		ic.stats.Unloads++
		ic.mu.Unlock()
	}

	return nil
}

// PreloadAll loads all templates marked for preloading
func (ic *ImageCache) PreloadAll() error {
	ic.mu.RLock()
	templates := make([]*CachedTemplate, 0, len(ic.templates))
	for _, t := range ic.templates {
		if t.preload {
			templates = append(templates, t)
		}
	}
	ic.mu.RUnlock()

	var errors []error
	for _, cached := range templates {
		if err := cached.load(); err != nil {
			errors = append(errors, fmt.Errorf("template %s: %w", cached.Name, err))
			ic.mu.Lock()
			ic.stats.PreloadFail++
			ic.mu.Unlock()
		} else {
			ic.mu.Lock()
			ic.stats.Loads++
			ic.mu.Unlock()
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to preload %d templates: %v", len(errors), errors[0])
	}

	return nil
}

// UnloadAll unloads all cached images
func (ic *ImageCache) UnloadAll() {
	ic.mu.RLock()
	templates := make([]*CachedTemplate, 0, len(ic.templates))
	for _, t := range templates {
		templates = append(templates, t)
	}
	ic.mu.RUnlock()

	for _, cached := range templates {
		cached.unload()
		ic.mu.Lock()
		ic.stats.Unloads++
		ic.mu.Unlock()
	}
}

// Stats returns cache statistics
func (ic *ImageCache) Stats() CacheStats {
	ic.mu.RLock()
	defer ic.mu.RUnlock()
	return ic.stats
}

// CachedTemplate methods

// getOrLoad returns the cached image or loads it if not cached
func (ct *CachedTemplate) getOrLoad() (*image.RGBA, error) {
	// Fast path: image already loaded
	ct.mu.RLock()
	if ct.image != nil {
		defer ct.mu.RUnlock()
		return ct.image, nil
	}
	ct.mu.RUnlock()

	// Slow path: need to load
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Double-check after acquiring write lock
	if ct.image != nil {
		return ct.image, nil
	}

	return ct.loadUnsafe()
}

// load loads the template image (thread-safe)
func (ct *CachedTemplate) load() error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if ct.image != nil {
		return nil // Already loaded
	}

	_, err := ct.loadUnsafe()
	return err
}

// loadUnsafe loads the image without locking (caller must hold lock)
func (ct *CachedTemplate) loadUnsafe() (*image.RGBA, error) {
	// Check if file exists
	if _, err := os.Stat(string(ct.Path)); os.IsNotExist(err) {
		return nil, fmt.Errorf("template image not found: %s", ct.Path)
	}

	// Open and decode PNG
	file, err := os.Open(string(ct.Path))
	if err != nil {
		return nil, fmt.Errorf("failed to open template: %w", err)
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

	ct.image = rgba
	ct.useCount++

	return ct.image, nil
}

// unload releases the template image (thread-safe)
func (ct *CachedTemplate) unload() error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if ct.image != nil {
		// No need to close image.RGBA - Go GC will handle it
		ct.image = nil
	}

	return nil
}

// IsLoaded returns true if the image is currently in memory
func (ct *CachedTemplate) IsLoaded() bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()
	return ct.image != nil
}
