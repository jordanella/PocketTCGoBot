package gui

import (
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"jordanella.com/pocket-tcg-go/internal/database"
)

// DatabaseErrorsTab displays error logs
type DatabaseErrorsTab struct {
	controller *Controller
	db         *database.DB

	// Filters
	filterAccount  *widget.Entry
	filterType     *widget.Select
	filterSeverity *widget.Select
	showRecovered  *widget.Check

	// Content containers
	contentArea *fyne.Container
}

// NewDatabaseErrorsTab creates a new database errors tab
func NewDatabaseErrorsTab(ctrl *Controller, db *database.DB) *DatabaseErrorsTab {
	return &DatabaseErrorsTab{
		controller: ctrl,
		db:         db,
	}
}

// Build constructs the UI
func (t *DatabaseErrorsTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Database - Error Logs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Filters
	t.filterAccount = widget.NewEntry()
	t.filterAccount.SetPlaceHolder("Account ID")

	t.filterType = widget.NewSelect([]string{
		"All",
		"communication",
		"stuck",
		"no_response",
		"popup",
		"maintenance",
		"update_required",
		"banned",
		"title_screen",
		"timeout",
		"custom",
	}, func(string) {
		t.refresh()
	})
	t.filterType.SetSelected("All")

	t.filterSeverity = widget.NewSelect([]string{
		"All",
		"critical",
		"high",
		"medium",
		"low",
	}, func(string) {
		t.refresh()
	})
	t.filterSeverity.SetSelected("All")

	t.showRecovered = widget.NewCheck("Show recovered", func(bool) {
		t.refresh()
	})
	t.showRecovered.SetChecked(true)

	// Refresh button
	refreshBtn := widget.NewButton("Refresh", func() {
		t.refresh()
	})

	// Clear filters button
	clearBtn := widget.NewButton("Clear Filters", func() {
		t.filterAccount.SetText("")
		t.filterType.SetSelected("All")
		t.filterSeverity.SetSelected("All")
		t.showRecovered.SetChecked(true)
		t.refresh()
	})

	// Stats button
	statsBtn := widget.NewButton("Error Statistics", func() {
		t.showErrorStats()
	})

	// Toolbar
	toolbar := container.NewHBox(
		widget.NewLabel("Account ID:"),
		t.filterAccount,
		widget.NewLabel("Type:"),
		t.filterType,
		widget.NewLabel("Severity:"),
		t.filterSeverity,
		t.showRecovered,
		refreshBtn,
		clearBtn,
		statsBtn,
	)

	// Content area - use Stack container to allow content to fill space
	t.contentArea = container.NewStack()
	t.refresh()

	content := container.NewVScroll(t.contentArea)

	// Return border with content area directly (no scroll - tables handle their own scrolling)
	return container.NewBorder(
		container.NewVBox(header, toolbar),
		nil,
		nil,
		nil,
		content,
	)
}

// refresh reloads the data
func (t *DatabaseErrorsTab) refresh() {
	// Don't refresh if content area not initialized yet
	if t.contentArea == nil {
		return
	}

	if t.db == nil {
		t.contentArea.Objects = []fyne.CanvasObject{
			widget.NewLabel("Database not initialized"),
		}
		t.contentArea.Refresh()
		return
	}

	// Get error logs based on filters
	errors, err := t.getFilteredErrors()
	if err != nil {
		if t.controller.window != nil {
			dialog.ShowError(err, t.controller.window)
		}
		return
	}

	if len(errors) == 0 {
		t.contentArea.Objects = []fyne.CanvasObject{
			widget.NewLabel("No error logs found"),
		}
		t.contentArea.Refresh()
		return
	}

	// Build table view
	t.contentArea.Objects = []fyne.CanvasObject{
		t.buildTableView(errors),
	}

	t.contentArea.Refresh()
}

// getFilteredErrors gets errors based on current filters
func (t *DatabaseErrorsTab) getFilteredErrors() ([]*database.ErrorLog, error) {
	// For now, get recent errors and filter client-side
	// In production, you'd want to add filtering to the database query

	query := `
		SELECT id, account_id, activity_log_id, error_type, error_severity,
		       error_message, stack_trace, screen_state, template_name,
		       action_name, was_recovered, recovery_action, recovery_time_ms, occurred_at
		FROM error_log
		ORDER BY occurred_at DESC
		LIMIT 1000
	`

	rows, err := t.db.Conn().Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var errors []*database.ErrorLog
	for rows.Next() {
		errorLog := &database.ErrorLog{}
		err := rows.Scan(
			&errorLog.ID,
			&errorLog.AccountID,
			&errorLog.ActivityLogID,
			&errorLog.ErrorType,
			&errorLog.ErrorSeverity,
			&errorLog.ErrorMessage,
			&errorLog.StackTrace,
			&errorLog.ScreenState,
			&errorLog.TemplateName,
			&errorLog.ActionName,
			&errorLog.WasRecovered,
			&errorLog.RecoveryAction,
			&errorLog.RecoveryTimeMs,
			&errorLog.OccurredAt,
		)
		if err != nil {
			return nil, err
		}

		// Apply client-side filters
		if !t.matchesFilters(errorLog) {
			continue
		}

		errors = append(errors, errorLog)
	}

	return errors, nil
}

// matchesFilters checks if error matches current filters
func (t *DatabaseErrorsTab) matchesFilters(errorLog *database.ErrorLog) bool {
	// Account ID filter
	if t.filterAccount != nil && t.filterAccount.Text != "" {
		if errorLog.AccountID != nil {
			accountIDStr := fmt.Sprintf("%d", *errorLog.AccountID)
			if accountIDStr != t.filterAccount.Text {
				return false
			}
		} else {
			return false
		}
	}

	// Type filter
	if t.filterType != nil && t.filterType.Selected != "All" {
		if errorLog.ErrorType != t.filterType.Selected {
			return false
		}
	}

	// Severity filter
	if t.filterSeverity != nil && t.filterSeverity.Selected != "All" {
		if errorLog.ErrorSeverity != t.filterSeverity.Selected {
			return false
		}
	}

	// Recovered filter
	if t.showRecovered != nil && !t.showRecovered.Checked {
		if errorLog.WasRecovered {
			return false
		}
	}

	return true
}

// buildTableView creates a table of error logs
func (t *DatabaseErrorsTab) buildTableView(errors []*database.ErrorLog) fyne.CanvasObject {
	// Create table
	table := widget.NewTable(
		func() (int, int) {
			return len(errors) + 1, 7 // +1 for header, 7 columns
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Cell")
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)

			// Header row
			if id.Row == 0 {
				headers := []string{"ID", "Account", "Type", "Severity", "Detected", "Recovered", "Message"}
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}

			// Data rows
			errorLog := errors[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(fmt.Sprintf("%d", errorLog.ID))
			case 1:
				if errorLog.AccountID != nil {
					label.SetText(fmt.Sprintf("%d", *errorLog.AccountID))
				} else {
					label.SetText("N/A")
				}
			case 2:
				label.SetText(errorLog.ErrorType)
			case 3:
				label.SetText(errorLog.ErrorSeverity)
			case 4:
				label.SetText(errorLog.OccurredAt.Format("01/02 15:04:05"))
			case 5:
				if errorLog.WasRecovered {
					label.SetText("Yes")
				} else {
					label.SetText("No")
				}
			case 6:
				// Truncate message to first 50 chars
				msg := errorLog.ErrorMessage
				if len(msg) > 50 {
					msg = msg[:47] + "..."
				}
				label.SetText(msg)
			}
		},
	)

	// Set column widths
	table.SetColumnWidth(0, 50)  // ID
	table.SetColumnWidth(1, 80)  // Account
	table.SetColumnWidth(2, 120) // Type
	table.SetColumnWidth(3, 80)  // Severity
	table.SetColumnWidth(4, 130) // Detected
	table.SetColumnWidth(5, 80)  // Recovered
	table.SetColumnWidth(6, 250) // Message

	// Handle row selection for details
	table.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 { // Skip header
			t.showErrorDetails(errors[id.Row-1])
		}
	}

	// Return table directly - it will fill the available space
	return table
}

// showErrorDetails shows a dialog with error details
func (t *DatabaseErrorsTab) showErrorDetails(errorLog *database.ErrorLog) {
	// Format account ID
	accountText := "N/A"
	if errorLog.AccountID != nil {
		accountText = fmt.Sprintf("%d", *errorLog.AccountID)
	}

	// Format activity ID
	activityText := "N/A"
	if errorLog.ActivityLogID != nil {
		activityText = fmt.Sprintf("%d", *errorLog.ActivityLogID)
	}

	// Format recovery info
	recoveredText := "Not recovered"
	if errorLog.WasRecovered {
		recoveredText = "Yes"
	}

	recoveryActionText := "N/A"
	if errorLog.RecoveryAction != nil && *errorLog.RecoveryAction != "" {
		recoveryActionText = *errorLog.RecoveryAction
	}

	recoveryTimeText := "N/A"
	if errorLog.RecoveryTimeMs != nil {
		duration := time.Duration(*errorLog.RecoveryTimeMs) * time.Millisecond
		recoveryTimeText = formatDuration(duration)
	}

	// Format stack trace
	stackTraceText := "(none)"
	if errorLog.StackTrace != nil && *errorLog.StackTrace != "" {
		stackTraceText = *errorLog.StackTrace
	}

	// Format screen state
	screenStateText := "(none)"
	if errorLog.ScreenState != nil && *errorLog.ScreenState != "" {
		screenStateText = *errorLog.ScreenState
	}

	// Format template name
	templateText := "(none)"
	if errorLog.TemplateName != nil && *errorLog.TemplateName != "" {
		templateText = *errorLog.TemplateName
	}

	// Format action name
	actionText := "(none)"
	if errorLog.ActionName != nil && *errorLog.ActionName != "" {
		actionText = *errorLog.ActionName
	}

	// Build detailed info
	details := fmt.Sprintf(`Error ID: %d
Account ID: %s
Activity ID: %s

Error Type: %s
Severity: %s

Occurred: %s
Recovered: %s
Recovery Action: %s
Recovery Time: %s

Template: %s
Action: %s

Message:
%s

Stack Trace:
%s

Screen State:
%s`,
		errorLog.ID,
		accountText,
		activityText,
		errorLog.ErrorType,
		errorLog.ErrorSeverity,
		errorLog.OccurredAt.Format("2006-01-02 15:04:05"),
		recoveredText,
		recoveryActionText,
		recoveryTimeText,
		templateText,
		actionText,
		errorLog.ErrorMessage,
		stackTraceText,
		screenStateText,
	)

	// Create dialog with scrollable content
	content := container.NewVScroll(widget.NewLabel(details))
	content.SetMinSize(fyne.NewSize(500, 400))

	dialog.ShowCustom(
		"Error Details",
		"Close",
		content,
		t.controller.window,
	)
}

// showErrorStats shows error statistics dialog
func (t *DatabaseErrorsTab) showErrorStats() {
	if t.db == nil {
		return
	}

	// Get stats for all errors (no date filtering for now)
	stats, err := t.db.GetErrorStatsByType(nil, time.Time{}, time.Now())
	if err != nil {
		if t.controller.window != nil {
			dialog.ShowError(err, t.controller.window)
		}
		return
	}

	if len(stats) == 0 {
		dialog.ShowInformation("Error Statistics", "No errors in database", t.controller.window)
		return
	}

	// Build stats text
	statsText := "Error Statistics\n\n"
	statsText += fmt.Sprintf("%-20s %10s\n", "Type", "Count")
	statsText += "================================\n"

	// Sort and display
	for errorType, count := range stats {
		statsText += fmt.Sprintf("%-20s %10d\n", errorType, count)
	}

	// Create dialog with scrollable content
	content := container.NewVScroll(widget.NewLabel(statsText))
	content.SetMinSize(fyne.NewSize(400, 300))

	dialog.ShowCustom(
		"Error Statistics",
		"Close",
		content,
		t.controller.window,
	)
}
