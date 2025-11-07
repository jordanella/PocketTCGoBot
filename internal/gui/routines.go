package gui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/yaml.v3"
	"jordanella.com/pocket-tcg-go/internal/actions"
	"jordanella.com/pocket-tcg-go/pkg/templates"
)

const routinesFolder = "routines"

// RoutinesTab displays routine files and allows building/validating them
type RoutinesTab struct {
	controller *Controller

	// UI components
	routineSelect *widget.Select
	buildBtn      *widget.Button
	treeWidget    *widget.Tree
	statusLabel   *widget.Label
	contentArea   *fyne.Container

	// Data
	routineFiles   []string
	currentRoutine *RoutineTreeNode
	nodeMap        map[string]*RoutineTreeNode
}

// RoutineTreeNode represents a node in the routine tree
type RoutineTreeNode struct {
	ID       string
	Label    string
	Parent   string
	Children []string
	IsStep   bool
	HasIssue bool
	Issue    string
	StepType string
}

// NewRoutinesTab creates a new routines browser tab
func NewRoutinesTab(ctrl *Controller) *RoutinesTab {
	return &RoutinesTab{
		controller: ctrl,
	}
}

// safeLog safely logs a message, only if logTab is available
func (t *RoutinesTab) safeLog(level LogLevel, instance int, message string) {
	if t.controller.logTab != nil {
		t.controller.logTab.AddLog(level, instance, message)
	}
}

// Build constructs the UI
func (t *RoutinesTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Routine Browser", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Ensure routines folder exists
	t.ensureRoutinesFolder()

	// Routine selector dropdown
	t.routineSelect = widget.NewSelect([]string{}, func(selected string) {
		// Selection changed - clear the tree until Build is pressed
		t.clearTree()
	})
	t.refreshRoutineList()

	// Build button
	t.buildBtn = widget.NewButton("Build & Validate", func() {
		t.buildSelectedRoutine()
	})
	t.buildBtn.Importance = widget.HighImportance

	// Refresh button
	refreshBtn := widget.NewButton("Refresh List", func() {
		t.refreshRoutineList()
	})

	// Open folder button
	openFolderBtn := widget.NewButton("Open Routines Folder", func() {
		t.openRoutinesFolder()
	})

	// Toolbar
	toolbar := container.NewBorder(
		nil,
		nil,
		container.NewHBox(
			widget.NewLabel("Select Routine:"),
			t.routineSelect,
			t.buildBtn,
		),
		container.NewHBox(
			refreshBtn,
			openFolderBtn,
		),
		nil,
	)

	// Status label
	t.statusLabel = widget.NewLabel("")
	statusContainer := container.NewHBox(t.statusLabel)

	// Tree widget for displaying routine structure
	t.treeWidget = widget.NewTree(
		t.treeChildUIDs,
		t.treeIsBranch,
		func(branch bool) fyne.CanvasObject {
			return t.treeCreate()
		},
		t.treeUpdate,
	)

	// Content area
	t.contentArea = container.NewStack(
		container.NewCenter(
			widget.NewLabel("Select a routine and click 'Build & Validate' to view its structure"),
		),
	)

	// Scrollable tree view
	treeScroll := container.NewVScroll(t.contentArea)

	return container.NewBorder(
		container.NewVBox(header, toolbar, statusContainer), // Top
		nil,        // Bottom
		nil,        // Left
		nil,        // Right
		treeScroll, // Center
	)
}

// ensureRoutinesFolder creates the routines folder if it doesn't exist
func (t *RoutinesTab) ensureRoutinesFolder() {
	if _, err := os.Stat(routinesFolder); os.IsNotExist(err) {
		os.MkdirAll(routinesFolder, 0755)

		// Create a sample routine file
		samplePath := filepath.Join(routinesFolder, "example.yaml")
		sampleContent := `routine_name: "Example Routine"

steps:
  - action: Click
    x: 100
    y: 200

  - action: Delay
    count: 1

  - action: Click
    x: 300
    y: 400
`
		os.WriteFile(samplePath, []byte(sampleContent), 0644)

		// Log folder creation
		t.safeLog(LogLevelInfo, 0, fmt.Sprintf("Created routines folder: %s", routinesFolder))
	}
}

// refreshRoutineList scans the routines folder and updates the dropdown
func (t *RoutinesTab) refreshRoutineList() {
	// Guard against calling this before UI is built
	if t.routineSelect == nil {
		return
	}

	files, err := os.ReadDir(routinesFolder)
	if err != nil {
		t.safeLog(LogLevelError, 0, fmt.Sprintf("Failed to read routines folder: %v", err))
		t.routineFiles = []string{}
		t.routineSelect.Options = []string{}
		t.routineSelect.Refresh()
		return
	}

	// Filter for YAML files
	var yamlFiles []string
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".yaml") || strings.HasSuffix(file.Name(), ".yml")) {
			yamlFiles = append(yamlFiles, file.Name())
		}
	}

	t.routineFiles = yamlFiles
	t.routineSelect.Options = yamlFiles
	t.routineSelect.Refresh()

	// Only update status label if it exists
	if t.statusLabel != nil {
		if len(yamlFiles) > 0 {
			t.statusLabel.SetText(fmt.Sprintf("Found %d routine(s)", len(yamlFiles)))
		} else {
			t.statusLabel.SetText("No routines found. Add .yaml files to the routines folder.")
		}
	}
}

// buildSelectedRoutine builds and validates the selected routine
func (t *RoutinesTab) buildSelectedRoutine() {
	selectedFile := t.routineSelect.Selected
	if selectedFile == "" {
		t.statusLabel.SetText("Please select a routine first")
		return
	}

	routinePath := filepath.Join(routinesFolder, selectedFile)

	t.statusLabel.SetText(fmt.Sprintf("Building routine: %s...", selectedFile))
	t.safeLog(LogLevelInfo, 0, fmt.Sprintf("Building routine from: %s", routinePath))

	// Create template registry (needed for validation)
	templateRegistry := templates.NewTemplateRegistry("templates/images")

	// Try to load from templates/registry folder first (recommended structure)
	registryPath := filepath.Join("templates", "registry")
	if _, err := os.Stat(registryPath); err == nil {
		if err := templateRegistry.LoadFromDirectory(registryPath); err != nil {
			t.safeLog(LogLevelWarn, 0, fmt.Sprintf("Failed to load templates from registry folder: %v", err))
		} else {
			t.safeLog(LogLevelInfo, 0, "Loaded templates from templates/registry")
		}
	}

	// Also load any templates from the root templates directory
	if err := templateRegistry.LoadFromDirectory("templates"); err != nil {
		t.safeLog(LogLevelWarn, 0, fmt.Sprintf("Failed to load templates from templates folder: %v", err))
	}

	// Create routine loader
	loader := actions.NewRoutineLoader().WithTemplateRegistry(templateRegistry)

	// Build the routine
	_, err := loader.LoadFromFile(routinePath)

	if err != nil {
		t.statusLabel.SetText(fmt.Sprintf("❌ Build failed: %s", err.Error()))
		t.safeLog(LogLevelError, 0, fmt.Sprintf("Routine build failed: %v", err))
		t.showError(err.Error())
		return
	}

	// Build succeeded - now parse the YAML again to get the tree structure
	t.statusLabel.SetText("✓ Build successful")
	t.safeLog(LogLevelInfo, 0, "Routine built successfully")

	// Build tree structure
	t.buildTreeFromFile(routinePath, templateRegistry)
}

// buildTreeFromFile reads the YAML file and builds a tree structure
func (t *RoutinesTab) buildTreeFromFile(filePath string, registry *templates.TemplateRegistry) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.showError(fmt.Sprintf("Failed to read file: %v", err))
		return
	}

	// Parse the routine using yaml.Unmarshal
	var routine actions.Routine
	if err := yaml.Unmarshal(data, &routine); err != nil {
		t.showError(fmt.Sprintf("Failed to parse YAML: %v", err))
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
	if registry != nil {
		ab.WithTemplateRegistry(registry)
	}

	// Build tree nodes from steps
	nodeMap := make(map[string]*RoutineTreeNode)
	nodeMap["root"] = rootNode

	for i, step := range routine.Steps {
		stepID := fmt.Sprintf("step_%d", i)
		rootNode.Children = append(rootNode.Children, stepID)

		// Validate the step
		var validationErr error
		if step != nil {
			validationErr = step.Validate(ab)
		}

		stepNode := t.createStepNode(stepID, step, i+1, validationErr, nodeMap)
		nodeMap[stepID] = stepNode
	}

	t.currentRoutine = rootNode
	t.nodeMap = nodeMap
	t.displayTree()
}

// createStepNode creates a tree node for a step and its nested actions
func (t *RoutinesTab) createStepNode(id string, step actions.ActionStep, stepNum int, validationErr error, nodeMap map[string]*RoutineTreeNode) *RoutineTreeNode {
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
	switch s := step.(type) {
	case *actions.WhileImageFound:
		if len(s.Actions) > 0 {
			for i, nestedStep := range s.Actions {
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
	case *actions.UntilImageFound:
		if len(s.Actions) > 0 {
			for i, nestedStep := range s.Actions {
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
	case *actions.UntilAnyImagesFound:
		if len(s.Actions) > 0 {
			for i, nestedStep := range s.Actions {
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
	case *actions.WhileAnyImagesFound:
		if len(s.Actions) > 0 {
			for i, nestedStep := range s.Actions {
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
	case *actions.Repeat:
		if len(s.Actions) > 0 {
			for i, nestedStep := range s.Actions {
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
	}

	return node
}

// showError displays an error message in the content area
func (t *RoutinesTab) showError(errMsg string) {
	t.contentArea.Objects = []fyne.CanvasObject{
		container.NewCenter(
			container.NewVBox(
				widget.NewLabelWithStyle("❌ Error", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
				widget.NewLabel(errMsg),
			),
		),
	}
	t.contentArea.Refresh()
}

// clearTree clears the tree display
func (t *RoutinesTab) clearTree() {
	t.currentRoutine = nil
	t.contentArea.Objects = []fyne.CanvasObject{
		container.NewCenter(
			widget.NewLabel("Select a routine and click 'Build & Validate' to view its structure"),
		),
	}
	t.contentArea.Refresh()
	t.statusLabel.SetText("")
}

// displayTree shows the tree widget with the current routine structure
func (t *RoutinesTab) displayTree() {
	if t.currentRoutine == nil {
		t.clearTree()
		return
	}

	t.treeWidget.Refresh()
	t.contentArea.Objects = []fyne.CanvasObject{t.treeWidget}
	t.contentArea.Refresh()
}

// Tree widget callback implementations

func (t *RoutinesTab) treeChildUIDs(uid string) []string {
	if t.currentRoutine == nil {
		return []string{}
	}

	if uid == "" {
		// Root level
		return []string{"root"}
	}

	// Find the node and return its children
	node := t.findNode(uid)
	if node != nil {
		return node.Children
	}

	return []string{}
}

func (t *RoutinesTab) treeIsBranch(uid string) bool {
	if uid == "" || uid == "root" {
		return true
	}

	node := t.findNode(uid)
	if node != nil {
		return len(node.Children) > 0
	}

	return false
}

func (t *RoutinesTab) treeCreate() fyne.CanvasObject {
	icon := widget.NewIcon(theme.DocumentIcon())
	label := widget.NewLabel("Template")
	return container.NewHBox(icon, label)
}

func (t *RoutinesTab) treeUpdate(uid string, branch bool, obj fyne.CanvasObject) {
	node := t.findNode(uid)
	if node == nil {
		return
	}

	c := obj.(*fyne.Container)
	icon := c.Objects[0].(*widget.Icon)
	label := c.Objects[1].(*widget.Label)

	// Set icon based on node type
	if branch {
		icon.SetResource(theme.FolderIcon())
	} else {
		if node.HasIssue {
			icon.SetResource(theme.ErrorIcon())
		} else {
			icon.SetResource(theme.ConfirmIcon())
		}
	}

	// Set label text
	labelText := node.Label
	if node.HasIssue {
		labelText += " ❌"
	} else if node.IsStep {
		labelText += " ✓"
	}

	label.SetText(labelText)

	// Set tooltip if there's an issue
	if node.HasIssue && node.Issue != "" {
		label.Importance = widget.DangerImportance
		// Note: Fyne doesn't support tooltips directly on labels
		// We could add a button next to it for details
	}
}

func (t *RoutinesTab) findNode(uid string) *RoutineTreeNode {
	if t.nodeMap == nil {
		return nil
	}

	return t.nodeMap[uid]
}

// openRoutinesFolder opens the routines folder in the file explorer
func (t *RoutinesTab) openRoutinesFolder() {
	absPath, err := filepath.Abs(routinesFolder)
	if err != nil {
		t.safeLog(LogLevelError, 0, fmt.Sprintf("Failed to get absolute path: %v", err))
		return
	}

	// Platform-specific command to open folder
	switch {
	case strings.Contains(strings.ToLower(os.Getenv("OS")), "windows"):
		// Execute command (but don't wait for it)
		go func() {
			cmd := exec.Command("explorer", absPath)
			if err := cmd.Start(); err != nil {
				t.safeLog(LogLevelError, 0, fmt.Sprintf("Failed to open folder: %v", err))
			}
		}()
	default:
		t.safeLog(LogLevelWarn, 0, "Open folder not supported on this platform")
	}
}
