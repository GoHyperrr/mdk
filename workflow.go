package mdk

import "context"

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

// Step is a single node in a workflow DAG.
type Step struct {
	ID         string
	Name       string
	DependsOn  []string    // step IDs this step waits for
	Handler    StepHandler
	Compensate StepHandler // called on rollback, may be nil
	MaxRetries int
}

// Workflow is a declarative DAG of steps.
type Workflow struct {
	ID    string
	Name  string
	Steps []Step
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
}
