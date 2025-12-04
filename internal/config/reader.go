package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func ReadYAMLConfig(path string) (*BPFStackConfig, error) {
	cfg := &BPFStackConfig{}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}
