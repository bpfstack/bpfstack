package action

import (
	"testing"
	"time"
)

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	action := NewPrintAction("test_action", "Hello", 1*time.Second)
	err := registry.Register(action)
	if err != nil {
		t.Fatalf("Failed to register action: %v", err)
	}

	// Verify action was registered
	retrieved, exists := registry.Get("test_action")
	if !exists {
		t.Error("Expected action to exist")
	}
	if retrieved.Name() != "test_action" {
		t.Errorf("Expected name 'test_action', got '%s'", retrieved.Name())
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewRegistry()

	action1 := NewPrintAction("duplicate", "First", 1*time.Second)
	action2 := NewPrintAction("duplicate", "Second", 1*time.Second)

	if err := registry.Register(action1); err != nil {
		t.Fatalf("Failed to register first action: %v", err)
	}

	err := registry.Register(action2)
	if err == nil {
		t.Error("Expected error when registering duplicate action, got nil")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	// Test getting non-existent action
	_, exists := registry.Get("nonexistent")
	if exists {
		t.Error("Expected non-existent action to not exist")
	}

	// Register and get
	action := NewPrintAction("my_action", "Test", 1*time.Second)
	registry.Register(action)

	retrieved, exists := registry.Get("my_action")
	if !exists {
		t.Error("Expected action to exist")
	}
	if retrieved != action {
		t.Error("Retrieved action is not the same as registered")
	}
}

func TestRegistry_GetAll(t *testing.T) {
	registry := NewRegistry()

	action1 := NewPrintAction("action1", "First", 1*time.Second)
	action2 := NewPrintAction("action2", "Second", 1*time.Second)

	registry.Register(action1)
	registry.Register(action2)

	all := registry.GetAll()
	if len(all) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(all))
	}

	if _, ok := all["action1"]; !ok {
		t.Error("Expected action1 in result")
	}
	if _, ok := all["action2"]; !ok {
		t.Error("Expected action2 in result")
	}
}

func TestRegistry_Names(t *testing.T) {
	registry := NewRegistry()

	action1 := NewPrintAction("alpha", "First", 1*time.Second)
	action2 := NewPrintAction("beta", "Second", 1*time.Second)

	registry.Register(action1)
	registry.Register(action2)

	names := registry.Names()
	if len(names) != 2 {
		t.Errorf("Expected 2 names, got %d", len(names))
	}

	// Check both names are present (order may vary)
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	if !nameMap["alpha"] {
		t.Error("Expected 'alpha' in names")
	}
	if !nameMap["beta"] {
		t.Error("Expected 'beta' in names")
	}
}
