package mdk

import (
	"context"
	"time"
)

// StepStatus represents the current state of a workflow step.
type StepStatus string

const (
	StepPending   StepStatus = "pending"
	StepRunning   StepStatus = "running"
	StepCompleted StepStatus = "completed"
	StepFailed    StepStatus = "failed"
	StepRetrying  StepStatus = "retrying"
	StepSkipped   StepStatus = "skipped"
)

// StepContext is passed to every step handler and compensation handler.
type StepContext struct {
	Ctx        context.Context
	Runtime    Runtime
	WorkflowID string
	RunID      string
	StepID     string
	Input      map[string]any // output from previous steps, merged
}

// StepResult is returned by a step handler.
type StepResult struct {
	Output map[string]any // merged into input for downstream steps
	Err    error
}

// StepHandler is the function signature for a workflow step.
type StepHandler func(ctx StepContext) StepResult

// Saga defines the compensation step on rollback.
type Saga struct {
	Uses       string     `json:"uses" yaml:"uses"`
	IsCritical bool       `json:"is_critical" yaml:"is_critical"`
}

// Step is a single node in a workflow DAG.
type Step struct {
	ID         string      `json:"id" yaml:"id"`
	Name       string      `json:"name" yaml:"name"`
	DependsOn  []string    `json:"depends_on" yaml:"depends_on"` // step IDs this step waits for
	Handler    StepHandler `json:"-" yaml:"-"`
	Compensate StepHandler `json:"-" yaml:"-"`
	MaxRetries int         `json:"max_retries" yaml:"max_retries"`
	Uses       string      `json:"uses" yaml:"uses"`      // For backwards compatibility and string-based resolution
	Saga       *Saga       `json:"saga,omitempty" yaml:"saga,omitempty"`       // For backwards compatibility saga rollback
}

// Workflow is a declarative DAG of steps.
type Workflow struct {
	ID          string         `json:"id" yaml:"id"`
	Name        string         `json:"name" yaml:"name"`
	Description string         `json:"description,omitempty" yaml:"description,omitempty"`
	ExposeToAI  bool           `json:"expose_to_ai" yaml:"expose_to_ai"`
	InputSchema map[string]any `json:"input_schema,omitempty" yaml:"input_schema,omitempty"`
	Steps       []Step         `json:"steps" yaml:"steps"`
}

// WorkflowEngine registers and executes workflow DAGs.
type WorkflowEngine interface {
	// Register makes a workflow available for execution.
	Register(w Workflow) error

	// Execute starts a workflow run and returns a run ID.
	Execute(ctx context.Context, workflowID string, input map[string]any) (runID string, err error)

	// Status returns the status of a specific run.
	Status(ctx context.Context, runID string) (StepStatus, error)

	// Cancel requests cancellation of a running workflow.
	Cancel(ctx context.Context, runID string) error

	// RegisterHandler registers a named step handler for string-based step resolution.
	RegisterHandler(name string, handler StepHandler) error
}

// LineageData defines the minimal interface for accessing workflow execution data.
type LineageData interface {
	GetID() string
	GetName() string
	GetState() string
	GetError() string
	GetStartedAt() time.Time
	GetEndedAt() *time.Time
	GetEvents() []Event
}

// Projector defines the interface for querying execution lineages.
type Projector interface {
	ListLineages() []LineageData
	QueryLineages(filter func(LineageData) bool) []LineageData
}

// ProjectorProvider is implemented by modules that expose an execution lineage Projector.
type ProjectorProvider interface {
	Projector() Projector
}
