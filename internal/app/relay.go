package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ricardoaldape/Heimda11/internal/core"
)

func (s *Service) CreateProvider(p core.RelayProvider) (core.RelayProvider, error) {
	p.Name = strings.TrimSpace(p.Name)
	p.BaseURL = strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if p.Name == "" || p.BaseURL == "" || p.SecretID == "" {
		return core.RelayProvider{}, errors.New("name, base_url and secret_id are required")
	}
	parsed, err := url.Parse(p.BaseURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return core.RelayProvider{}, errors.New("base_url must use http or https")
	}
	if _, err := s.store.Secret(p.SecretID); err != nil {
		return core.RelayProvider{}, fmt.Errorf("secret: %w", err)
	}
	if p.TimeoutSeconds <= 0 {
		p.TimeoutSeconds = 45
	}
	if p.TimeoutSeconds > 300 {
		return core.RelayProvider{}, errors.New("timeout_seconds cannot exceed 300")
	}
	p.ID = core.NewID("prv")
	p.Enabled = true
	p.CreatedAt = s.now()
	if err := s.store.PutProvider(p); err != nil {
		return core.RelayProvider{}, err
	}
	_ = s.audit("", "relay.provider_created", "success", "provider "+p.Name+" created", map[string]any{"provider_id": p.ID})
	return p, nil
}

func (s *Service) ListProviders() []core.RelayProvider { return s.store.Providers() }

func (s *Service) RelayChat(ctx context.Context, req core.RelayRequest) (int, []byte, string, error) {
	if req.AgentID == "" || len(req.Payload) == 0 {
		return 0, nil, "", errors.New("agent_id and payload are required")
	}
	agent, err := s.store.Agent(req.AgentID)
	if err != nil {
		return 0, nil, "", err
	}
	if agent.Status != core.AgentActive {
		return 0, nil, "", errors.New("agent is not active")
	}
	gate, err := s.Evaluate(core.ActionRequest{AgentID: req.AgentID, Tool: "llm", Action: "chat.completions", TraceID: req.TraceID})
	if err != nil {
		return 0, nil, "", err
	}
	if gate.Decision != core.DecisionAllow {
		if gate.ApprovalID != "" {
			return 0, nil, "", fmt.Errorf("relay blocked by Gate: %s (approval %s)", gate.Reason, gate.ApprovalID)
		}
		return 0, nil, "", errors.New("relay blocked by Gate: " + gate.Reason)
	}
	providers := s.store.Providers()
	var errs []string
	for _, provider := range providers {
		if !provider.Enabled {
			continue
		}
		secret, err := s.secretValueForAgent(provider.SecretID, req.AgentID)
		if err != nil {
			errs = append(errs, provider.Name+": secret unavailable")
			continue
		}
		payload := req.Payload
		if provider.Model != "" {
			var obj map[string]any
			if err := json.Unmarshal(req.Payload, &obj); err != nil {
				return 0, nil, "", errors.New("payload must be a JSON object")
			}
			obj["model"] = provider.Model
			payload, _ = json.Marshal(obj)
		}
		timeout := time.Duration(provider.TimeoutSeconds) * time.Second
		pctx, cancel := context.WithTimeout(ctx, timeout)
		httpReq, err := http.NewRequestWithContext(pctx, http.MethodPost, provider.BaseURL+"/chat/completions", bytes.NewReader(payload))
		if err != nil {
			cancel()
			errs = append(errs, provider.Name+": "+err.Error())
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+secret)
		start := time.Now()
		resp, err := http.DefaultClient.Do(httpReq)
		latency := time.Since(start).Milliseconds()
		if err != nil {
			cancel()
			errs = append(errs, provider.Name+": "+err.Error())
			_ = s.store.AddEvent(core.Event{ID: core.NewID("evt"), AgentID: req.AgentID, Type: "relay.attempt", Status: "error", TraceID: req.TraceID, LatencyMS: latency, Message: provider.Name + ": " + err.Error(), OccurredAt: s.now()})
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
		resp.Body.Close()
		cancel()
		if readErr != nil {
			errs = append(errs, provider.Name+": "+readErr.Error())
			continue
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			_ = s.store.AddEvent(core.Event{ID: core.NewID("evt"), AgentID: req.AgentID, Type: "relay.attempt", Status: "success", TraceID: req.TraceID, LatencyMS: latency, Message: provider.Name, OccurredAt: s.now()})
			return resp.StatusCode, body, provider.Name, nil
		}
		msg := fmt.Sprintf("%s: upstream status %d", provider.Name, resp.StatusCode)
		errs = append(errs, msg)
		_ = s.store.AddEvent(core.Event{ID: core.NewID("evt"), AgentID: req.AgentID, Type: "relay.attempt", Status: "failed", TraceID: req.TraceID, LatencyMS: latency, Message: msg, OccurredAt: s.now()})
	}
	if len(errs) == 0 {
		return 0, nil, "", errors.New("no relay providers are enabled")
	}
	return 0, nil, "", errors.New("all relay providers failed: " + strings.Join(errs, "; "))
}
