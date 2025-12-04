package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadYAMLConfig_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `version: "1.0"
actions:
  - cpu_metrics: true
  - memory_metrics: false
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := ReadYAMLConfig(configPath)
	if err != nil {
		t.Fatalf("ReadYAMLConfig failed: %v", err)
	}

	if cfg.Version != "1.0" {
		t.Errorf("Expected version '1.0', got '%s'", cfg.Version)
	}

	if len(cfg.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(cfg.Actions))
	}
}

func TestReadYAMLConfig_FileNotFound(t *testing.T) {
	_, err := ReadYAMLConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestReadYAMLConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Invalid YAML content
	content := `{{{invalid yaml content`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err := ReadYAMLConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestReadYAMLConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.yaml")

	if err := os.WriteFile(configPath, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := ReadYAMLConfig(configPath)
	if err != nil {
		t.Fatalf("ReadYAMLConfig failed: %v", err)
	}

	// Empty file should result in zero-value config
	if cfg.Version != "" {
		t.Errorf("Expected empty version, got '%s'", cfg.Version)
	}
}

func TestReadYAMLConfig_PartialConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "partial.yaml")

	// Only version, no actions
	content := `version: "3.0"`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := ReadYAMLConfig(configPath)
	if err != nil {
		t.Fatalf("ReadYAMLConfig failed: %v", err)
	}

	if cfg.Version != "3.0" {
		t.Errorf("Expected version '3.0', got '%s'", cfg.Version)
	}
}

func TestReadYAMLConfig_MultipleActionsPerEntry(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "multi.yaml")

	// Multiple actions in single map entry (unusual but valid)
	content := `version: "1.0"
actions:
  - action1: true
    action2: false
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := ReadYAMLConfig(configPath)
	if err != nil {
		t.Fatalf("ReadYAMLConfig failed: %v", err)
	}

	if len(cfg.Actions) != 1 {
		t.Errorf("Expected 1 action map, got %d", len(cfg.Actions))
	}

	if len(cfg.Actions[0]) != 2 {
		t.Errorf("Expected 2 entries in action map, got %d", len(cfg.Actions[0]))
	}
}

func TestReadYAMLConfig_AllActionsEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "all_enabled.yaml")

	content := `version: "1.0"
actions:
  - cpu_metrics: true
  - memory_metrics: true
  - vmexit: true
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := ReadYAMLConfig(configPath)
	if err != nil {
		t.Fatalf("ReadYAMLConfig failed: %v", err)
	}

	for i, action := range cfg.Actions {
		for name, enabled := range action {
			if !enabled {
				t.Errorf("Action %d (%s) should be enabled", i, name)
			}
		}
	}
}

func TestReadYAMLConfig_AllActionsDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "all_disabled.yaml")

	content := `version: "1.0"
actions:
  - cpu_metrics: false
  - memory_metrics: false
  - vmexit: false
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := ReadYAMLConfig(configPath)
	if err != nil {
		t.Fatalf("ReadYAMLConfig failed: %v", err)
	}

	for i, action := range cfg.Actions {
		for name, enabled := range action {
			if enabled {
				t.Errorf("Action %d (%s) should be disabled", i, name)
			}
		}
	}
}

func TestReadYAMLConfig_PermissionDenied(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "noperm.yaml")

	content := `version: "1.0"`
	if err := os.WriteFile(configPath, []byte(content), 0000); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	defer os.Chmod(configPath, 0644) // Restore permissions for cleanup

	_, err := ReadYAMLConfig(configPath)
	if err == nil {
		t.Error("Expected error for permission denied")
	}
}

func TestReadYAMLConfig_LargeConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "large.yaml")

	// Generate config with many actions
	content := `version: "1.0"
actions:
`
	for i := 0; i < 100; i++ {
		content += "  - action_" + string(rune('a'+i%26)) + "_" + string(rune('0'+i/26)) + ": true\n"
	}

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := ReadYAMLConfig(configPath)
	if err != nil {
		t.Fatalf("ReadYAMLConfig failed: %v", err)
	}

	if len(cfg.Actions) != 100 {
		t.Errorf("Expected 100 actions, got %d", len(cfg.Actions))
	}
}
