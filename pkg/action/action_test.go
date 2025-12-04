package action

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/bpfstack/bpfstack/pkg/logger"
)

// setupTestLogger configures logger to capture output for testing
func setupTestLogger() *bytes.Buffer {
	buf := &bytes.Buffer{}
	logger.SetGlobalConfig(&logger.Config{
		Level:  logger.DEBUG,
		Format: logger.FormatText,
		Output: buf,
	})
	return buf
}

func TestBaseAction_Name(t *testing.T) {
	base := NewBaseAction("test_action")

	if base.Name() != "test_action" {
		t.Errorf("Expected name 'test_action', got '%s'", base.Name())
	}
}

func TestBaseAction_Logging(t *testing.T) {
	buf := setupTestLogger()
	base := NewBaseAction("test_action")

	base.LogInfo("info message")
	base.LogDebug("debug message")
	base.LogWarn("warn message")
	base.LogError("error message")

	output := buf.String()

	if !strings.Contains(output, "[test_action]") {
		t.Errorf("Expected action name in output, got: %s", output)
	}
	if !strings.Contains(output, "info message") {
		t.Error("Expected info message in output")
	}
	if !strings.Contains(output, "debug message") {
		t.Error("Expected debug message in output")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Expected warn message in output")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Expected error message in output")
	}
}

func TestBaseAction_LoggingWithFields(t *testing.T) {
	buf := setupTestLogger()
	base := NewBaseAction("test_action")

	base.LogInfo("collected metrics", logger.Fields{
		"cpu":    "45%",
		"memory": "2GB",
	})

	output := buf.String()

	if !strings.Contains(output, "cpu=45%") {
		t.Errorf("Expected cpu field in output, got: %s", output)
	}
	if !strings.Contains(output, "memory=2GB") {
		t.Errorf("Expected memory field in output, got: %s", output)
	}
}

func TestPrintAction_Lifecycle(t *testing.T) {
	buf := setupTestLogger()
	action := NewPrintAction("test_printer", "Test Message", 100*time.Millisecond)

	// Test Name()
	if action.Name() != "test_printer" {
		t.Errorf("Expected name 'test_printer', got '%s'", action.Name())
	}

	// Test Init()
	if err := action.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Test Start()
	if err := action.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Let it run for a bit
	time.Sleep(250 * time.Millisecond)

	// Test Stop()
	if err := action.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "initialized") {
		t.Error("Expected 'initialized' in logs")
	}
	if !strings.Contains(output, "starting") {
		t.Error("Expected 'starting' in logs")
	}
	if !strings.Contains(output, "stopped") {
		t.Error("Expected 'stopped' in logs")
	}
}

// CPU and Memory metrics tests moved to pkg/action/compute package

func TestVMExitAction_Lifecycle(t *testing.T) {
	buf := setupTestLogger()
	action := NewVMExitAction()

	if action.Name() != "vmexit" {
		t.Errorf("Expected name 'vmexit', got '%s'", action.Name())
	}

	if err := action.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if err := action.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := action.Stop(); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[vmexit]") {
		t.Errorf("Expected [vmexit] in logs, got: %s", output)
	}
}

func TestActionInterface_Implementation(t *testing.T) {
	// Verify example actions implement ActionInterface
	var _ ActionInterface = &PrintAction{}
	var _ ActionInterface = &VMExitAction{}
}

func TestRegisterDefaultActions(t *testing.T) {
	registry := NewRegistry()
	RegisterDefaultActions(registry)

	// Verify default actions are registered (vmexit)
	names := registry.Names()
	if len(names) != 1 {
		t.Errorf("Expected 1 default action, got %d", len(names))
	}

	expectedActions := []string{"vmexit"}
	for _, expected := range expectedActions {
		if _, exists := registry.Get(expected); !exists {
			t.Errorf("Expected action '%s' to be registered", expected)
		}
	}
}
