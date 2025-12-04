package config

import (
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ConfigWatcher watches for config file changes and triggers callbacks
type ConfigWatcher struct {
	path       string
	watcher    *fsnotify.Watcher
	callbacks  []func(*BPFStackConfig)
	mu         sync.RWMutex
	stopCh     chan struct{}
	debounce   time.Duration
	lastConfig *BPFStackConfig
	configMu   sync.RWMutex // Separate mutex for lastConfig to avoid deadlock
}

// NewConfigWatcher creates a new config watcher for the given path
func NewConfigWatcher(path string) (*ConfigWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	cw := &ConfigWatcher{
		path:      path,
		watcher:   watcher,
		callbacks: make([]func(*BPFStackConfig), 0),
		stopCh:    make(chan struct{}),
		debounce:  100 * time.Millisecond,
	}

	return cw, nil
}

// OnChange registers a callback to be called when config changes
func (cw *ConfigWatcher) OnChange(callback func(*BPFStackConfig)) {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	cw.callbacks = append(cw.callbacks, callback)
}

// Start begins watching the config file for changes
func (cw *ConfigWatcher) Start() error {
	cfg, err := ReadYAMLConfig(cw.path)
	if err != nil {
		return err
	}

	cw.configMu.Lock()
	cw.lastConfig = cfg
	cw.configMu.Unlock()

	// Notify callbacks with initial config
	cw.notifyCallbacks(cfg)

	if err := cw.watcher.Add(cw.path); err != nil {
		return err
	}

	// Start watching in goroutine
	go cw.watch()

	return nil
}

// watch handles file system events
func (cw *ConfigWatcher) watch() {
	var debounceTimer *time.Timer

	for {
		select {
		case event, ok := <-cw.watcher.Events:
			if !ok {
				return
			}

			// Only handle write events
			if event.Op&fsnotify.Write == fsnotify.Write {
				// Debounce: reset timer on each event
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(cw.debounce, func() {
					cw.reloadConfig()
				})
			}

		case err, ok := <-cw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Config watcher error: %v", err)

		case <-cw.stopCh:
			if debounceTimer != nil {
				debounceTimer.Stop()
			}
			return
		}
	}
}

// reloadConfig reads the config file and notifies callbacks if changed
func (cw *ConfigWatcher) reloadConfig() {
	cfg, err := ReadYAMLConfig(cw.path)
	if err != nil {
		log.Printf("Failed to reload config: %v", err)
		return
	}

	log.Printf("Config reloaded successfully")

	cw.configMu.Lock()
	cw.lastConfig = cfg
	cw.configMu.Unlock()

	cw.notifyCallbacks(cfg)
}

// notifyCallbacks calls all registered callbacks with the new config
func (cw *ConfigWatcher) notifyCallbacks(cfg *BPFStackConfig) {
	cw.mu.RLock()
	defer cw.mu.RUnlock()

	for _, callback := range cw.callbacks {
		callback(cfg)
	}
}

// GetCurrentConfig returns the current config
func (cw *ConfigWatcher) GetCurrentConfig() *BPFStackConfig {
	cw.configMu.RLock()
	defer cw.configMu.RUnlock()
	return cw.lastConfig
}

// Stop stops the config watcher
func (cw *ConfigWatcher) Stop() error {
	close(cw.stopCh)
	return cw.watcher.Close()
}
