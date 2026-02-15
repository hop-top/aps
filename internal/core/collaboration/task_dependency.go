package collaboration

import (
	"fmt"
	"sync"
)

// DependencyGraph tracks task dependencies and supports topological ordering
// and cycle detection. All operations are thread-safe.
type DependencyGraph struct {
	mu    sync.RWMutex
	nodes map[string][]string // task ID -> list of dependency task IDs
}

// NewDependencyGraph creates an empty dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string][]string),
	}
}

// AddTask registers a task with its dependencies in the graph.
// If the task already exists, its dependencies are replaced.
func (dg *DependencyGraph) AddTask(taskID string, dependencies []string) {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	deps := make([]string, len(dependencies))
	copy(deps, dependencies)
	dg.nodes[taskID] = deps

	// Ensure all dependency nodes are present in the graph.
	for _, dep := range deps {
		if _, exists := dg.nodes[dep]; !exists {
			dg.nodes[dep] = nil
		}
	}
}

// TopologicalSort returns the tasks in a valid execution order such that all
// dependencies are satisfied before a task appears. Returns an error if the
// graph contains a cycle.
func (dg *DependencyGraph) TopologicalSort() ([]string, error) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	// Kahn's algorithm.
	inDegree := make(map[string]int, len(dg.nodes))
	for id := range dg.nodes {
		if _, exists := inDegree[id]; !exists {
			inDegree[id] = 0
		}
	}

	// Build reverse adjacency (dependsOn -> depended-by) and compute in-degrees.
	dependedBy := make(map[string][]string, len(dg.nodes))
	for id, deps := range dg.nodes {
		for _, dep := range deps {
			dependedBy[dep] = append(dependedBy[dep], id)
			inDegree[id]++
		}
	}

	// Seed queue with nodes that have no dependencies.
	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		sorted = append(sorted, node)

		for _, downstream := range dependedBy[node] {
			inDegree[downstream]--
			if inDegree[downstream] == 0 {
				queue = append(queue, downstream)
			}
		}
	}

	if len(sorted) != len(dg.nodes) {
		return nil, fmt.Errorf("circular dependency detected: cannot produce valid task ordering")
	}

	return sorted, nil
}

// DetectCycles returns the task IDs forming a cycle, or nil if no cycle exists.
// Uses DFS with coloring: white (unvisited), gray (in-progress), black (done).
func (dg *DependencyGraph) DetectCycles() []string {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	const (
		white = 0 // unvisited
		gray  = 1 // in current DFS path
		black = 2 // fully processed
	)

	color := make(map[string]int, len(dg.nodes))
	parent := make(map[string]string, len(dg.nodes))

	for id := range dg.nodes {
		color[id] = white
	}

	// dfs returns the cycle-starting node if a cycle is found from this node.
	var dfs func(node string) string
	dfs = func(node string) string {
		color[node] = gray

		for _, dep := range dg.nodes[node] {
			if color[dep] == gray {
				// Back edge found: dep is the start of the cycle.
				parent[dep] = node
				return dep
			}
			if color[dep] == white {
				parent[dep] = node
				if cycleStart := dfs(dep); cycleStart != "" {
					return cycleStart
				}
			}
		}

		color[node] = black
		return ""
	}

	for id := range dg.nodes {
		if color[id] == white {
			if cycleStart := dfs(id); cycleStart != "" {
				return dg.reconstructCycle(parent, cycleStart)
			}
		}
	}

	return nil
}

// reconstructCycle traces the parent map to build the cycle path.
func (dg *DependencyGraph) reconstructCycle(parent map[string]string, cycleStart string) []string {
	var cycle []string
	current := cycleStart
	for {
		cycle = append(cycle, current)
		current = parent[current]
		if current == cycleStart {
			cycle = append(cycle, current)
			break
		}
	}

	// Reverse to get forward-order cycle.
	for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
		cycle[i], cycle[j] = cycle[j], cycle[i]
	}

	return cycle
}

// CanStart checks whether all dependencies of the given task have been completed.
// The completed map holds task IDs that are finished.
func (dg *DependencyGraph) CanStart(taskID string, completed map[string]bool) (bool, error) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	deps, exists := dg.nodes[taskID]
	if !exists {
		return false, fmt.Errorf("task %q not found in dependency graph", taskID)
	}

	for _, dep := range deps {
		if !completed[dep] {
			return false, nil
		}
	}
	return true, nil
}
