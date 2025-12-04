package action

import (
	"testing"
)

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	action := NewMockTestAction("test_action")
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

	action1 := NewMockTestAction("duplicate")
	action2 := NewMockTestAction("duplicate")

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
	action := NewMockTestAction("my_action")
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

	action1 := NewMockTestAction("action1")
	action2 := NewMockTestAction("action2")

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

	action1 := NewMockTestAction("alpha")
	action2 := NewMockTestAction("beta")

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
