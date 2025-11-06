package gui

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// LogLevel represents log severity
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Instance  int
	Message   string
}

// LogTab displays bot event logs
type LogTab struct {
	controller *Controller

	// Log storage
	logs   []LogEntry
	logsMu sync.RWMutex

	// Widgets
	logList       *widget.List
	clearBtn      *widget.Button
	filterSelect  *widget.Select
	autoScrollCheck *widget.Check
	maxLogs       int
}

// NewLogTab creates a new log tab
func NewLogTab(ctrl *Controller) *LogTab {
	tab := &LogTab{
		controller: ctrl,
		logs:       make([]LogEntry, 0, 1000),
		maxLogs:    1000,
	}

	// Add some sample logs for demonstration
	tab.AddLog(LogLevelInfo, 0, "Bot system initialized")
	tab.AddLog(LogLevelInfo, 1, "Instance 1 connected")
	tab.AddLog(LogLevelDebug, 1, "Screen detection enabled")

	return tab
}

// Build constructs the log viewer UI
func (l *LogTab) Build() fyne.CanvasObject {
	// Header
	header := widget.NewLabelWithStyle("Event Log", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	// Filter dropdown
	l.filterSelect = widget.NewSelect(
		[]string{"All", "DEBUG", "INFO", "WARN", "ERROR"},
		func(selected string) {
			if l.logList != nil {
				l.logList.Refresh()
			}
		},
	)
	l.filterSelect.PlaceHolder = "All"

	// Auto-scroll checkbox
	l.autoScrollCheck = widget.NewCheck("Auto-scroll", nil)
	l.autoScrollCheck.SetChecked(true)

	// Clear button
	l.clearBtn = widget.NewButton("Clear Logs", func() {
		l.ClearLogs()
	})

	// Controls
	controls := container.NewHBox(
		widget.NewLabel("Filter:"),
		l.filterSelect,
		l.autoScrollCheck,
		l.clearBtn,
	)

	// Log list
	l.logList = widget.NewList(
		func() int {
			return l.getFilteredLogCount()
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("timestamp"),
				widget.NewLabel("level"),
				widget.NewLabel("instance"),
				widget.NewLabel("message"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			entry := l.getFilteredLog(id)
			if entry == nil {
				return
			}

			box := item.(*fyne.Container)

			// Timestamp
			timestampLabel := box.Objects[0].(*widget.Label)
			timestampLabel.SetText(entry.Timestamp.Format("15:04:05"))

			// Level
			levelLabel := box.Objects[1].(*widget.Label)
			levelLabel.SetText(fmt.Sprintf("[%s]", entry.Level.String()))

			// Apply color based on level
			switch entry.Level {
			case LogLevelDebug:
				levelLabel.Importance = widget.LowImportance
			case LogLevelInfo:
				levelLabel.Importance = widget.MediumImportance
			case LogLevelWarn:
				levelLabel.Importance = widget.WarningImportance
			case LogLevelError:
				levelLabel.Importance = widget.DangerImportance
			}

			// Instance
			instanceLabel := box.Objects[2].(*widget.Label)
			if entry.Instance > 0 {
				instanceLabel.SetText(fmt.Sprintf("[I%d]", entry.Instance))
			} else {
				instanceLabel.SetText("[SYS]")
			}

			// Message
			messageLabel := box.Objects[3].(*widget.Label)
			messageLabel.SetText(entry.Message)
		},
	)

	// Layout
	content := container.NewBorder(
		container.NewVBox(header, controls),
		nil,
		nil,
		nil,
		l.logList,
	)

	return content
}

// AddLog adds a new log entry
func (l *LogTab) AddLog(level LogLevel, instance int, message string) {
	l.logsMu.Lock()

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Instance:  instance,
		Message:   message,
	}

	l.logs = append(l.logs, entry)

	// Trim if exceeds max
	if len(l.logs) > l.maxLogs {
		l.logs = l.logs[len(l.logs)-l.maxLogs:]
	}

	l.logsMu.Unlock()

	// Refresh list if created - use fyne.Do for thread safety
	if l.logList != nil {
		fyne.Do(func() {
			l.logList.Refresh()

			// Auto-scroll to bottom
			if l.autoScrollCheck != nil && l.autoScrollCheck.Checked {
				l.logList.ScrollToBottom()
			}
		})
	}
}

// ClearLogs removes all log entries
func (l *LogTab) ClearLogs() {
	l.logsMu.Lock()
	defer l.logsMu.Unlock()

	l.logs = make([]LogEntry, 0, 1000)

	if l.logList != nil {
		l.logList.Refresh()
	}
}

// getFilteredLogCount returns count of logs matching filter
func (l *LogTab) getFilteredLogCount() int {
	l.logsMu.RLock()
	defer l.logsMu.RUnlock()

	// Default to "All" if nothing selected
	selected := "All"
	if l.filterSelect != nil && l.filterSelect.Selected != "" {
		selected = l.filterSelect.Selected
	}

	if selected == "All" {
		return len(l.logs)
	}

	count := 0
	for _, entry := range l.logs {
		if entry.Level.String() == selected {
			count++
		}
	}
	return count
}

// getFilteredLog returns the Nth filtered log entry
func (l *LogTab) getFilteredLog(index int) *LogEntry {
	l.logsMu.RLock()
	defer l.logsMu.RUnlock()

	// Default to "All" if nothing selected
	selected := "All"
	if l.filterSelect != nil && l.filterSelect.Selected != "" {
		selected = l.filterSelect.Selected
	}

	if selected == "All" {
		if index >= 0 && index < len(l.logs) {
			return &l.logs[index]
		}
		return nil
	}

	// Filter logs
	currentIndex := 0
	for i := range l.logs {
		if l.logs[i].Level.String() == selected {
			if currentIndex == index {
				return &l.logs[i]
			}
			currentIndex++
		}
	}

	return nil
}

// GetLogs returns all logs (for export, etc.)
func (l *LogTab) GetLogs() []LogEntry {
	l.logsMu.RLock()
	defer l.logsMu.RUnlock()

	logs := make([]LogEntry, len(l.logs))
	copy(logs, l.logs)
	return logs
}
