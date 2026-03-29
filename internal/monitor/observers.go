package monitor

import (
	"bytes"
	"crypto/sha256"
	"sort"

	"vps-monitor/internal/model"
)

type observerCandidate struct {
	digest [sha256.Size]byte
	member model.ClusterMember
}

func BuildObserverAssignments(members []model.ClusterMember, states []model.NodeState, observersPerTarget int) map[string][]model.ClusterMember {
	activeMembers := activeMembersOnly(members)
	assignments := make(map[string][]model.ClusterMember, len(activeMembers))
	if len(activeMembers) == 0 {
		return assignments
	}

	if observersPerTarget <= 0 {
		for _, target := range activeMembers {
			observers := make([]model.ClusterMember, 0, len(activeMembers)-1)
			for _, member := range activeMembers {
				if member.NodeID == target.NodeID {
					continue
				}
				observers = append(observers, member)
			}
			assignments[target.NodeID] = observers
		}
		return assignments
	}

	stateMap := make(map[string]model.NodeState, len(states))
	for _, state := range states {
		stateMap[state.NodeID] = state
	}

	for _, target := range activeMembers {
		candidates := make([]observerCandidate, 0, len(activeMembers)-1)
		for _, member := range activeMembers {
			if member.NodeID == target.NodeID {
				continue
			}
			state, ok := stateMap[member.NodeID]
			if ok && !state.ReplicatedFresh {
				continue
			}
			candidates = append(candidates, observerCandidate{
				digest: sha256.Sum256([]byte(target.NodeID + "\n" + member.NodeID)),
				member: member,
			})
		}

		sort.Slice(candidates, func(i, j int) bool {
			if cmp := bytes.Compare(candidates[i].digest[:], candidates[j].digest[:]); cmp != 0 {
				return cmp < 0
			}
			return candidates[i].member.NodeID < candidates[j].member.NodeID
		})

		limit := observersPerTarget
		if limit > len(candidates) {
			limit = len(candidates)
		}
		observers := make([]model.ClusterMember, 0, limit)
		for i := 0; i < limit; i++ {
			observers = append(observers, candidates[i].member)
		}
		assignments[target.NodeID] = observers
	}

	return assignments
}

func ProbeTargetsForObserver(observerNodeID string, members []model.ClusterMember, assignments map[string][]model.ClusterMember) []model.ClusterMember {
	activeMembers := activeMembersOnly(members)
	memberMap := make(map[string]model.ClusterMember, len(activeMembers))
	for _, member := range activeMembers {
		memberMap[member.NodeID] = member
	}

	targets := make([]model.ClusterMember, 0, len(activeMembers))
	for _, member := range activeMembers {
		if member.NodeID == observerNodeID {
			continue
		}
		for _, observer := range assignments[member.NodeID] {
			if observer.NodeID == observerNodeID {
				targets = append(targets, memberMap[member.NodeID])
				break
			}
		}
	}
	return targets
}

func activeMembersOnly(members []model.ClusterMember) []model.ClusterMember {
	out := make([]model.ClusterMember, 0, len(members))
	for _, member := range members {
		if !member.IsActive() {
			continue
		}
		out = append(out, member)
	}
	return out
}
