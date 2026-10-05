package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ricardoaldape/Heimda11/internal/app"
	"github.com/ricardoaldape/Heimda11/internal/secure"
	"github.com/ricardoaldape/Heimda11/internal/store"
)

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	st, err := store.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := secure.NewCipher(strings.Repeat("22", 32))
	if err != nil {
		t.Fatal(err)
	}
	return New(app.NewService(st, cipher, "pro"), "01234567890123456789012345678901")
}

func request(t *testing.T, h http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var raw []byte
	if body != nil {
		var err error
		raw, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(raw))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAdminAndAgentAuthorization(t *testing.T) {
	h := testHandler(t)
	if rec := request(t, h, http.MethodGet, "/v1/agents", "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	admin := "01234567890123456789012345678901"
	rec := request(t, h, http.MethodPost, "/v1/agents", admin, map[string]any{
		"name": "worker-a", "role": "worker", "human_owner": "owner", "autonomy_level": 1, "allowed_tools": []string{"crm"},
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create agent: %d %s", rec.Code, rec.Body.String())
	}
	var created struct {
		Agent struct {
			ID string `json:"id"`
		} `json:"agent"`
		APIKey string `json:"api_key"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.APIKey == "" || created.Agent.ID == "" {
		t.Fatal("missing created credential or id")
	}

	rec = request(t, h, http.MethodPost, "/v1/watch/events", created.APIKey, map[string]any{
		"agent_id": created.Agent.ID, "type": "task.completed", "status": "success",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("agent event: %d %s", rec.Code, rec.Body.String())
	}

	rec = request(t, h, http.MethodPost, "/v1/watch/events", created.APIKey, map[string]any{
		"agent_id": "agt_someone_else", "type": "task.completed", "status": "success",
	})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-agent action should be forbidden, got %d", rec.Code)
	}
}

func TestSecurityHeadersAndHealth(t *testing.T) {
	h := testHandler(t)
	rec := request(t, h, http.MethodGet, "/healthz", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("health: %d", rec.Code)
	}
	if rec.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatal("missing frame protection")
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("missing CSP")
	}
}