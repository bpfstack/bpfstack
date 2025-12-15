// Package core contains the core logic for the agent.
package core

import (
	"context"
	"fmt"
	"sync"
)

// ProbeFactoryFunc is the function type for the probe factory.
type ProbeFactoryFunc func() Prober

// ProbeManager is the manager for the probes.
type ProbeManager struct {
	// dataChan is double-buffered channels for data
	dataChan  chan TelemetryEvent
    // errorChan is double-buffered channels for errors.
	errorChan chan error
	// registry is the collection of available probes.
	registry map[string]ProbeFactoryFunc
	// activeProbes is the collection of active probes.
	activeProbes map[string]Prober
	// cancelFuncs is the collection of cancel functions for the probes.
	cancelFuncs map[string]context.CancelFunc
	// mu is the mutex for the activeProbes map.
	mu sync.RWMutex
}

// NewProbeManager creates a new probe manager.
// [수정 2] 인자를 받지 않고 내부에서 채널을 생성합니다.
func NewProbeManager() *ProbeManager {
	return &ProbeManager{
		// Buffer to prevent blocking
		dataChan:     make(chan TelemetryEvent, 100),
		errorChan:    make(chan error, 100),
		registry:     make(map[string]ProbeFactoryFunc),
		activeProbes: make(map[string]Prober),
		cancelFuncs:  make(map[string]context.CancelFunc),
	}
}

// DataChan getter
// Returns a read-only channel for the data.
// It enhances the safety of the data channel.
func (pm *ProbeManager) DataChan() <-chan TelemetryEvent {
	return pm.dataChan
}

// ErrorChan getter
func (pm *ProbeManager) ErrorChan() <-chan error {
	return pm.errorChan
}

// Register registers a new probe.
func (pm *ProbeManager) Register(name string, factory ProbeFactoryFunc) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.registry[name] = factory
}

// Reconcile is the function that reconciles the active probes with the config.
func (pm *ProbeManager) Reconcile(ctx context.Context, config map[string]bool) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, shouldRun := range config {
		_, isRunning := pm.activeProbes[name]

		// Start the probe if it should run but is not running
		if shouldRun && !isRunning {
			if err := pm.startProbe(ctx, name); err != nil {
				// Non-blocking error channel
				select {
				case pm.errorChan <- fmt.Errorf("failed to start probe %s: %w", name, err):
				default:
					fmt.Printf("Error channel full, dropping start error for %s\n", name)
				}
			}
		}

		// Stop the probe if it is running but should not run
		if !shouldRun && isRunning {
			if err := pm.stopProbe(name); err != nil {
				select {
				case pm.errorChan <- fmt.Errorf("failed to stop probe %s: %w", name, err):
				default:
					fmt.Printf("Error channel full, dropping stop error for %s\n", name)
				}
			}
		}
	}
	return nil
}

// startProbe starts a probe. (Called inside Lock)
func (pm *ProbeManager) startProbe(ctx context.Context, name string) error {
	factory, exists := pm.registry[name]
	if !exists {
		return fmt.Errorf("probe %s not found", name)
	}

	probe := factory()
	if err := probe.Load(); err != nil {
		return fmt.Errorf("failed to load probe %s: %w", name, err)
	}

	// Create a derived context for cancellation
	probeCtx, cancel := context.WithCancel(ctx)

	pm.cancelFuncs[name] = cancel
	pm.activeProbes[name] = probe

	// Each probe runs in its own goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				select {
				case pm.errorChan <- fmt.Errorf("probe %s panicked: %v", name, r):
				default:
				}
			}
		}()
		
		if err := probe.Run(probeCtx, pm.dataChan); err != nil {
			if probeCtx.Err() == nil { // Only report if not canceled intentionally
				select {
				case pm.errorChan <- fmt.Errorf("probe %s runtime error: %w", name, err):
				default:
				}
			}
		}
	}()

	fmt.Printf("Probe started: %s\n", name)
	return nil
}

// stopProbe stops a probe. (Called inside Lock)
func (pm *ProbeManager) stopProbe(name string) error {
	// Cancel the context first to stop the Run loop
	if cancel, ok := pm.cancelFuncs[name]; ok {
		cancel()
		delete(pm.cancelFuncs, name)
	}

	// Close the probe resources
	if probe, exists := pm.activeProbes[name]; exists {
		if err := probe.Close(); err != nil {
			return err
		}
		delete(pm.activeProbes, name)
		fmt.Printf("Probe stopped: %s\n", name)
	}
	return nil
}

// Shutdown stops all active probes.
func (pm *ProbeManager) Shutdown() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	fmt.Println("Shutting down ProbeManager...")

	for name := range pm.activeProbes {
		if err := pm.stopProbe(name); err != nil {
			fmt.Printf("Failed to stop probe %s during shutdown: %v\n", name, err)
		}
	}
	close(pm.dataChan)
	close(pm.errorChan)
}