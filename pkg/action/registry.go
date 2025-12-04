package action

import (
	"fmt"
	"sync"
)

// Registry holds all registered actions
type Registry struct {
	actions map[string]ActionInterface
	mu      sync.RWMutex
}

// NewRegistry creates a new action registry
func NewRegistry() *Registry {
	return &Registry{
		actions: make(map[string]ActionInterface),
	}
}

// Register adds an action to the registry
func (r *Registry) Register(action ActionInterface) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := action.Name()
	if _, exists := r.actions[name]; exists {
		return fmt.Errorf("action %q already registered", name)
	}

	r.actions[name] = action
	return nil
}

// Get returns an action by name
func (r *Registry) Get(name string) (ActionInterface, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	action, exists := r.actions[name]
	return action, exists
}

// GetAll returns all registered actions
func (r *Registry) GetAll() map[string]ActionInterface {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]ActionInterface)
	for k, v := range r.actions {
		result[k] = v
	}
	return result
}

// Names returns all registered action names
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.actions))
	for name := range r.actions {
		names = append(names, name)
	}
	return names
}
