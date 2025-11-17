package logging

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelFatal LogLevel = "FATAL"
)

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     LogLevel               `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
	Error     error                  `json:"error,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// Logger provides structured logging functionality
type Logger struct {
	component string
	minLevel  LogLevel
	outputs   []io.Writer
	mu        sync.Mutex
	formatter LogFormatter
}

// LogFormatter formats log entries for output
type LogFormatter interface {
	Format(entry *LogEntry) string
}

// TextFormatter formats logs as human-readable text
type TextFormatter struct{}

func (f *TextFormatter) Format(entry *LogEntry) string {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05.000")
	msg := fmt.Sprintf("[%s] %s [%s] %s", timestamp, entry.Level, entry.Component, entry.Message)

	if entry.Error != nil {
		msg += fmt.Sprintf(" | error=%v", entry.Error)
	}

	if len(entry.Context) > 0 {
		msg += " |"
		for k, v := range entry.Context {
			msg += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	return msg + "\n"
}

// NewLogger creates a new logger for a specific component
func NewLogger(component string) *Logger {
	return &Logger{
		component: component,
		minLevel:  LogLevelInfo,
		outputs:   []io.Writer{os.Stdout},
		formatter: &TextFormatter{},
	}
}

// SetMinLevel sets the minimum log level to output
func (l *Logger) SetMinLevel(level LogLevel) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.minLevel = level
	return l
}

// AddOutput adds an output writer for logs
func (l *Logger) AddOutput(w io.Writer) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.outputs = append(l.outputs, w)
	return l
}

// SetFormatter sets the log formatter
func (l *Logger) SetFormatter(formatter LogFormatter) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.formatter = formatter
	return l
}

// log writes a log entry
func (l *Logger) log(level LogLevel, message string, err error, context map[string]interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if this level should be logged
	if !l.shouldLog(level) {
		return
	}

	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Component: l.component,
		Message:   message,
		Error:     err,
		Context:   context,
	}

	formatted := l.formatter.Format(entry)

	for _, output := range l.outputs {
		output.Write([]byte(formatted))
	}
}

// shouldLog checks if a log level should be output
func (l *Logger) shouldLog(level LogLevel) bool {
	levels := map[LogLevel]int{
		LogLevelDebug: 0,
		LogLevelInfo:  1,
		LogLevelWarn:  2,
		LogLevelError: 3,
		LogLevelFatal: 4,
	}

	return levels[level] >= levels[l.minLevel]
}

// Debug logs a debug message
func (l *Logger) Debug(message string) {
	l.log(LogLevelDebug, message, nil, nil)
}

// DebugWithContext logs a debug message with context
func (l *Logger) DebugWithContext(message string, context map[string]interface{}) {
	l.log(LogLevelDebug, message, nil, context)
}

// Info logs an info message
func (l *Logger) Info(message string) {
	l.log(LogLevelInfo, message, nil, nil)
}

// InfoWithContext logs an info message with context
func (l *Logger) InfoWithContext(message string, context map[string]interface{}) {
	l.log(LogLevelInfo, message, nil, context)
}

// Warn logs a warning message
func (l *Logger) Warn(message string) {
	l.log(LogLevelWarn, message, nil, nil)
}

// WarnWithContext logs a warning message with context
func (l *Logger) WarnWithContext(message string, context map[string]interface{}) {
	l.log(LogLevelWarn, message, nil, context)
}

// Error logs an error message
func (l *Logger) Error(message string, err error) {
	l.log(LogLevelError, message, err, nil)
}

// ErrorWithContext logs an error message with context
func (l *Logger) ErrorWithContext(message string, err error, context map[string]interface{}) {
	l.log(LogLevelError, message, err, context)
}

// Fatal logs a fatal error message
func (l *Logger) Fatal(message string, err error) {
	l.log(LogLevelFatal, message, err, nil)
}

// FatalWithContext logs a fatal error message with context
func (l *Logger) FatalWithContext(message string, err error, context map[string]interface{}) {
	l.log(LogLevelFatal, message, err, context)
}

// WithContext returns a log function that includes context
func (l *Logger) WithContext(context map[string]interface{}) *ContextLogger {
	return &ContextLogger{
		logger:  l,
		context: context,
	}
}

// ContextLogger is a logger with pre-set context
type ContextLogger struct {
	logger  *Logger
	context map[string]interface{}
}

// Debug logs a debug message with pre-set context
func (cl *ContextLogger) Debug(message string) {
	cl.logger.log(LogLevelDebug, message, nil, cl.context)
}

// Info logs an info message with pre-set context
func (cl *ContextLogger) Info(message string) {
	cl.logger.log(LogLevelInfo, message, nil, cl.context)
}

// Warn logs a warning message with pre-set context
func (cl *ContextLogger) Warn(message string) {
	cl.logger.log(LogLevelWarn, message, nil, cl.context)
}

// Error logs an error message with pre-set context
func (cl *ContextLogger) Error(message string, err error) {
	cl.logger.log(LogLevelError, message, err, cl.context)
}

// Fatal logs a fatal error message with pre-set context
func (cl *ContextLogger) Fatal(message string, err error) {
	cl.logger.log(LogLevelFatal, message, err, cl.context)
}
