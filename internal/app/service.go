package app

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ricardoaldape/Heimda11/internal/core"
	"github.com/ricardoaldape/Heimda11/internal/secure"
	"github.com/ricardoaldape/Heimda11/internal/store"
)

const Version = "0.1.0"

type Service struct {
	store   *store.FileStore
	cipher  *secure.Cipher
	edition string
	now     func() time.Time
}

func NewService(st *store.FileStore, cipher *secure.Cipher, edition string) *Service {
	if edition == "" {
		edition = "community"
	}
	return &Service{
		store: st,
		cipher: cipher,
		edition: strings.ToLower(edition),
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Meta() core.Meta {
	return core.Meta{Name: "Heimda11", Version: Version, Edition: s.edition, SelfHosted: true}
}

func credentialHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (s *Service) AuthenticateAgentKey(key string) (string, bool) {
	if !strings.HasPrefix(key, "hmd_") {
		return "", false
	}
	return s.store.CredentialAgent(credentialHash(key))
}

func (s *Service) CreateAgent(a core.Agent) (core.AgentCreateResult, error) {
	a.Name = strings.TrimSpace(a.Name)
	a.Role = strings.TrimSpace(a.Role)
	a.HumanOwner = strings.TrimSpace(a.HumanOwner)
	if a.Name == "" || a.Role == "" || a.HumanOwner == "" {
		return core.AgentCreateResult{}, errors.New("name, role and human_owner are required")
	}
	if a.AutonomyLevel < 0 || a.AutonomyLevel > 4 {
		return core.AgentCreateResult{}, errors.New("autonomy_level must be between 0 and 4")
	}
	if a.MonthlyBudget < 0 {
		return core.AgentCreateResult{}, errors.New("monthly_budget_usd cannot be negative")
	}
	if s.edition == "community" && len(s.store.Agents()) >= 10 {
		return core.AgentCreateResult{}, errors.New("community edition supports up to 10 registered agents")
	}
	a.ID = core.NewID("agt")
	a.Status = core.AgentActive
	now := s.now()
	a.CreatedAt = now
	a.UpdatedAt = now
	if err := s.store.UpsertAgent(a); err != nil {
		return core.AgentCreateResult{}, err
	}
	key := core.NewCredential()
	if err := s.store.SetCredential(credentialHash(key), a.ID); err != nil {
		return core.AgentCreateResult{}, err
	}
	_ = s.audit(a.ID, "registry.agent_created", "success", "AI worker registered", nil)
	return core.AgentCreateResult{Agent: a, APIKey: key}, nil
}

func (s *Service) RotateCredential(agentID string) (string, error) {
	if _, err := s.store.Agent(agentID); err != nil {
		return "", err
	}
	key := core.NewCredential()
	if err := s.store.SetCredential(credentialHash(key), agentID); err != nil {
		return "", err
	}
	_ = s.audit(agentID, "registry.credential_rotated", "success", "agent credential rotated", nil)
	return key, nil
}

func (s *Service) ListAgents() []core.Agent { return s.store.Agents() }

func (s *Service) Agent(id string) (core.Agent, error) { return s.store.Agent(id) }

func (s *Service) SetAgentStatus(id string, status core.AgentStatus) (core.Agent, error) {
	a, err := s.store.Agent(id)
	if err != nil {
		return core.Agent{}, err
	}
	switch status {
	case core.AgentActive, core.AgentSuspended, core.AgentRetired:
	default:
		return core.Agent{}, errors.New("status must be active, suspended or retired")
	}
	a.Status = status
	a.UpdatedAt = s.now()
	if err := s.store.UpsertAgent(a); err != nil {
		return core.Agent{}, err
	}
	_ = s.audit(a.ID, "registry.status_changed", "success", "agent status changed to "+string(status), nil)
	return a, nil
}

func (s *Service) CreatePolicy(p core.Policy) (core.Policy, error) {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return core.Policy{}, errors.New("name is required")
	}
	if p.AgentID == "" { p.AgentID = "*" }
	if p.Tool == "" { p.Tool = "*" }
	if p.Action == "" { p.Action = "*" }
	switch p.Effect {
	case core.DecisionAllow, core.DecisionDeny, core.DecisionApproval:
	default:
		return core.Policy{}, errors.New("effect must be allow, deny or approval_required")
	}
	if p.MinAmountUSD < 0 || p.MaxAmountUSD < 0 {
		return core.Policy{}, errors.New("amount conditions cannot be negative")
	}
	if p.MaxAmountUSD > 0 && p.MinAmountUSD > p.MaxAmountUSD {
		return core.Policy{}, errors.New("min_amount_usd cannot exceed max_amount_usd")
	}
	p.ID = core.NewID("pol")
	p.Enabled = true
	p.CreatedAt = s.now()
	if err := s.store.UpsertPolicy(p); err != nil {
		return core.Policy{}, err
	}
	_ = s.audit(p.AgentID, "gate.policy_created", "success", "policy "+p.Name+" created", map[string]any{"policy_id":p.ID})
	return p, nil
}

func (s *Service) ListPolicies() []core.Policy { return s.store.Policies() }

func (s *Service) Evaluate(req core.ActionRequest) (core.GateResult, error) {
	req.Tool = strings.TrimSpace(req.Tool)
	req.Action = strings.TrimSpace(req.Action)
	if req.AgentID == "" || req.Tool == "" || req.Action == "" {
		return core.GateResult{}, errors.New("agent_id, tool and action are required")
	}
	agent, err := s.store.Agent(req.AgentID)
	if err != nil {
		return core.GateResult{}, fmt.Errorf("agent: %w", err)
	}
	if agent.Status != core.AgentActive {
		result := core.GateResult{Decision: core.DecisionDeny, Reason: "agent is not active"}
		_ = s.recordGateEvent(req, result)
		return result, nil
	}
	if len(agent.AllowedTools) > 0 && !containsFold(agent.AllowedTools, req.Tool) {
		result := core.GateResult{Decision: core.DecisionDeny, Reason: "tool is outside the agent allow-list"}
		_ = s.recordGateEvent(req, result)
		return result, nil
	}

	for _, p := range s.store.Policies() {
		if !p.Enabled || !matches(p.AgentID, req.AgentID) || !matches(p.Tool, req.Tool) || !matches(p.Action, req.Action) {
			continue
		}
		if req.AmountUSD < p.MinAmountUSD {
			continue
		}
		if p.MaxAmountUSD > 0 && req.AmountUSD > p.MaxAmountUSD {
			continue
		}
		reason := p.Reason
		if reason == "" {
			reason = "matched policy " + p.Name
		}
		result := core.GateResult{Decision: p.Effect, PolicyID: p.ID, Reason: reason}
		if p.Effect == core.DecisionApproval {
			approval := core.Approval{
				ID: core.NewID("apr"), AgentID: req.AgentID, PolicyID: p.ID,
				Tool: req.Tool, Action: req.Action, AmountUSD: req.AmountUSD,
				Status: core.ApprovalPending, RequestedAt: s.now(),
			}
			if err := s.store.PutApproval(approval); err != nil {
				return core.GateResult{}, err
			}
			result.ApprovalID = approval.ID
		}
		_ = s.recordGateEvent(req, result)
		return result, nil
	}

	result := core.GateResult{Decision: core.DecisionDeny, Reason: "default deny: no enabled policy matched"}
	_ = s.recordGateEvent(req, result)
	return result, nil
}

func matches(pattern, value string) bool {
	return pattern == "*" || strings.EqualFold(pattern, value)
}

func containsFold(values []string, want string) bool {
	for _, value := range values {
		if value == "*" || strings.EqualFold(value, want) {
			return true
		}
	}
	return false
}

func (s *Service) recordGateEvent(req core.ActionRequest, result core.GateResult) error {
	return s.store.AddEvent(core.Event{
		ID: core.NewID("evt"), AgentID: req.AgentID, Type: "gate.decision",
		Tool: req.Tool, Action: req.Action, Status: string(result.Decision),
		TraceID: req.TraceID, Message: result.Reason, Metadata: req.Metadata, OccurredAt: s.now(),
	})
}

func (s *Service) ListApprovals() []core.Approval { return s.store.Approvals() }

func (s *Service) ResolveApproval(id string, status core.ApprovalStatus, by, note string) (core.Approval, error) {
	if status != core.ApprovalApproved && status != core.ApprovalRejected {
		return core.Approval{}, errors.New("status must be approved or rejected")
	}
	a, err := s.store.Approval(id)
	if err != nil { return core.Approval{}, err }
	if a.Status != core.ApprovalPending {
		return core.Approval{}, errors.New("approval is already resolved")
	}
	by = strings.TrimSpace(by)
	if by == "" {
		return core.Approval{}, errors.New("resolved_by is required")
	}
	now := s.now()
	a.Status = status
	a.ResolvedAt = &now
	a.ResolvedBy = by
	a.Note = strings.TrimSpace(note)
	if err := s.store.PutApproval(a); err != nil { return core.Approval{}, err }
	_ = s.store.AddEvent(core.Event{
		ID: core.NewID("evt"), AgentID: a.AgentID, Type: "approval.resolved",
		Tool: a.Tool, Action: a.Action, Status: string(a.Status), Message: a.Note, OccurredAt: now,
	})
	return a, nil
}

func (s *Service) RecordUsage(u core.UsageRecord) (core.UsageRecord, float64, bool, error) {
	agent, err := s.store.Agent(u.AgentID)
	if err != nil { return core.UsageRecord{}, 0, false, fmt.Errorf("agent: %w", err) }
	if u.CostUSD < 0 || u.InputTokens < 0 || u.OutputTokens < 0 {
		return core.UsageRecord{}, 0, false, errors.New("token counts and cost cannot be negative")
	}
	u.ID = core.NewID("use")
	if u.OccurredAt.IsZero() { u.OccurredAt = s.now() }
	if err := s.store.AddUsage(u); err != nil { return core.UsageRecord{}, 0, false, err }
	total := s.MonthSpend(u.AgentID, u.OccurredAt)
	over := agent.MonthlyBudget > 0 && total > agent.MonthlyBudget
	if over {
		_ = s.audit(u.AgentID, "meter.budget_exceeded", "warning", fmt.Sprintf("monthly spend %.4f USD exceeds budget %.4f USD", total, agent.MonthlyBudget), map[string]any{"trace_id":u.TraceID})
	}
	return u, total, over, nil
}

func (s *Service) Usage(agentID string) []core.UsageRecord { return s.store.Usage(agentID) }

func (s *Service) MonthSpend(agentID string, when time.Time) float64 {
	var total float64
	for _, u := range s.store.Usage(agentID) {
		if u.OccurredAt.Year() == when.Year() && u.OccurredAt.Month() == when.Month() {
			total += u.CostUSD
		}
	}
	return total
}

func (s *Service) RecordEvent(e core.Event) (core.Event, error) {
	if _, err := s.store.Agent(e.AgentID); err != nil {
		return core.Event{}, fmt.Errorf("agent: %w", err)
	}
	if strings.TrimSpace(e.Type) == "" || strings.TrimSpace(e.Status) == "" {
		return core.Event{}, errors.New("type and status are required")
	}
	e.ID = core.NewID("evt")
	if e.OccurredAt.IsZero() { e.OccurredAt = s.now() }
	if err := s.store.AddEvent(e); err != nil { return core.Event{}, err }
	a, _ := s.store.Agent(e.AgentID)
	now := s.now()
	a.LastSeenAt = &now
	a.UpdatedAt = now
	_ = s.store.UpsertAgent(a)
	return e, nil
}

func (s *Service) Events(agentID string) []core.Event { return s.store.Events(agentID) }

func (s *Service) Performance(agentID string) (core.AgentPerformance, error) {
	if _, err := s.store.Agent(agentID); err != nil {
		return core.AgentPerformance{}, err
	}
	p := core.AgentPerformance{AgentID: agentID}
	for _, e := range s.store.Events(agentID) {
		if e.Type == "task.completed" || e.Type == "task.failed" {
			p.Tasks++
			if e.Type == "task.completed" || strings.EqualFold(e.Status,"success") {
				p.Successes++
			} else {
				p.Failures++
			}
		}
		p.ValueUSD += e.ValueUSD
	}
	for _, u := range s.store.Usage(agentID) { p.CostUSD += u.CostUSD }
	if p.Tasks > 0 { p.SuccessRate = float64(p.Successes) / float64(p.Tasks) }
	if p.CostUSD > 0 { p.ROI = (p.ValueUSD - p.CostUSD) / p.CostUSD }
	return p, nil
}

func (s *Service) Dashboard() core.Dashboard {
	now := s.now()
	d := core.Dashboard{}
	for _, a := range s.store.Agents() {
		d.AgentsTotal++
		if a.Status == core.AgentActive { d.AgentsActive++ }
	}
	for _, a := range s.store.Approvals() {
		if a.Status == core.ApprovalPending { d.PendingApprovals++ }
	}
	for _, e := range s.store.Events("") {
		d.EventsTotal++
		if strings.EqualFold(e.Status,"failed") || strings.EqualFold(e.Status,"error") { d.FailuresTotal++ }
		if e.Type == "gate.decision" && e.Status == string(core.DecisionDeny) { d.BlockedTotal++ }
		if e.OccurredAt.Year()==now.Year() && e.OccurredAt.Month()==now.Month() { d.MonthValueUSD += e.ValueUSD }
	}
	for _, u := range s.store.Usage("") {
		if u.OccurredAt.Year()==now.Year() && u.OccurredAt.Month()==now.Month() { d.MonthCostUSD += u.CostUSD }
	}
	d.SecretsTotal = len(s.store.Secrets())
	d.MemoryItems = len(s.store.MemoryItems())
	for _, run := range s.store.FlowRuns() {
		if run.Status == "running" || run.Status == "waiting_approval" { d.FlowsRunning++ }
	}
	return d
}

func (s *Service) Export() store.State { return s.store.Snapshot() }

func (s *Service) audit(agentID, eventType, status, message string, metadata map[string]any) error {
	return s.store.AddEvent(core.Event{
		ID: core.NewID("evt"), AgentID: agentID, Type: eventType,
		Status: status, Message: message, Metadata: metadata, OccurredAt: s.now(),
	})
}
