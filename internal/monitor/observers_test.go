package monitor

import (
	"reflect"
	"testing"

	"vps-monitor/internal/model"
)

func TestBuildObserverAssignmentsFullMesh(t *testing.T) {
	t.Parallel()

	members := []model.ClusterMember{
		testMember("node-a"),
		testMember("node-b"),
		testMember("node-c"),
	}

	assignments := BuildObserverAssignments(members, nil, 0)
	if got := observerNodeIDs(assignments["node-a"]); !reflect.DeepEqual(got, []string{"node-b", "node-c"}) {
		t.Fatalf("node-a observers = %#v, want %#v", got, []string{"node-b", "node-c"})
	}
	if got := observerNodeIDs(assignments["node-b"]); !reflect.DeepEqual(got, []string{"node-a", "node-c"}) {
		t.Fatalf("node-b observers = %#v, want %#v", got, []string{"node-a", "node-c"})
	}
}

func TestBuildObserverAssignmentsSparseStableAcrossInputOrder(t *testing.T) {
	t.Parallel()

	membersA := []model.ClusterMember{
		testMember("node-a"),
		testMember("node-b"),
		testMember("node-c"),
		testMember("node-d"),
	}
	membersB := []model.ClusterMember{
		testMember("node-d"),
		testMember("node-b"),
		testMember("node-a"),
		testMember("node-c"),
	}

	first := observerNodeIDs(BuildObserverAssignments(membersA, nil, 2)["node-a"])
	second := observerNodeIDs(BuildObserverAssignments(membersB, nil, 2)["node-a"])
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("stable sparse observers mismatch: first=%#v second=%#v", first, second)
	}
	if len(first) != 2 {
		t.Fatalf("sparse observer count = %d, want 2", len(first))
	}
}

func TestBuildObserverAssignmentsTruncatesToEligibleObservers(t *testing.T) {
	t.Parallel()

	members := []model.ClusterMember{
		testMember("node-a"),
		testMember("node-b"),
		testMember("node-c"),
	}

	assignments := BuildObserverAssignments(members, nil, 5)
	if got := len(assignments["node-a"]); got != 2 {
		t.Fatalf("node-a sparse observer count = %d, want 2", got)
	}
}

func TestBuildObserverAssignmentsReplacesStaleObserver(t *testing.T) {
	t.Parallel()

	members := []model.ClusterMember{
		testMember("node-a"),
		testMember("node-b"),
		testMember("node-c"),
		testMember("node-d"),
	}

	initial := observerNodeIDs(BuildObserverAssignments(members, nil, 2)["node-a"])
	if len(initial) != 2 {
		t.Fatalf("initial sparse observer count = %d, want 2", len(initial))
	}

	states := []model.NodeState{
		{NodeID: initial[0], ReplicatedFresh: false},
	}
	next := observerNodeIDs(BuildObserverAssignments(members, states, 2)["node-a"])
	if len(next) != 2 {
		t.Fatalf("replacement sparse observer count = %d, want 2", len(next))
	}
	if reflect.DeepEqual(initial, next) {
		t.Fatalf("expected replacement observers to change, got %#v", next)
	}
	for _, nodeID := range next {
		if nodeID == initial[0] {
			t.Fatalf("stale observer %q was not replaced: %#v", initial[0], next)
		}
	}
}

func testMember(nodeID string) model.ClusterMember {
	return model.ClusterMember{
		NodeID:  nodeID,
		Status:  model.ClusterMemberStatusActive,
		APIAddr: nodeID + ":8443",
	}
}

func observerNodeIDs(members []model.ClusterMember) []string {
	out := make([]string, 0, len(members))
	for _, member := range members {
		out = append(out, member.NodeID)
	}
	return out
}
