package dag

import "fmt"

// TaskKind identifies the category of code generation for a task.
type TaskKind string

const (
	TaskKindDataSchemas    TaskKind = "data.schemas"
	TaskKindDataMigrations TaskKind = "data.migrations"

	// Service tasks are split into four focused layers so each agent call
	// produces a small, independently-verifiable unit of code rather than
	// an entire multi-thousand-line codebase in one shot.
	TaskKindServicePlan          TaskKind = "backend.service.plan"       // architect phase: interfaces + go.mod skeleton
	TaskKindDependencyResolution TaskKind = "backend.service.deps"       // resolve + lock all dependencies (no LLM)
	TaskKindServiceRepository    TaskKind = "backend.service.repository" // data-access layer
	TaskKindServiceLogic         TaskKind = "backend.service.logic"      // business logic layer
	TaskKindServiceHandler       TaskKind = "backend.service.handler"    // HTTP handlers + routing
	TaskKindServiceBootstrap     TaskKind = "backend.service.bootstrap"  // main.go, go.mod, config

	TaskKindAuth      TaskKind = "backend.auth"
	TaskKindMessaging TaskKind = "backend.messaging"
	TaskKindGateway   TaskKind = "backend.gateway"

	TaskKindContracts       TaskKind = "contracts"
	TaskKindFrontend        TaskKind = "frontend"
	TaskKindInfraDocker     TaskKind = "infra.docker"
	TaskKindInfraTerraform  TaskKind = "infra.terraform"
	TaskKindInfraCI         TaskKind = "infra.cicd"
	TaskKindCrossCutTesting TaskKind = "crosscut.testing"
	TaskKindCrossCutDocs    TaskKind = "crosscut.docs"

	// TaskKindReconciliation runs after all backend service chains complete but
	// before infrastructure generation. It performs a project-wide go build ./...
	// across all generated Go modules and patches only the files that fail to
	// compile, fixing cross-task type mismatches, wrong import paths, and
	// constructor-signature drift that per-task verification cannot catch.
	TaskKindReconciliation TaskKind = "backend.reconciliation"

	// TaskKindIntegrationRepair is a synthetic kind used exclusively by the
	// post-pipeline integration repair phase. It never appears in the DAG; it
	// provides a role-description hook and tier assignment for the LLM repair agent.
	TaskKindIntegrationRepair TaskKind = "integration.repair"
)

// GeneratedFile is one file produced by an agent for a task.
type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// Task is the atomic unit of work executed by one agent invocation.
type Task struct {
	ID           string // unique stable identifier, e.g. "service.user-api"
	Kind         TaskKind
	Label        string
	Dependencies []string    // IDs of tasks that must complete before this one
	Payload      TaskPayload // scoped manifest slice for this task's agent
}

// DAG is the full dependency graph for a single manifest realization.
type DAG struct {
	Tasks map[string]*Task
	order []string // topologically sorted task IDs
}

// Levels returns task IDs grouped into parallel execution waves.
// All tasks in wave N can run concurrently; wave N+1 starts only after N completes.
func (d *DAG) Levels() [][]string {
	// For each task, compute its level = max(level of deps) + 1.
	levels := make(map[string]int, len(d.Tasks))
	var assignLevel func(id string) int
	assignLevel = func(id string) int {
		if v, ok := levels[id]; ok {
			return v
		}
		task := d.Tasks[id]
		maxDep := -1
		for _, dep := range task.Dependencies {
			if l := assignLevel(dep); l > maxDep {
				maxDep = l
			}
		}
		levels[id] = maxDep + 1
		return maxDep + 1
	}
	for id := range d.Tasks {
		assignLevel(id)
	}

	// Find the maximum level (-1 means no tasks → 0 waves).
	maxLevel := -1
	for _, l := range levels {
		if l > maxLevel {
			maxLevel = l
		}
	}

	// Group tasks by level.
	waves := make([][]string, maxLevel+1)
	for id, l := range levels {
		waves[l] = append(waves[l], id)
	}
	return waves
}

// Order returns topologically sorted task IDs (dependencies before dependents).
func (d *DAG) Order() []string {
	return d.order
}

// build finalizes the DAG after all tasks have been added.
// It validates that all dependency IDs exist and computes the topological order.
func (d *DAG) build() error {
	// Validate all dependency references.
	for id, task := range d.Tasks {
		for _, dep := range task.Dependencies {
			if _, ok := d.Tasks[dep]; !ok {
				return fmt.Errorf("task %q references unknown dependency %q", id, dep)
			}
		}
	}

	// Kahn's algorithm for topological sort.
	inDegree := make(map[string]int, len(d.Tasks))
	for id := range d.Tasks {
		inDegree[id] = 0
	}
	for _, task := range d.Tasks {
		for _, dep := range task.Dependencies {
			inDegree[task.ID]++
			_ = dep
		}
	}
	// Recompute: count how many tasks depend on each task.
	// Actually count incoming edges: for each task, count its Dependencies.
	for id := range inDegree {
		inDegree[id] = len(d.Tasks[id].Dependencies)
	}

	queue := []string{}
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	result := make([]string, 0, len(d.Tasks))
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		result = append(result, cur)
		// Find tasks that depend on cur and reduce their in-degree.
		for id, task := range d.Tasks {
			for _, dep := range task.Dependencies {
				if dep == cur {
					inDegree[id]--
					if inDegree[id] == 0 {
						queue = append(queue, id)
					}
				}
			}
		}
	}

	if len(result) != len(d.Tasks) {
		return fmt.Errorf("task dependency graph contains a cycle")
	}
	d.order = result
	return nil
}
