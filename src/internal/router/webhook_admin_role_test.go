package router_test

// F-D002-005 — admin-only role check on POST /v1/webhooks.
//
// D-002 review §3.3 (F-D002-005) noted that POST /v1/webhooks was
// previously gated by `writeRole` (developer OR admin). Combined
// with the X-Project-ID IDOR (F-D002-004), a developer in project
// A could register a webhook in project B by spoofing the header.
// This file's edit (router.go line 115) tightens the gate to
// `adminRole` so only admins can register; the F-D002-004 fix
// (project_memberships) is the second leg.
//
// This test file covers the 4 sub-cases per the dispatch:
//   - viewer with token   → 403 (role gate denies)
//   - developer with token → 403 (role gate denies — the F-D002-005 fix)
//   - admin with token    → 201 (role gate passes; service creates webhook)
//   - no token            → 401 (auth gate denies)
//
// Helpers `fakeAuthService`, `buildRouter`, `issueToken`, and `send`
// are defined in router_role_matrix_test.go and reused from this
// file (same package). The new helpers in this file are
// `buildRouterWithWebhook` and `sendJSON`.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"github.com/fadhilfathi/AI-Software-Factory/internal/middleware"
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/router"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// buildRouterWithWebhook wires router.New with a *service.Services
// that includes a real WebhookService (so the admin allow-case
// reaches the handler and gets a 201). The other services are
// populated the same way as buildRouter in router_role_matrix_test.go.
func buildRouterWithWebhook(t *testing.T, auth *fakeAuthService) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	st := store.NewMemoryStore()
	log := zap.NewNop()
	svc := &service.Services{
		Auth:      auth,
		Log:       log,
		Task:      service.NewTaskService(st, log),
		Execution: service.NewExecutionService(st, log, nil, aion.NewMockRuntime()),
		Webhook:   service.NewWebhookService(st, log),
	}

	return router.New(svc, middleware.CORSConfig{}, middleware.RateLimitConfig{
		RequestsPerMinute: 10000,
		Burst:             10000,
	})
}

// sendJSON is the body-aware version of `send` from
// router_role_matrix_test.go. Used by the admin allow-case where
// the handler actually runs and validates the body.
func sendJSON(t *testing.T, r *gin.Engine, method, path, bearer string, body any) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	var buf []byte
	if body != nil {
		var err error
		buf, err = json.Marshal(body)
		require.NoError(t, err, "marshal body")
	}
	req, err := http.NewRequest(method, path, bytes.NewReader(buf))
	require.NoError(t, err, "new request")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	r.ServeHTTP(w, req)
	return w
}

// validWebhookBody returns a body that passes the WebhookService
// validation: a non-empty URL, a non-empty event list with at
// least one valid event, and a non-empty secret.
func validWebhookBody() map[string]any {
	return map[string]any{
		"url":    "https://example.com/webhook",
		"events": []string{string(model.EventProjectCreated)},
		"secret": "test-secret",
	}
}

// TestWebhookHandler_AdminRole_Required is the 4-sub-case matrix
// for the F-D002-005 fix. The deny-cases (viewer, developer, no
// token) short-circuit at the auth/role middleware and never reach
// the handler, so their bodies are irrelevant. The allow-case
// (admin) sends a valid body and expects a 201.
func TestWebhookHandler_AdminRole_Required(t *testing.T) {
	t.Run("viewer_rejected_at_role_gate", func(t *testing.T) {
		auth := newFakeAuth()
		tok := issueToken(t, auth, "viewer")
		r := buildRouterWithWebhook(t, auth)

		w := sendJSON(t, r, http.MethodPost, "/v1/webhooks", tok, validWebhookBody())
		assert.Equal(t, http.StatusForbidden, w.Code,
			"viewer must be 403 on POST /v1/webhooks (admin-only after F-D002-005), got %d: %s",
			w.Code, w.Body.String())
	})

	t.Run("developer_rejected_at_role_gate", func(t *testing.T) {
		// The F-D002-005 fix: developers used to be allowed
		// (writeRole) — now they're rejected (adminRole).
		auth := newFakeAuth()
		tok := issueToken(t, auth, "developer")
		r := buildRouterWithWebhook(t, auth)

		w := sendJSON(t, r, http.MethodPost, "/v1/webhooks", tok, validWebhookBody())
		assert.Equal(t, http.StatusForbidden, w.Code,
			"developer must be 403 on POST /v1/webhooks (was 200 pre-F-D002-005), got %d: %s",
			w.Code, w.Body.String())
	})

	t.Run("admin_allowed_creates_webhook", func(t *testing.T) {
		auth := newFakeAuth()
		tok := issueToken(t, auth, "admin")
		r := buildRouterWithWebhook(t, auth)

		w := sendJSON(t, r, http.MethodPost, "/v1/webhooks", tok, validWebhookBody())
		assert.Equal(t, http.StatusCreated, w.Code,
			"admin must be 201 on POST /v1/webhooks (role gate passes; service registers), got %d: %s",
			w.Code, w.Body.String())
		// The response body wraps the created webhook in a
		// {data: ...} envelope per the spec; sanity-check
		// the data field is non-empty.
		assert.Contains(t, w.Body.String(), `"data"`,
			"admin response should have a {data: ...} envelope")
	})

	t.Run("unauthenticated_rejected_at_auth_gate", func(t *testing.T) {
		auth := newFakeAuth()
		r := buildRouterWithWebhook(t, auth)

		w := sendJSON(t, r, http.MethodPost, "/v1/webhooks", "", validWebhookBody())
		assert.Equal(t, http.StatusUnauthorized, w.Code,
			"no token must be 401 on POST /v1/webhooks, got %d: %s",
			w.Code, w.Body.String())
	})
}
