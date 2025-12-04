package config

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewConfigWatcher_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `version: "1.0"
actions: []
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	if watcher.path != configPath {
		t.Errorf("Expected path '%s', got '%s'", configPath, watcher.path)
	}
}

func TestConfigWatcher_OnChange(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `version: "1.0"
actions: []
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	callCount := 0
	watcher.OnChange(func(cfg *BPFStackConfig) {
		callCount++
	})

	watcher.OnChange(func(cfg *BPFStackConfig) {
		callCount++
	})

	// Should have 2 callbacks registered
	if len(watcher.callbacks) != 2 {
		t.Errorf("Expected 2 callbacks, got %d", len(watcher.callbacks))
	}
}

func TestConfigWatcher_Start(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `version: "1.0"
actions:
  - test_action: true
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	var receivedConfig *BPFStackConfig
	watcher.OnChange(func(cfg *BPFStackConfig) {
		receivedConfig = cfg
	})

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Should receive initial config
	if receivedConfig == nil {
		t.Fatal("Expected to receive initial config")
	}

	if receivedConfig.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", receivedConfig.Version)
	}
}

func TestConfigWatcher_Start_FileNotFound(t *testing.T) {
	watcher, err := NewConfigWatcher("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	err = watcher.Start()
	if err == nil {
		t.Error("Expected error when starting with non-existent file")
	}
}

func TestConfigWatcher_GetCurrentConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `version: "2.0"
actions:
  - my_action: true
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	// Before start, should be nil
	if watcher.GetCurrentConfig() != nil {
		t.Error("Expected nil config before Start()")
	}

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// After start, should have config
	cfg := watcher.GetCurrentConfig()
	if cfg == nil {
		t.Fatal("Expected config after Start()")
	}

	if cfg.Version != "2.0" {
		t.Errorf("Expected version '2.0', got '%s'", cfg.Version)
	}
}

func TestConfigWatcher_HotReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	initialContent := `version: "1.0"
actions:
  - action_a: true
`
	if err := os.WriteFile(configPath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	configCh := make(chan *BPFStackConfig, 10)
	watcher.OnChange(func(cfg *BPFStackConfig) {
		configCh <- cfg
	})

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Receive initial config
	select {
	case cfg := <-configCh:
		if cfg.Version != "1.0" {
			t.Errorf("Expected initial version '1.0', got '%s'", cfg.Version)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for initial config")
	}

	// Update config file
	updatedContent := `version: "2.0"
actions:
  - action_a: false
  - action_b: true
`
	if err := os.WriteFile(configPath, []byte(updatedContent), 0644); err != nil {
		t.Fatalf("Failed to update config file: %v", err)
	}

	// Wait for reload
	select {
	case cfg := <-configCh:
		if cfg.Version != "2.0" {
			t.Errorf("Expected updated version '2.0', got '%s'", cfg.Version)
		}
		if len(cfg.Actions) != 2 {
			t.Errorf("Expected 2 actions, got %d", len(cfg.Actions))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for config reload")
	}
}

func TestConfigWatcher_Stop(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `version: "1.0"
actions: []
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Stop should not panic or error
	if err := watcher.Stop(); err != nil {
		t.Errorf("Stop returned error: %v", err)
	}
}

func TestConfigWatcher_MultipleCallbacks(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `version: "1.0"
actions: []
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	var callCount atomic.Int32

	// Register multiple callbacks
	for i := 0; i < 5; i++ {
		watcher.OnChange(func(cfg *BPFStackConfig) {
			callCount.Add(1)
		})
	}

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Wait a bit for callbacks to be called
	time.Sleep(100 * time.Millisecond)

	// All 5 callbacks should have been called once
	if callCount.Load() != 5 {
		t.Errorf("Expected 5 callbacks to be called, got %d", callCount.Load())
	}
}

func TestConfigWatcher_ConcurrentOnChange(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `version: "1.0"
actions: []
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	// Concurrently add callbacks
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			watcher.OnChange(func(cfg *BPFStackConfig) {})
		}()
	}
	wg.Wait()

	if len(watcher.callbacks) != 10 {
		t.Errorf("Expected 10 callbacks, got %d", len(watcher.callbacks))
	}
}

func TestConfigWatcher_Debounce(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `version: "1.0"
actions: []
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	var callCount atomic.Int32
	watcher.OnChange(func(cfg *BPFStackConfig) {
		callCount.Add(1)
	})

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Initial callback
	time.Sleep(50 * time.Millisecond)
	initialCount := callCount.Load()

	// Rapidly write to file multiple times
	for i := 0; i < 5; i++ {
		newContent := `version: "` + string(rune('1'+i)) + `.0"
actions: []
`
		os.WriteFile(configPath, []byte(newContent), 0644)
		time.Sleep(20 * time.Millisecond) // Faster than debounce interval
	}

	// Wait for debounce to complete
	time.Sleep(300 * time.Millisecond)

	// Due to debouncing, we should have fewer callbacks than file writes
	finalCount := callCount.Load()
	additionalCalls := finalCount - initialCount

	// Should have at most 2-3 calls due to debouncing (not 5)
	if additionalCalls > 3 {
		t.Errorf("Debouncing failed: expected at most 3 additional calls, got %d", additionalCalls)
	}
}

func TestConfigWatcher_InvalidConfigOnReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	validContent := `version: "1.0"
actions:
  - test: true
`
	if err := os.WriteFile(configPath, []byte(validContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	var lastConfig *BPFStackConfig
	watcher.OnChange(func(cfg *BPFStackConfig) {
		lastConfig = cfg
	})

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Remember the valid config
	validConfigVersion := lastConfig.Version

	// Write invalid YAML
	invalidContent := `{{{invalid`
	os.WriteFile(configPath, []byte(invalidContent), 0644)

	time.Sleep(300 * time.Millisecond)

	// Last config should still be the valid one (reload failed gracefully)
	currentConfig := watcher.GetCurrentConfig()
	if currentConfig.Version != validConfigVersion {
		t.Errorf("Config should remain unchanged after invalid reload, got version '%s'", currentConfig.Version)
	}
}

func TestConfigWatcher_MultipleReloads(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `version: "1.0"
actions: []
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	versions := make([]string, 0)
	var mu sync.Mutex

	watcher.OnChange(func(cfg *BPFStackConfig) {
		mu.Lock()
		versions = append(versions, cfg.Version)
		mu.Unlock()
	})

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Make several sequential updates
	for i := 2; i <= 5; i++ {
		time.Sleep(200 * time.Millisecond)
		newContent := `version: "` + string(rune('0'+i)) + `.0"
actions: []
`
		os.WriteFile(configPath, []byte(newContent), 0644)
	}

	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should have received multiple version updates
	if len(versions) < 2 {
		t.Errorf("Expected at least 2 version updates, got %d: %v", len(versions), versions)
	}

	// First version should be "1.0"
	if versions[0] != "1.0" {
		t.Errorf("Expected first version '1.0', got '%s'", versions[0])
	}
}

func TestConfigWatcher_CallbackOrder(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `version: "1.0"
actions: []
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	order := make([]int, 0)
	var mu sync.Mutex

	// Register callbacks in order
	for i := 1; i <= 3; i++ {
		idx := i
		watcher.OnChange(func(cfg *BPFStackConfig) {
			mu.Lock()
			order = append(order, idx)
			mu.Unlock()
		})
	}

	if err := watcher.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Callbacks should be called in registration order
	expected := []int{1, 2, 3}
	if len(order) != len(expected) {
		t.Errorf("Expected %d callbacks, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if i < len(order) && order[i] != v {
			t.Errorf("Callback order mismatch at index %d: expected %d, got %d", i, v, order[i])
		}
	}
}

func TestConfigWatcher_NoCallbacks(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `version: "1.0"
actions: []
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	watcher, err := NewConfigWatcher(configPath)
	if err != nil {
		t.Fatalf("NewConfigWatcher failed: %v", err)
	}
	defer watcher.Stop()

	// Start without registering any callbacks - should not panic
	if err := watcher.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Update config - should not panic
	newContent := `version: "2.0"
actions: []
`
	os.WriteFile(configPath, []byte(newContent), 0644)

	time.Sleep(300 * time.Millisecond)

	// Config should still be updated
	cfg := watcher.GetCurrentConfig()
	if cfg.Version != "2.0" {
		t.Errorf("Expected version '2.0', got '%s'", cfg.Version)
	}
}
