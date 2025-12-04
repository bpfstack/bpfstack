package config

// BPFStackConfig is the configuration for the BPFStack
type BPFStackConfig struct {
	// Version is the version of the BPFStack configuration
	Version string `yaml:"version"`
	// Actions is the list of actions to enable
	// e.g.) actions:
	//   - vmexit: true
	Actions []map[string]bool `yaml:"actions"`
}
