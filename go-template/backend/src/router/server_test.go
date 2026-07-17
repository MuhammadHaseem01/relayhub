package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewServerRegistersRoutes(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("route registration panicked: %v", recovered)
		}
	}()
	_ = NewServer(nil)
}

func TestNewRouterRegistersCargonexRoutes(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("gin route registration panicked: %v", recovered)
		}
	}()
	_ = NewRouter(nil)
}

func TestRouterRoot(t *testing.T) {
	router := NewRouter(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api", nil)

	router.Engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestRouterProtectedRouteUnauthorized(t *testing.T) {
	router := NewRouter(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/client", nil)

	router.Engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/client status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestRouterPreflight(t *testing.T) {
	router := NewRouter(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/client", nil)

	router.Engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS /api/client status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestRouterTrailingSlash(t *testing.T) {
	router := NewRouter(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/client/", nil)

	router.Engine.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/client/ status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
