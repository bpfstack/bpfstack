package test

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bpfstack/bpfstack/internal/config"
	"github.com/bpfstack/bpfstack/pkg/action"
	"github.com/bpfstack/bpfstack/pkg/action/compute"
)

// MockAction is a test action that tracks its lifecycle
type MockAction struct {
	name        string
	initCount   atomic.Int32
	startCount  atomic.Int32
	stopCount   atomic.Int32
	isRunning   atomic.Bool
	mu          sync.Mutex
	stopCh      chan struct{}
	onStartFunc func()
	onStopFunc  func()
}

func NewMockAction(name string) *MockAction {
	return &MockAction{
		name: name,
	}
}

func (m *MockAction) Name() string {
	return m.name
}

func (m *MockAction) Init() error {
	m.initCount.Add(1)
	m.mu.Lock()
	m.stopCh = make(chan struct{})
	m.mu.Unlock()
	return nil
}

func (m *MockAction) Start() error {
	m.startCount.Add(1)
	m.isRunning.Store(true)
	if m.onStartFunc != nil {
		m.onStartFunc()
	}
	return nil
}

func (m *MockAction) Stop() error {
	m.stopCount.Add(1)
	m.isRunning.Store(false)
	m.mu.Lock()
	if m.stopCh != nil {
		close(m.stopCh)
	}
	m.mu.Unlock()
	if m.onStopFunc != nil {
		m.onStopFunc()
	}
	return nil
}

func (m *MockAction) GetInitCount() int32 {
	return m.initCount.Load()
}

func (m *MockAction) GetStartCount() int32 {
	return m.startCount.Load()
}

func (m *MockAction) GetStopCount() int32 {
	return m.stopCount.Load()
}

func (m *MockAction) IsRunning() bool {
	return m.isRunning.Load()
}

// TestIntegration_FullLifecycle tests the complete lifecycle of the application
func TestIntegration_FullLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Create initial config with action_a enabled
	initialConfig := `version: "1.0"
actions:
  - action_a: true
  - action_b: false
  - action_c: false
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create mock actions
	actionA := NewMockAction("action_a")
	actionB := NewMockAction("action_b")
	actionC := NewMockAction("action_c")

	// Setup registry
	registry := action.NewRegistry()
	registry.Register(actionA)
	registry.Register(actionB)
	registry.Register(actionC)

	// Create manager and watcher
	manager := action.NewManager(registry)
	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Start manager with watcher
	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.StopAll()

	// Wait for initial config to be processed
	time.Sleep(300 * time.Millisecond)

	// Verify initial state
	t.Run("InitialState", func(t *testing.T) {
		if !actionA.IsRunning() {
			t.Error("action_a should be running")
		}
		if actionB.IsRunning() {
			t.Error("action_b should not be running")
		}
		if actionC.IsRunning() {
			t.Error("action_c should not be running")
		}

		if actionA.GetInitCount() != 1 {
			t.Errorf("action_a init count should be 1, got %d", actionA.GetInitCount())
		}
		if actionA.GetStartCount() != 1 {
			t.Errorf("action_a start count should be 1, got %d", actionA.GetStartCount())
		}
	})

	// Update config to enable action_b and disable action_a
	t.Run("HotReload_EnableDisable", func(t *testing.T) {
		updatedConfig := `version: "2.0"
actions:
  - action_a: false
  - action_b: true
  - action_c: false
`
		if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
			t.Fatalf("Failed to update config: %v", err)
		}

		// Wait for hot reload
		time.Sleep(500 * time.Millisecond)

		if actionA.IsRunning() {
			t.Error("action_a should be stopped after config update")
		}
		if !actionB.IsRunning() {
			t.Error("action_b should be running after config update")
		}
		if actionC.IsRunning() {
			t.Error("action_c should still not be running")
		}

		if actionA.GetStopCount() != 1 {
			t.Errorf("action_a stop count should be 1, got %d", actionA.GetStopCount())
		}
		if actionB.GetStartCount() != 1 {
			t.Errorf("action_b start count should be 1, got %d", actionB.GetStartCount())
		}
	})

	// Enable all actions
	t.Run("HotReload_EnableAll", func(t *testing.T) {
		allEnabledConfig := `version: "3.0"
actions:
  - action_a: true
  - action_b: true
  - action_c: true
`
		if err := os.WriteFile(configPath, []byte(allEnabledConfig), 0644); err != nil {
			t.Fatalf("Failed to update config: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		if !actionA.IsRunning() {
			t.Error("action_a should be running")
		}
		if !actionB.IsRunning() {
			t.Error("action_b should be running")
		}
		if !actionC.IsRunning() {
			t.Error("action_c should be running")
		}

		// action_a was restarted
		if actionA.GetStartCount() != 2 {
			t.Errorf("action_a should have been started twice, got %d", actionA.GetStartCount())
		}
	})

	// Disable all actions
	t.Run("HotReload_DisableAll", func(t *testing.T) {
		allDisabledConfig := `version: "4.0"
actions:
  - action_a: false
  - action_b: false
  - action_c: false
`
		if err := os.WriteFile(configPath, []byte(allDisabledConfig), 0644); err != nil {
			t.Fatalf("Failed to update config: %v", err)
		}

		time.Sleep(500 * time.Millisecond)

		if actionA.IsRunning() {
			t.Error("action_a should be stopped")
		}
		if actionB.IsRunning() {
			t.Error("action_b should be stopped")
		}
		if actionC.IsRunning() {
			t.Error("action_c should be stopped")
		}
	})
}

// TestIntegration_RapidConfigChanges tests that the system handles rapid config changes correctly
func TestIntegration_RapidConfigChanges(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `version: "1.0"
actions:
  - rapid_action: false
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	rapidAction := NewMockAction("rapid_action")

	registry := action.NewRegistry()
	registry.Register(rapidAction)

	manager := action.NewManager(registry)
	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.StopAll()

	time.Sleep(200 * time.Millisecond)

	// Rapidly toggle the action multiple times
	for i := 0; i < 5; i++ {
		enabled := i%2 == 0
		cfg := `version: "` + string(rune('1'+i)) + `.0"
actions:
  - rapid_action: ` + boolToString(enabled) + `
`
		if err := os.WriteFile(configPath, []byte(cfg), 0644); err != nil {
			t.Fatalf("Failed to update config: %v", err)
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Final state should be enabled (4%2 == 0 -> true)
	time.Sleep(300 * time.Millisecond)

	if !rapidAction.IsRunning() {
		t.Error("rapid_action should be running after rapid changes")
	}
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// TestIntegration_UnregisteredAction tests behavior when config references unregistered action
func TestIntegration_UnregisteredAction(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config references an action that doesn't exist
	configContent := `version: "1.0"
actions:
  - known_action: true
  - unknown_action: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	knownAction := NewMockAction("known_action")

	registry := action.NewRegistry()
	registry.Register(knownAction)
	// Note: unknown_action is NOT registered

	manager := action.NewManager(registry)
	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Should not crash even with unknown action
	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.StopAll()

	time.Sleep(300 * time.Millisecond)

	// Known action should still work
	if !knownAction.IsRunning() {
		t.Error("known_action should be running despite unknown_action in config")
	}
}

// TestIntegration_EmptyConfig tests behavior with empty actions
func TestIntegration_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	emptyConfig := `version: "1.0"
actions: []
`
	if err := os.WriteFile(configPath, []byte(emptyConfig), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	testAction := NewMockAction("test_action")

	registry := action.NewRegistry()
	registry.Register(testAction)

	manager := action.NewManager(registry)
	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.StopAll()

	time.Sleep(200 * time.Millisecond)

	// No actions should be running
	if testAction.IsRunning() {
		t.Error("test_action should not be running with empty config")
	}
}

// TestIntegration_ConcurrentAccess tests thread safety
func TestIntegration_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `version: "1.0"
actions:
  - concurrent_action: true
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	concurrentAction := NewMockAction("concurrent_action")

	registry := action.NewRegistry()
	registry.Register(concurrentAction)

	manager := action.NewManager(registry)
	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.StopAll()

	time.Sleep(200 * time.Millisecond)

	// Concurrently read state and update config
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)

		// Reader goroutine
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = manager.IsRunning("concurrent_action")
				_ = manager.GetRunningActions()
			}
		}()

		// Config updater goroutine
		go func(idx int) {
			defer wg.Done()
			enabled := idx%2 == 0
			cfg := `version: "` + string(rune('a'+idx)) + `"
actions:
  - concurrent_action: ` + boolToString(enabled) + `
`
			os.WriteFile(configPath, []byte(cfg), 0644)
			time.Sleep(50 * time.Millisecond)
		}(i)
	}

	wg.Wait()

	// Should not panic or deadlock - if we reach here, the test passes
}

// TestIntegration_WithDefaultActions tests with the actual default actions
func TestIntegration_WithDefaultActions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `version: "1.0"
actions:
  - cpu_metrics: true
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	registry := action.NewRegistry()
	compute.RegisterActions(registry)

	manager := action.NewManager(registry)
	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.StopAll()

	time.Sleep(300 * time.Millisecond)

	// Check initial state
	if !manager.IsRunning("cpu_metrics") {
		t.Error("cpu_metrics should be running")
	}

	// Disable cpu_metrics
	disabledConfig := `version: "2.0"
actions:
  - cpu_metrics: false
`
	if err := os.WriteFile(configPath, []byte(disabledConfig), 0644); err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	running := manager.GetRunningActions()
	if len(running) != 0 {
		t.Errorf("Expected 0 running actions, got %d: %v", len(running), running)
	}
}

// TestIntegration_ConfigWatcherReconnect tests watcher behavior after file recreation
func TestIntegration_ConfigWatcherReconnect(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `version: "1.0"
actions:
  - reconnect_action: true
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	reconnectAction := NewMockAction("reconnect_action")

	registry := action.NewRegistry()
	registry.Register(reconnectAction)

	manager := action.NewManager(registry)
	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.StopAll()

	time.Sleep(300 * time.Millisecond)

	if !reconnectAction.IsRunning() {
		t.Error("reconnect_action should be running initially")
	}

	// Modify file content (simulating external edit)
	updatedConfig := `version: "2.0"
actions:
  - reconnect_action: false
`
	if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	if reconnectAction.IsRunning() {
		t.Error("reconnect_action should be stopped after config update")
	}
}

// TestIntegration_StopAllGracefully tests that StopAll stops all actions gracefully
func TestIntegration_StopAllGracefully(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `version: "1.0"
actions:
  - graceful_a: true
  - graceful_b: true
  - graceful_c: true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	actionA := NewMockAction("graceful_a")
	actionB := NewMockAction("graceful_b")
	actionC := NewMockAction("graceful_c")

	var stopOrder []string
	var stopMu sync.Mutex

	actionA.onStopFunc = func() {
		stopMu.Lock()
		stopOrder = append(stopOrder, "a")
		stopMu.Unlock()
	}
	actionB.onStopFunc = func() {
		stopMu.Lock()
		stopOrder = append(stopOrder, "b")
		stopMu.Unlock()
	}
	actionC.onStopFunc = func() {
		stopMu.Lock()
		stopOrder = append(stopOrder, "c")
		stopMu.Unlock()
	}

	registry := action.NewRegistry()
	registry.Register(actionA)
	registry.Register(actionB)
	registry.Register(actionC)

	manager := action.NewManager(registry)
	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	// Verify all running
	if !actionA.IsRunning() || !actionB.IsRunning() || !actionC.IsRunning() {
		t.Error("All actions should be running before StopAll")
	}

	// Stop all
	manager.StopAll()

	// Verify all stopped
	if actionA.IsRunning() || actionB.IsRunning() || actionC.IsRunning() {
		t.Error("All actions should be stopped after StopAll")
	}

	// Verify stop was called for all
	stopMu.Lock()
	if len(stopOrder) != 3 {
		t.Errorf("Expected 3 stops, got %d", len(stopOrder))
	}
	stopMu.Unlock()
}

// TestIntegration_VersionTracking tests that config version changes are tracked
func TestIntegration_VersionTracking(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialConfig := `version: "1.0.0"
actions:
  - version_action: true
`
	if err := os.WriteFile(configPath, []byte(initialConfig), 0644); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	versionAction := NewMockAction("version_action")

	registry := action.NewRegistry()
	registry.Register(versionAction)

	manager := action.NewManager(registry)
	watcher, err := config.NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	if err := manager.StartWithWatcher(watcher); err != nil {
		t.Fatalf("Failed to start manager: %v", err)
	}
	defer manager.StopAll()

	time.Sleep(200 * time.Millisecond)

	// Check initial version
	cfg := watcher.GetCurrentConfig()
	if cfg.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", cfg.Version)
	}

	// Update to new version
	updatedConfig := `version: "2.0.0"
actions:
  - version_action: true
`
	if err := os.WriteFile(configPath, []byte(updatedConfig), 0644); err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	cfg = watcher.GetCurrentConfig()
	if cfg.Version != "2.0.0" {
		t.Errorf("Expected version '2.0.0', got '%s'", cfg.Version)
	}
}
