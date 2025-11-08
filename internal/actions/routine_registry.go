package actions

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

// RoutineMetadata stores information about a routine
type RoutineMetadata struct {
	Filename    string   // e.g., "common_navigation"
	DisplayName string   // e.g., "Common Navigation Routine"
	Description string   // Optional description of the routine's purpose
	Tags        []string // Optional tags for organization and filtering (e.g., "sentry", "navigation")
}

// RoutineRegistryExtendedInterface provides full access to the routine registry
// This extends the basic RoutineRegistryInterface with additional methods
type RoutineRegistryExtendedInterface interface {
	RoutineRegistryInterface

	// ListValid returns only valid routine filenames
	ListValid() []string

	// ListInvalid returns routine filenames that failed validation
	ListInvalid() []string
}

// RoutineRegistry manages routine loading and validation
// All routines are eagerly loaded and validated at initialization
type RoutineRegistry struct {
	mu               sync.RWMutex
	templateRegistry TemplateRegistryInterface
	routinesPath     string // Base path for routines (e.g., "routines/")

	// Pre-loaded routines (filename -> builder)
	routines map[string]*ActionBuilder

	// Routine sentries (filename -> sentries)
	sentries map[string][]Sentry

	// Routine metadata (filename -> metadata)
	metadata map[string]*RoutineMetadata

	// Validation errors (filename -> error)
	validationErrors map[string]error
}

// NewRoutineRegistry creates a new routine registry
// It scans the routines folder and eagerly loads all routines
func NewRoutineRegistry(routinesPath string) *RoutineRegistry {
	rr := &RoutineRegistry{
		routinesPath:     routinesPath,
		routines:         make(map[string]*ActionBuilder),
		sentries:         make(map[string][]Sentry),
		metadata:         make(map[string]*RoutineMetadata),
		validationErrors: make(map[string]error),
	}

	return rr
}

// WithTemplateRegistry sets the template registry and loads all routines
func (rr *RoutineRegistry) WithTemplateRegistry(registry TemplateRegistryInterface) *RoutineRegistry {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.templateRegistry = registry

	// Load all routines now that we have the template registry
	log.Printf("[RoutineRegistry] Loading routines from: %s", rr.routinesPath)
	rr.loadAllRoutines()

	validCount := len(rr.routines)
	invalidCount := len(rr.validationErrors)
	log.Printf("[RoutineRegistry] Loaded %d valid routine(s), %d invalid routine(s)", validCount, invalidCount)

	// Log invalid routines
	if invalidCount > 0 {
		for filename, err := range rr.validationErrors {
			log.Printf("[RoutineRegistry] ⚠️  Invalid routine '%s': %v", filename, err)
		}
	}

	return rr
}

// loadAllRoutines discovers and loads all routine files
func (rr *RoutineRegistry) loadAllRoutines() {
	// Check if the routines folder exists
	if _, err := os.Stat(rr.routinesPath); os.IsNotExist(err) {
		log.Printf("[RoutineRegistry] Routines folder not found: %s", rr.routinesPath)
		return
	}

	// Scan for .yaml and .yml files
	patterns := []string{
		filepath.Join(rr.routinesPath, "*.yaml"),
		filepath.Join(rr.routinesPath, "*.yml"),
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, path := range matches {
			filename := filepath.Base(path)
			// Remove extension to get routine filename
			ext := filepath.Ext(filename)
			routineFilename := filename[:len(filename)-len(ext)]

			// Load and validate the routine
			rr.loadRoutine(routineFilename, path)
		}
	}
}

// loadRoutine loads a single routine file
func (rr *RoutineRegistry) loadRoutine(filename string, path string) {
	// First, parse YAML to extract the routine_name for metadata
	data, err := os.ReadFile(path)
	if err != nil {
		rr.validationErrors[filename] = fmt.Errorf("failed to read file: %w", err)
		return
	}

	var routine Routine
	if err := yaml.Unmarshal(data, &routine); err != nil {
		rr.validationErrors[filename] = fmt.Errorf("failed to parse YAML: %w", err)
		return
	}

	// Store metadata
	displayName := routine.RoutineName
	if displayName == "" {
		displayName = filename // Fallback if routine_name not specified
	}
	rr.metadata[filename] = &RoutineMetadata{
		Filename:    filename,
		DisplayName: displayName,
		Description: routine.Description,
		Tags:        routine.Tags,
	}

	// Now load and validate with the loader
	loader := NewRoutineLoader()
	if rr.templateRegistry != nil {
		loader.WithTemplateRegistry(rr.templateRegistry)
	}

	builder, sentries, err := loader.LoadFromFile(path)
	if err != nil {
		// Store the validation error
		rr.validationErrors[filename] = fmt.Errorf("validation failed: %w", err)
		return
	}

	// Store the successfully loaded routine
	rr.routines[filename] = builder

	// Store sentries if any exist
	if len(sentries) > 0 {
		rr.sentries[filename] = sentries
		log.Printf("[RoutineRegistry] ✓ Loaded: %s (%s) with %d sentry/sentries", displayName, filename, len(sentries))
	} else {
		log.Printf("[RoutineRegistry] ✓ Loaded: %s (%s)", displayName, filename)
	}
}

// Get retrieves a pre-loaded routine by filename
func (rr *RoutineRegistry) Get(filename string) (*ActionBuilder, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	// Check if there was a validation error
	if err, hasError := rr.validationErrors[filename]; hasError {
		return nil, err
	}

	// Return the pre-loaded routine
	if builder, ok := rr.routines[filename]; ok {
		return builder, nil
	}

	return nil, fmt.Errorf("routine '%s' not found", filename)
}

// GetWithSentries retrieves a pre-loaded routine with its sentries
func (rr *RoutineRegistry) GetWithSentries(filename string) (*ActionBuilder, []Sentry, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	// Check if there was a validation error
	if err, hasError := rr.validationErrors[filename]; hasError {
		return nil, nil, err
	}

	// Return the pre-loaded routine and its sentries
	if builder, ok := rr.routines[filename]; ok {
		sentries := rr.sentries[filename] // Will be nil/empty if no sentries
		return builder, sentries, nil
	}

	return nil, nil, fmt.Errorf("routine '%s' not found", filename)
}

// GetSentries retrieves just the sentries for a routine
func (rr *RoutineRegistry) GetSentries(filename string) ([]Sentry, error) {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	// Check if routine exists
	if _, ok := rr.routines[filename]; !ok {
		return nil, fmt.Errorf("routine '%s' not found", filename)
	}

	return rr.sentries[filename], nil
}

// Has checks if a routine exists in the registry (valid or invalid)
func (rr *RoutineRegistry) Has(filename string) bool {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	_, inRoutines := rr.routines[filename]
	_, inErrors := rr.validationErrors[filename]
	return inRoutines || inErrors
}

// GetMetadata returns metadata for a routine (interface{} for interface compliance)
func (rr *RoutineRegistry) GetMetadata(filename string) interface{} {
	return rr.getMetadataTyped(filename)
}

// getMetadataTyped returns typed metadata for internal use
func (rr *RoutineRegistry) getMetadataTyped(filename string) *RoutineMetadata {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	if meta, ok := rr.metadata[filename]; ok {
		return meta
	}

	// Return basic metadata if not found
	return &RoutineMetadata{
		Filename:    filename,
		DisplayName: filename,
	}
}

// GetValidationError returns the validation error for a routine (if any)
func (rr *RoutineRegistry) GetValidationError(filename string) error {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	if err, ok := rr.validationErrors[filename]; ok {
		return err
	}

	return nil
}

// ListAvailable returns all discovered routine filenames (valid and invalid)
func (rr *RoutineRegistry) ListAvailable() []string {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	// Combine valid and invalid routine filenames
	names := make([]string, 0, len(rr.routines)+len(rr.validationErrors))

	for filename := range rr.routines {
		names = append(names, filename)
	}

	for filename := range rr.validationErrors {
		names = append(names, filename)
	}

	// Sort for consistent ordering
	sort.Strings(names)

	return names
}

// ListValid returns only valid routine filenames
func (rr *RoutineRegistry) ListValid() []string {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	names := make([]string, 0, len(rr.routines))
	for filename := range rr.routines {
		names = append(names, filename)
	}

	// Sort for consistent ordering
	sort.Strings(names)

	return names
}

// ListInvalid returns routine filenames that failed validation
func (rr *RoutineRegistry) ListInvalid() []string {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	names := make([]string, 0, len(rr.validationErrors))
	for filename := range rr.validationErrors {
		names = append(names, filename)
	}

	// Sort for consistent ordering
	sort.Strings(names)

	return names
}

// ListByTag returns routine filenames that have a specific tag
func (rr *RoutineRegistry) ListByTag(tag string) []string {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	names := make([]string, 0)
	for filename, meta := range rr.metadata {
		// Check if routine has this tag
		for _, t := range meta.Tags {
			if t == tag {
				names = append(names, filename)
				break
			}
		}
	}

	// Sort for consistent ordering
	sort.Strings(names)
	return names
}

// HasTag checks if a routine has a specific tag
func (rr *RoutineRegistry) HasTag(filename string, tag string) bool {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	meta, ok := rr.metadata[filename]
	if !ok {
		return false
	}

	for _, t := range meta.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// Reload clears and reloads all routines from disk
func (rr *RoutineRegistry) Reload() error {
	rr.mu.Lock()
	defer rr.mu.Unlock()

	// Clear existing data
	rr.routines = make(map[string]*ActionBuilder)
	rr.sentries = make(map[string][]Sentry)
	rr.metadata = make(map[string]*RoutineMetadata)
	rr.validationErrors = make(map[string]error)

	// Reload all routines
	log.Printf("[RoutineRegistry] Reloading routines from: %s", rr.routinesPath)
	rr.loadAllRoutines()

	validCount := len(rr.routines)
	invalidCount := len(rr.validationErrors)
	log.Printf("[RoutineRegistry] Reloaded %d valid routine(s), %d invalid routine(s)", validCount, invalidCount)

	return nil
}
