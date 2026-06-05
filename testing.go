package mdk

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

// TestRuntime is a concrete, in-memory implementation of Runtime designed for unit testing.
type TestRuntime struct {
	db             *gorm.DB
	bus            *TestEventBus
	workflowEngine *TestWorkflowEngine
	configs        map[string]any
	logger         *slog.Logger
	modules        map[string]Module
	mu             sync.RWMutex
}

// NewTestRuntime creates a new TestRuntime instance.
func NewTestRuntime(db *gorm.DB) *TestRuntime {
	tr := &TestRuntime{
		db:      db,
		configs: make(map[string]any),
		logger:  slog.Default(),
		modules: make(map[string]Module),
	}
	tr.bus = NewTestEventBus(tr)
	tr.workflowEngine = NewTestWorkflowEngine(tr)
	return tr
}

func (tr *TestRuntime) DB() *gorm.DB {
	return tr.db
}

func (tr *TestRuntime) Bus() EventBus {
	return tr.bus
}

func (tr *TestRuntime) Workflows() WorkflowEngine {
	return tr.workflowEngine
}

func (tr *TestRuntime) Config(key string) any {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.configs[key]
}

func (tr *TestRuntime) SetConfig(key string, val any) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.configs[key] = val
}

func (tr *TestRuntime) Logger() *slog.Logger {
	return tr.logger
}

func (tr *TestRuntime) SetLogger(l *slog.Logger) {
	tr.logger = l
}

func (tr *TestRuntime) Module(id string) (Module, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	m, ok := tr.modules[id]
	return m, ok
}

func (tr *TestRuntime) SetModule(id string, m Module) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.modules[id] = m
}

// TestEventBus is an in-memory implementation of EventBus for testing.
type TestEventBus struct {
	rt        *TestRuntime
	mu        sync.RWMutex
	handlers  map[string][]EventHandler
	Published []Event
}

func NewTestEventBus(rt *TestRuntime) *TestEventBus {
	return &TestEventBus{
		rt:       rt,
		handlers: make(map[string][]EventHandler),
	}
}

func (teb *TestEventBus) Publish(ctx context.Context, e Event) error {
	teb.mu.Lock()
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now()
	}
	teb.Published = append(teb.Published, e)
	teb.mu.Unlock()

	teb.mu.RLock()
	// Match key "namespace.type" or namespace.*
	key := e.Namespace + "." + e.Type
	handlers := append([]EventHandler{}, teb.handlers[key]...)
	wildcardHandlers := append([]EventHandler{}, teb.handlers[e.Namespace+".*"]...)
	teb.mu.RUnlock()

	for _, h := range handlers {
		_ = h(ctx, e)
	}
	for _, h := range wildcardHandlers {
		_ = h(ctx, e)
	}
	return nil
}

func (teb *TestEventBus) Subscribe(namespace, eventType string, handler EventHandler) (func(), error) {
	teb.mu.Lock()
	defer teb.mu.Unlock()
	key := namespace + "." + eventType
	teb.handlers[key] = append(teb.handlers[key], handler)
	
	return func() {
		teb.mu.Lock()
		defer teb.mu.Unlock()
		handlers := teb.handlers[key]
		for i, h := range handlers {
			// Basic clean up if unsubscribe is needed.
			// Since we can't easily compare func pointers in Go directly,
			// a simplified unsubscribe is fine for test usage.
			_ = h
			_ = i
		}
	}, nil
}

var runIDCounter int64

// TestWorkflowEngine is a simple, synchronous implementation of WorkflowEngine for unit tests.
type TestWorkflowEngine struct {
	rt        *TestRuntime
	mu        sync.RWMutex
	workflows map[string]Workflow
	handlers  map[string]StepHandler
	runs      map[string]StepStatus
	outputs   map[string]map[string]any
}

func NewTestWorkflowEngine(rt *TestRuntime) *TestWorkflowEngine {
	return &TestWorkflowEngine{
		rt:        rt,
		workflows: make(map[string]Workflow),
		handlers:  make(map[string]StepHandler),
		runs:      make(map[string]StepStatus),
		outputs:   make(map[string]map[string]any),
	}
}

func (twe *TestWorkflowEngine) Register(w Workflow) error {
	twe.mu.Lock()
	defer twe.mu.Unlock()
	twe.workflows[w.ID] = w
	return nil
}

func (twe *TestWorkflowEngine) RegisterHandler(name string, handler StepHandler) error {
	twe.mu.Lock()
	defer twe.mu.Unlock()
	twe.handlers[name] = handler
	return nil
}

func (twe *TestWorkflowEngine) Execute(ctx context.Context, workflowID string, input map[string]any) (string, error) {
	val := atomic.AddInt64(&runIDCounter, 1)
	runID := fmt.Sprintf("wf_run_%d_%d", time.Now().UnixNano(), val)
	go func() {
		_, _ = twe.ExecuteSync(context.Background(), runID, workflowID, input)
	}()
	return runID, nil
}

func (twe *TestWorkflowEngine) Status(ctx context.Context, runID string) (StepStatus, error) {
	twe.mu.RLock()
	defer twe.mu.RUnlock()
	return twe.runs[runID], nil
}

func (twe *TestWorkflowEngine) Cancel(ctx context.Context, runID string) error {
	twe.mu.Lock()
	defer twe.mu.Unlock()
	twe.runs[runID] = StepFailed
	return nil
}

func (twe *TestWorkflowEngine) ExecuteSync(ctx context.Context, runID, workflowID string, input map[string]any) (map[string]any, error) {
	twe.mu.RLock()
	wf, ok := twe.workflows[workflowID]
	twe.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("workflow not found: %s", workflowID)
	}

	twe.mu.Lock()
	twe.runs[runID] = StepRunning
	twe.mu.Unlock()

	results := make(map[string]any)
	for k, v := range input {
		results[k] = v
	}
	results["input"] = input
	results["_workflow_id"] = runID

	completed := make(map[string]bool)
	launched := make(map[string]bool)
	var history []Step

	for len(completed) < len(wf.Steps) {
		var ready []Step
		for _, step := range wf.Steps {
			if launched[step.ID] {
				continue
			}
			canRun := true
			for _, dep := range step.DependsOn {
				if !completed[dep] {
					canRun = false
					break
				}
			}
			if canRun {
				ready = append(ready, step)
			}
		}

		if len(ready) == 0 {
			twe.mu.Lock()
			twe.runs[runID] = StepFailed
			twe.mu.Unlock()
			return results, fmt.Errorf("deadlock detected or unresolved dependencies in test workflow execution")
		}

		for _, step := range ready {
			launched[step.ID] = true
			twe.mu.RLock()
			handler := step.Handler
			if handler == nil && step.Uses != "" {
				handler = twe.handlers[step.Uses]
			}
			twe.mu.RUnlock()

			if handler == nil {
				twe.mu.Lock()
				twe.runs[runID] = StepFailed
				twe.mu.Unlock()
				return results, fmt.Errorf("handler not found for step %s (uses %s)", step.ID, step.Uses)
			}

			sCtx := StepContext{
				Ctx:        ctx,
				Runtime:    twe.rt,
				WorkflowID: workflowID,
				RunID:      runID,
				StepID:     step.ID,
				Input:      results,
			}

			res := handler(sCtx)
			if res.Err != nil {
				twe.mu.Lock()
				twe.runs[runID] = StepFailed
				twe.mu.Unlock()

				// Run compensations in reverse order
				for i := len(history) - 1; i >= 0; i-- {
					hStep := history[i]
					compensate := hStep.Compensate
					if compensate == nil && hStep.Saga != nil && hStep.Saga.Uses != "" {
						twe.mu.RLock()
						h := twe.handlers[hStep.Saga.Uses]
						twe.mu.RUnlock()
						if h != nil {
							compensate = h
						}
					}
					if compensate != nil {
						sCtxComp := StepContext{
							Ctx:        ctx,
							Runtime:    twe.rt,
							WorkflowID: workflowID,
							RunID:      runID,
							StepID:     hStep.ID,
							Input:      results,
						}
						_ = compensate(sCtxComp)
					}
				}

				return results, fmt.Errorf("step %s failed: %w", step.ID, res.Err)
			}

			for k, v := range res.Output {
				results[k] = v
			}
			results[step.ID] = res.Output
			completed[step.ID] = true
			history = append(history, step)
		}
	}

	twe.mu.Lock()
	twe.runs[runID] = StepCompleted
	twe.outputs[runID] = results
	twe.mu.Unlock()

	return results, nil
}

// TestLineageData implements LineageData for testing.
type TestLineageData struct {
	ID        string
	Name      string
	State     string
	Error     string
	StartedAt time.Time
	EndedAt   *time.Time
	Events    []Event
}

func (tld TestLineageData) GetID() string            { return tld.ID }
func (tld TestLineageData) GetName() string          { return tld.Name }
func (tld TestLineageData) GetState() string         { return tld.State }
func (tld TestLineageData) GetError() string         { return tld.Error }
func (tld TestLineageData) GetStartedAt() time.Time  { return tld.StartedAt }
func (tld TestLineageData) GetEndedAt() *time.Time   { return tld.EndedAt }
func (tld TestLineageData) GetEvents() []Event       { return tld.Events }

// TestProjector implements Projector for testing.
type TestProjector struct {
	Lineages []LineageData
}

func (tp *TestProjector) ListLineages() []LineageData {
	return tp.Lineages
}

func (tp *TestProjector) QueryLineages(filter func(LineageData) bool) []LineageData {
	var out []LineageData
	for _, l := range tp.Lineages {
		if filter(l) {
			out = append(out, l)
		}
	}
	return out
}

// TestContextModule mock implementation of core.context module.
type TestContextModule struct {
	Proj Projector
}

func (tcm *TestContextModule) ID() string {
	return "core.context"
}

func (tcm *TestContextModule) Models() []any {
	return nil
}

func (tcm *TestContextModule) Routes() []Route {
	return nil
}

func (tcm *TestContextModule) Init(ctx context.Context, rt Runtime) error {
	return nil
}

func (tcm *TestContextModule) Shutdown(ctx context.Context) error {
	return nil
}

func (tcm *TestContextModule) Projector() Projector {
	return tcm.Proj
}
