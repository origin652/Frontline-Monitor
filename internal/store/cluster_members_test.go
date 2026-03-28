package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"vps-monitor/internal/model"
)

func TestClusterMembersSnapshotAndRestore(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "source.db"))
	if err != nil {
		t.Fatalf("open source store: %v", err)
	}
	defer st.Close()

	now := time.Now().UTC().Round(time.Second)
	removedAt := now.Add(2 * time.Minute)
	items := []model.ClusterMember{
		{
			NodeID:      "node-a",
			DisplayName: "Tokyo-1",
			APIAddr:     "10.0.0.11:8443",
			RaftAddr:    "10.0.0.11:7000",
			PublicIPv4:  "203.0.113.11",
			Priority:    300,
			DesiredRole: model.ClusterMemberRoleVoter,
			Status:      model.ClusterMemberStatusActive,
			JoinedAt:    now,
			UpdatedAt:   now,
		},
		{
			NodeID:      "node-b",
			DisplayName: "Singapore-1",
			APIAddr:     "10.0.0.12:8443",
			RaftAddr:    "10.0.0.12:7000",
			PublicIPv4:  "203.0.113.12",
			Priority:    200,
			DesiredRole: model.ClusterMemberRoleNonvoter,
			Status:      model.ClusterMemberStatusRemoved,
			JoinedAt:    now.Add(time.Minute),
			UpdatedAt:   removedAt,
			RemovedAt:   &removedAt,
		},
	}
	for _, item := range items {
		if err := st.UpsertClusterMember(ctx, item); err != nil {
			t.Fatalf("UpsertClusterMember(%s) error = %v", item.NodeID, err)
		}
	}

	snap, err := st.Snapshot(ctx)
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if len(snap.ClusterMembers) != 2 {
		t.Fatalf("Snapshot() cluster members = %d, want 2", len(snap.ClusterMembers))
	}

	restoreStore, err := Open(filepath.Join(t.TempDir(), "restore.db"))
	if err != nil {
		t.Fatalf("open restore store: %v", err)
	}
	defer restoreStore.Close()
	if err := restoreStore.Restore(ctx, *snap); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	restored, err := restoreStore.ListClusterMembers(ctx)
	if err != nil {
		t.Fatalf("ListClusterMembers() error = %v", err)
	}
	if len(restored) != 2 {
		t.Fatalf("restored member count = %d, want 2", len(restored))
	}
	if restored[1].NodeID != "node-b" {
		t.Fatalf("restored second node_id = %q, want %q", restored[1].NodeID, "node-b")
	}
	if restored[1].RemovedAt == nil {
		t.Fatal("restored removed member should keep removed_at")
	}
	if restored[1].DesiredRole != model.ClusterMemberRoleNonvoter {
		t.Fatalf("restored desired_role = %q, want %q", restored[1].DesiredRole, model.ClusterMemberRoleNonvoter)
	}
}
