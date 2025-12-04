package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestBPFStackConfig_YAMLMarshal(t *testing.T) {
	cfg := &BPFStackConfig{
		Version: "1.0",
		Actions: []map[string]bool{
			{"cpu_metrics": true},
			{"memory_metrics": false},
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// Verify it can be unmarshaled back
	var result BPFStackConfig
	if err := yaml.Unmarshal(data, &result); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	if result.Version != cfg.Version {
		t.Errorf("Version mismatch: expected %s, got %s", cfg.Version, result.Version)
	}

	if len(result.Actions) != len(cfg.Actions) {
		t.Errorf("Actions length mismatch: expected %d, got %d", len(cfg.Actions), len(result.Actions))
	}
}

func TestBPFStackConfig_YAMLUnmarshal(t *testing.T) {
	yamlData := `
version: "2.0"
actions:
  - action1: true
  - action2: false
  - action3: true
`
	var cfg BPFStackConfig
	if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if cfg.Version != "2.0" {
		t.Errorf("Expected version '2.0', got '%s'", cfg.Version)
	}

	if len(cfg.Actions) != 3 {
		t.Errorf("Expected 3 actions, got %d", len(cfg.Actions))
	}

	// Check action values
	if enabled, ok := cfg.Actions[0]["action1"]; !ok || !enabled {
		t.Error("Expected action1 to be true")
	}
	if enabled, ok := cfg.Actions[1]["action2"]; !ok || enabled {
		t.Error("Expected action2 to be false")
	}
	if enabled, ok := cfg.Actions[2]["action3"]; !ok || !enabled {
		t.Error("Expected action3 to be true")
	}
}

func TestBPFStackConfig_EmptyActions(t *testing.T) {
	yamlData := `
version: "1.0"
actions: []
`
	var cfg BPFStackConfig
	if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(cfg.Actions) != 0 {
		t.Errorf("Expected 0 actions, got %d", len(cfg.Actions))
	}
}

func TestBPFStackConfig_NoActions(t *testing.T) {
	yamlData := `version: "1.0"`

	var cfg BPFStackConfig
	if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(cfg.Actions) != 0 {
		t.Errorf("Expected nil or empty actions, got %v", cfg.Actions)
	}
}

func TestBPFStackConfig_VersionOnly(t *testing.T) {
	yamlData := `version: "3.5.1"`

	var cfg BPFStackConfig
	if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if cfg.Version != "3.5.1" {
		t.Errorf("Expected version '3.5.1', got '%s'", cfg.Version)
	}
}

func TestBPFStackConfig_MultipleActionsInSingleMap(t *testing.T) {
	yamlData := `
version: "1.0"
actions:
  - action1: true
    action2: false
    action3: true
`
	var cfg BPFStackConfig
	if err := yaml.Unmarshal([]byte(yamlData), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if len(cfg.Actions) != 1 {
		t.Errorf("Expected 1 action map, got %d", len(cfg.Actions))
	}

	if len(cfg.Actions[0]) != 3 {
		t.Errorf("Expected 3 entries in map, got %d", len(cfg.Actions[0]))
	}
}
