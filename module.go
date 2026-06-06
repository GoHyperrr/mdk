package mdk

import (
	"context"
	"net/http"
)

// Module is the interface every hyperrr module must implement.
// Register your module in an init() function using mdk.Register().
type Module interface {
	// ID returns a unique, stable identifier for this module e.g. "order", "auth.emailpass"
	ID() string

	// Models returns GORM model structs for automigration.
	// Return nil if the module has no DB models.
	Models() []any

	// Routes returns HTTP handlers to mount on the server.
	// Return nil if the module exposes no routes.
	Routes() []Route

	// Init is called once when the runtime starts, after DB migration.
	Init(ctx context.Context, rt Runtime) error

	// Shutdown is called on graceful shutdown.
	Shutdown(ctx context.Context) error
}

// Route is an HTTP route a module wants to register.
type Route struct {
	Method  string       // "GET", "POST", etc. Empty means all methods.
	Pattern string       // e.g. "/orders", "/orders/{id}"
	Handler http.Handler
}

// Factory is a constructor function for a Module.
type Factory func() Module

// HTMLUIProvider can be implemented by modules that want to expose a dashboard UI to MCP.
type HTMLUIProvider interface {
	RenderHTML(ctx context.Context) (title, html string, err error)
}

