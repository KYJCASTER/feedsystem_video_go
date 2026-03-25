package observability

import (
	"net/http"
	"net/http/httptest"
	"testing"
)
func TestNewPprofMux(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	rr := httptest.NewRecorder()

	NewPprofMux().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", rr.Code)
	}
}
	
func TestStartPprofServerWithDisabled(t *testing.T) {
	t.Parallel()

	server, err := StartPprofServer("api", false, "localhost:6060")
	if err != nil {
		t.Fatalf("Failed to start pprof server: %v", err)
	}
	if server != nil {
		t.Fatalf("Expected nil server when pprof is disabled, got non-nil")
	}
}
