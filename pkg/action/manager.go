package action

import (
	"sync"

	"github.com/bpfstack/bpfstack/internal/config"
	"github.com/bpfstack/bpfstack/pkg/logger"
)

// Manager manages the lifecycle of actions based on config
type Manager struct {
	registry      *Registry
	runningMu     sync.RWMutex
	running       map[string]bool // tracks which actions are currently running
	configWatcher *config.ConfigWatcher
	logger        *logger.Logger
}

// NewManager creates a new action manager
func NewManager(registry *Registry) *Manager {
	return &Manager{
		registry: registry,
		running:  make(map[string]bool),
		logger:   logger.New("manager"),
	}
}

// StartWithWatcher starts the manager with a config watcher for hot-reloading
func (m *Manager) StartWithWatcher(watcher *config.ConfigWatcher) error {
	m.configWatcher = watcher

	// Register callback for config changes
	watcher.OnChange(m.handleConfigChange)

	// Start the watcher
	return watcher.Start()
}

// handleConfigChange is called when config changes
func (m *Manager) handleConfigChange(cfg *config.BPFStackConfig) {
	m.logger.Info("handling config change", logger.Fields{
		"version": cfg.Version,
	})

	// Build a map of desired action states from config
	desiredState := make(map[string]bool)
	for _, actionMap := range cfg.Actions {
		for name, enabled := range actionMap {
			desiredState[name] = enabled
		}
	}

	// Get current running state
	m.runningMu.RLock()
	currentRunning := make(map[string]bool)
	for k, v := range m.running {
		currentRunning[k] = v
	}
	m.runningMu.RUnlock()

	// Stop actions that should no longer be running
	for name, isRunning := range currentRunning {
		if isRunning && !desiredState[name] {
			m.stopAction(name)
		}
	}

	// Start actions that should be running
	for name, shouldRun := range desiredState {
		if shouldRun && !currentRunning[name] {
			m.startAction(name)
		}
	}
}

// startAction starts an action by name
func (m *Manager) startAction(name string) {
	action, exists := m.registry.Get(name)
	if !exists {
		m.logger.Warn("action not found in registry", logger.Fields{
			"action": name,
		})
		return
	}

	// Initialize the action
	if err := action.Init(); err != nil {
		m.logger.Error("failed to initialize action", logger.Fields{
			"action": name,
			"error":  err.Error(),
		})
		return
	}

	// Start the action
	if err := action.Start(); err != nil {
		m.logger.Error("failed to start action", logger.Fields{
			"action": name,
			"error":  err.Error(),
		})
		return
	}

	m.runningMu.Lock()
	m.running[name] = true
	m.runningMu.Unlock()

	m.logger.Info("action started", logger.Fields{
		"action": name,
	})
}

// stopAction stops an action by name
func (m *Manager) stopAction(name string) {
	action, exists := m.registry.Get(name)
	if !exists {
		m.logger.Warn("action not found in registry", logger.Fields{
			"action": name,
		})
		return
	}

	if err := action.Stop(); err != nil {
		m.logger.Error("failed to stop action", logger.Fields{
			"action": name,
			"error":  err.Error(),
		})
		return
	}

	m.runningMu.Lock()
	m.running[name] = false
	m.runningMu.Unlock()

	m.logger.Info("action stopped", logger.Fields{
		"action": name,
	})
}

// StopAll stops all running actions
func (m *Manager) StopAll() {
	m.runningMu.RLock()
	runningActions := make([]string, 0)
	for name, isRunning := range m.running {
		if isRunning {
			runningActions = append(runningActions, name)
		}
	}
	m.runningMu.RUnlock()

	for _, name := range runningActions {
		m.stopAction(name)
	}

	if m.configWatcher != nil {
		if err := m.configWatcher.Stop(); err != nil {
			m.logger.Error("failed to stop config watcher", logger.Fields{
				"error": err.Error(),
			})
		}
	}
}

// IsRunning checks if an action is currently running
func (m *Manager) IsRunning(name string) bool {
	m.runningMu.RLock()
	defer m.runningMu.RUnlock()
	return m.running[name]
}

// GetRunningActions returns a list of currently running action names
func (m *Manager) GetRunningActions() []string {
	m.runningMu.RLock()
	defer m.runningMu.RUnlock()

	result := make([]string, 0)
	for name, isRunning := range m.running {
		if isRunning {
			result = append(result, name)
		}
	}
	return result
}
