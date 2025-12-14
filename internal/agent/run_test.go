// agent is for the agent main logic.
package agent

import (
	"context"
	"syscall"
	"testing"
	"time"

	"github.com/bpfstack/bpfstack/pkg/agent/core"
)


// Mock Probe definition (similar to core package's test)
type mockProbe struct{}

// Name returns the name of the probe.
func (m *mockProbe) Name() string { return "mock_probe" }

// Load does nothing.
func (m *mockProbe) Load() error  { return nil } 

// Run sends fake data every 10ms.
func (m *mockProbe) Run(ctx context.Context, outCh chan<- core.TelemetryEvent) error {

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			outCh <- core.TelemetryEvent{
				ProbeName: "mock_probe",
				Data:      "mock data",
			}
		}
	}
}

// Close does nothing.
func (m *mockProbe) Close() error { return nil }

// TestRun_GracefulShutdown tests the graceful shutdown of the agent.
func TestRun_GracefulShutdown(t *testing.T) {
	// Prepare Mock Factory
	factories := map[string]core.ProbeFactoryFunc{
		"mock_probe": func() core.Prober {
			return &mockProbe{}
		},
	}

	// Run the agent in a separate goroutine (blocking so we can send the signal)
	errCh := make(chan error)
	go func() {
		err := Run(factories)
		errCh <- err
	}()

	// Wait for the agent to start
	time.Sleep(100 * time.Millisecond)

	// Send SIGINT to self to simulate Ctrl+C
	t.Log("Sending SIGINT to self...")
	err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	if err != nil {
		t.Fatalf("Failed to send signal: %v", err)
	}

	// Check if the agent shut down gracefully
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Run() returned an error: %v", err)
		}
		t.Log("Agent shut down gracefully")
	case <-time.After(2 * time.Second):
		t.Error("Agent did not shut down within timeout")
	}
}