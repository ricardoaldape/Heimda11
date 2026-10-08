package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ricardoaldape/Heimda11/internal/app"
	"github.com/ricardoaldape/Heimda11/internal/core"
	"github.com/ricardoaldape/Heimda11/internal/store"
	"github.com/ricardoaldape/Heimda11/internal/webui"
)

type actorKey struct{}

type actor struct {
	admin   bool
	agentID string
}

type API struct {
	service *app.Service
	token   string
}

func New(service *app.Service, token string) http.Handler {
	api := &API{service: service, token: token}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("GET /v1/meta", api.meta)
	mux.HandleFunc("GET /", api.ui)
	mux.HandleFunc("GET /assets/app.js", api.appJS)
	mux.HandleFunc("GET /assets/styles.css", api.styles)

	mux.Handle("GET /v1/dashboard", api.auth(true, http.HandlerFunc(api.dashboard)))
	mux.Handle("GET /v1/workforce/headcount", api.auth(true, http.HandlerFunc(api.headcount)))
	mux.Handle("GET /v1/export", api.auth(true, http.HandlerFunc(api.export)))

	mux.Handle("GET /v1/agents", api.auth(true, http.HandlerFunc(api.listAgents)))
	mux.Handle("POST /v1/agents", api.auth(true, http.HandlerFunc(api.createAgent)))
	mux.Handle("GET /v1/agents/{id}", api.auth(true, http.HandlerFunc(api.getAgent)))
	mux.Handle("POST /v1/agents/{id}/status", api.auth(true, http.HandlerFunc(api.agentStatus)))
	mux.Handle("POST /v1/agents/{id}/credentials/rotate", api.auth(true, http.HandlerFunc(api.rotateCredential)))
	mux.Handle("GET /v1/agents/{id}/performance", api.auth(false, http.HandlerFunc(api.performance)))
	mux.Handle("GET /v1/agents/{id}/usage", api.auth(false, http.HandlerFunc(api.agentUsage)))
	mux.Handle("GET /v1/agents/{id}/events", api.auth(false, http.HandlerFunc(api.agentEvents)))
	mux.Handle("GET /v1/agents/{id}/certifications", api.auth(false, http.HandlerFunc(api.certifications)))

	mux.Handle("GET /v1/policies", api.auth(true, http.HandlerFunc(api.listPolicies)))
	mux.Handle("POST /v1/policies", api.auth(true, http.HandlerFunc(api.createPolicy)))
	mux.Handle("POST /v1/gate/evaluate", api.auth(false, http.HandlerFunc(api.evaluate)))

	mux.Handle("GET /v1/approvals", api.auth(true, http.HandlerFunc(api.listApprovals)))
	mux.Handle("POST /v1/approvals/{id}/resolve", api.auth(true, http.HandlerFunc(api.resolveApproval)))

	mux.Handle("POST /v1/meter/usage", api.auth(false, http.HandlerFunc(api.recordUsage)))
	mux.Handle("POST /v1/watch/events", api.auth(false, http.HandlerFunc(api.recordEvent)))

	mux.Handle("GET /v1/vault/secrets", api.auth(true, http.HandlerFunc(api.listSecrets)))
	mux.Handle("POST /v1/vault/secrets", api.auth(true, http.HandlerFunc(api.createSecret)))
	mux.Handle("POST /v1/vault/secrets/{id}/rotate", api.auth(true, http.HandlerFunc(api.rotateSecret)))
	mux.Handle("POST /v1/vault/leases", api.auth(true, http.HandlerFunc(api.createLease)))
	mux.Handle("POST /v1/vault/leases/{id}/redeem", api.auth(false, http.HandlerFunc(api.redeemLease)))

	mux.Handle("GET /v1/relay/providers", api.auth(true, http.HandlerFunc(api.listProviders)))
	mux.Handle("POST /v1/relay/providers", api.auth(true, http.HandlerFunc(api.createProvider)))
	mux.Handle("POST /v1/relay/chat/completions", api.auth(false, http.HandlerFunc(api.relayChat)))

	mux.Handle("GET /v1/memory", api.auth(false, http.HandlerFunc(api.searchMemory)))
	mux.Handle("POST /v1/memory", api.auth(true, http.HandlerFunc(api.createMemory)))
	mux.Handle("POST /v1/training/certifications", api.auth(true, http.HandlerFunc(api.addCertification)))

	mux.Handle("GET /v1/flows", api.auth(true, http.HandlerFunc(api.listFlows)))
	mux.Handle("POST /v1/flows", api.auth(true, http.HandlerFunc(api.createFlow)))
	mux.Handle("GET /v1/flows/runs", api.auth(true, http.HandlerFunc(api.listFlowRuns)))
	mux.Handle("POST /v1/flows/{id}/runs", api.auth(true, http.HandlerFunc(api.startFlow)))
	mux.Handle("GET /v1/flow-runs/{id}/current", api.auth(true, http.HandlerFunc(api.currentFlowStep)))
	mux.Handle("POST /v1/flow-runs/{id}/complete", api.auth(true, http.HandlerFunc(api.completeFlowStep)))

	return securityHeaders(mux)
}

func (a *API) auth(adminOnly bool, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimSpace(r.Header.Get("Authorization"))
		if !strings.HasPrefix(raw, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		provided := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
		current := actor{}
		if len(provided) == len(a.token) && subtle.ConstantTimeCompare([]byte(provided), []byte(a.token)) == 1 {
			current.admin = true
		} else if agentID, ok := a.service.AuthenticateAgentKey(provided); ok {
			current.agentID = agentID
		} else {
			writeError(w, http.StatusUnauthorized, "invalid credential")
			return
		}
		if adminOnly && !current.admin {
			writeError(w, http.StatusForbidden, "administrator credential required")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), actorKey{}, current)))
	})
}

func currentActor(r *http.Request) actor {
	value, _ := r.Context().Value(actorKey{}).(actor)
	return value
}

func authorizeAgent(w http.ResponseWriter, r *http.Request, agentID string) bool {
	current := currentActor(r)
	if current.admin || current.agentID == agentID {
		return true
	}
	writeError(w, http.StatusForbidden, "agent credential cannot act for another agent")
	return false
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "version": app.Version})
}

func (a *API) meta(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.service.Meta())
}

func (a *API) ui(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(webui.Index)
}
func (a *API) appJS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	_, _ = w.Write(webui.AppJS)
}
func (a *API) styles(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write(webui.Styles)
}

func (a *API) dashboard(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.service.Dashboard())
}
func (a *API) headcount(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.service.WorkforceHeadcount())
}
func (a *API) export(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=heimda11-export.json")
	writeJSON(w, http.StatusOK, a.service.Export())
}

func (a *API) listAgents(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.service.ListAgents())
}
func (a *API) createAgent(w http.ResponseWriter, r *http.Request) {
	var input core.Agent
	if !decode(w, r, &input) {
		return
	}
	result, err := a.service.CreateAgent(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
func (a *API) getAgent(w http.ResponseWriter, r *http.Request) {
	value, err := a.service.Agent(r.PathValue("id"))
	writeValue(w, value, err, "agent")
}
func (a *API) agentStatus(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status core.AgentStatus `json:"status"`
	}
	if !decode(w, r, &input) {
		return
	}
	value, err := a.service.SetAgentStatus(r.PathValue("id"), input.Status)
	writeValue(w, value, err, "agent")
}
func (a *API) rotateCredential(w http.ResponseWriter, r *http.Request) {
	value, err := a.service.RotateCredential(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err, "agent")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"api_key": value})
}
func (a *API) performance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !authorizeAgent(w, r, id) {
		return
	}
	value, err := a.service.Performance(id)
	writeValue(w, value, err, "agent")
}
func (a *API) agentUsage(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !authorizeAgent(w, r, id) {
		return
	}
	writeJSON(w, http.StatusOK, a.service.Usage(id))
}
func (a *API) agentEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !authorizeAgent(w, r, id) {
		return
	}
	writeJSON(w, http.StatusOK, a.service.Events(id))
}
func (a *API) certifications(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !authorizeAgent(w, r, id) {
		return
	}
	writeJSON(w, http.StatusOK, a.service.Certifications(id))
}

func (a *API) listPolicies(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.service.ListPolicies())
}
func (a *API) createPolicy(w http.ResponseWriter, r *http.Request) {
	var input core.Policy
	if !decode(w, r, &input) {
		return
	}
	value, err := a.service.CreatePolicy(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func (a *API) evaluate(w http.ResponseWriter, r *http.Request) {
	var input core.ActionRequest
	if !decode(w, r, &input) {
		return
	}
	if !authorizeAgent(w, r, input.AgentID) {
		return
	}
	value, err := a.service.Evaluate(input)
	if err != nil {
		writeServiceError(w, err, "agent")
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (a *API) listApprovals(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.service.ListApprovals())
}
func (a *API) resolveApproval(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status     core.ApprovalStatus `json:"status"`
		ResolvedBy string              `json:"resolved_by"`
		Note       string              `json:"note"`
	}
	if !decode(w, r, &input) {
		return
	}
	value, err := a.service.ResolveApproval(r.PathValue("id"), input.Status, input.ResolvedBy, input.Note)
	writeValue(w, value, err, "approval")
}

func (a *API) recordUsage(w http.ResponseWriter, r *http.Request) {
	var input core.UsageRecord
	if !decode(w, r, &input) {
		return
	}
	if !authorizeAgent(w, r, input.AgentID) {
		return
	}
	usage, total, over, err := a.service.RecordUsage(input)
	if err != nil {
		writeServiceError(w, err, "agent")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"usage": usage, "month_spend_usd": total, "budget_exceeded": over})
}
func (a *API) recordEvent(w http.ResponseWriter, r *http.Request) {
	var input core.Event
	if !decode(w, r, &input) {
		return
	}
	if !authorizeAgent(w, r, input.AgentID) {
		return
	}
	value, err := a.service.RecordEvent(input)
	if err != nil {
		writeServiceError(w, err, "agent")
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (a *API) listSecrets(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.service.ListSecrets())
}
func (a *API) createSecret(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name         string `json:"name"`
		Value        string `json:"value"`
		ScopeAgentID string `json:"scope_agent_id"`
	}
	if !decode(w, r, &input) {
		return
	}
	value, err := a.service.CreateSecret(input.Name, input.Value, input.ScopeAgentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func (a *API) rotateSecret(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Value string `json:"value"`
	}
	if !decode(w, r, &input) {
		return
	}
	value, err := a.service.RotateSecret(r.PathValue("id"), input.Value)
	writeValue(w, value, err, "secret")
}
func (a *API) createLease(w http.ResponseWriter, r *http.Request) {
	var input struct {
		SecretID   string `json:"secret_id"`
		AgentID    string `json:"agent_id"`
		TTLSeconds int    `json:"ttl_seconds"`
	}
	if !decode(w, r, &input) {
		return
	}
	value, err := a.service.CreateLease(input.SecretID, input.AgentID, time.Duration(input.TTLSeconds)*time.Second)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func (a *API) redeemLease(w http.ResponseWriter, r *http.Request) {
	current := currentActor(r)
	if current.admin {
		writeError(w, http.StatusBadRequest, "redeem requires an agent credential")
		return
	}
	value, err := a.service.RedeemLease(r.PathValue("id"), current.agentID)
	if err != nil {
		writeServiceError(w, err, "lease")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"value": value})
}

func (a *API) listProviders(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.service.ListProviders())
}
func (a *API) createProvider(w http.ResponseWriter, r *http.Request) {
	var input core.RelayProvider
	if !decode(w, r, &input) {
		return
	}
	value, err := a.service.CreateProvider(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func (a *API) relayChat(w http.ResponseWriter, r *http.Request) {
	var input core.RelayRequest
	if !decodeLimit(w, r, &input, 16<<20) {
		return
	}
	if !authorizeAgent(w, r, input.AgentID) {
		return
	}
	status, body, provider, err := a.service.RelayChat(r.Context(), input)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Heimda11-Provider", provider)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func (a *API) searchMemory(w http.ResponseWriter, r *http.Request) {
	current := currentActor(r)
	agentID := current.agentID
	if current.admin {
		agentID = strings.TrimSpace(r.URL.Query().Get("agent_id"))
	}
	writeJSON(w, http.StatusOK, a.service.SearchMemory(r.URL.Query().Get("q"), r.URL.Query().Get("namespace"), agentID))
}
func (a *API) createMemory(w http.ResponseWriter, r *http.Request) {
	var input core.MemoryItem
	if !decode(w, r, &input) {
		return
	}
	value, err := a.service.CreateMemoryItem(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func (a *API) addCertification(w http.ResponseWriter, r *http.Request) {
	var input core.Certification
	if !decode(w, r, &input) {
		return
	}
	value, err := a.service.AddCertification(input)
	if err != nil {
		writeServiceError(w, err, "agent")
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (a *API) listFlows(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.service.Flows())
}
func (a *API) createFlow(w http.ResponseWriter, r *http.Request) {
	var input core.FlowDefinition
	if !decode(w, r, &input) {
		return
	}
	value, err := a.service.CreateFlow(input)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, value)
}
func (a *API) listFlowRuns(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.service.FlowRuns())
}
func (a *API) startFlow(w http.ResponseWriter, r *http.Request) {
	value, err := a.service.StartFlow(r.PathValue("id"))
	writeValue(w, value, err, "flow")
}
func (a *API) currentFlowStep(w http.ResponseWriter, r *http.Request) {
	run, step, gate, err := a.service.CurrentFlowStep(r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err, "flow run")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"run": run, "step": step, "gate": gate})
}
func (a *API) completeFlowStep(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if !decode(w, r, &input) {
		return
	}
	value, err := a.service.CompleteFlowStep(r.PathValue("id"), input.Status, input.Message)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	return decodeLimit(w, r, dst, 1<<20)
}
func decodeLimit(w http.ResponseWriter, r *http.Request, dst any, limit int64) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "request body must contain one JSON value")
		return false
	}
	return true
}
func writeServiceError(w http.ResponseWriter, err error, kind string) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, kind+" not found")
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}
func writeValue(w http.ResponseWriter, value any, err error, kind string) {
	if err != nil {
		writeServiceError(w, err, kind)
		return
	}
	writeJSON(w, http.StatusOK, value)
}
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func parseInt(value string, fallback int) int {
	v, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return v
}
