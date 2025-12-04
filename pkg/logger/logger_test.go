package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.level.String(); got != tt.expected {
			t.Errorf("Level(%d).String() = %s, want %s", tt.level, got, tt.expected)
		}
	}
}

func TestNew(t *testing.T) {
	log := New("test_action")

	if log.name != "test_action" {
		t.Errorf("Expected name 'test_action', got '%s'", log.name)
	}
}

func TestNewWithConfig(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{
		Level:  DEBUG,
		Format: FormatJSON,
		Output: buf,
	}

	log := NewWithConfig("test_action", cfg)

	if log.level != DEBUG {
		t.Error("Expected DEBUG level")
	}
	if log.format != FormatJSON {
		t.Error("Expected JSON format")
	}
}

func TestLogger_TextFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{
		Level:  INFO,
		Format: FormatText,
		Output: buf,
	}

	log := NewWithConfig("cpu_metrics", cfg)
	log.Info("test message")

	output := buf.String()

	// Check format: timestamp LEVEL [action] message
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected INFO in output, got: %s", output)
	}
	if !strings.Contains(output, "[cpu_metrics]") {
		t.Errorf("Expected [cpu_metrics] in output, got: %s", output)
	}
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected 'test message' in output, got: %s", output)
	}
}

func TestLogger_TextFormatWithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{
		Level:  INFO,
		Format: FormatText,
		Output: buf,
	}

	log := NewWithConfig("cpu_metrics", cfg)
	log.Info("collected metrics", Fields{
		"cpu_usage": "45.2%",
		"cores":     4,
	})

	output := buf.String()

	if !strings.Contains(output, "cpu_usage=45.2%") {
		t.Errorf("Expected cpu_usage field in output, got: %s", output)
	}
	if !strings.Contains(output, "cores=4") {
		t.Errorf("Expected cores field in output, got: %s", output)
	}
}

func TestLogger_JSONFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{
		Level:  INFO,
		Format: FormatJSON,
		Output: buf,
	}

	log := NewWithConfig("memory_metrics", cfg)
	log.Info("collected metrics", Fields{
		"used":  "2.1GB",
		"total": "8GB",
	})

	output := buf.String()

	var entry LogEntry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	if entry.Level != "INFO" {
		t.Errorf("Expected level 'INFO', got '%s'", entry.Level)
	}
	if entry.Action != "memory_metrics" {
		t.Errorf("Expected action 'memory_metrics', got '%s'", entry.Action)
	}
	if entry.Message != "collected metrics" {
		t.Errorf("Expected message 'collected metrics', got '%s'", entry.Message)
	}
	if entry.Fields["used"] != "2.1GB" {
		t.Errorf("Expected used '2.1GB', got '%v'", entry.Fields["used"])
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{
		Level:  WARN,
		Format: FormatText,
		Output: buf,
	}

	log := NewWithConfig("test", cfg)

	log.Debug("debug message")
	log.Info("info message")
	log.Warn("warn message")
	log.Error("error message")

	output := buf.String()

	if strings.Contains(output, "debug message") {
		t.Error("DEBUG message should be filtered")
	}
	if strings.Contains(output, "info message") {
		t.Error("INFO message should be filtered")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("WARN message should be logged")
	}
	if !strings.Contains(output, "error message") {
		t.Error("ERROR message should be logged")
	}
}

func TestLogger_AllLevels(t *testing.T) {
	tests := []struct {
		name  string
		logFn func(l *Logger)
		level string
	}{
		{"Debug", func(l *Logger) { l.Debug("debug") }, "DEBUG"},
		{"Info", func(l *Logger) { l.Info("info") }, "INFO"},
		{"Warn", func(l *Logger) { l.Warn("warn") }, "WARN"},
		{"Error", func(l *Logger) { l.Error("error") }, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			cfg := &Config{
				Level:  DEBUG,
				Format: FormatText,
				Output: buf,
			}

			log := NewWithConfig("test", cfg)
			tt.logFn(log)

			if !strings.Contains(buf.String(), tt.level) {
				t.Errorf("Expected %s in output", tt.level)
			}
		})
	}
}

func TestLogger_WithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{
		Level:  INFO,
		Format: FormatText,
		Output: buf,
	}

	log := NewWithConfig("test", cfg)
	entry := log.WithFields(Fields{"request_id": "123"})
	entry.Info("processing request", Fields{"user": "admin"})

	output := buf.String()

	if !strings.Contains(output, "request_id=123") {
		t.Errorf("Expected request_id field in output, got: %s", output)
	}
	if !strings.Contains(output, "user=admin") {
		t.Errorf("Expected user field in output, got: %s", output)
	}
}

func TestEntry_AllLevels(t *testing.T) {
	tests := []struct {
		name  string
		logFn func(e *Entry)
		level string
	}{
		{"Debug", func(e *Entry) { e.Debug("debug") }, "DEBUG"},
		{"Info", func(e *Entry) { e.Info("info") }, "INFO"},
		{"Warn", func(e *Entry) { e.Warn("warn") }, "WARN"},
		{"Error", func(e *Entry) { e.Error("error") }, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			cfg := &Config{
				Level:  DEBUG,
				Format: FormatText,
				Output: buf,
			}

			log := NewWithConfig("test", cfg)
			entry := log.WithFields(Fields{"key": "value"})
			tt.logFn(entry)

			output := buf.String()
			if !strings.Contains(output, tt.level) {
				t.Errorf("Expected %s in output", tt.level)
			}
			if !strings.Contains(output, "key=value") {
				t.Errorf("Expected key=value in output")
			}
		})
	}
}

func TestGlobalConfig(t *testing.T) {
	// Save original config
	original := GetGlobalConfig()
	defer SetGlobalConfig(original)

	newCfg := &Config{
		Level:  DEBUG,
		Format: FormatJSON,
		Output: &bytes.Buffer{},
	}

	SetGlobalConfig(newCfg)

	retrieved := GetGlobalConfig()
	if retrieved.Level != DEBUG {
		t.Error("Expected DEBUG level in global config")
	}
	if retrieved.Format != FormatJSON {
		t.Error("Expected JSON format in global config")
	}
}

func TestLogger_SetLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{
		Level:  ERROR,
		Format: FormatText,
		Output: buf,
	}

	log := NewWithConfig("test", cfg)

	log.Info("should not appear")
	if strings.Contains(buf.String(), "should not appear") {
		t.Error("INFO should be filtered at ERROR level")
	}

	log.SetLevel(INFO)
	log.Info("should appear")
	if !strings.Contains(buf.String(), "should appear") {
		t.Error("INFO should be logged after SetLevel(INFO)")
	}
}

func TestLogger_SetFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{
		Level:  INFO,
		Format: FormatText,
		Output: buf,
	}

	log := NewWithConfig("test", cfg)

	log.SetFormat(FormatJSON)
	log.Info("json message")

	var entry LogEntry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Errorf("Expected JSON format after SetFormat: %v", err)
	}
}

func TestLogger_SetOutput(t *testing.T) {
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	cfg := &Config{
		Level:  INFO,
		Format: FormatText,
		Output: buf1,
	}

	log := NewWithConfig("test", cfg)
	log.Info("to buf1")

	if !strings.Contains(buf1.String(), "to buf1") {
		t.Error("Expected output to buf1")
	}

	log.SetOutput(buf2)
	log.Info("to buf2")

	if !strings.Contains(buf2.String(), "to buf2") {
		t.Error("Expected output to buf2")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Level != INFO {
		t.Errorf("Expected default level INFO, got %v", cfg.Level)
	}
	if cfg.Format != FormatText {
		t.Errorf("Expected default format Text, got %v", cfg.Format)
	}
	if cfg.Output == nil {
		t.Error("Expected non-nil output")
	}
}

func TestMergeFields(t *testing.T) {
	result := mergeFields([]Fields{
		{"a": 1, "b": 2},
		{"c": 3, "b": 4}, // b should be overwritten
	})

	if result["a"] != 1 {
		t.Error("Expected a=1")
	}
	if result["b"] != 4 {
		t.Error("Expected b=4 (overwritten)")
	}
	if result["c"] != 3 {
		t.Error("Expected c=3")
	}
}

func TestMergeFields_Empty(t *testing.T) {
	result := mergeFields(nil)
	if result != nil {
		t.Error("Expected nil for empty input")
	}

	result = mergeFields([]Fields{})
	if result != nil {
		t.Error("Expected nil for empty slice")
	}
}
