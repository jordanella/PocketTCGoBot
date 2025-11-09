package gui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"jordanella.com/pocket-tcg-go/internal/accountpool"
)

// QueryBuilder provides a visual interface for building SQL queries
type QueryBuilder struct {
	window fyne.Window
	onSave func(*accountpool.QueryDefinition)

	// Pool info
	poolName        string
	poolDescription string

	// Filters
	filters []*FilterRow

	// Sort
	sortFields []*SortRow

	// Limit
	limitValue int

	// Pool config
	retryFailed      bool
	maxFailures      int
	waitForAccounts  bool
	maxWaitTime      time.Duration
	bufferSize       int
	refreshInterval  time.Duration

	// UI Components
	content           *fyne.Container
	filtersContainer  *fyne.Container
	sortContainer     *fyne.Container
	previewText       *widget.Entry
}

// FilterRow represents a single filter condition
type FilterRow struct {
	fieldSelect    *widget.Select
	operatorSelect *widget.Select
	valueEntry     *widget.Entry
	removeBtn      *widget.Button
	container      *fyne.Container
}

// SortRow represents a single sort field
type SortRow struct {
	fieldSelect     *widget.Select
	directionSelect *widget.Select
	removeBtn       *widget.Button
	container       *fyne.Container
}

// Available fields for filtering and sorting
var availableFields = []string{
	"status",
	"pack_count",
	"failure_count",
	"completed_at",
	"last_modified",
	"last_error",
}

// Available operators
var availableOperators = map[string][]string{
	"status":         {"=", "!=", "IN"},
	"pack_count":     {"=", "!=", "<", ">", "<=", ">=", "BETWEEN"},
	"failure_count":  {"=", "!=", "<", ">", "<=", ">=", "BETWEEN"},
	"completed_at":   {"IS NULL", "IS NOT NULL", "<", ">"},
	"last_modified":  {"<", ">", "<=", ">="},
	"last_error":     {"LIKE", "NOT LIKE", "IS NULL", "IS NOT NULL"},
}

// NewQueryBuilder creates a new visual query builder
func NewQueryBuilder(window fyne.Window, poolName string, onSave func(*accountpool.QueryDefinition)) *QueryBuilder {
	qb := &QueryBuilder{
		window:          window,
		poolName:        poolName,
		onSave:          onSave,
		filters:         make([]*FilterRow, 0),
		sortFields:      make([]*SortRow, 0),
		limitValue:      50,
		retryFailed:     false,
		maxFailures:     3,
		waitForAccounts: true,
		maxWaitTime:     5 * time.Minute,
		bufferSize:      50,
		refreshInterval: 30 * time.Second,
	}

	qb.buildUI()
	return qb
}

// buildUI constructs the query builder UI
func (qb *QueryBuilder) buildUI() {
	// Pool name and description
	nameEntry := widget.NewEntry()
	nameEntry.SetText(qb.poolName)
	nameEntry.OnChanged = func(s string) {
		qb.poolName = s
		qb.updatePreview()
	}

	descEntry := widget.NewEntry()
	descEntry.SetPlaceHolder("Enter pool description")
	descEntry.OnChanged = func(s string) {
		qb.poolDescription = s
	}

	poolInfoCard := widget.NewCard("Pool Information", "", container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Name", nameEntry),
			widget.NewFormItem("Description", descEntry),
		),
	))

	// Filters section
	qb.filtersContainer = container.NewVBox()
	addFilterBtn := widget.NewButton("+ Add Filter", qb.addFilter)

	filtersCard := widget.NewCard("Filters", "Define which accounts to include", container.NewVBox(
		qb.filtersContainer,
		addFilterBtn,
	))

	// Sort section
	qb.sortContainer = container.NewVBox()
	addSortBtn := widget.NewButton("+ Add Sort Field", qb.addSortField)

	sortCard := widget.NewCard("Sorting", "Define the order of results", container.NewVBox(
		qb.sortContainer,
		addSortBtn,
	))

	// Limit section
	limitEntry := widget.NewEntry()
	limitEntry.SetText(strconv.Itoa(qb.limitValue))
	limitEntry.OnChanged = func(s string) {
		if val, err := strconv.Atoi(s); err == nil {
			qb.limitValue = val
			qb.updatePreview()
		}
	}

	limitCard := widget.NewCard("Limit", "Maximum number of accounts to return", limitEntry)

	// Pool config section
	poolConfigCard := qb.buildPoolConfigSection()

	// Query preview
	qb.previewText = widget.NewMultiLineEntry()
	qb.previewText.Disable()
	qb.previewText.SetMinRowsVisible(10)

	previewCard := widget.NewCard("Query Preview", "Generated SQL query", qb.previewText)

	// Left column: Filters, Sort, Limit
	leftColumn := container.NewVBox(
		filtersCard,
		sortCard,
		limitCard,
	)

	// Right column: Pool config and preview
	rightColumn := container.NewVBox(
		poolConfigCard,
		previewCard,
	)

	// Main layout
	mainContent := container.NewHSplit(
		container.NewVScroll(leftColumn),
		container.NewVScroll(rightColumn),
	)
	mainContent.Offset = 0.5

	// Action buttons
	saveBtn := widget.NewButton("Save Pool", qb.savePool)
	cancelBtn := widget.NewButton("Cancel", func() {
		// Close dialog
	})

	actions := container.NewHBox(
		layout.NewSpacer(),
		cancelBtn,
		saveBtn,
	)

	qb.content = container.NewBorder(
		poolInfoCard,
		actions,
		nil,
		nil,
		mainContent,
	)

	// Add initial filter
	qb.addFilter()
	qb.updatePreview()
}

// buildPoolConfigSection creates the pool configuration section
func (qb *QueryBuilder) buildPoolConfigSection() *widget.Card {
	retryCheck := widget.NewCheck("Retry Failed Accounts", func(checked bool) {
		qb.retryFailed = checked
	})
	retryCheck.SetChecked(qb.retryFailed)

	maxFailuresEntry := widget.NewEntry()
	maxFailuresEntry.SetText(strconv.Itoa(qb.maxFailures))
	maxFailuresEntry.OnChanged = func(s string) {
		if val, err := strconv.Atoi(s); err == nil {
			qb.maxFailures = val
		}
	}

	waitCheck := widget.NewCheck("Wait for Accounts", func(checked bool) {
		qb.waitForAccounts = checked
	})
	waitCheck.SetChecked(qb.waitForAccounts)

	bufferSizeEntry := widget.NewEntry()
	bufferSizeEntry.SetText(strconv.Itoa(qb.bufferSize))
	bufferSizeEntry.OnChanged = func(s string) {
		if val, err := strconv.Atoi(s); err == nil {
			qb.bufferSize = val
		}
	}

	refreshSelect := widget.NewSelect(
		[]string{"No auto-refresh", "30 seconds", "1 minute", "2 minutes", "5 minutes"},
		func(s string) {
			switch s {
			case "No auto-refresh":
				qb.refreshInterval = 0
			case "30 seconds":
				qb.refreshInterval = 30 * time.Second
			case "1 minute":
				qb.refreshInterval = 1 * time.Minute
			case "2 minutes":
				qb.refreshInterval = 2 * time.Minute
			case "5 minutes":
				qb.refreshInterval = 5 * time.Minute
			}
		},
	)
	refreshSelect.SetSelected("30 seconds")

	form := widget.NewForm(
		widget.NewFormItem("", retryCheck),
		widget.NewFormItem("Max Failures", maxFailuresEntry),
		widget.NewFormItem("", waitCheck),
		widget.NewFormItem("Buffer Size", bufferSizeEntry),
		widget.NewFormItem("Auto Refresh", refreshSelect),
	)

	return widget.NewCard("Pool Configuration", "Behavior settings", form)
}

// addFilter adds a new filter row
func (qb *QueryBuilder) addFilter() {
	row := &FilterRow{}

	row.fieldSelect = widget.NewSelect(availableFields, func(field string) {
		// Update operators when field changes
		operators := availableOperators[field]
		row.operatorSelect.Options = operators
		if len(operators) > 0 {
			row.operatorSelect.SetSelected(operators[0])
		}
		qb.updatePreview()
	})

	row.operatorSelect = widget.NewSelect([]string{}, func(s string) {
		qb.updatePreview()
	})

	row.valueEntry = widget.NewEntry()
	row.valueEntry.SetPlaceHolder("value")
	row.valueEntry.OnChanged = func(s string) {
		qb.updatePreview()
	}

	row.removeBtn = widget.NewButton("Remove", func() {
		qb.removeFilter(row)
	})

	row.container = container.NewHBox(
		row.fieldSelect,
		row.operatorSelect,
		row.valueEntry,
		row.removeBtn,
	)

	qb.filters = append(qb.filters, row)
	qb.filtersContainer.Add(row.container)

	// Set default field
	if len(availableFields) > 0 {
		row.fieldSelect.SetSelected(availableFields[0])
	}
}

// removeFilter removes a filter row
func (qb *QueryBuilder) removeFilter(row *FilterRow) {
	// Find and remove from filters slice
	for i, f := range qb.filters {
		if f == row {
			qb.filters = append(qb.filters[:i], qb.filters[i+1:]...)
			break
		}
	}

	qb.filtersContainer.Remove(row.container)
	qb.updatePreview()
}

// addSortField adds a new sort field row
func (qb *QueryBuilder) addSortField() {
	row := &SortRow{}

	row.fieldSelect = widget.NewSelect(availableFields, func(s string) {
		qb.updatePreview()
	})

	row.directionSelect = widget.NewSelect([]string{"ASC", "DESC"}, func(s string) {
		qb.updatePreview()
	})
	row.directionSelect.SetSelected("DESC")

	row.removeBtn = widget.NewButton("Remove", func() {
		qb.removeSortField(row)
	})

	row.container = container.NewHBox(
		row.fieldSelect,
		row.directionSelect,
		row.removeBtn,
	)

	qb.sortFields = append(qb.sortFields, row)
	qb.sortContainer.Add(row.container)

	// Set default field
	if len(availableFields) > 0 {
		row.fieldSelect.SetSelected(availableFields[0])
	}
}

// removeSortField removes a sort field row
func (qb *QueryBuilder) removeSortField(row *SortRow) {
	for i, s := range qb.sortFields {
		if s == row {
			qb.sortFields = append(qb.sortFields[:i], qb.sortFields[i+1:]...)
			break
		}
	}

	qb.sortContainer.Remove(row.container)
	qb.updatePreview()
}

// updatePreview generates and displays the SQL query preview
func (qb *QueryBuilder) updatePreview() {
	query := qb.generateQuery()
	qb.previewText.SetText(query)
}

// generateQuery builds the SQL query from current settings
func (qb *QueryBuilder) generateQuery() string {
	var sb strings.Builder

	sb.WriteString("SELECT\n")
	sb.WriteString("  id,\n")
	sb.WriteString("  xml_path,\n")
	sb.WriteString("  pack_count,\n")
	sb.WriteString("  last_modified,\n")
	sb.WriteString("  status,\n")
	sb.WriteString("  failure_count,\n")
	sb.WriteString("  last_error\n")
	sb.WriteString("FROM accounts\n")

	// WHERE clause
	if len(qb.filters) > 0 {
		sb.WriteString("WHERE ")

		validFilters := 0
		for _, filter := range qb.filters {
			if filter.fieldSelect.Selected == "" {
				continue
			}

			if validFilters > 0 {
				sb.WriteString("\n  AND ")
			}

			field := filter.fieldSelect.Selected
			operator := filter.operatorSelect.Selected
			value := filter.valueEntry.Text

			sb.WriteString(field)
			sb.WriteString(" ")
			sb.WriteString(operator)

			// Add value based on operator
			switch operator {
			case "IS NULL", "IS NOT NULL":
				// No value needed
			case "IN":
				sb.WriteString(" (?)")  // Parameterized
			case "BETWEEN":
				sb.WriteString(" ? AND ?")  // Parameterized
			default:
				sb.WriteString(" ?")  // Parameterized
			}

			validFilters++
		}
		sb.WriteString("\n")
	}

	// ORDER BY clause
	if len(qb.sortFields) > 0 {
		sb.WriteString("ORDER BY ")

		validSorts := 0
		for _, sort := range qb.sortFields {
			if sort.fieldSelect.Selected == "" {
				continue
			}

			if validSorts > 0 {
				sb.WriteString(", ")
			}

			sb.WriteString(sort.fieldSelect.Selected)
			sb.WriteString(" ")
			sb.WriteString(sort.directionSelect.Selected)

			validSorts++
		}
		sb.WriteString("\n")
	}

	// LIMIT clause
	sb.WriteString(fmt.Sprintf("LIMIT %d", qb.limitValue))

	return sb.String()
}

// savePool saves the query definition
func (qb *QueryBuilder) savePool() {
	if qb.poolName == "" {
		dialog.ShowError(fmt.Errorf("pool name is required"), qb.window)
		return
	}

	// Build parameters
	parameters := make([]accountpool.Parameter, 0)

	for _, filter := range qb.filters {
		if filter.fieldSelect.Selected == "" {
			continue
		}

		operator := filter.operatorSelect.Selected
		if operator == "IS NULL" || operator == "IS NOT NULL" {
			continue
		}

		value := filter.valueEntry.Text
		if value == "" {
			continue
		}

		paramName := filter.fieldSelect.Selected
		paramType := "string"

		// Determine type based on field
		switch filter.fieldSelect.Selected {
		case "pack_count", "failure_count":
			paramType = "int"
		}

		// Parse value based on type
		var paramValue interface{}
		if paramType == "int" {
			if val, err := strconv.Atoi(value); err == nil {
				paramValue = val
			} else {
				continue
			}
		} else {
			paramValue = value
		}

		parameters = append(parameters, accountpool.Parameter{
			Name:  paramName,
			Value: paramValue,
			Type:  paramType,
		})
	}

	// Build query definition
	queryDef := &accountpool.QueryDefinition{
		Name:        qb.poolName,
		Description: qb.poolDescription,
		Type:        "sql",
		Version:     "1.0",
		Query: accountpool.QueryConfig{
			Select:     qb.generateQuery(),
			Parameters: parameters,
		},
		PoolConfig: accountpool.PoolConfig{
			RetryFailed:     qb.retryFailed,
			MaxFailures:     qb.maxFailures,
			WaitForAccounts: qb.waitForAccounts,
			MaxWaitTime:     qb.maxWaitTime,
			BufferSize:      qb.bufferSize,
			RefreshInterval: qb.refreshInterval,
		},
	}

	// Call save callback
	if qb.onSave != nil {
		qb.onSave(queryDef)
	}
}

// Show displays the query builder dialog
func (qb *QueryBuilder) Show() {
	dlg := dialog.NewCustom("Visual Query Builder", "Close", qb.content, qb.window)
	dlg.Resize(fyne.NewSize(1000, 700))
	dlg.Show()
}
