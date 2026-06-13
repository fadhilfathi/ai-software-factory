package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestHealthzHandlerSupportsHEAD is the regression test for the Sprint 5
// 6th-bug fix: the Docker / wget --spider healthcheck in
// docker-compose.yml:106 and src/Dockerfile:39 sends HEAD, and
// /v1/healthz must respond 200 on HEAD as well as GET.
func TestHealthzHandlerSupportsHEAD(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/v1/healthz", healthzHandler)
	r.HEAD("/v1/healthz", healthzHandler)

	tests := []struct {
		name   string
		method string
		want   int
	}{
		{"GET /v1/healthz returns 200", http.MethodGet, http.StatusOK},
		{"HEAD /v1/healthz returns 200", http.MethodHead, http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/v1/healthz", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("method=%s code=%d want=%d", tc.method, w.Code, tc.want)
			}
		})
	}
}

// TestHealthzHandlerInPublicRouteSet asserts HEAD is a public route.
// If a future change drops HEAD /v1/healthz from publicRouteSet the
// Docker healthcheck will get 401 and the container will be marked
// unhealthy.
func TestHealthzHandlerInPublicRouteSet(t *testing.T) {
	if !publicRouteSet["HEAD /v1/healthz"] {
		t.Fatal("publicRouteSet missing HEAD /v1/healthz — Docker healthcheck will 401")
	}
	if !publicRouteSet["GET /v1/healthz"] {
		t.Fatal("publicRouteSet missing GET /v1/healthz")
	}
}
