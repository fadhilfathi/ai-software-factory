package router_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"github.com/fadhilfathi/AI-Software-Factory/internal/middleware"
	"github.com/fadhilfathi/AI-Software-Factory/internal/router"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeAuthService stands in for service.AuthService. It returns canned claims
// for the JWT path (so the role-route matrix can be exercised) and rejects
// every API-key path (we don't need it for the role checks). The other
// interface methods (Login / Refresh / Logout / ValidateRefreshToken) are
// unreachable from these tests because we never POST to /v1/auth/* — we only
// hit protected write/delete routes where the Auth middleware consumes the
// token.
type fakeAuthService struct {
	claimsByToken map[string]*service.Claims
}

func newFakeAuth() *fakeAuthService {
	return &fakeAuthService{claimsByToken: make(map[string]*service.Claims)}
}

func (f *fakeAuthService) Login(_ service.LoginRequest) (*service.LoginResult, *service.Error) {
	return nil, &service.Error{Code: "UNAUTHORIZED", Message: "not used in router tests"}
}

func (f *fakeAuthService) Refresh(_ string) (*service.LoginResult, *service.Error) {
	return nil, &service.Error{Code: "UNAUTHORIZED", Message: "not used in router tests"}
}

func (f *fakeAuthService) Logout(_ string) *service.Error {
	return nil
}

func (f *fakeAuthService) ValidateRefreshToken(_ string) (uuid.UUID, error) {
	return uuid.Nil, errors.New("not used in router tests")
}

func (f *fakeAuthService) ValidateToken(tokenString string) (*service.Claims, error) {
	if c, ok := f.claimsByToken[tokenString]; ok {
		return c, nil
	}
	return nil, errors.New("unknown token")
}

func (f *fakeAuthService) ValidateAPIKey(_ context.Context, _ string) (*service.ValidateAPIKeyResult, *service.Error) {
	return nil, &service.Error{Code: "UNAUTHORIZED", Message: "API key path disabled in router tests"}
}

// buildRouter wires router.New with a *service.Services that has just the
// Auth field populated. The remaining handler constructors receive nil
// dependencies, but that's fine: every test in this file relies on the role
// middleware short-circuiting BEFORE the handler runs, so the nil handlers
// are never invoked.
func buildRouter(t *testing.T, auth *fakeAuthService) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// Build a memory-backed Services so the "role-allow" tests
	// (admin DELETE /v1/tasks/:id, developer PUT /v1/tasks/:id,
	// admin PATCH /v1/executions/:id) reach the handler cleanly
	// and get a 404 "no such task/execution" rather than a
	// nil-pointer panic. The role-deny tests short-circuit at
	// the role middleware and never touch the handlers, so
	// leaving the other services nil is safe.
	st := store.NewMemoryStore()
	log := zap.NewNop()
	svc := &service.Services{
		Auth:      auth,
		Log:       log,
		Task:      service.NewTaskService(st, log),
		Execution: service.NewExecutionService(st, log, nil, aion.NewMockRuntime()),
	}

	r := router.New(svc, middleware.CORSConfig{}, middleware.RateLimitConfig{
		RequestsPerMinute: 10000,
		Burst:             10000,
	})
	return r
}

// issueToken maps a role to a deterministic token so the test body can be
// read like "issueToken(t, auth, "admin")" and stay focused on the role
// branch under test.
func issueToken(t *testing.T, auth *fakeAuthService, role string) string {
	t.Helper()
	tok := "router-test-" + role
	auth.claimsByToken[tok] = &service.Claims{
		UserID: uuid.New().String(),
		Role:   role,
	}
	return tok
}

// send is a tiny helper that issues a method/path/bearer against the engine
// and returns the recorder. We keep the body empty because every route under
// test is short-circuited by the role middleware before the handler would
// read the body.
func send(r *gin.Engine, method, path, bearer string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, err := http.NewRequest(method, path, strings.NewReader(""))
	if err != nil {
		panic(err)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	r.ServeHTTP(w, req)
	return w
}

// -----------------------------------------------------------------------
// 3 admin-only tests
// -----------------------------------------------------------------------

// Admin-only #1: developer on DELETE /v1/projects/:id must be 403, not 200.
func TestRoleMatrix_AdminOnly_DELETE_Project_RejectsDeveloper(t *testing.T) {
	auth := newFakeAuth()
	tok := issueToken(t, auth, "developer")
	r := buildRouter(t, auth)

	w := send(r, http.MethodDelete, "/v1/projects/00000000-0000-0000-0000-000000000000", tok)
	assert.Equal(t, http.StatusForbidden, w.Code, "developer on admin-only DELETE must be 403")
}

// Admin-only #2: viewer on POST /v1/users/register must be 403 (this is the
// route that USED to be public and is the entire reason F-021 is filed).
func TestRoleMatrix_AdminOnly_Register_RejectsViewer(t *testing.T) {
	auth := newFakeAuth()
	tok := issueToken(t, auth, "viewer")
	r := buildRouter(t, auth)

	w := send(r, http.MethodPost, "/v1/users/register", tok)
	assert.Equal(t, http.StatusForbidden, w.Code, "viewer on register must be 403 — was public pre-TASK-425")
}

// Admin-only #3: admin on DELETE /v1/tasks/:id must pass the role gate. The
// handler downstream may 404 (no such task) or 500 (nil service) — either
// way, the request MUST NOT be 401/403. We accept anything that proves the
// role gate let the admin through.
func TestRoleMatrix_AdminOnly_DELETE_Task_AllowsAdmin(t *testing.T) {
	auth := newFakeAuth()
	tok := issueToken(t, auth, "admin")
	r := buildRouter(t, auth)

	w := send(r, http.MethodDelete, "/v1/tasks/00000000-0000-0000-0000-000000000000", tok)
	require.NotEqual(t, http.StatusUnauthorized, w.Code, "admin must clear the auth gate")
	require.NotEqual(t, http.StatusForbidden, w.Code, "admin must clear the role gate")
}

// -----------------------------------------------------------------------
// 3 write-any tests (developer OR admin; viewer is denied)
// -----------------------------------------------------------------------

// Write #1: viewer on POST /v1/projects must be 403 — this is the matrix's
// central promise (viewer is the implicit no-write role).
func TestRoleMatrix_Write_POST_Project_RejectsViewer(t *testing.T) {
	auth := newFakeAuth()
	tok := issueToken(t, auth, "viewer")
	r := buildRouter(t, auth)

	w := send(r, http.MethodPost, "/v1/projects", tok)
	assert.Equal(t, http.StatusForbidden, w.Code, "viewer on POST /v1/projects must be 403")
}

// Write #2: developer on PUT /v1/tasks/:id must clear the role gate.
func TestRoleMatrix_Write_PUT_Task_AllowsDeveloper(t *testing.T) {
	auth := newFakeAuth()
	tok := issueToken(t, auth, "developer")
	r := buildRouter(t, auth)

	w := send(r, http.MethodPut, "/v1/tasks/00000000-0000-0000-0000-000000000000", tok)
	require.NotEqual(t, http.StatusUnauthorized, w.Code)
	require.NotEqual(t, http.StatusForbidden, w.Code, "developer must clear the write-role gate on PUT /v1/tasks/:id")
}

// Write #3: admin on PATCH /v1/executions/:id must clear the role gate.
func TestRoleMatrix_Write_PATCH_Execution_AllowsAdmin(t *testing.T) {
	auth := newFakeAuth()
	tok := issueToken(t, auth, "admin")
	r := buildRouter(t, auth)

	w := send(r, http.MethodPatch, "/v1/executions/00000000-0000-0000-0000-000000000000", tok)
	require.NotEqual(t, http.StatusUnauthorized, w.Code)
	require.NotEqual(t, http.StatusForbidden, w.Code, "admin must clear the write-role gate on PATCH /v1/executions/:id")
}

// -----------------------------------------------------------------------
// 1 regression test: the public-route set must still let login through
// without a token. This guards against accidentally re-adding register to
// publicRouteSet or accidentally tightening auth globally.
// -----------------------------------------------------------------------

func TestRoleMatrix_Public_Login_NoToken(t *testing.T) {
	auth := newFakeAuth()
	r := buildRouter(t, auth)

	// No Authorization header. The Auth middleware should see this route in
	// publicRouteSet and skip JWT validation. The handler downstream may
	// 400/500 because service is nil; we only care that the request is not
	// rejected at the auth layer.
	w := send(r, http.MethodPost, "/v1/auth/login", "")
	require.NotEqual(t, http.StatusUnauthorized, w.Code, "login must remain in the public-route set")
}
