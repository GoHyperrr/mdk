package mdk

import (
	"fmt"
	"sync"
)

var global = &registry{
	factories: make(map[string]Factory),
}

type registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

// Register adds a module factory to the global registry.
// Call this inside an init() function in your module package.
//
// Example:
//
//    func init() {
//        mdk.Register(func() mdk.Module { return &MyModule{} })
//    }
func Register(factory Factory) {
	m := factory()
	id := m.ID()

	global.mu.Lock()
	defer global.mu.Unlock()

	if _, exists := global.factories[id]; exists {
		panic(fmt.Sprintf("mdk: module %q already registered", id))
	}
	global.factories[id] = factory
}

// Registered returns a snapshot of all registered module factories.
func Registered() map[string]Factory {
	global.mu.RLock()
	defer global.mu.RUnlock()

	out := make(map[string]Factory, len(global.factories))
	for k, v := range global.factories {
		out[k] = v
	}
	return out
}
