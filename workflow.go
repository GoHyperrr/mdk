package mdk

import (
	"context"
	"fmt"
	"sync"
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
	ID         string   `json:"id" yaml:"id"`
	Name       string   `json:"name" yaml:"name"`
	DependsOn  []string `json:"depends_on" yaml:"depends_on"` // step IDs this step waits for
	MaxRetries int      `json:"max_retries" yaml:"max_retries"`
	Uses       string   `json:"uses" yaml:"uses"` // String-based resolution key
	Saga       *Saga    `json:"saga,omitempty" yaml:"saga,omitempty"`
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

// Validate checks if the workflow definition is valid.
func (w *Workflow) Validate() error {
	if w.ID == "" && w.Name == "" {
		return fmt.Errorf("workflow ID and Name cannot both be empty")
	}

	wfID := w.ID
	if wfID == "" {
		wfID = w.Name
	}

	stepIDs := make(map[string]bool)
	for _, step := range w.Steps {
		if step.ID == "" {
			return fmt.Errorf("step ID cannot be empty in workflow %s", wfID)
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("duplicate step ID %q in workflow %s", step.ID, wfID)
		}
		stepIDs[step.ID] = true
		if step.Uses == "" {
			return fmt.Errorf("step %q must specify a 'uses' handler in workflow %s", step.ID, wfID)
		}
		for _, dep := range step.DependsOn {
			if dep == step.ID {
				return fmt.Errorf("step %q cannot depend on itself in workflow %s", step.ID, wfID)
			}
		}
	}
	for _, step := range w.Steps {
		for _, dep := range step.DependsOn {
			if !stepIDs[dep] {
				return fmt.Errorf("step %q depends on non-existent step %q in workflow %s", step.ID, dep, wfID)
			}
		}
	}
	return nil
}

// WorkflowStatus represents the execution status and history details of a workflow.
type WorkflowStatus struct {
	State     StepStatus        `json:"state"`
	Steps     map[string]string `json:"steps,omitempty"` // stepID -> state
	Error     string            `json:"error,omitempty"`
	StartedAt time.Time         `json:"started_at,omitempty"`
	EndedAt   *time.Time        `json:"ended_at,omitempty"`
}

// WorkflowEngine registers and executes workflow DAGs.
type WorkflowEngine interface {
	// Register makes a workflow available for execution.
	Register(w Workflow) error

	// Execute starts a workflow run and returns a run ID.
	Execute(ctx context.Context, workflowID string, input map[string]any) (runID string, err error)

	// Status returns the status of a specific run.
	Status(ctx context.Context, runID string) (WorkflowStatus, error)

	// Cancel requests cancellation of a running workflow.
	Cancel(ctx context.Context, runID string) error

	// RegisterHandler registers a named step handler for string-based step resolution.
	RegisterHandler(name string, handler StepHandler) error
}

var (
	lineageEventsMu sync.RWMutex
	lineageEvents   = map[string]bool{
		"workflow.started":        true,
		"workflow.step_started":   true,
		"workflow.step_completed": true,
		"workflow.step_failed":    true,
		"workflow.step_retrying":   true,
		"workflow.step_fallback":   true,
		"workflow.step.started":   true,
		"workflow.step.completed": true,
		"workflow.step.failed":    true,
		"workflow.step.retrying":  true,
		"workflow.step.fallback":  true,
		"workflow.waiting_human":   true,
		"workflow.completed":      true,
		"workflow.failed":         true,
		"order.created":           true,
		"order.paid":              true,
	}
	onRegisterLineageEvent func(string)
)

// RegisterLineageEvent registers an event type to be tracked by the context engine.
func RegisterLineageEvent(eventType string) {
	lineageEventsMu.Lock()
	lineageEvents[eventType] = true
	cb := onRegisterLineageEvent
	lineageEventsMu.Unlock()
	if cb != nil {
		cb(eventType)
	}
}

// GetLineageEvents returns all registered lineage events.
func GetLineageEvents() []string {
	lineageEventsMu.RLock()
	defer lineageEventsMu.RUnlock()
	events := make([]string, 0, len(lineageEvents))
	for e := range lineageEvents {
		events = append(events, e)
	}
	return events
}

// OnRegisterLineageEvent sets a callback to execute when a new lineage event is registered.
func OnRegisterLineageEvent(cb func(string)) {
	lineageEventsMu.Lock()
	onRegisterLineageEvent = cb
	lineageEventsMu.Unlock()
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
