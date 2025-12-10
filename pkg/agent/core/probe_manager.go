package core

import (
	"context"
	"fmt"
	"sync"
)

// ProbeFactoryFunc is the function type for the probe factory.
type ProbeFactoryFunc func() Probe

// ProbeManager is the manager for the probes.
type ProbeManager struct {
    // dataChan is the channel where the probe will send the telemetry events.
    dataChan chan<- TelemetryEvent
    // errorChan is the channel where the probe will send the errors.
    errorChan chan<- error
    // registry is the collection of available probes.
    registry map[string]ProbeFactoryFunc
    // activeProbes is the collection of active probes.
    activeProbes map[string]Probe
    // mu is the mutex for the activeProbes map.
    // cancelFuncs is the collection of cancel functions for the probes.
    cancelFuncs map[string]context.CancelFunc
    mu sync.RWMutex
}

func NewProbeManager(ch chan<- TelemetryEvent, errChan chan<- error) *ProbeManager {
	return &ProbeManager{
		dataChan:     ch,
		errorChan:    errChan,
		registry:     make(map[string]ProbeFactoryFunc),
		activeProbes: make(map[string]Probe),
		cancelFuncs:  make(map[string]context.CancelFunc),
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
            if err := pm.startProbe(ctx, name); err != nil {
                pm.errorChan <- FailedToStartProbe(name, err)
            }
        }

        // if the probe is running and should not be run,
        // stop it
        if !shouldRun && isRunning {
            if err := pm.stopProbe(name); err != nil {
                pm.errorChan <- FailedToStopProbe(name, err)
            }
        }
   }
}

// startProbe is the function that starts a probe.
func (pm *ProbeManager) startProbe(ctx context.Context, name string)error{
    factory, exists := pm.registry[name]
    if !exists {
        // If the probe is not found in the registry, return an error.
        return ProbeNotFound(name)
    }

    // When we register a probe, 
    // create a new instance of the probe.
    probe := factory()
    if err := probe.Load(); err != nil {
        return ProbeLoadFailed(name, err)
    }

    // Make context that comes from parent context 
    probeCtx, cancel := context.WithCancel(ctx)

    pm.cancelFuncs[name] = cancel
    pm.activeProbes[name] = probe

    // Each probe runs in its own goroutine.
    // Send events to the same channel.
    go func() {
        defer func() {
            if r := recover(); r != nil {
                pm.errorChan <- ProbePanic(name, r)
            }
        }()
        if err := probe.Run(probeCtx, pm.dataChan); err != nil {
            // report error only if the context is not canceled.
            if probeCtx.Err() != nil {
                pm.errorChan <- ProbeContextCanceled(name)
            }
        }
    }()
    
    fmt.Printf("probe %s started\n", name)
    return nil
}

// stopProbe cloes the probe and removes it from the active probes map.
func (pm *ProbeManager) stopProbe(name string)error{
    if cancel, ok := pm.cancelFuncs[name]; ok {
        cancel()
        delete(pm.cancelFuncs, name)
    }

    if probe, exists := pm.activeProbes[name]; exists {
        if err := probe.Close(); err != nil {
            return FailedToCloseProbe(name, err)
        }

        delete(pm.activeProbes, name)
        fmt.Printf("probe %s stopped\n", name)
    }
    return nil
}

func (pm *ProbeManager) Shutdown() {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    fmt.Printf("Shutting down the probe manager")
    
    // Stop all the active probes.
    for name := range pm.activeProbes {
        pm.stopProbe(name)
    }
}