package mdktest

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GoHyperrr/mdk"
	"gorm.io/gorm"
)

// TestRuntime is a concrete, in-memory implementation of Runtime designed for unit testing.
type TestRuntime struct {
	db             *gorm.DB
	bus            *TestEventBus
	workflowEngine *TestWorkflowEngine
	configs        map[string]any
	logger         *slog.Logger
	modules        map[string]mdk.Module
	mu             sync.RWMutex
}

// NewTestRuntime creates a new TestRuntime instance.
func NewTestRuntime(db *gorm.DB) *TestRuntime {
	tr := &TestRuntime{
		db:      db,
		configs: make(map[string]any),
		logger:  slog.Default(),
		modules: make(map[string]mdk.Module),
	}
	tr.bus = NewTestEventBus(tr)
	tr.workflowEngine = NewTestWorkflowEngine(tr)
	return tr
}

func (tr *TestRuntime) DB() *gorm.DB {
	return tr.db
}

func (tr *TestRuntime) Bus() mdk.EventBus {
	return tr.bus
}

func (tr *TestRuntime) Workflows() mdk.WorkflowEngine {
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

func (tr *TestRuntime) Module(id string) (mdk.Module, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	m, ok := tr.modules[id]
	return m, ok
}

func (tr *TestRuntime) SetModule(id string, m mdk.Module) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.modules[id] = m
}

// TestEventBus is an in-memory implementation of EventBus for testing.
type TestEventBus struct {
	rt        *TestRuntime
	mu        sync.RWMutex
	handlers  map[string][]mdk.EventHandler
	Published []mdk.Event
}

func NewTestEventBus(rt *TestRuntime) *TestEventBus {
	return &TestEventBus{
		rt:       rt,
		handlers: make(map[string][]mdk.EventHandler),
	}
}

func (teb *TestEventBus) Publish(ctx context.Context, e mdk.Event) error {
	teb.mu.Lock()
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now()
	}
	teb.Published = append(teb.Published, e)
	teb.mu.Unlock()

	teb.mu.RLock()
	key := e.Namespace + "." + e.Type
	handlers := append([]mdk.EventHandler{}, teb.handlers[key]...)
	wildcardHandlers := append([]mdk.EventHandler{}, teb.handlers[e.Namespace+".*"]...)
	teb.mu.RUnlock()

	for _, h := range handlers {
		_ = h(ctx, e)
	}
	for _, h := range wildcardHandlers {
		_ = h(ctx, e)
	}
	return nil
}

func (teb *TestEventBus) Subscribe(namespace, eventType string, handler mdk.EventHandler) (func(), error) {
	teb.mu.Lock()
	defer teb.mu.Unlock()
	key := namespace + "." + eventType
	teb.handlers[key] = append(teb.handlers[key], handler)
	
	return func() {
		teb.mu.Lock()
		defer teb.mu.Unlock()
		handlers := teb.handlers[key]
		for i, h := range handlers {
			if reflect.ValueOf(h).Pointer() == reflect.ValueOf(handler).Pointer() {
				teb.handlers[key] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}, nil
}

var runIDCounter int64

// TestWorkflowEngine is a simple, synchronous implementation of WorkflowEngine for unit tests.
type TestWorkflowEngine struct {
	rt        *TestRuntime
	mu        sync.RWMutex
	workflows map[string]mdk.Workflow
	handlers  map[string]mdk.StepHandler
	runs      map[string]mdk.WorkflowStatus
	outputs   map[string]map[string]any
}

func NewTestWorkflowEngine(rt *TestRuntime) *TestWorkflowEngine {
	return &TestWorkflowEngine{
		rt:        rt,
		workflows: make(map[string]mdk.Workflow),
		handlers:  make(map[string]mdk.StepHandler),
		runs:      make(map[string]mdk.WorkflowStatus),
		outputs:   make(map[string]map[string]any),
	}
}

func (twe *TestWorkflowEngine) Register(w mdk.Workflow) error {
	twe.mu.Lock()
	defer twe.mu.Unlock()
	twe.workflows[w.ID] = w
	return nil
}

func (twe *TestWorkflowEngine) RegisterHandler(name string, handler mdk.StepHandler) error {
	twe.mu.Lock()
	defer twe.mu.Unlock()
	twe.handlers[name] = handler
	return nil
}

func (twe *TestWorkflowEngine) Execute(ctx context.Context, workflowID string, input map[string]any) (string, error) {
	val := atomic.AddInt64(&runIDCounter, 1)
	runID := fmt.Sprintf("wf_run_%d_%d", time.Now().UnixNano(), val)
	go func() {
		_, _ = twe.ExecuteSync(ctx, runID, workflowID, input)
	}()
	return runID, nil
}

func (twe *TestWorkflowEngine) Status(ctx context.Context, runID string) (mdk.WorkflowStatus, error) {
	twe.mu.RLock()
	defer twe.mu.RUnlock()
	return twe.runs[runID], nil
}

func (twe *TestWorkflowEngine) Cancel(ctx context.Context, runID string) error {
	twe.mu.Lock()
	defer twe.mu.Unlock()
	run := twe.runs[runID]
	run.State = mdk.StepFailed
	run.Error = "cancelled"
	twe.runs[runID] = run
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
	twe.runs[runID] = mdk.WorkflowStatus{State: mdk.StepRunning, StartedAt: time.Now()}
	twe.mu.Unlock()

	results := make(map[string]any)
	for k, v := range input {
		results[k] = v
	}
	results["input"] = input
	results["_workflow_id"] = runID

	completed := make(map[string]bool)
	launched := make(map[string]bool)
	var history []mdk.Step

	for len(completed) < len(wf.Steps) {
		var ready []mdk.Step
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
			twe.runs[runID] = mdk.WorkflowStatus{
				State: mdk.StepFailed,
				Error: "deadlock detected or unresolved dependencies in test workflow execution",
			}
			twe.mu.Unlock()
			return results, fmt.Errorf("deadlock detected or unresolved dependencies in test workflow execution")
		}

		for _, step := range ready {
			launched[step.ID] = true
			twe.mu.RLock()
			handler := twe.handlers[step.Uses]
			twe.mu.RUnlock()

			if handler == nil {
				twe.mu.Lock()
				twe.runs[runID] = mdk.WorkflowStatus{
					State: mdk.StepFailed,
					Error: fmt.Sprintf("handler not found for step %s (uses %s)", step.ID, step.Uses),
				}
				twe.mu.Unlock()
				return results, fmt.Errorf("handler not found for step %s (uses %s)", step.ID, step.Uses)
			}

			sCtx := mdk.StepContext{
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
				twe.runs[runID] = mdk.WorkflowStatus{
					State: mdk.StepFailed,
					Error: res.Err.Error(),
				}
				twe.mu.Unlock()

				// Run compensations in reverse order
				for i := len(history) - 1; i >= 0; i-- {
					hStep := history[i]
					var compensate mdk.StepHandler
					if hStep.Saga != nil && hStep.Saga.Uses != "" {
						twe.mu.RLock()
						h := twe.handlers[hStep.Saga.Uses]
						twe.mu.RUnlock()
						if h != nil {
							compensate = h
						}
					}
					if compensate != nil {
						sCtxComp := mdk.StepContext{
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
	run := twe.runs[runID]
	run.State = mdk.StepCompleted
	now := time.Now()
	run.EndedAt = &now
	twe.runs[runID] = run
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
	Events    []mdk.Event
}

func (tld TestLineageData) GetID() string            { return tld.ID }
func (tld TestLineageData) GetName() string          { return tld.Name }
func (tld TestLineageData) GetState() string         { return tld.State }
func (tld TestLineageData) GetError() string         { return tld.Error }
func (tld TestLineageData) GetStartedAt() time.Time  { return tld.StartedAt }
func (tld TestLineageData) GetEndedAt() *time.Time   { return tld.EndedAt }
func (tld TestLineageData) GetEvents() []mdk.Event   { return tld.Events }

// TestProjector implements Projector for testing.
type TestProjector struct {
	Lineages []mdk.LineageData
}

func (tp *TestProjector) ListLineages() []mdk.LineageData {
	return tp.Lineages
}

func (tp *TestProjector) QueryLineages(filter func(mdk.LineageData) bool) []mdk.LineageData {
	var out []mdk.LineageData
	for _, l := range tp.Lineages {
		if filter(l) {
			out = append(out, l)
		}
	}
	return out
}

// ProjectorModule is a generic mock implementation of a module that provides a Projector.
type ProjectorModule struct {
	ModuleID string
	Proj     mdk.Projector
}

// ID returns the module ID, defaulting to "core.context".
func (pm *ProjectorModule) ID() string {
	if pm.ModuleID != "" {
		return pm.ModuleID
	}
	return "core.context"
}

func (pm *ProjectorModule) Init(ctx context.Context, rt mdk.Runtime) error {
	return nil
}

func (pm *ProjectorModule) Shutdown(ctx context.Context) error {
	return nil
}

func (pm *ProjectorModule) Models() []any {
	return nil
}

func (pm *ProjectorModule) Routes() []mdk.Route {
	return nil
}

func (pm *ProjectorModule) Projector() mdk.Projector {
	return pm.Proj
}
