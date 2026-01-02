// Package main is the main package for the agent.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bpfstack/bpfstack/internal/agent"
	"github.com/bpfstack/bpfstack/pkg/agent/core"
	"github.com/bpfstack/bpfstack/pkg/probes/iowait"
)

func main() {
	// Check if the agent is running as root.
	if os.Geteuid() != 0 {
		log.Printf("Warning: Agent is not running as root. Ensure adequate capabilities (CAP_BPF, etc.) are set.")
	}

	// Parse the command line flags.
	// Example: sudo ./agent --controller=192.168.0.5:50051 --id=worker-node-1
	// TODO: It'll be set by config file.
	controllerAddr := flag.String("controller", "localhost:50051", "Address of the controller gRPC server")
	agentID := flag.String("id", "default-agent", "Unique ID of this agent")
	flag.Parse()

	log.Printf("Starting BPFStack Agent [%s]...", *agentID)
	log.Printf("Connecting to Controller at: %s", *controllerAddr)

	// Create a new probe manager and register the iowait probe.
	pm := core.NewProbeManager()
	pm.Register("iowait_top5", iowait.New)

	// Handle the context and signals (Graceful Shutdown).
	// If Ctrl+C is pressed, the agent will be safely shut down.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n Shutting down agent...")
		// Cancel the context -> Stop all the probes and the gRPC client.
		cancel()
	}()

	// If you want to run the agent locally before connecting to the controller, you can use this code.
	// TODO: It'll be set by config file.
	log.Println("Enabling 'iowait_top5' probe by default...")
	if err := pm.Reconcile(ctx, map[string]bool{"iowait_top5": true}); err != nil {
		log.Printf("Failed to enable default probes: %v", err)
	}

	// Create a new agent client and start it.
	client := agent.NewClient(*controllerAddr, *agentID, pm, map[string]bool{"iowait_top5": false})
	
	if err := client.Start(ctx); err != nil {
		// If the context is not cancelled, print the error.
		if ctx.Err() == nil {
			log.Fatalf("AgentClient error: %v", err)
		}
	}

	log.Println("Agent stopped successfully.")
}
