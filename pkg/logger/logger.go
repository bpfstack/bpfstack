package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// Level represents the severity level of a log message
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Format represents the output format of logs
type Format int

const (
	FormatText Format = iota
	FormatJSON
)

// Fields represents key-value pairs for structured logging
type Fields map[string]interface{}

// Logger provides structured logging for actions
type Logger struct {
	name   string
	level  Level
	format Format
	output io.Writer
	mu     sync.Mutex
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Action    string                 `json:"action"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// Config holds logger configuration
type Config struct {
	Level  Level
	Format Format
	Output io.Writer
}

// DefaultConfig returns default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:  INFO,
		Format: FormatText,
		Output: os.Stdout,
	}
}

// globalConfig holds the global logger configuration
var globalConfig = DefaultConfig()
var globalMu sync.RWMutex

// SetGlobalConfig sets the global logger configuration
func SetGlobalConfig(cfg *Config) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalConfig = cfg
}

// GetGlobalConfig returns the current global logger configuration
func GetGlobalConfig() *Config {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalConfig
}

// New creates a new logger for an action
func New(name string) *Logger {
	cfg := GetGlobalConfig()
	return &Logger{
		name:   name,
		level:  cfg.Level,
		format: cfg.Format,
		output: cfg.Output,
	}
}

// NewWithConfig creates a new logger with specific configuration
func NewWithConfig(name string, cfg *Config) *Logger {
	return &Logger{
		name:   name,
		level:  cfg.Level,
		format: cfg.Format,
		output: cfg.Output,
	}
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetFormat sets the output format
func (l *Logger) SetFormat(format Format) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.format = format
}

// SetOutput sets the output writer
func (l *Logger) SetOutput(output io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = output
}

// log writes a log entry
func (l *Logger) log(level Level, msg string, fields Fields) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Level:     level.String(),
		Action:    l.name,
		Message:   msg,
		Fields:    fields,
	}

	var output string
	if l.format == FormatJSON {
		output = l.formatJSON(entry)
	} else {
		output = l.formatText(entry)
	}

	fmt.Fprintln(l.output, output)
}

// formatText formats the log entry as human-readable text
func (l *Logger) formatText(entry LogEntry) string {
	// Format: 2024-12-02T19:30:00.000+09:00 INFO  [action_name] message key=value
	timestamp := entry.Timestamp[:23] // Trim to milliseconds
	levelPadded := fmt.Sprintf("%-5s", entry.Level)

	base := fmt.Sprintf("%s %s [%s] %s", timestamp, levelPadded, entry.Action, entry.Message)

	if len(entry.Fields) > 0 {
		for k, v := range entry.Fields {
			base += fmt.Sprintf(" %s=%v", k, v)
		}
	}

	return base
}

// formatJSON formats the log entry as JSON
func (l *Logger) formatJSON(entry LogEntry) string {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to marshal log entry: %v"}`, err)
	}
	return string(data)
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...Fields) {
	l.log(DEBUG, msg, mergeFields(fields))
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...Fields) {
	l.log(INFO, msg, mergeFields(fields))
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...Fields) {
	l.log(WARN, msg, mergeFields(fields))
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...Fields) {
	l.log(ERROR, msg, mergeFields(fields))
}

// WithFields returns a new logger entry with pre-set fields
func (l *Logger) WithFields(fields Fields) *Entry {
	return &Entry{
		logger: l,
		fields: fields,
	}
}

// mergeFields merges multiple Fields into one
func mergeFields(fields []Fields) Fields {
	if len(fields) == 0 {
		return nil
	}
	result := make(Fields)
	for _, f := range fields {
		for k, v := range f {
			result[k] = v
		}
	}
	return result
}

// Entry represents a log entry with pre-set fields
type Entry struct {
	logger *Logger
	fields Fields
}

// Debug logs a debug message with pre-set fields
func (e *Entry) Debug(msg string, fields ...Fields) {
	merged := mergeFields(append([]Fields{e.fields}, fields...))
	e.logger.log(DEBUG, msg, merged)
}

// Info logs an info message with pre-set fields
func (e *Entry) Info(msg string, fields ...Fields) {
	merged := mergeFields(append([]Fields{e.fields}, fields...))
	e.logger.log(INFO, msg, merged)
}

// Warn logs a warning message with pre-set fields
func (e *Entry) Warn(msg string, fields ...Fields) {
	merged := mergeFields(append([]Fields{e.fields}, fields...))
	e.logger.log(WARN, msg, merged)
}

// Error logs an error message with pre-set fields
func (e *Entry) Error(msg string, fields ...Fields) {
	merged := mergeFields(append([]Fields{e.fields}, fields...))
	e.logger.log(ERROR, msg, merged)
}
