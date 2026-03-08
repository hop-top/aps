package collaboration_test

import (
	"testing"

	"hop.top/aps/internal/core/collaboration"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDependencyGraph_AddTask(t *testing.T) {
	dg := collaboration.NewDependencyGraph()

	dg.AddTask("A", nil)
	dg.AddTask("B", []string{"A"})
	dg.AddTask("C", []string{"A", "B"})

	// Verify all tasks are present by checking CanStart with no completions.
	canStart, err := dg.CanStart("A", map[string]bool{})
	require.NoError(t, err)
	assert.True(t, canStart, "A has no dependencies")

	canStart, err = dg.CanStart("B", map[string]bool{})
	require.NoError(t, err)
	assert.False(t, canStart, "B depends on A")
}

func TestDependencyGraph_TopologicalSort_Linear(t *testing.T) {
	dg := collaboration.NewDependencyGraph()

	// Linear chain: A -> B -> C
	dg.AddTask("A", nil)
	dg.AddTask("B", []string{"A"})
	dg.AddTask("C", []string{"B"})

	sorted, err := dg.TopologicalSort()
	require.NoError(t, err)
	require.Len(t, sorted, 3)

	// Build index map to check ordering.
	idx := make(map[string]int)
	for i, s := range sorted {
		idx[s] = i
	}

	assert.Less(t, idx["A"], idx["B"], "A must come before B")
	assert.Less(t, idx["B"], idx["C"], "B must come before C")
}

func TestDependencyGraph_TopologicalSort_Diamond(t *testing.T) {
	dg := collaboration.NewDependencyGraph()

	// Diamond: A -> B, A -> C, B -> D, C -> D
	dg.AddTask("A", nil)
	dg.AddTask("B", []string{"A"})
	dg.AddTask("C", []string{"A"})
	dg.AddTask("D", []string{"B", "C"})

	sorted, err := dg.TopologicalSort()
	require.NoError(t, err)
	require.Len(t, sorted, 4)

	idx := make(map[string]int)
	for i, s := range sorted {
		idx[s] = i
	}

	assert.Less(t, idx["A"], idx["B"])
	assert.Less(t, idx["A"], idx["C"])
	assert.Less(t, idx["B"], idx["D"])
	assert.Less(t, idx["C"], idx["D"])
}

func TestDependencyGraph_TopologicalSort_Cycle(t *testing.T) {
	dg := collaboration.NewDependencyGraph()

	// A -> B -> A (cycle)
	dg.AddTask("A", []string{"B"})
	dg.AddTask("B", []string{"A"})

	_, err := dg.TopologicalSort()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestDependencyGraph_DetectCycles_None(t *testing.T) {
	dg := collaboration.NewDependencyGraph()

	dg.AddTask("A", nil)
	dg.AddTask("B", []string{"A"})
	dg.AddTask("C", []string{"B"})

	cycle := dg.DetectCycles()
	assert.Nil(t, cycle)
}

func TestDependencyGraph_DetectCycles_Simple(t *testing.T) {
	dg := collaboration.NewDependencyGraph()

	// A -> B -> A
	dg.AddTask("A", []string{"B"})
	dg.AddTask("B", []string{"A"})

	cycle := dg.DetectCycles()
	require.NotNil(t, cycle)
	assert.GreaterOrEqual(t, len(cycle), 2, "cycle path should contain at least 2 nodes")

	// The cycle should contain both A and B.
	cycleSet := make(map[string]bool)
	for _, node := range cycle {
		cycleSet[node] = true
	}
	assert.True(t, cycleSet["A"], "cycle should contain A")
	assert.True(t, cycleSet["B"], "cycle should contain B")
}

func TestDependencyGraph_DetectCycles_Complex(t *testing.T) {
	dg := collaboration.NewDependencyGraph()

	// A -> B -> C -> A
	dg.AddTask("A", []string{"C"})
	dg.AddTask("B", []string{"A"})
	dg.AddTask("C", []string{"B"})

	cycle := dg.DetectCycles()
	require.NotNil(t, cycle)
	assert.GreaterOrEqual(t, len(cycle), 3, "cycle path should contain at least 3 nodes")

	cycleSet := make(map[string]bool)
	for _, node := range cycle {
		cycleSet[node] = true
	}
	assert.True(t, cycleSet["A"])
	assert.True(t, cycleSet["B"])
	assert.True(t, cycleSet["C"])
}

func TestDependencyGraph_CanStart_AllDepsCompleted(t *testing.T) {
	dg := collaboration.NewDependencyGraph()

	dg.AddTask("A", nil)
	dg.AddTask("B", []string{"A"})
	dg.AddTask("C", []string{"A", "B"})

	completed := map[string]bool{"A": true, "B": true}

	canStart, err := dg.CanStart("C", completed)
	require.NoError(t, err)
	assert.True(t, canStart)
}

func TestDependencyGraph_CanStart_MissingDeps(t *testing.T) {
	dg := collaboration.NewDependencyGraph()

	dg.AddTask("A", nil)
	dg.AddTask("B", []string{"A"})
	dg.AddTask("C", []string{"A", "B"})

	// Only A is completed; B is still pending.
	completed := map[string]bool{"A": true}

	canStart, err := dg.CanStart("C", completed)
	require.NoError(t, err)
	assert.False(t, canStart)
}

func TestDependencyGraph_CanStart_NoDeps(t *testing.T) {
	dg := collaboration.NewDependencyGraph()

	dg.AddTask("root", nil)

	canStart, err := dg.CanStart("root", map[string]bool{})
	require.NoError(t, err)
	assert.True(t, canStart)
}

func TestDependencyGraph_CanStart_UnknownTask(t *testing.T) {
	dg := collaboration.NewDependencyGraph()

	dg.AddTask("A", nil)

	_, err := dg.CanStart("unknown", map[string]bool{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
