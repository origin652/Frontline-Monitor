package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hashicorp/raft"

	"vps-monitor/internal/model"
	"vps-monitor/internal/store"
)

const (
	CommandHeartbeat       = "heartbeat"
	CommandProbe           = "probe"
	CommandNodeState       = "node_state"
	CommandIncident        = "incident"
	CommandEvent           = "event"
	CommandIngress         = "ingress"
	CommandAlertClaim      = "alert_claim"
	CommandAlertCompletion = "alert_completion"
	CommandAdminSettings   = "admin_settings"
	CommandAdminSession    = "admin_session"
	CommandDeleteSession   = "delete_session"
	CommandMonitorCheck    = "monitor_check"
	CommandDeleteCheck     = "delete_check"
	CommandNodeDisplayName = "node_display_name"
	CommandDeleteNodeName  = "delete_node_display_name"
)

type commandEnvelope struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type CommandResult struct {
	Claimed bool   `json:"claimed,omitempty"`
	Error   string `json:"error,omitempty"`
}

type FSM struct {
	store *store.Store
}

func NewFSM(s *store.Store) *FSM {
	return &FSM{store: s}
}

func (f *FSM) Apply(logEntry *raft.Log) any {
	var env commandEnvelope
	if err := json.Unmarshal(logEntry.Data, &env); err != nil {
		return CommandResult{Error: err.Error()}
	}

	ctx := context.Background()
	switch env.Type {
	case CommandHeartbeat:
		var hb model.NodeHeartbeat
		if err := json.Unmarshal(env.Payload, &hb); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.RecordHeartbeat(ctx, hb); err != nil {
			return CommandResult{Error: err.Error()}
		}
	case CommandProbe:
		var probe model.ProbeObservation
		if err := json.Unmarshal(env.Payload, &probe); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.RecordProbe(ctx, probe); err != nil {
			return CommandResult{Error: err.Error()}
		}
	case CommandNodeState:
		var state model.NodeState
		if err := json.Unmarshal(env.Payload, &state); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.UpsertNodeState(ctx, state); err != nil {
			return CommandResult{Error: err.Error()}
		}
	case CommandIncident:
		var inc model.Incident
		if err := json.Unmarshal(env.Payload, &inc); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.UpsertIncident(ctx, inc); err != nil {
			return CommandResult{Error: err.Error()}
		}
	case CommandEvent:
		var event model.Event
		if err := json.Unmarshal(env.Payload, &event); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.AddEvent(ctx, event); err != nil {
			return CommandResult{Error: err.Error()}
		}
	case CommandIngress:
		var ingress model.IngressState
		if err := json.Unmarshal(env.Payload, &ingress); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.UpsertIngressState(ctx, ingress); err != nil {
			return CommandResult{Error: err.Error()}
		}
	case CommandAlertClaim:
		var claim model.AlertClaim
		if err := json.Unmarshal(env.Payload, &claim); err != nil {
			return CommandResult{Error: err.Error()}
		}
		claimed, err := f.store.ClaimAlertDelivery(ctx, model.AlertDelivery{
			IncidentID:  claim.IncidentID,
			Channel:     claim.Channel,
			DeliveryKey: claim.DeliveryKey,
			Status:      "pending",
			Response:    "",
			CreatedAt:   claim.CreatedAt,
		})
		if err != nil {
			return CommandResult{Error: err.Error()}
		}
		return CommandResult{Claimed: claimed}
	case CommandAlertCompletion:
		var completion model.AlertCompletion
		if err := json.Unmarshal(env.Payload, &completion); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.CompleteAlertDelivery(ctx, completion.DeliveryKey, completion.Status, completion.Response, completion.SentAt); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if completion.IncidentID != "" && completion.Status == "sent" {
			if err := f.store.UpdateIncidentLastNotified(ctx, completion.IncidentID, completion.SentAt); err != nil {
				return CommandResult{Error: err.Error()}
			}
		}
	case CommandAdminSettings:
		var settings model.AdminSettings
		if err := json.Unmarshal(env.Payload, &settings); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.UpsertAdminSettings(ctx, settings); err != nil {
			return CommandResult{Error: err.Error()}
		}
	case CommandAdminSession:
		var session model.AdminSession
		if err := json.Unmarshal(env.Payload, &session); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.UpsertAdminSession(ctx, session); err != nil {
			return CommandResult{Error: err.Error()}
		}
	case CommandDeleteSession:
		var payload struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.DeleteAdminSession(ctx, payload.ID); err != nil {
			return CommandResult{Error: err.Error()}
		}
	case CommandMonitorCheck:
		var check model.MonitorCheck
		if err := json.Unmarshal(env.Payload, &check); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.UpsertMonitorCheck(ctx, check); err != nil {
			return CommandResult{Error: err.Error()}
		}
	case CommandDeleteCheck:
		var payload struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.DeleteMonitorCheck(ctx, payload.ID); err != nil {
			return CommandResult{Error: err.Error()}
		}
	case CommandNodeDisplayName:
		var item model.NodeDisplayName
		if err := json.Unmarshal(env.Payload, &item); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.UpsertNodeDisplayName(ctx, item); err != nil {
			return CommandResult{Error: err.Error()}
		}
	case CommandDeleteNodeName:
		var payload struct {
			NodeID string `json:"node_id"`
		}
		if err := json.Unmarshal(env.Payload, &payload); err != nil {
			return CommandResult{Error: err.Error()}
		}
		if err := f.store.DeleteNodeDisplayName(ctx, payload.NodeID); err != nil {
			return CommandResult{Error: err.Error()}
		}
	default:
		return CommandResult{Error: fmt.Sprintf("unknown command type %q", env.Type)}
	}

	return CommandResult{}
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	snap, err := f.store.Snapshot(context.Background())
	if err != nil {
		return nil, err
	}
	return &fsmSnapshot{snapshot: snap}, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	var snap store.SnapshotData
	if err := json.NewDecoder(rc).Decode(&snap); err != nil {
		return err
	}
	return f.store.Restore(context.Background(), snap)
}

type fsmSnapshot struct {
	snapshot *store.SnapshotData
}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	defer sink.Close()
	if err := json.NewEncoder(sink).Encode(s.snapshot); err != nil {
		_ = sink.Cancel()
		return err
	}
	return nil
}

func (s *fsmSnapshot) Release() {}
