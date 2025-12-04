package action

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bpfstack/bpfstack/internal/config"
)

// MockAction for testing - tracks lifecycle
type MockTestAction struct {
	*BaseAction
	initCalled  atomic.Bool
	startCalled atomic.Bool
	stopCalled  atomic.Bool
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

func NewMockTestAction(name string) *MockTestAction {
	return &MockTestAction{
		BaseAction: NewBaseAction(name),
	}
}

func (m *MockTestAction) Init() error {
	m.initCalled.Store(true)
	m.stopCh = make(chan struct{})
	return nil
}

func (m *MockTestAction) Start() error {
	m.startCalled.Store(true)
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		<-m.stopCh
	}()
	return nil
}

func (m *MockTestAction) Stop() error {
	m.stopCalled.Store(true)
	close(m.stopCh)
	m.wg.Wait()
	return nil
}

func TestManager_StartStopActions(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	// Initial config with cpu_metrics enabled
	initialConfig := `version: "1.0"
actions:
  - cpu_metrics: true
  - memory_metrics: false
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Create registry and register test actions
	registry := NewRegistry()
	registry.Register(NewMockTestAction("cpu_metrics"))
	registry.Register(NewMockTestAction("memory_metrics"))

	// Create manager
	manager := NewManager(registry)

	// Create watcher
	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Start manager with watcher
	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.StopAll()

	// Give time for actions to start
	time.Sleep(200 * time.Millisecond)

	// Check cpu_metrics is running
	if !manager.IsRunning("cpu_metrics") {
		t.Error("Expected cpu_metrics to be running")
	}

	// Check memory_metrics is not running
	if manager.IsRunning("memory_metrics") {
		t.Error("Expected memory_metrics to not be running")
	}
}

func TestManager_HotReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	// Initial config - nothing enabled
	initialConfig := `version: "1.0"
actions:
  - cpu_metrics: false
  - memory_metrics: false
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	registry := NewRegistry()
	registry.Register(NewMockTestAction("cpu_metrics"))
	registry.Register(NewMockTestAction("memory_metrics"))

	manager := NewManager(registry)

	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.StopAll()

	time.Sleep(200 * time.Millisecond)

	// Initially nothing should be running
	if manager.IsRunning("cpu_metrics") {
		t.Error("Expected cpu_metrics to not be running initially")
	}
	if manager.IsRunning("memory_metrics") {
		t.Error("Expected memory_metrics to not be running initially")
	}

	// Update config to enable both
	updatedConfig := `version: "2.0"
actions:
  - cpu_metrics: true
  - memory_metrics: true
`
	if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Wait for hot reload
	time.Sleep(500 * time.Millisecond)

	// Now both should be running
	if !manager.IsRunning("cpu_metrics") {
		t.Error("Expected cpu_metrics to be running after config update")
	}
	if !manager.IsRunning("memory_metrics") {
		t.Error("Expected memory_metrics to be running after config update")
	}

	// Update config to disable cpu_metrics
	finalConfig := `version: "3.0"
actions:
  - cpu_metrics: false
  - memory_metrics: true
`
	if err := os.WriteFile(configPath, []byte(finalConfig), 0644); err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Wait for hot reload
	time.Sleep(500 * time.Millisecond)

	// cpu_metrics should be stopped, memory_metrics still running
	if manager.IsRunning("cpu_metrics") {
		t.Error("Expected cpu_metrics to be stopped after final config update")
	}
	if !manager.IsRunning("memory_metrics") {
		t.Error("Expected memory_metrics to still be running after final config update")
	}
}

func TestManager_GetRunningActions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `version: "1.0"
actions:
  - cpu_metrics: true
  - memory_metrics: true
  - vmexit: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	registry := NewRegistry()
	registry.Register(NewMockTestAction("cpu_metrics"))
	registry.Register(NewMockTestAction("memory_metrics"))
	registry.Register(NewMockTestAction("vmexit"))

	manager := NewManager(registry)

	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.StopAll()

	time.Sleep(200 * time.Millisecond)

	running := manager.GetRunningActions()
	if len(running) != 2 {
		t.Errorf("Expected 2 running actions, got %d", len(running))
	}

	// Check that correct actions are running
	runningMap := make(map[string]bool)
	for _, name := range running {
		runningMap[name] = true
	}

	if !runningMap["cpu_metrics"] {
		t.Error("Expected cpu_metrics to be in running list")
	}
	if !runningMap["memory_metrics"] {
		t.Error("Expected memory_metrics to be in running list")
	}
	if runningMap["vmexit"] {
		t.Error("Expected vmexit to not be in running list")
	}
}

func TestManager_StopAll(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `version: "1.0"
actions:
  - cpu_metrics: true
  - memory_metrics: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	registry := NewRegistry()
	registry.Register(NewMockTestAction("cpu_metrics"))
	registry.Register(NewMockTestAction("memory_metrics"))

	manager := NewManager(registry)

	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	// Verify actions are running
	if !manager.IsRunning("cpu_metrics") || !manager.IsRunning("memory_metrics") {
		t.Error("Expected actions to be running before StopAll")
	}

	// Stop all
	manager.StopAll()

	// Verify all stopped
	if manager.IsRunning("cpu_metrics") {
		t.Error("Expected cpu_metrics to be stopped after StopAll")
	}
	if manager.IsRunning("memory_metrics") {
		t.Error("Expected memory_metrics to be stopped after StopAll")
	}
}
