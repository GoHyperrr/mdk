package mdk

import (
	"context"
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
)

// EventBusCloser extends EventBus to support graceful shutdowns.
type EventBusCloser interface {
	EventBus
	Close() error
}

// Locker defines the interface for distributed locking.
type Locker interface {
	Acquire(ctx context.Context, key string, ttl time.Duration, timeout time.Duration) (bool, error)
	Release(ctx context.Context, key string) error
	Close() error
}

var (
	ErrLockAcquisitionTimeout = errors.New("lock acquisition timed out")
	ErrLockNotHeld           = errors.New("lock not held")
)

type lockContextKey string

const LockOwnerKey lockContextKey = "lock_owner"

// StateStore defines the interface for checkpointing workflow states.
type StateStore interface {
	SaveState(ctx context.Context, execID string, stepID string, state string) error
	GetState(ctx context.Context, execID string) (map[string]string, error)
	InitializeExecution(ctx context.Context, execID string, input []byte) error
	SaveInput(ctx context.Context, execID string, input []byte) error
	GetInput(ctx context.Context, execID string) ([]byte, error)
	SetTTL(ctx context.Context, execID string, ttl time.Duration) error
	SaveStepOutput(ctx context.Context, execID string, stepID string, output []byte) error
	GetStepOutput(ctx context.Context, execID string, stepID string) ([]byte, error)
	ListExecutions(ctx context.Context, state string) ([]string, error)
	RecordEventEmitted(ctx context.Context, execID string, eventType string) error
	IsEventEmitted(ctx context.Context, execID string, eventType string) (bool, error)
}

// DialectProvider constructor signature for database dialects.
type DialectProvider func(dsn string) gorm.Dialector

// BusProvider constructor signature for event buses.
type BusProvider func(url string) (EventBusCloser, error)

// LockerProvider constructor signature for lockers.
type LockerProvider func(url string, bucketOrPrefix string) (Locker, error)

// StoreProvider constructor signature for workflow state stores.
type StoreProvider func(url string, bucketOrPrefix string) (StateStore, error)

var (
	dialectsMu     sync.RWMutex
	dialects       = make(map[string]DialectProvider)
	busProvidersMu sync.RWMutex
	busProviders   = make(map[string]BusProvider)
	lockersMu      sync.RWMutex
	lockers        = make(map[string]LockerProvider)
	storesMu       sync.RWMutex
	stores         = make(map[string]StoreProvider)
)

// RegisterDialect registers a database dialect provider.
func RegisterDialect(name string, provider DialectProvider) {
	dialectsMu.Lock()
	defer dialectsMu.Unlock()
	dialects[name] = provider
}

// GetDialect retrieves a database dialect provider.
func GetDialect(name string) (DialectProvider, bool) {
	dialectsMu.RLock()
	defer dialectsMu.RUnlock()
	d, ok := dialects[name]
	return d, ok
}

// RegisterEventBusProvider registers an event bus provider.
func RegisterEventBusProvider(name string, provider BusProvider) {
	busProvidersMu.Lock()
	defer busProvidersMu.Unlock()
	busProviders[name] = provider
}

// GetEventBusProvider retrieves an event bus provider.
func GetEventBusProvider(name string) (BusProvider, bool) {
	busProvidersMu.RLock()
	defer busProvidersMu.RUnlock()
	bp, ok := busProviders[name]
	return bp, ok
}

// RegisterLocker registers a lock manager provider.
func RegisterLocker(name string, provider LockerProvider) {
	lockersMu.Lock()
	defer lockersMu.Unlock()
	lockers[name] = provider
}

// GetLocker retrieves a lock manager provider.
func GetLocker(name string) (LockerProvider, bool) {
	lockersMu.RLock()
	defer lockersMu.RUnlock()
	l, ok := lockers[name]
	return l, ok
}

// RegisterStateStore registers a state store provider.
func RegisterStateStore(name string, provider StoreProvider) {
	storesMu.Lock()
	defer storesMu.Unlock()
	stores[name] = provider
}

// GetStateStore retrieves a state store provider.
func GetStateStore(name string) (StoreProvider, bool) {
	storesMu.RLock()
	defer storesMu.RUnlock()
	s, ok := stores[name]
	return s, ok
}
