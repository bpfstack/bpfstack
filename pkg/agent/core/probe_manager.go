package core

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// ProbeFactoryFunc is the function type for the probe factory.
type ProbeFactoryFunc func() Probe

// ProbeManager is the manager for the probes.
type ProbeManager struct {
    // dataChan is the channel where the probe will send the telemetry events.
    dataChan chan<- TelemetryEvent
    // registry is the collection of available probes.
    registry map[string]ProbeFactoryFunc
    // activeProbes is the collection of active probes.
    activeProbes map[string]Probe
    // mu is the mutex for the activeProbes map.
    mu sync.RWMutex
}

func NewProbeManager(ch chan<- TelemetryEvent) *ProbeManager {
	return &ProbeManager{
		dataChan:     ch,
		registry:     make(map[string]ProbeFactoryFunc),
		activeProbes: make(map[string]Probe),
	}
}

func (pm *ProbeManager) Register(name string, factory ProbeFactoryFunc) {
    // Lock the mutex to prevent race conditions.
    pm.mu.Lock()
    defer pm.mu.Unlock()
    pm.registry[name] = factory
}

// Reconcile is the function that reconciles the active probes with the config.
func (pm *ProbeManager) Reconcile(ctx context.Context, config map[string]bool){
    // Lock the mutext to prevent race conditions.
    pm.mu.Lock()
    defer pm.mu.Unlock()

    for name, shouldRun := range config {
        _, isRunning := pm.activeProbes[name]
        
        // if the probe is not running and should be run,
        // start it 
        if shouldRun && !isRunning {
            pm.startProbe(ctx, name)
        }

        // if the probe is running and should not be run,
        // stop it
        if !shouldRun && isRunning {
            pm.stopProbe(name)
        }
    } 
}

// startProbe is the function that starts a probe.
func (pm *ProbeManager) startProbe(ctx context.Context, name string){
    factory, exists := pm.registry[name]
    if !exists {
        log.Printf("probe %s not found in registry", name)
        // If the probe is not found in the registry, do not start it.
        return
    }

    // When we register a probe, 
    // create a new instance of the probe.
    probe := factory()
    if err := probe.Load(); err != nil {
        log.Printf("probe %s failed to load: %v", name, err)
        return
    }

    // Each probe runs in its own goroutine.
    // Send events to the same channel.
    go func() {
        if err := probe.Run(ctx, pm.dataChan); err != nil {
            log.Printf("probe %s failed: %v", name, err)
        }
    }()
    
    pm.activeProbes[name] = probe
    fmt.Printf("probe %s started\n", name)
}

// stopProbe cloes the probe and removes it from the active probes map.
func (pm *ProbeManager) stopProbe(name string) {
    if probe, exists := pm.activeProbes[name]; exists {
        probe.Close()
        delete(pm.activeProbes, name)
        fmt.Printf("probe %s stopped\n", name)
    }
}
