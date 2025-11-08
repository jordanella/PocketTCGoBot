package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/yaml.v3"
	"jordanella.com/pocket-tcg-go/internal/actions"
	"jordanella.com/pocket-tcg-go/internal/bot"
)

// RoutinesEnhancedTab displays routines as cards with metadata and filtering
type RoutinesEnhancedTab struct {
	controller *Controller

	// UI components
	searchEntry     *widget.Entry
	tagFilterChecks map[string]*widget.Check
	hideSentryCheck *widget.Check
	cardList        *fyne.Container
	detailsPanel    *fyne.Container
	treeWidget      *widget.Tree

	// Data
	manager         *bot.Manager
	allTags         []string
	selectedRoutine string
	currentRoutine  *RoutineTreeNode
	nodeMap         map[string]*RoutineTreeNode
}

// NewRoutinesEnhancedTab creates a new enhanced routines browser tab
func NewRoutinesEnhancedTab(ctrl *Controller, manager *bot.Manager) *RoutinesEnhancedTab {
	return &RoutinesEnhancedTab{
		controller:      ctrl,
		manager:         manager,
		tagFilterChecks: make(map[string]*widget.Check),
	}
}

// Build constructs the UI
func (t *RoutinesEnhancedTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Routine Library", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Search bar
	t.searchEntry = widget.NewEntry()
	t.searchEntry.SetPlaceHolder("Search routines...")
	t.searchEntry.OnChanged = func(_ string) {
		t.refreshCardList()
	}

	// Collect all unique tags
	t.collectAllTags()

	// Tag filter section
	tagFilterSection := t.createTagFilterSection()

	// Hide sentry checkbox
	t.hideSentryCheck = widget.NewCheck("Hide Sentry Routines", func(_ bool) {
		t.refreshCardList()
	})

	// Filters container
	filtersContainer := container.NewVBox(
		widget.NewLabelWithStyle("Filters", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		t.hideSentryCheck,
		widget.NewSeparator(),
		tagFilterSection,
	)

	// Card list (scrollable)
	t.cardList = container.NewVBox()
	t.refreshCardList()
	cardScroll := container.NewVScroll(t.cardList)
	cardScroll.SetMinSize(fyne.NewSize(400, 600))

	// Details panel (shows tree when routine is clicked)
	// Add a colored rectangle as background to make it visible
	detailsBg := widget.NewLabel("DETAILS PANEL - Click a routine card to see tree")
	detailsBg.Importance = widget.HighImportance

	t.detailsPanel = container.NewVBox(
		detailsBg,
	)

	// Don't wrap - just scroll directly
	detailsScroll := container.NewVScroll(t.detailsPanel)
	detailsScroll.SetMinSize(fyne.NewSize(500, 600))

	// Main content: filters | cards | details
	split := container.NewHSplit(cardScroll, detailsScroll)
	split.Offset = 0.5 // 50/50 split

	mainContent := container.NewBorder(
		nil, nil,
		container.NewVBox(filtersContainer),
		nil,
		split,
	)

	// Refresh button
	refreshBtn := widget.NewButton("Refresh Routines", func() {
		if t.manager != nil && t.manager.RoutineRegistry() != nil {
			t.manager.RoutineRegistry().Reload()
			t.collectAllTags()
			t.refreshCardList()
		}
	})

	// Top toolbar
	toolbar := container.NewHBox(
		t.searchEntry,
		refreshBtn,
	)

	return container.NewBorder(
		container.NewVBox(header, toolbar), // Top
		nil,                                // Bottom
		nil,                                // Left
		nil,                                // Right
		mainContent,                        // Center
	)
}

// collectAllTags collects all unique tags from all routines
func (t *RoutinesEnhancedTab) collectAllTags() {
	if t.manager == nil || t.manager.RoutineRegistry() == nil {
		t.allTags = []string{}
		return
	}

	registry := t.manager.RoutineRegistry()
	tagSet := make(map[string]bool)

	for _, filename := range registry.ListAvailable() {
		metaInterface := registry.GetMetadata(filename)
		meta, ok := metaInterface.(*actions.RoutineMetadata)
		if !ok {
			continue
		}

		for _, tag := range meta.Tags {
			tagSet[tag] = true
		}
	}

	// Convert to sorted slice
	t.allTags = make([]string, 0, len(tagSet))
	for tag := range tagSet {
		t.allTags = append(t.allTags, tag)
	}
}

// createTagFilterSection creates checkboxes for each tag
func (t *RoutinesEnhancedTab) createTagFilterSection() fyne.CanvasObject {
	if len(t.allTags) == 0 {
		return widget.NewLabel("No tags found")
	}

	tagChecks := container.NewVBox()
	tagChecks.Add(widget.NewLabel("Filter by tags:"))

	for _, tag := range t.allTags {
		check := widget.NewCheck(tag, func(_ bool) {
			t.refreshCardList()
		})
		t.tagFilterChecks[tag] = check
		tagChecks.Add(check)
	}

	return tagChecks
}

// refreshCardList updates the card list based on current filters
func (t *RoutinesEnhancedTab) refreshCardList() {
	if t.manager == nil || t.manager.RoutineRegistry() == nil {
		t.cardList.Objects = []fyne.CanvasObject{
			widget.NewLabel("No routine registry available"),
		}
		t.cardList.Refresh()
		return
	}

	registry := t.manager.RoutineRegistry()
	searchTerm := strings.ToLower(t.searchEntry.Text)

	// Get active tag filters
	activeTagFilters := make([]string, 0)
	for tag, check := range t.tagFilterChecks {
		if check.Checked {
			activeTagFilters = append(activeTagFilters, tag)
		}
	}

	// Clear card list
	t.cardList.Objects = []fyne.CanvasObject{}

	// Iterate through all routines
	for _, filename := range registry.ListAvailable() {
		metaInterface := registry.GetMetadata(filename)
		meta, ok := metaInterface.(*actions.RoutineMetadata)
		if !ok {
			// Debug: log type assertion failure
			if t.controller.logTab != nil {
				t.controller.logTab.AddLog(LogLevelWarn, 0, fmt.Sprintf("Failed to cast metadata for routine: %s", filename))
			}
			continue
		}

		// Debug: log what we got
		if t.controller.logTab != nil {
			t.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Routine: %s, Display: %s, Desc: %s, Tags: %v",
				filename, meta.DisplayName, meta.Description, meta.Tags))
		}

		// Apply filters
		if !t.shouldShowRoutine(meta, filename, searchTerm, activeTagFilters) {
			continue
		}

		// Check if valid
		validationErr := registry.GetValidationError(filename)
		isValid := validationErr == nil

		// Create card
		card := NewRoutineCard(filename, meta, isValid, validationErr, func() {
			t.showRoutineDetails(filename)
		})

		t.cardList.Add(card)
		t.cardList.Add(widget.NewSeparator())
	}

	if len(t.cardList.Objects) == 0 {
		t.cardList.Add(widget.NewLabel("No routines match the current filters"))
	}

	t.cardList.Refresh()
}

// shouldShowRoutine checks if routine passes all filters
func (t *RoutinesEnhancedTab) shouldShowRoutine(meta *actions.RoutineMetadata, filename string, searchTerm string, activeTagFilters []string) bool {
	// Hide sentry filter
	if t.hideSentryCheck != nil && t.hideSentryCheck.Checked {
		for _, tag := range meta.Tags {
			if strings.ToLower(tag) == "sentry" {
				return false
			}
		}
	}

	// Search filter
	if searchTerm != "" {
		nameMatch := strings.Contains(strings.ToLower(meta.DisplayName), searchTerm)
		descMatch := strings.Contains(strings.ToLower(meta.Description), searchTerm)
		fileMatch := strings.Contains(strings.ToLower(filename), searchTerm)

		if !nameMatch && !descMatch && !fileMatch {
			return false
		}
	}

	// Tag filter (show if routine has ANY of the active tags)
	if len(activeTagFilters) > 0 {
		hasMatchingTag := false
		for _, tag := range meta.Tags {
			for _, filterTag := range activeTagFilters {
				if tag == filterTag {
					hasMatchingTag = true
					break
				}
			}
			if hasMatchingTag {
				break
			}
		}

		if !hasMatchingTag {
			return false
		}
	}

	return true
}

// showRoutineDetails displays the tree structure of the selected routine
func (t *RoutinesEnhancedTab) showRoutineDetails(filename string) {
	t.selectedRoutine = filename

	// Debug log
	if t.controller.logTab != nil {
		t.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("showRoutineDetails called for: %s", filename))
	}

	// Build the routine tree
	if t.manager == nil || t.manager.RoutineRegistry() == nil {
		t.detailsPanel.Objects = []fyne.CanvasObject{
			widget.NewLabel("No manager available to load routine details"),
		}
		t.detailsPanel.Refresh()
		return
	}

	// Get metadata
	metaInterface := t.manager.RoutineRegistry().GetMetadata(filename)
	meta, ok := metaInterface.(*actions.RoutineMetadata)
	if !ok {
		t.detailsPanel.Objects = []fyne.CanvasObject{
			widget.NewLabel("Failed to load routine metadata"),
		}
		t.detailsPanel.Refresh()
		return
	}

	// Build the routine tree from file
	routinePath := filepath.Join("routines", filename+".yaml")
	if _, err := os.Stat(routinePath); os.IsNotExist(err) {
		routinePath = filepath.Join("routines", filename+".yml")
	}

	t.buildTreeFromFile(routinePath, meta)
}

// buildTreeFromFile reads the YAML file and builds a tree structure
func (t *RoutinesEnhancedTab) buildTreeFromFile(filePath string, meta *actions.RoutineMetadata) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.detailsPanel.Objects = []fyne.CanvasObject{
			widget.NewLabel(fmt.Sprintf("Failed to read file: %v", err)),
		}
		t.detailsPanel.Refresh()
		return
	}

	// Parse the routine using yaml.Unmarshal
	var routine actions.Routine
	if err := yaml.Unmarshal(data, &routine); err != nil {
		t.detailsPanel.Objects = []fyne.CanvasObject{
			widget.NewLabel(fmt.Sprintf("Failed to parse YAML: %v", err)),
		}
		t.detailsPanel.Refresh()
		return
	}

	// Build tree structure from the parsed routine
	rootNode := &RoutineTreeNode{
		ID:       "root",
		Label:    fmt.Sprintf("📋 %s (%s)", routine.RoutineName, filepath.Base(filePath)),
		Parent:   "",
		Children: []string{},
		IsStep:   false,
		HasIssue: false,
	}

	// Create temporary action builder for validation
	ab := actions.NewActionBuilder()
	if t.manager != nil && t.manager.TemplateRegistry() != nil {
		ab.WithTemplateRegistry(t.manager.TemplateRegistry())
	}

	// Build tree nodes from steps
	t.nodeMap = make(map[string]*RoutineTreeNode)
	t.nodeMap["root"] = rootNode

	for i, step := range routine.Steps {
		stepID := fmt.Sprintf("step_%d", i)
		rootNode.Children = append(rootNode.Children, stepID)

		// Validate the step
		var validationErr error
		if step != nil {
			validationErr = step.Validate(ab)
		}

		stepNode := t.createStepNode(stepID, step, i+1, validationErr, t.nodeMap)
		t.nodeMap[stepID] = stepNode
	}

	t.currentRoutine = rootNode

	// Debug log
	if t.controller.logTab != nil {
		t.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("Built tree with %d steps, calling displayTree", len(rootNode.Children)))
	}

	t.displayTree()
}

// createStepNode creates a tree node for a step and its nested actions
func (t *RoutinesEnhancedTab) createStepNode(id string, step actions.ActionStep, stepNum int, validationErr error, nodeMap map[string]*RoutineTreeNode) *RoutineTreeNode {
	stepType := fmt.Sprintf("%T", step)
	// Clean up the type name
	if idx := strings.LastIndex(stepType, "."); idx != -1 {
		stepType = stepType[idx+1:]
	}
	stepType = strings.TrimPrefix(stepType, "*")

	node := &RoutineTreeNode{
		ID:       id,
		Label:    fmt.Sprintf("Step %d: %s", stepNum, stepType),
		Parent:   "root",
		Children: []string{},
		IsStep:   true,
		HasIssue: validationErr != nil,
		StepType: stepType,
	}

	if validationErr != nil {
		node.Issue = validationErr.Error()
		node.Label += fmt.Sprintf(" (⚠️ %s)", validationErr.Error())
	}

	// Check for nested actions using type assertions
	t.addNestedActions(node, step, id, nodeMap)

	return node
}

// addNestedActions adds nested actions to a node based on step type
func (t *RoutinesEnhancedTab) addNestedActions(node *RoutineTreeNode, step actions.ActionStep, id string, nodeMap map[string]*RoutineTreeNode) {
	var nestedActions []actions.ActionStep

	switch s := step.(type) {
	case *actions.WhileImageFound:
		nestedActions = s.Actions
	case *actions.UntilImageFound:
		nestedActions = s.Actions
	case *actions.UntilAnyImagesFound:
		nestedActions = s.Actions
	case *actions.WhileAnyImagesFound:
		nestedActions = s.Actions
	case *actions.IfImageFound:
		nestedActions = s.Actions
	case *actions.IfImageNotFound:
		nestedActions = s.Actions
	case *actions.Repeat:
		nestedActions = s.Actions
	case *actions.RunRoutine:
		// Special handling for nested routines
		return
	}

	// Add nested action nodes
	for i, nestedStep := range nestedActions {
		nestedID := fmt.Sprintf("%s_nested_%d", id, i)
		node.Children = append(node.Children, nestedID)

		var nestedErr error
		if nestedStep != nil {
			nestedErr = nestedStep.Validate(actions.NewActionBuilder())
		}
		nestedNode := t.createStepNode(nestedID, nestedStep, i+1, nestedErr, nodeMap)
		nestedNode.Parent = id
		nodeMap[nestedID] = nestedNode
	}
}

// displayTree updates the details panel with the tree widget
func (t *RoutinesEnhancedTab) displayTree() {
	if t.currentRoutine == nil {
		if t.controller.logTab != nil {
			t.controller.logTab.AddLog(LogLevelWarn, 0, "displayTree: currentRoutine is nil")
		}
		return
	}

	// Debug log
	if t.controller.logTab != nil {
		t.controller.logTab.AddLog(LogLevelInfo, 0, fmt.Sprintf("displayTree: Creating tree widget with root: %s", t.currentRoutine.Label))
	}

	// Create tree widget
	t.treeWidget = widget.NewTree(
		t.treeChildUIDs,
		t.treeIsBranch,
		func(branch bool) fyne.CanvasObject {
			return t.treeCreate()
		},
		t.treeUpdate,
	)

	// Update details panel
	t.detailsPanel.Objects = []fyne.CanvasObject{
		t.treeWidget,
	}
	t.detailsPanel.Refresh()

	// Debug log
	if t.controller.logTab != nil {
		t.controller.logTab.AddLog(LogLevelInfo, 0, "displayTree: Details panel refreshed")
	}
}

// Tree widget callbacks (same as original RoutinesTab)

func (t *RoutinesEnhancedTab) treeChildUIDs(uid string) []string {
	if t.currentRoutine == nil {
		return []string{}
	}

	if uid == "" {
		// Root level - return the root node ID
		return []string{"root"}
	}

	if t.nodeMap == nil {
		return []string{}
	}

	node, ok := t.nodeMap[uid]
	if !ok {
		return []string{}
	}

	return node.Children
}

func (t *RoutinesEnhancedTab) treeIsBranch(uid string) bool {
	if t.nodeMap == nil {
		return false
	}

	node, ok := t.nodeMap[uid]
	if !ok {
		return false
	}

	return len(node.Children) > 0
}

func (t *RoutinesEnhancedTab) treeCreate() fyne.CanvasObject {
	return widget.NewLabel("")
}

func (t *RoutinesEnhancedTab) treeUpdate(uid string, branch bool, obj fyne.CanvasObject) {
	label, ok := obj.(*widget.Label)
	if !ok {
		return
	}

	if t.nodeMap == nil {
		label.SetText("No data")
		return
	}

	node, ok := t.nodeMap[uid]
	if !ok {
		label.SetText(fmt.Sprintf("Node not found: %s", uid))
		return
	}

	// Set the label text
	label.SetText(node.Label)

	// Color-code based on issues
	if node.HasIssue {
		label.Importance = widget.DangerImportance
	} else if node.IsStep {
		label.Importance = widget.MediumImportance
	} else {
		label.Importance = widget.HighImportance
	}
}
