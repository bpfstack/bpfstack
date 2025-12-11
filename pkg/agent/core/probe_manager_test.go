package core

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// mockProbe is a fake probe for testing.
type mockProbe struct {
    // mu is a mutex to protect the state.
	mu          sync.Mutex
    // isRunning is whether the probe is running.
	isRunning   bool
    // isClosed is whether the probe is closed.
	isClosed    bool
	// shouldError is if true, Run will return error immediately.
	shouldError bool 
}

// Name is a fake implementation of the Name method.
func (m *mockProbe) Name() string {
	return "mock_probe"
}

// Load is a fake implementation of the Load method.
func (m *mockProbe) Load() error {
	return nil
}

// Run is a fake implementation of the Run method
// It simulates sending data to the output channel 
// and returning an error if shouldError is true.
func (m *mockProbe) Run(ctx context.Context, outCh chan<- TelemetryEvent) error {
	m.mu.Lock()
	if m.shouldError {
		m.mu.Unlock()
		return errors.New("simulated run error")
	}
	m.isRunning = true
	m.mu.Unlock()

	// Simulate sending data.
	select {
	case outCh <- TelemetryEvent{ProbeName: "mock", Data: "test-data"}:
	case <-ctx.Done():
	}

	// Wait until context is cancelled.
	<-ctx.Done()
	return nil
}

// Close is a fake implementation of the Close method.
// It sets the isRunning and isClosed fields to false.
func (m *mockProbe) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.isRunning = false
	m.isClosed = true
	return nil
}

// Running is a fake implementation of the Running method.
func (m *mockProbe) Running() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isRunning
}

// Closed is a fake implementation of the Closed method.
func (m *mockProbe) Closed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.isClosed
}

// TestProbeManager_RegisterAndReconcile tests the RegisterAndReconcile method.
// It tests that the probe is registered and reconciled correctly.
func TestProbeManager_RegisterAndReconcile(t *testing.T) {
	// NOTE: Use buffered channels to prevent blocking in tests.
	dataChan := make(chan TelemetryEvent, 10)
	errChan := make(chan error, 10)

	pm := NewProbeManager(dataChan, errChan)

	var capturedProbe *mockProbe
	pm.Register("mock_probe", func() Prober {
		p := &mockProbe{}
        // Capture reference to verify state later.
		capturedProbe = p 
		return p
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Log("Step 1: Enabling probe...")
	config := map[string]bool{"mock_probe": true}
	pm.Reconcile(ctx, config)

	// Allow goroutine time to start.
	time.Sleep(50 * time.Millisecond)

	if capturedProbe == nil {
		t.Fatal("Probe factory was not called")
	}
	if !capturedProbe.Running() {
		t.Error("Probe should be running, but it is not")
	}

	select {
	case event := <-dataChan:
		if event.Data != "test-data" {
			t.Errorf("Expected 'test-data', got %s", event.Data)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Timed out waiting for data from probe")
	}

	t.Log("Step 2: Disabling probe...")
	config["mock_probe"] = false
	pm.Reconcile(ctx, config)

	// Allow goroutine time to stop.
	time.Sleep(50 * time.Millisecond)

	if !capturedProbe.Closed() {
		t.Error("Probe should be closed, but it is not")
	}
}

func TestProbeManager_Idempotency(t *testing.T) {
	// Ensure starting an already running probe doesn't crash or restart it
	dataChan := make(chan TelemetryEvent, 10)
	errChan := make(chan error, 10)
	pm := NewProbeManager(dataChan, errChan)

	creationCount := 0
	pm.Register("mock_probe", func() Prober {
		creationCount++
		return &mockProbe{}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()


	pm.Reconcile(ctx, map[string]bool{"mock_probe": true})
	time.Sleep(10 * time.Millisecond)

	pm.Reconcile(ctx, map[string]bool{"mock_probe": true})
	time.Sleep(10 * time.Millisecond)

	if creationCount != 1 {
		t.Errorf("Expected probe to be created once, but was created %d times", creationCount)
	}
}

func TestProbeManager_Shutdown(t *testing.T) {
	dataChan := make(chan TelemetryEvent, 10)
	errChan := make(chan error, 10)
	pm := NewProbeManager(dataChan, errChan)

	var p1 *mockProbe
	pm.Register("probe1", func() Prober {
		p1 = &mockProbe{}
		return p1
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pm.Reconcile(ctx, map[string]bool{"probe1": true})
	time.Sleep(10 * time.Millisecond)

	if !p1.Running() {
		t.Fatal("Probe 1 failed to start")
	}

	pm.Shutdown()
	time.Sleep(10 * time.Millisecond)

	if !p1.Closed() {
		t.Error("Probe 1 should be closed after Shutdown()")
	}
}