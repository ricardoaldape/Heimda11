package core

import (
	"encoding/json"
	"time"
)

type AgentStatus string

const (
	AgentActive    AgentStatus = "active"
	AgentSuspended AgentStatus = "suspended"
	AgentRetired   AgentStatus = "retired"
)

type Decision string

const (
	DecisionAllow    Decision = "allow"
	DecisionDeny     Decision = "deny"
	DecisionApproval Decision = "approval_required"
)

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

type Agent struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Role           string      `json:"role"`
	Department     string      `json:"department"`
	HumanOwner     string      `json:"human_owner"`
	ManagerAgentID string      `json:"manager_agent_id,omitempty"`
	Status         AgentStatus `json:"status"`
	AutonomyLevel  int         `json:"autonomy_level"`
	MonthlyBudget  float64     `json:"monthly_budget_usd"`
	AllowedTools   []string    `json:"allowed_tools,omitempty"`
	Labels         []string    `json:"labels,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	LastSeenAt     *time.Time  `json:"last_seen_at,omitempty"`
}

type AgentCreateResult struct {
	Agent  Agent  `json:"agent"`
	APIKey string `json:"api_key"`
}

type Policy struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Priority     int       `json:"priority"`
	AgentID      string    `json:"agent_id"`
	Tool         string    `json:"tool"`
	Action       string    `json:"action"`
	MinAmountUSD float64   `json:"min_amount_usd,omitempty"`
	MaxAmountUSD float64   `json:"max_amount_usd,omitempty"`
	Effect       Decision  `json:"effect"`
	Reason       string    `json:"reason,omitempty"`
	Enabled      bool      `json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
}

type ActionRequest struct {
	AgentID   string         `json:"agent_id"`
	Tool      string         `json:"tool"`
	Action    string         `json:"action"`
	AmountUSD float64        `json:"amount_usd,omitempty"`
	Risk      string         `json:"risk,omitempty"`
	TraceID   string         `json:"trace_id,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type GateResult struct {
	Decision   Decision `json:"decision"`
	PolicyID   string   `json:"policy_id,omitempty"`
	Reason     string   `json:"reason"`
	ApprovalID string   `json:"approval_id,omitempty"`
}

type Approval struct {
	ID          string         `json:"id"`
	AgentID     string         `json:"agent_id"`
	PolicyID    string         `json:"policy_id"`
	Tool        string         `json:"tool"`
	Action      string         `json:"action"`
	AmountUSD   float64        `json:"amount_usd,omitempty"`
	Status      ApprovalStatus `json:"status"`
	RequestedAt time.Time      `json:"requested_at"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
	ResolvedBy  string         `json:"resolved_by,omitempty"`
	Note        string         `json:"note,omitempty"`
}

type UsageRecord struct {
	ID           string    `json:"id"`
	AgentID      string    `json:"agent_id"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	TraceID      string    `json:"trace_id,omitempty"`
	OccurredAt   time.Time `json:"occurred_at"`
}

type Event struct {
	ID         string         `json:"id"`
	AgentID    string         `json:"agent_id"`
	Type       string         `json:"type"`
	Tool       string         `json:"tool,omitempty"`
	Action     string         `json:"action,omitempty"`
	Status     string         `json:"status"`
	TraceID    string         `json:"trace_id,omitempty"`
	LatencyMS  int64          `json:"latency_ms,omitempty"`
	ValueUSD   float64        `json:"value_usd,omitempty"`
	Message    string         `json:"message,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	OccurredAt time.Time      `json:"occurred_at"`
}

type SecretRecord struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	ScopeAgentID string    `json:"scope_agent_id,omitempty"`
	Ciphertext   string    `json:"ciphertext"`
	CreatedAt    time.Time `json:"created_at"`
	RotatedAt    time.Time `json:"rotated_at"`
}

type SecretView struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	ScopeAgentID string    `json:"scope_agent_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	RotatedAt    time.Time `json:"rotated_at"`
}

type SecretLease struct {
	ID         string     `json:"id"`
	SecretID   string     `json:"secret_id"`
	AgentID    string     `json:"agent_id"`
	ExpiresAt  time.Time  `json:"expires_at"`
	Redeemed   bool       `json:"redeemed"`
	CreatedAt  time.Time  `json:"created_at"`
	RedeemedAt *time.Time `json:"redeemed_at,omitempty"`
}

type RelayProvider struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	BaseURL        string    `json:"base_url"`
	Model          string    `json:"model,omitempty"`
	SecretID       string    `json:"secret_id"`
	Priority       int       `json:"priority"`
	TimeoutSeconds int       `json:"timeout_seconds"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
}

type RelayRequest struct {
	AgentID string          `json:"agent_id"`
	Payload json.RawMessage `json:"payload"`
	TraceID string          `json:"trace_id,omitempty"`
}

type MemoryItem struct {
	ID              string    `json:"id"`
	Namespace       string    `json:"namespace"`
	Title           string    `json:"title"`
	Content         string    `json:"content"`
	Tags            []string  `json:"tags,omitempty"`
	AllowedAgentIDs []string  `json:"allowed_agent_ids,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Certification struct {
	ID         string     `json:"id"`
	AgentID    string     `json:"agent_id"`
	Skill      string     `json:"skill"`
	Score      float64    `json:"score"`
	Issuer     string     `json:"issuer"`
	IssuedAt   time.Time  `json:"issued_at"`
	ValidUntil *time.Time `json:"valid_until,omitempty"`
	Notes      string     `json:"notes,omitempty"`
}

type FlowStep struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	AgentID   string  `json:"agent_id"`
	Tool      string  `json:"tool"`
	Action    string  `json:"action"`
	AmountUSD float64 `json:"amount_usd,omitempty"`
}

type FlowDefinition struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Steps     []FlowStep `json:"steps"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type FlowStepResult struct {
	StepID      string    `json:"step_id"`
	Status      string    `json:"status"`
	Message     string    `json:"message,omitempty"`
	CompletedAt time.Time `json:"completed_at"`
}

type FlowRun struct {
	ID        string           `json:"id"`
	FlowID    string           `json:"flow_id"`
	Status    string           `json:"status"`
	Current   int              `json:"current_step"`
	History   []FlowStepResult `json:"history"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type AgentPerformance struct {
	AgentID     string  `json:"agent_id"`
	Tasks       int     `json:"tasks"`
	Successes   int     `json:"successes"`
	Failures    int     `json:"failures"`
	SuccessRate float64 `json:"success_rate"`
	CostUSD     float64 `json:"cost_usd"`
	ValueUSD    float64 `json:"value_usd"`
	ROI         float64 `json:"roi"`
}

type Dashboard struct {
	AgentsTotal      int     `json:"agents_total"`
	AgentsActive     int     `json:"agents_active"`
	PendingApprovals int     `json:"pending_approvals"`
	EventsTotal      int     `json:"events_total"`
	FailuresTotal    int     `json:"failures_total"`
	BlockedTotal     int     `json:"blocked_total"`
	MonthCostUSD     float64 `json:"month_cost_usd"`
	MonthValueUSD    float64 `json:"month_value_usd"`
	SecretsTotal     int     `json:"secrets_total"`
	FlowsRunning     int     `json:"flows_running"`
	MemoryItems      int     `json:"memory_items"`
}

type Meta struct {
	Name       string `json:"name"`
	Version    string `json:"version"`
	Edition    string `json:"edition"`
	SelfHosted bool   `json:"self_hosted"`
}
