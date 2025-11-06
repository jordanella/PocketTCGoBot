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

// DatabaseActivityTab displays activity logs
type DatabaseActivityTab struct {
	controller *Controller
	db         *database.DB

	// Filters
	filterAccount *widget.Entry
	filterType    *widget.Select
	filterStatus  *widget.Select
	showCompleted *widget.Check

	// Content containers
	contentArea *fyne.Container
}

// NewDatabaseActivityTab creates a new database activity tab
func NewDatabaseActivityTab(ctrl *Controller, db *database.DB) *DatabaseActivityTab {
	return &DatabaseActivityTab{
		controller: ctrl,
		db:         db,
	}
}

// Build constructs the UI
func (t *DatabaseActivityTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Database - Activity Logs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Filters
	t.filterAccount = widget.NewEntry()
	t.filterAccount.SetPlaceHolder("Account ID")

	t.filterType = widget.NewSelect([]string{
		"All",
		"pack_opening",
		"wonder_pick",
		"mission_completion",
		"battle",
		"daily_login",
		"shop_purchase",
	}, func(string) {
		t.refresh()
	})
	t.filterType.SetSelected("All")

	t.filterStatus = widget.NewSelect([]string{
		"All",
		"running",
		"completed",
		"failed",
	}, func(string) {
		t.refresh()
	})
	t.filterStatus.SetSelected("All")

	t.showCompleted = widget.NewCheck("Show completed", func(bool) {
		t.refresh()
	})
	t.showCompleted.SetChecked(true)

	// Refresh button
	refreshBtn := widget.NewButton("Refresh", func() {
		t.refresh()
	})

	// Clear filters button
	clearBtn := widget.NewButton("Clear Filters", func() {
		t.filterAccount.SetText("")
		t.filterType.SetSelected("All")
		t.filterStatus.SetSelected("All")
		t.showCompleted.SetChecked(false)
		t.refresh()
	})

	// Toolbar
	toolbar := container.NewHBox(
		widget.NewLabel("Account ID:"),
		t.filterAccount,
		widget.NewLabel("Type:"),
		t.filterType,
		widget.NewLabel("Status:"),
		t.filterStatus,
		t.showCompleted,
		refreshBtn,
		clearBtn,
	)

	// Content area - use Stack container to allow content to fill space
	t.contentArea = container.NewStack()
	t.refresh()

	// Return border with content area directly (no scroll - tables handle their own scrolling)
	return container.NewBorder(
		container.NewVBox(header, toolbar),
		nil,
		nil,
		nil,
		t.contentArea,
	)
}

// refresh reloads the data
func (t *DatabaseActivityTab) refresh() {
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

	// Get activity logs based on filters
	activities, err := t.getFilteredActivities()
	if err != nil {
		if t.controller.window != nil {
			dialog.ShowError(err, t.controller.window)
		}
		return
	}

	if len(activities) == 0 {
		t.contentArea.Objects = []fyne.CanvasObject{
			widget.NewLabel("No activity logs found"),
		}
		t.contentArea.Refresh()
		return
	}

	// Build table view directly (not in VBox, to allow proper sizing)
	table := t.buildTableView(activities)

	// Replace content area with just the table
	t.contentArea.Objects = []fyne.CanvasObject{table}
	t.contentArea.Refresh()
}

// getFilteredActivities gets activities based on current filters
func (t *DatabaseActivityTab) getFilteredActivities() ([]*database.ActivityLog, error) {
	// For now, get all activities and filter client-side
	// In production, you'd want to add filtering to the database query

	// If only showing running activities (check if checkbox exists first)
	if t.showCompleted != nil && !t.showCompleted.Checked {
		return t.db.GetRunningActivities()
	}

	// Get all recent activities (last 1000)
	query := `
		SELECT id, account_id, activity_type, started_at, completed_at,
		       duration_seconds, status, error_message, bot_version, routine_name
		FROM activity_log
		ORDER BY started_at DESC
		LIMIT 1000
	`

	rows, err := t.db.Conn().Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*database.ActivityLog
	for rows.Next() {
		activity := &database.ActivityLog{}
		err := rows.Scan(
			&activity.ID,
			&activity.AccountID,
			&activity.ActivityType,
			&activity.StartedAt,
			&activity.CompletedAt,
			&activity.DurationSeconds,
			&activity.Status,
			&activity.ErrorMessage,
			&activity.BotVersion,
			&activity.RoutineName,
		)
		if err != nil {
			return nil, err
		}

		// Apply client-side filters
		if !t.matchesFilters(activity) {
			continue
		}

		activities = append(activities, activity)
	}

	return activities, nil
}

// matchesFilters checks if activity matches current filters
func (t *DatabaseActivityTab) matchesFilters(activity *database.ActivityLog) bool {
	// Account ID filter
	if t.filterAccount != nil && t.filterAccount.Text != "" {
		accountIDStr := fmt.Sprintf("%d", activity.AccountID)
		if accountIDStr != t.filterAccount.Text {
			return false
		}
	}

	// Type filter
	if t.filterType != nil && t.filterType.Selected != "All" {
		if activity.ActivityType != t.filterType.Selected {
			return false
		}
	}

	// Status filter
	if t.filterStatus != nil && t.filterStatus.Selected != "All" {
		status := t.getActivityStatus(activity)
		if status != t.filterStatus.Selected {
			return false
		}
	}

	return true
}

// getActivityStatus returns the status of an activity
func (t *DatabaseActivityTab) getActivityStatus(activity *database.ActivityLog) string {
	return activity.Status
}

// buildTableView creates a table of activity logs
func (t *DatabaseActivityTab) buildTableView(activities []*database.ActivityLog) fyne.CanvasObject {
	// Create table
	table := widget.NewTable(
		func() (int, int) {
			return len(activities) + 1, 7 // +1 for header, 7 columns
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Cell")
		},
		func(id widget.TableCellID, cell fyne.CanvasObject) {
			label := cell.(*widget.Label)

			// Header row
			if id.Row == 0 {
				headers := []string{"ID", "Account", "Type", "Started", "Duration", "Status", "Result"}
				label.SetText(headers[id.Col])
				label.TextStyle = fyne.TextStyle{Bold: true}
				return
			}

			// Data rows
			activity := activities[id.Row-1]
			switch id.Col {
			case 0:
				label.SetText(fmt.Sprintf("%d", activity.ID))
			case 1:
				label.SetText(fmt.Sprintf("%d", activity.AccountID))
			case 2:
				label.SetText(activity.ActivityType)
			case 3:
				label.SetText(activity.StartedAt.Format("01/02 15:04:05"))
			case 4:
				if activity.DurationSeconds != nil {
					duration := time.Duration(*activity.DurationSeconds) * time.Second
					label.SetText(formatDuration(duration))
				} else if activity.CompletedAt == nil {
					// Still running
					duration := time.Since(activity.StartedAt)
					label.SetText(formatDuration(duration) + " (running)")
				} else {
					label.SetText("N/A")
				}
			case 5:
				label.SetText(activity.Status)
			case 6:
				if activity.Status == "completed" {
					label.SetText("Success")
				} else if activity.Status == "failed" {
					label.SetText("Failed")
				} else {
					label.SetText("Running")
				}
			}
		},
	)

	// Set column widths
	table.SetColumnWidth(0, 50)  // ID
	table.SetColumnWidth(1, 80)  // Account
	table.SetColumnWidth(2, 150) // Type
	table.SetColumnWidth(3, 130) // Started
	table.SetColumnWidth(4, 120) // Duration
	table.SetColumnWidth(5, 80)  // Status
	table.SetColumnWidth(6, 80)  // Result

	// Handle row selection for details
	table.OnSelected = func(id widget.TableCellID) {
		if id.Row > 0 { // Skip header
			t.showActivityDetails(activities[id.Row-1])
		}
	}

	// Return table directly - it will fill the available space
	return table
}

// showActivityDetails shows a dialog with activity details
func (t *DatabaseActivityTab) showActivityDetails(activity *database.ActivityLog) {
	// Format completed time
	completedText := "Still running"
	if activity.CompletedAt != nil {
		completedText = activity.CompletedAt.Format("2006-01-02 15:04:05")
	}

	// Format duration
	durationText := "N/A"
	if activity.DurationSeconds != nil {
		duration := time.Duration(*activity.DurationSeconds) * time.Second
		durationText = formatDuration(duration)
	} else if activity.CompletedAt == nil {
		duration := time.Since(activity.StartedAt)
		durationText = formatDuration(duration) + " (still running)"
	}

	// Format bot version
	botVersionText := "(unknown)"
	if activity.BotVersion != nil && *activity.BotVersion != "" {
		botVersionText = *activity.BotVersion
	}

	// Format routine name
	routineText := "(none)"
	if activity.RoutineName != nil && *activity.RoutineName != "" {
		routineText = *activity.RoutineName
	}

	// Format error message
	errorText := "(none)"
	if activity.ErrorMessage != nil && *activity.ErrorMessage != "" {
		errorText = *activity.ErrorMessage
	}

	// Build detailed info
	details := fmt.Sprintf(`Activity ID: %d
Account ID: %d
Activity Type: %s
Routine: %s

Started: %s
Completed: %s
Duration: %s

Status: %s
Bot Version: %s

Error Message:
%s`,
		activity.ID,
		activity.AccountID,
		activity.ActivityType,
		routineText,
		activity.StartedAt.Format("2006-01-02 15:04:05"),
		completedText,
		durationText,
		activity.Status,
		botVersionText,
		errorText,
	)

	// Create dialog with scrollable content
	content := container.NewVScroll(widget.NewLabel(details))
	content.SetMinSize(fyne.NewSize(500, 400))

	dialog.ShowCustom(
		"Activity Details",
		"Close",
		content,
		t.controller.window,
	)
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}
