package gui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"jordanella.com/pocket-tcg-go/internal/accountpool"
)

// UnifiedPoolWizard provides a multi-step wizard for creating unified pools
type UnifiedPoolWizard struct {
	window     fyne.Window
	poolName   string
	onComplete func(*accountpool.UnifiedPoolDefinition)

	// Pool configuration
	description   string
	queries       []QueryConfig
	includes      []string
	excludes      []string
	watchedPaths  []string
	sortMethod    string
	retryFailed   bool
	maxFailures   int
	refreshInterval int

	// UI state
	currentStep int
	wizard      *dialog.CustomDialog
}

// QueryConfig holds a single query configuration with structured filters
type QueryConfig struct {
	Name    string
	Filters []accountpool.QueryFilter
	Sort    []accountpool.SortOrder
	Limit   int
}

// NewUnifiedPoolWizard creates a new wizard for unified pool creation
func NewUnifiedPoolWizard(window fyne.Window, poolName string, onComplete func(*accountpool.UnifiedPoolDefinition)) *UnifiedPoolWizard {
	return &UnifiedPoolWizard{
		window:     window,
		poolName:   poolName,
		onComplete: onComplete,
		queries:    make([]QueryConfig, 0),
		includes:   make([]string, 0),
		excludes:   make([]string, 0),
		watchedPaths: make([]string, 0),
		sortMethod:  "packs_desc",
		retryFailed: true,
		maxFailures: 3,
		refreshInterval: 300,
		currentStep: 0,
	}
}

// Show displays the wizard
func (w *UnifiedPoolWizard) Show() {
	w.showStep1BasicInfo()
}

// Step 1: Basic Information
func (w *UnifiedPoolWizard) showStep1BasicInfo() {
	descEntry := widget.NewMultiLineEntry()
	descEntry.SetPlaceHolder("Enter a description for this pool...")
	descEntry.SetMinRowsVisible(3)
	descEntry.SetText(w.description)

	content := container.NewVBox(
		widget.NewLabel("Step 1 of 5: Basic Information"),
		widget.NewSeparator(),
		widget.NewLabel(fmt.Sprintf("Pool Name: %s", w.poolName)),
		widget.NewLabel("Description:"),
		descEntry,
		widget.NewLabel(""),
		widget.NewLabel("This pool will support:"),
		widget.NewLabel("• SQL queries for dynamic account selection"),
		widget.NewLabel("• Manual account inclusions"),
		widget.NewLabel("• Manual account exclusions"),
		widget.NewLabel("• Watched paths for automatic imports"),
	)

	w.wizard = dialog.NewCustom("Create Unified Pool", "Cancel", content, w.window)
	w.wizard.SetButtons([]fyne.CanvasObject{
		widget.NewButton("Cancel", func() {
			w.wizard.Hide()
		}),
		widget.NewButton("Next →", func() {
			w.description = descEntry.Text
			w.wizard.Hide()
			w.showStep2Queries()
		}),
	})
	w.wizard.Resize(fyne.NewSize(600, 400))
	w.wizard.Show()
}

// Step 2: SQL Queries
func (w *UnifiedPoolWizard) showStep2Queries() {
	var queryList *widget.List

	queryList = widget.NewList(
		func() int { return len(w.queries) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("Query Name"),
				widget.NewButton("Edit", nil),
				widget.NewButton("Remove", nil),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(w.queries) {
				return
			}
			query := w.queries[id]
			hbox := obj.(*fyne.Container)
			label := hbox.Objects[0].(*widget.Label)
			editBtn := hbox.Objects[1].(*widget.Button)
			removeBtn := hbox.Objects[2].(*widget.Button)

			label.SetText(query.Name)
			editBtn.OnTapped = func() {
				w.showQueryEditor(id)
			}
			removeBtn.OnTapped = func() {
				w.queries = append(w.queries[:id], w.queries[id+1:]...)
				queryList.Refresh()
			}
		},
	)

	content := container.NewVBox(
		widget.NewLabel("Step 2 of 5: SQL Queries (Optional)"),
		widget.NewSeparator(),
		widget.NewLabel("Add SQL queries to dynamically select accounts from the database."),
		widget.NewLabel("Query results will be combined."),
		widget.NewLabel(""),
		widget.NewButton("+ Add Query", func() {
			w.showQueryEditor(-1) // -1 means new query
		}),
		queryList,
	)

	w.wizard = dialog.NewCustom("Create Unified Pool", "Cancel", content, w.window)
	w.wizard.SetButtons([]fyne.CanvasObject{
		widget.NewButton("← Back", func() {
			w.wizard.Hide()
			w.showStep1BasicInfo()
		}),
		widget.NewButton("Skip", func() {
			w.wizard.Hide()
			w.showStep3Inclusions()
		}),
		widget.NewButton("Next →", func() {
			w.wizard.Hide()
			w.showStep3Inclusions()
		}),
	})
	w.wizard.Resize(fyne.NewSize(600, 450))
	w.wizard.Show()
}

// Query Editor Dialog - Visual query builder
func (w *UnifiedPoolWizard) showQueryEditor(index int) {
	var existingQuery *QueryConfig
	if index >= 0 && index < len(w.queries) {
		existingQuery = &w.queries[index]
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("e.g., high_pack_accounts")
	if existingQuery != nil {
		nameEntry.SetText(existingQuery.Name)
	}

	// Create a visual filter builder
	vqb := NewVisualQueryBuilder()
	if existingQuery != nil {
		// TODO: Load existing structured filters into vqb
		// For now, this is empty - queries created will use new structure
	}

	// Preview of generated SQL
	previewLabel := widget.NewLabel("SQL Preview:")
	previewText := widget.NewMultiLineEntry()
	previewText.Wrapping = fyne.TextWrapWord
	previewText.SetMinRowsVisible(6)
	previewText.Disable()

	updatePreview := func() {
		previewText.SetText(vqb.GenerateSQL())
	}
	vqb.SetOnChange(updatePreview)
	updatePreview()

	// Visual builder (structured filters)
	form := container.NewVBox(
		widget.NewLabel("Query Name:"),
		nameEntry,
		widget.NewSeparator(),
		widget.NewLabel("Filters (combined with AND):"),
		vqb.BuildUI(w.window),
		widget.NewSeparator(),
		previewLabel,
		previewText,
	)

	dlg := dialog.NewCustomConfirm("Add/Edit Query", "Save", "Cancel", form, func(save bool) {
		if !save {
			return
		}

		name := strings.TrimSpace(nameEntry.Text)
		if name == "" {
			dialog.ShowError(fmt.Errorf("query name is required"), w.window)
			return
		}

		// Get structured query from visual builder (always use visual builder now)
		filters := vqb.ExportFilters()
		sort := vqb.ExportSort()
		limit := vqb.ExportLimit()

		if len(filters) == 0 {
			dialog.ShowError(fmt.Errorf("at least one filter is required"), w.window)
			return
		}

		newQuery := QueryConfig{
			Name:    name,
			Filters: filters,
			Sort:    sort,
			Limit:   limit,
		}

		if index >= 0 {
			// Edit existing
			w.queries[index] = newQuery
		} else {
			// Add new
			w.queries = append(w.queries, newQuery)
		}

		// Re-show step 2 to refresh list
		w.showStep2Queries()
	}, w.window)

	dlg.Resize(fyne.NewSize(800, 650))
	dlg.Show()
}

// Step 3: Manual Inclusions
func (w *UnifiedPoolWizard) showStep3Inclusions() {
	includesEntry := widget.NewMultiLineEntry()
	includesEntry.SetPlaceHolder("Enter device accounts (one per line):\naccount1@example.com\naccount2@example.com")
	includesEntry.SetMinRowsVisible(8)
	includesEntry.SetText(strings.Join(w.includes, "\n"))

	content := container.NewVBox(
		widget.NewLabel("Step 3 of 5: Manual Inclusions (Optional)"),
		widget.NewSeparator(),
		widget.NewLabel("Add specific accounts to the pool by device_account."),
		widget.NewLabel("These accounts will be fetched from the database and added to the pool."),
		widget.NewLabel(""),
		includesEntry,
	)

	w.wizard = dialog.NewCustom("Create Unified Pool", "Cancel", content, w.window)
	w.wizard.SetButtons([]fyne.CanvasObject{
		widget.NewButton("← Back", func() {
			w.wizard.Hide()
			w.showStep2Queries()
		}),
		widget.NewButton("Skip", func() {
			w.wizard.Hide()
			w.showStep4Exclusions()
		}),
		widget.NewButton("Next →", func() {
			// Parse includes
			text := strings.TrimSpace(includesEntry.Text)
			if text != "" {
				lines := strings.Split(text, "\n")
				w.includes = make([]string, 0, len(lines))
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" {
						w.includes = append(w.includes, line)
					}
				}
			} else {
				w.includes = make([]string, 0)
			}
			w.wizard.Hide()
			w.showStep4Exclusions()
		}),
	})
	w.wizard.Resize(fyne.NewSize(600, 450))
	w.wizard.Show()
}

// Step 4: Manual Exclusions
func (w *UnifiedPoolWizard) showStep4Exclusions() {
	excludesEntry := widget.NewMultiLineEntry()
	excludesEntry.SetPlaceHolder("Enter device accounts to exclude (one per line):\nbanned@example.com\ntest@example.com")
	excludesEntry.SetMinRowsVisible(8)
	excludesEntry.SetText(strings.Join(w.excludes, "\n"))

	content := container.NewVBox(
		widget.NewLabel("Step 4 of 5: Manual Exclusions (Optional)"),
		widget.NewSeparator(),
		widget.NewLabel("Exclude specific accounts from the pool."),
		widget.NewLabel("Exclusions are applied LAST (after queries, includes, and watched paths)."),
		widget.NewLabel(""),
		excludesEntry,
	)

	w.wizard = dialog.NewCustom("Create Unified Pool", "Cancel", content, w.window)
	w.wizard.SetButtons([]fyne.CanvasObject{
		widget.NewButton("← Back", func() {
			w.wizard.Hide()
			w.showStep3Inclusions()
		}),
		widget.NewButton("Skip", func() {
			w.wizard.Hide()
			w.showStep5Configuration()
		}),
		widget.NewButton("Next →", func() {
			// Parse excludes
			text := strings.TrimSpace(excludesEntry.Text)
			if text != "" {
				lines := strings.Split(text, "\n")
				w.excludes = make([]string, 0, len(lines))
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" {
						w.excludes = append(w.excludes, line)
					}
				}
			} else {
				w.excludes = make([]string, 0)
			}
			w.wizard.Hide()
			w.showStep5Configuration()
		}),
	})
	w.wizard.Resize(fyne.NewSize(600, 450))
	w.wizard.Show()
}

// Step 5: Configuration
func (w *UnifiedPoolWizard) showStep5Configuration() {
	sortMethodSelect := widget.NewSelect([]string{"packs_desc", "packs_asc", "modified_desc", "modified_asc"}, nil)
	sortMethodSelect.SetSelected(w.sortMethod)

	retryCheck := widget.NewCheck("Retry failed accounts", nil)
	retryCheck.SetChecked(w.retryFailed)

	maxFailuresEntry := widget.NewEntry()
	maxFailuresEntry.SetPlaceHolder("3")
	maxFailuresEntry.SetText(fmt.Sprintf("%d", w.maxFailures))

	refreshEntry := widget.NewEntry()
	refreshEntry.SetPlaceHolder("300")
	refreshEntry.SetText(fmt.Sprintf("%d", w.refreshInterval))

	watchedPathsEntry := widget.NewMultiLineEntry()
	watchedPathsEntry.SetPlaceHolder("Enter watched paths (one per line):\nC:/accounts/premium\n./imported")
	watchedPathsEntry.SetMinRowsVisible(4)
	watchedPathsEntry.SetText(strings.Join(w.watchedPaths, "\n"))

	content := container.NewVBox(
		widget.NewLabel("Step 5 of 5: Configuration"),
		widget.NewSeparator(),
		widget.NewLabel("Sort Method:"),
		sortMethodSelect,
		retryCheck,
		widget.NewLabel("Max Failures:"),
		maxFailuresEntry,
		widget.NewLabel("Auto-Refresh Interval (seconds, 0 = disabled):"),
		refreshEntry,
		widget.NewLabel("Watched Paths (optional):"),
		watchedPathsEntry,
	)

	w.wizard = dialog.NewCustom("Create Unified Pool", "Cancel", content, w.window)
	w.wizard.SetButtons([]fyne.CanvasObject{
		widget.NewButton("← Back", func() {
			w.wizard.Hide()
			w.showStep4Exclusions()
		}),
		widget.NewButton("Create Pool", func() {
			// Parse watched paths
			pathsText := strings.TrimSpace(watchedPathsEntry.Text)
			if pathsText != "" {
				lines := strings.Split(pathsText, "\n")
				w.watchedPaths = make([]string, 0, len(lines))
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if line != "" {
						w.watchedPaths = append(w.watchedPaths, line)
					}
				}
			}

			w.sortMethod = sortMethodSelect.Selected
			w.retryFailed = retryCheck.Checked

			// Parse max failures
			fmt.Sscanf(maxFailuresEntry.Text, "%d", &w.maxFailures)
			if w.maxFailures <= 0 {
				w.maxFailures = 3
			}

			// Parse refresh interval
			fmt.Sscanf(refreshEntry.Text, "%d", &w.refreshInterval)
			if w.refreshInterval < 0 {
				w.refreshInterval = 0
			}

			// Create pool definition
			w.createPoolDefinition()
			w.wizard.Hide()
		}),
	})
	w.wizard.Resize(fyne.NewSize(600, 550))
	w.wizard.Show()
}

// createPoolDefinition builds the final pool definition and calls the completion callback
func (w *UnifiedPoolWizard) createPoolDefinition() {
	// Convert queries to structured format
	queries := make([]accountpool.QuerySource, len(w.queries))
	for i, q := range w.queries {
		queries[i] = accountpool.QuerySource{
			Name:    q.Name,
			Filters: q.Filters,
			Sort:    q.Sort,
			Limit:   q.Limit,
		}
	}

	// Build unified pool definition
	poolDef := &accountpool.UnifiedPoolDefinition{
		PoolName:     w.poolName,
		Description:  w.description,
		Queries:      queries,
		Include:      w.includes,
		Exclude:      w.excludes,
		WatchedPaths: w.watchedPaths,
		Config: accountpool.UnifiedPoolConfig{
			SortMethod:      w.sortMethod,
			RetryFailed:     w.retryFailed,
			MaxFailures:     w.maxFailures,
			RefreshInterval: w.refreshInterval,
		},
	}

	// Call completion callback
	if w.onComplete != nil {
		w.onComplete(poolDef)
	}
}

// VisualQueryBuilder provides a simple visual query builder for the wizard
type VisualQueryBuilder struct {
	filters []* QueryFilter
	sortBy string
	sortDir string
	limit int
	onChange func()
	filtersContainer *fyne.Container
}

// QueryFilter represents a single filter condition
type QueryFilter struct {
	field string
	operator string
	value string
	container *fyne.Container
}

// NewVisualQueryBuilder creates a new visual query builder
func NewVisualQueryBuilder() *VisualQueryBuilder {
	return &VisualQueryBuilder{
		filters: make([]*QueryFilter, 0),
		sortBy: "packs_opened",
		sortDir: "DESC",
		limit: 100,
	}
}

// SetOnChange sets the callback when query changes
func (vqb *VisualQueryBuilder) SetOnChange(fn func()) {
	vqb.onChange = fn
}

// BuildUI creates the UI for the visual query builder
func (vqb *VisualQueryBuilder) BuildUI(window fyne.Window) *fyne.Container {
	// Filters section
	vqb.filtersContainer = container.NewVBox()
	addFilterBtn := widget.NewButton("+ Add Filter", func() {
		vqb.addFilter()
	})

	// Sort section
	sortFieldSelect := widget.NewSelect(
		[]string{"packs_opened", "shinedust", "last_used_at", "created_at"},
		func(s string) {
			vqb.sortBy = s
			if vqb.onChange != nil {
				vqb.onChange()
			}
		},
	)
	sortFieldSelect.SetSelected(vqb.sortBy)

	sortDirSelect := widget.NewSelect(
		[]string{"DESC", "ASC"},
		func(s string) {
			vqb.sortDir = s
			if vqb.onChange != nil {
				vqb.onChange()
			}
		},
	)
	sortDirSelect.SetSelected(vqb.sortDir)

	limitEntry := widget.NewEntry()
	limitEntry.SetText(strconv.Itoa(vqb.limit))
	limitEntry.OnChanged = func(s string) {
		if val, err := strconv.Atoi(s); err == nil {
			vqb.limit = val
			if vqb.onChange != nil {
				vqb.onChange()
			}
		}
	}

	return container.NewVBox(
		widget.NewLabel("Filters:"),
		vqb.filtersContainer,
		addFilterBtn,
		widget.NewSeparator(),
		widget.NewLabel("Sort By:"),
		container.NewHBox(sortFieldSelect, sortDirSelect),
		widget.NewLabel("Limit:"),
		limitEntry,
	)
}

// addFilter adds a new filter to the builder
func (vqb *VisualQueryBuilder) addFilter() {
	filter := &QueryFilter{}

	fieldSelect := widget.NewSelect(
		[]string{"device_account", "packs_opened", "shinedust", "last_used_at", "is_active"},
		func(s string) {
			filter.field = s
			if vqb.onChange != nil {
				vqb.onChange()
			}
		},
	)
	fieldSelect.SetSelected("packs_opened")
	filter.field = "packs_opened"

	operatorSelect := widget.NewSelect(
		[]string{"=", "!=", "<", ">", "<=", ">="},
		func(s string) {
			filter.operator = s
			if vqb.onChange != nil {
				vqb.onChange()
			}
		},
	)
	operatorSelect.SetSelected(">=")
	filter.operator = ">="

	valueEntry := widget.NewEntry()
	valueEntry.SetPlaceHolder("value")
	valueEntry.OnChanged = func(s string) {
		filter.value = s
		if vqb.onChange != nil {
			vqb.onChange()
		}
	}

	removeBtn := widget.NewButton("Remove", func() {
		vqb.removeFilter(filter)
	})

	filter.container = container.NewHBox(fieldSelect, operatorSelect, valueEntry, removeBtn)
	vqb.filters = append(vqb.filters, filter)
	vqb.filtersContainer.Add(filter.container)

	if vqb.onChange != nil {
		vqb.onChange()
	}
}

// removeFilter removes a filter from the builder
func (vqb *VisualQueryBuilder) removeFilter(filter *QueryFilter) {
	for i, f := range vqb.filters {
		if f == filter {
			vqb.filters = append(vqb.filters[:i], vqb.filters[i+1:]...)
			break
		}
	}
	vqb.filtersContainer.Remove(filter.container)
	if vqb.onChange != nil {
		vqb.onChange()
	}
}

// GenerateSQL generates the SQL query from current settings
func (vqb *VisualQueryBuilder) GenerateSQL() string {
	var sb strings.Builder

	sb.WriteString("SELECT device_account, device_password, shinedust, packs_opened, last_used_at\n")
	sb.WriteString("FROM accounts\n")

	// WHERE clause
	if len(vqb.filters) > 0 {
		sb.WriteString("WHERE ")
		first := true
		for _, filter := range vqb.filters {
			if filter.field == "" || filter.value == "" {
				continue
			}
			if !first {
				sb.WriteString("\n  AND ")
			}
			sb.WriteString(filter.field)
			sb.WriteString(" ")
			sb.WriteString(filter.operator)
			sb.WriteString(" ")

			// Quote string values if needed
			if filter.field == "device_account" {
				sb.WriteString("'")
				sb.WriteString(filter.value)
				sb.WriteString("'")
			} else {
				sb.WriteString(filter.value)
			}
			first = false
		}
		sb.WriteString("\n")
	}

	// ORDER BY
	sb.WriteString("ORDER BY ")
	sb.WriteString(vqb.sortBy)
	sb.WriteString(" ")
	sb.WriteString(vqb.sortDir)
	sb.WriteString("\n")

	// LIMIT
	sb.WriteString("LIMIT ")
	sb.WriteString(strconv.Itoa(vqb.limit))

	return sb.String()
}

// ParseSQL parses an existing SQL query (simplified - just sets defaults)
func (vqb *VisualQueryBuilder) ParseSQL(sql string) {
	// For simplicity, just parse ORDER BY and LIMIT
	// TODO: Could implement full SQL parsing, but for now just set defaults
	sql = strings.ToUpper(sql)

	// Parse ORDER BY
	if strings.Contains(sql, "ORDER BY PACKS_OPENED DESC") {
		vqb.sortBy = "packs_opened"
		vqb.sortDir = "DESC"
	} else if strings.Contains(sql, "ORDER BY PACKS_OPENED ASC") {
		vqb.sortBy = "packs_opened"
		vqb.sortDir = "ASC"
	} else if strings.Contains(sql, "ORDER BY SHINEDUST DESC") {
		vqb.sortBy = "shinedust"
		vqb.sortDir = "DESC"
	}

	// Parse LIMIT
	if idx := strings.Index(sql, "LIMIT "); idx != -1 {
		limitStr := sql[idx+6:]
		if val, err := strconv.Atoi(strings.TrimSpace(limitStr)); err == nil {
			vqb.limit = val
		}
	}
}

// ExportFilters exports the current filters as structured QueryFilter slice
func (vqb *VisualQueryBuilder) ExportFilters() []accountpool.QueryFilter {
	filters := make([]accountpool.QueryFilter, 0, len(vqb.filters))
	for _, f := range vqb.filters {
		if f.field != "" && f.value != "" {
			filters = append(filters, accountpool.QueryFilter{
				Column:     f.field,
				Comparator: f.operator,
				Value:      f.value,
			})
		}
	}
	return filters
}

// ExportSort exports the current sort configuration as structured SortOrder slice
func (vqb *VisualQueryBuilder) ExportSort() []accountpool.SortOrder {
	if vqb.sortBy == "" {
		return nil
	}
	return []accountpool.SortOrder{
		{
			Column:    vqb.sortBy,
			Direction: strings.ToLower(vqb.sortDir),
		},
	}
}

// ExportLimit returns the configured limit
func (vqb *VisualQueryBuilder) ExportLimit() int {
	return vqb.limit
}
