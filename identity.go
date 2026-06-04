package mdk

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// ActorType represents the type of entity performing an action.
type ActorType string

const (
	ActorHuman   ActorType = "HUMAN"
	ActorAIAgent ActorType = "AI_AGENT"
	ActorSystem  ActorType = "SYSTEM"
)

// JSONMap is a custom type for map[string]string that implements GORM/SQL scanner/valuer.
type JSONMap map[string]string

func (m JSONMap) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

func (m *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	return json.Unmarshal(bytes, m)
}

// Actor represents a generic identity in the system.
type Actor struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	Type      ActorType `gorm:"index" json:"type"`
	Name      string    `json:"name"`
	Metadata  JSONMap   `gorm:"type:text" json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type contextKey struct{}
var actorKey = contextKey{}

// WithActor stores the Actor in the context.
func WithActor(ctx context.Context, actor *Actor) context.Context {
	return context.WithValue(ctx, actorKey, actor)
}

// ActorFromContext retrieves the Actor from the context.
func ActorFromContext(ctx context.Context) (*Actor, bool) {
	actor, ok := ctx.Value(actorKey).(*Actor)
	return actor, ok
}

// TokenValidator defines the interface for validating authentication tokens.
type TokenValidator interface {
	ValidateToken(ctx context.Context, token string) (*Actor, error)
}

