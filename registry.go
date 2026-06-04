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

// CLICommand represents a dynamic subcommand registered by a module.
type CLICommand struct {
	Group       string   // Command group: "auth", "commerce", etc.
	Name        string   // Subcommand name: "apikey", "product", etc.
	Aliases     []string // Alternative names
	Short       string   // One-line description
	Long        string   // Detailed description (shown in --help)
	Usage       string   // Args pattern: "generate", "<email> <password>"
	Run         func(rt Runtime, args []string) error
	NeedsDB     bool     // If true, auto-connect DB before Run
	NeedsServer bool     // If true, requires running server (validates connectivity)
}

var (
	commandsMu sync.RWMutex
	commands   = make(map[string]CLICommand)
)

// RegisterCommand adds a dynamic CLI subcommand to the global mdk registry.
func RegisterCommand(cmd CLICommand) {
	commandsMu.Lock()
	defer commandsMu.Unlock()
	key := cmd.Name
	if cmd.Group != "" {
		key = cmd.Group + "/" + cmd.Name
	}
	commands[key] = cmd
}

// Commands returns a list of all registered custom CLI commands.
func Commands() []CLICommand {
	commandsMu.RLock()
	defer commandsMu.RUnlock()
	res := make([]CLICommand, 0, len(commands))
	for _, cmd := range commands {
		res = append(res, cmd)
	}
	return res
}

