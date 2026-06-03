package mdk

import (
	"log/slog"

	"gorm.io/gorm"
)

// Runtime is provided by hyperrr to every module at Init time.
// Modules use this to access shared infrastructure.
type Runtime interface {
	// DB returns the shared GORM database connection.
	DB() *gorm.DB

	// Bus returns the event bus for pub/sub.
	Bus() EventBus

	// Workflows returns the workflow engine.
	Workflows() WorkflowEngine

	// Config returns a config value by key.
	Config(key string) any

	// Logger returns the structured logger.
	Logger() *slog.Logger
}
