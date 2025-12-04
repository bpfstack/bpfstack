package action

import (
	"sync"
	"time"

	"github.com/bpfstack/bpfstack/pkg/logger"
)

// PrintAction is an example action that prints messages periodically
type PrintAction struct {
	*BaseAction
	message  string
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

// NewPrintAction creates a new print action
func NewPrintAction(name, message string, interval time.Duration) *PrintAction {
	return &PrintAction{
		BaseAction: NewBaseAction(name),
		message:    message,
		interval:   interval,
	}
}

func (p *PrintAction) Init() error {
	p.stopCh = make(chan struct{})
	p.LogInfo("initialized")
	return nil
}

func (p *PrintAction) Start() error {
	p.LogInfo("starting")

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.LogInfo(p.message)
			case <-p.stopCh:
				return
			}
		}
	}()

	return nil
}

func (p *PrintAction) Stop() error {
	p.LogInfo("stopping")
	close(p.stopCh)
	p.wg.Wait()
	p.LogInfo("stopped")
	return nil
}

// VMExitAction is an example action that simulates VM exit tracking
type VMExitAction struct {
	*BaseAction
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func NewVMExitAction() *VMExitAction {
	return &VMExitAction{
		BaseAction: NewBaseAction("vmexit"),
	}
}

func (v *VMExitAction) Init() error {
	v.stopCh = make(chan struct{})
	v.LogInfo("initialized VM exit tracker")
	return nil
}

func (v *VMExitAction) Start() error {
	v.LogInfo("starting VM exit tracking")

	v.wg.Add(1)
	go func() {
		defer v.wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		counter := 0
		for {
			select {
			case <-ticker.C:
				counter++
				v.LogInfo("tracked VM exits", logger.Fields{
					"count":     counter * 1234,
					"exit_type": "EPT_VIOLATION",
				})
			case <-v.stopCh:
				return
			}
		}
	}()

	return nil
}

func (v *VMExitAction) Stop() error {
	v.LogInfo("stopping VM exit tracking")
	close(v.stopCh)
	v.wg.Wait()
	v.LogInfo("stopped")
	return nil
}

// RegisterDefaultActions registers all default example actions
func RegisterDefaultActions(registry *Registry) {
	if err := registry.Register(NewVMExitAction()); err != nil {
		// Log error but don't fail - allow partial registration
		_ = err
	}
}
