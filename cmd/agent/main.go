package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bpfstack/bpfstack/pkg/agent/core"
	"github.com/bpfstack/bpfstack/pkg/probes/file_open"
	"github.com/bpfstack/bpfstack/pkg/probes/helloworld"
)

func main() {
    dataChan := make(chan core.TelemetryEvent)
    probeMgr := core.NewProbeManager(dataChan)

    // Just register the ProbeFactoryFunc to the probe manager.
    // If the probe is true in the config,
    // the probe initialized and started.
    probeMgr.Register("helloworld", helloworld.New)
    probeMgr.Register("file_open", file_open.New)
    
    // TODO: Convert to YAML config file
    currentConfig := map[string]bool{
		"file_open":   true,
		"helloworld": true,
	}
    
	ctx, cancel := context.WithCancel(context.Background())
	probeMgr.Reconcile(ctx, currentConfig)

    // consume and print telemetry events
    // TODO: Convert to output format object 
    go func() {
        for event := range dataChan {
            fmt.Printf("[%s] %s\n", event.ProbeName, event.Data)
        }
    }()

    // Wait for the context to be done
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	
	cancel()
}