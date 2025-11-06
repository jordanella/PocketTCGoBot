package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
	"jordanella.com/pocket-tcg-go/internal/cv"
)

// TemplateRegistry manages a dynamic collection of templates loaded from YAML files
type TemplateRegistry struct {
	mu         sync.RWMutex
	templates  map[string]cv.Template
	basePath   string      // Base path for template image files
	imageCache *ImageCache // Optional: for caching loaded images
}

// TemplateDefinition represents a template in the YAML file
type TemplateDefinition struct {
	Name        string     `yaml:"name"`
	Path        string     `yaml:"path"`
	Threshold   float64    `yaml:"threshold"`
	Region      *RegionDef `yaml:"region,omitempty"`
	Scale       float64    `yaml:"scale,omitempty"`
	Preload     bool       `yaml:"preload,omitempty"`      // Load image at startup
	UnloadAfter bool       `yaml:"unload_after,omitempty"` // Unload after use
}

// RegionDef represents a region in the YAML file
type RegionDef struct {
	X1 int `yaml:"x1"`
	Y1 int `yaml:"y1"`
	X2 int `yaml:"x2"`
	Y2 int `yaml:"y2"`
}

// TemplateFile represents the structure of a template YAML file
type TemplateFile struct {
	Templates []TemplateDefinition `yaml:"templates"`
}

// NewTemplateRegistry creates a new template registry
// basePath is the root directory where template image files are stored
func NewTemplateRegistry(basePath string) *TemplateRegistry {
	return &TemplateRegistry{
		templates:  make(map[string]cv.Template),
		basePath:   basePath,
		imageCache: NewImageCache(),
	}
}

// WithoutImageCache disables image caching for this registry
func (tr *TemplateRegistry) WithoutImageCache() *TemplateRegistry {
	tr.imageCache = nil
	return tr
}

// LoadFromFile loads templates from a YAML file
func (tr *TemplateRegistry) LoadFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", filePath, err)
	}

	var templateFile TemplateFile
	if err := yaml.Unmarshal(data, &templateFile); err != nil {
		return fmt.Errorf("failed to unmarshal template YAML: %w", err)
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()

	for i, def := range templateFile.Templates {
		if def.Name == "" {
			return fmt.Errorf("template %d: name cannot be empty", i+1)
		}
		if def.Path == "" {
			return fmt.Errorf("template %d (%s): path cannot be empty", i+1, def.Name)
		}

		// Convert the definition to a cv.Template
		template := cv.Template{
			Name:      def.Name,
			Path:      filepath.Join(tr.basePath, def.Path),
			Threshold: def.Threshold,
			Scale:     def.Scale,
		}

		// Convert region if present
		if def.Region != nil {
			template.Region = &cv.Region{
				X1: def.Region.X1,
				Y1: def.Region.Y1,
				X2: def.Region.X2,
				Y2: def.Region.Y2,
			}
		}

		// Set default threshold if not specified
		if template.Threshold == 0 {
			template.Threshold = 0.8
		}

		tr.templates[def.Name] = template

		// Register with image cache if enabled
		if tr.imageCache != nil {
			if err := tr.imageCache.Register(template, def.Preload, def.UnloadAfter); err != nil {
				// Don't fail loading, just log the preload failure
				// The image can still be loaded on-demand
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}
		}
	}

	return nil
}

// LoadFromDirectory loads all YAML files from a directory
func (tr *TemplateRegistry) LoadFromDirectory(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read template directory %s: %w", dirPath, err)
	}

	var loadErrors []error
	loadedCount := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .yaml and .yml files
		ext := filepath.Ext(entry.Name())
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		fullPath := filepath.Join(dirPath, entry.Name())
		if err := tr.LoadFromFile(fullPath); err != nil {
			loadErrors = append(loadErrors, fmt.Errorf("file %s: %w", entry.Name(), err))
		} else {
			loadedCount++
		}
	}

	if len(loadErrors) > 0 {
		// Return first error but log that there were multiple
		return fmt.Errorf("failed to load %d template files (first error): %w", len(loadErrors), loadErrors[0])
	}

	return nil
}

// Get retrieves a template by name
// Returns the template and true if found, or an empty template and false if not found
func (tr *TemplateRegistry) Get(name string) (cv.Template, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	template, ok := tr.templates[name]
	return template, ok
}

// MustGet retrieves a template by name and panics if not found
// Use this only during initialization or when the template is guaranteed to exist
func (tr *TemplateRegistry) MustGet(name string) cv.Template {
	template, ok := tr.Get(name)
	if !ok {
		panic(fmt.Sprintf("template '%s' not found in registry", name))
	}
	return template
}

// GetOrDefault retrieves a template by name, or returns a default template if not found
func (tr *TemplateRegistry) GetOrDefault(name string, defaultThreshold float64) cv.Template {
	template, ok := tr.Get(name)
	if !ok {
		// Return a basic template with the name and default threshold
		return cv.Template{
			Name:      name,
			Path:      filepath.Join(tr.basePath, name+".png"),
			Threshold: defaultThreshold,
		}
	}
	return template
}

// Register adds a template to the registry programmatically
func (tr *TemplateRegistry) Register(template cv.Template) error {
	if template.Name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.templates[template.Name] = template
	return nil
}

// RegisterBatch adds multiple templates to the registry
func (tr *TemplateRegistry) RegisterBatch(templates []cv.Template) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	for i, template := range templates {
		if template.Name == "" {
			return fmt.Errorf("template %d: name cannot be empty", i)
		}
		tr.templates[template.Name] = template
	}

	return nil
}

// Has checks if a template exists in the registry
func (tr *TemplateRegistry) Has(name string) bool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	_, ok := tr.templates[name]
	return ok
}

// List returns all template names in the registry
func (tr *TemplateRegistry) List() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	names := make([]string, 0, len(tr.templates))
	for name := range tr.templates {
		names = append(names, name)
	}
	return names
}

// Count returns the number of templates in the registry
func (tr *TemplateRegistry) Count() int {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	return len(tr.templates)
}

// Clear removes all templates from the registry
func (tr *TemplateRegistry) Clear() {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.templates = make(map[string]cv.Template)
}

// Remove removes a template from the registry
func (tr *TemplateRegistry) Remove(name string) bool {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if _, ok := tr.templates[name]; ok {
		delete(tr.templates, name)
		// Also unload from cache if present
		if tr.imageCache != nil {
			tr.imageCache.Release(name)
		}
		return true
	}
	return false
}

// ImageCache returns the image cache (if enabled)
func (tr *TemplateRegistry) ImageCache() *ImageCache {
	return tr.imageCache
}

// PreloadAll preloads all templates marked for preloading
func (tr *TemplateRegistry) PreloadAll() error {
	if tr.imageCache == nil {
		return fmt.Errorf("image cache not enabled")
	}
	return tr.imageCache.PreloadAll()
}

// UnloadAll unloads all cached images
func (tr *TemplateRegistry) UnloadAll() {
	if tr.imageCache != nil {
		tr.imageCache.UnloadAll()
	}
}

// CacheStats returns image cache statistics
func (tr *TemplateRegistry) CacheStats() CacheStats {
	if tr.imageCache == nil {
		return CacheStats{}
	}
	return tr.imageCache.Stats()
}

// Global registry instance (for backward compatibility)
var globalRegistry *TemplateRegistry
var once sync.Once

// GlobalRegistry returns the singleton global template registry
// This is initialized lazily on first access
func GlobalRegistry() *TemplateRegistry {
	once.Do(func() {
		globalRegistry = NewTemplateRegistry("templates")
	})
	return globalRegistry
}

// InitializeGlobalRegistry initializes the global registry with a specific base path
// This should be called once during application startup
func InitializeGlobalRegistry(basePath string) *TemplateRegistry {
	once.Do(func() {
		globalRegistry = NewTemplateRegistry(basePath)
	})
	return globalRegistry
}

// Convenience functions that use the global registry

// Get retrieves a template from the global registry
func Get(name string) (cv.Template, bool) {
	return GlobalRegistry().Get(name)
}

// MustGet retrieves a template from the global registry and panics if not found
func MustGet(name string) cv.Template {
	return GlobalRegistry().MustGet(name)
}

// Has checks if a template exists in the global registry
func Has(name string) bool {
	return GlobalRegistry().Has(name)
}

// LoadFromFile loads templates from a file into the global registry
func LoadFromFile(filePath string) error {
	return GlobalRegistry().LoadFromFile(filePath)
}

// LoadFromDirectory loads templates from a directory into the global registry
func LoadFromDirectory(dirPath string) error {
	return GlobalRegistry().LoadFromDirectory(dirPath)
}
