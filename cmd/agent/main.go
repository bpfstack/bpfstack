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
    if err := run(); err != nil {
        fmt.Printf("Error: %v", err)
        os.Exit(1)
    }
}
func run() error {
    // if sigint(ctrl+c) or sigterm(kill) is received,
    // the context is done and the program should exit.
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    dataChan := make(chan core.TelemetryEvent, 100) 
    errChan := make(chan error, 100)
    
    fmt.Println("Starting the probe manager")
    probeMgr := core.NewProbeManager(dataChan, errChan)
    probeMgr.Register("helloworld", helloworld.New)
    probeMgr.Register("file_open", file_open.New)

    currentConfig := map[string]bool{
        "file_open":   true,
        "helloworld": true,
    }

    // receive the data in a separate goroutine.
    // if an error occurs during reconcile,
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

    // reconcile the probes
    probeMgr.Reconcile(ctx, currentConfig)
    
    fmt.Println("Agent started") 

    // wait for the context to be done
    <-ctx.Done()
    
    fmt.Println("Shutting down the agent")
    probeMgr.Shutdown()
    
    return nil
}