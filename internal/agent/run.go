// Package agent is for the agent main logic.
package agent

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/bpfstack/bpfstack/pkg/agent/core"
)

// Run executes the agent main logic.
func Run(factories map[string]core.ProbeFactoryFunc) error {
	// If sigint(ctrl+c) or sigterm(kill) is received,
	// the context is done and the program should exit.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dataChan := make(chan core.TelemetryEvent, 100)
	errChan := make(chan error, 100)

	fmt.Println("Starting the probe manager")
	probeMgr := core.NewProbeManager(dataChan, errChan)

    config := make(map[string]bool)
    for name, factory := range factories {
        probeMgr.Register(name, factory)
        // If that probe is in the config,
        // it always gets loaded.
        
        // TODO: Add a way to disable a probe
        // in the first time the agent starts.
        config[name] = true
    }

	// Receive the data in a separate goroutine.
	// If an error occurs during reconcile,
	// the error is sent to the errChan.
	go func() {
		for {
			select {
			case event := <-dataChan:
				fmt.Printf("[%s] %s\n", event.ProbeName, event.Data)
			case err := <-errChan:
				fmt.Printf("Error: %v\n", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Reconcile the probes.
	probeMgr.Reconcile(ctx, config)

	fmt.Println("Agent started")

	// Wait for the context to be done.
	<-ctx.Done()

	fmt.Println("Shutting down the agent")
	probeMgr.Shutdown()

	return nil
}
