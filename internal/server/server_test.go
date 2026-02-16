package server_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/allyourbase/ayb/internal/config"
	"github.com/allyourbase/ayb/internal/schema"
	"github.com/allyourbase/ayb/internal/server"
	"github.com/allyourbase/ayb/internal/testutil"
)

func newTestServer(t *testing.T, schemaCache *schema.CacheHolder) *server.Server {
	t.Helper()
	cfg := config.Default()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return server.New(cfg, logger, schemaCache, nil, nil, nil)
}

// newCacheHolderWithSchema creates a CacheHolder with an optional pre-loaded schema for tests.
func newCacheHolderWithSchema(sc *schema.SchemaCache) *schema.CacheHolder {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ch := schema.NewCacheHolder(nil, logger)
	if sc != nil {
		ch.SetForTesting(sc)
	}
	return ch
}

func TestHealthEndpoint(t *testing.T) {
	ch := newCacheHolderWithSchema(nil)
	srv := newTestServer(t, ch)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, http.StatusOK, w.Code)
	testutil.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	testutil.NoError(t, err)
	testutil.Equal(t, "ok", body["status"])
}

func TestSchemaEndpointNotReady(t *testing.T) {
	ch := newCacheHolderWithSchema(nil)
	srv := newTestServer(t, ch)

	req := httptest.NewRequest(http.MethodGet, "/api/schema", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, http.StatusServiceUnavailable, w.Code)

	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &body)
	testutil.NoError(t, err)
	testutil.Equal(t, 503, body.Code)
}

func TestSchemaEndpointReady(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ch := schema.NewCacheHolder(nil, logger)
	ch.SetForTesting(&schema.SchemaCache{
		Tables: map[string]*schema.Table{
			"public.users": {Schema: "public", Name: "users", Kind: "table"},
		},
		Schemas: []string{"public"},
		BuiltAt: time.Now(),
	})

	srv := newTestServer(t, ch)

	req := httptest.NewRequest(http.MethodGet, "/api/schema", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, http.StatusOK, w.Code)
	testutil.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	testutil.NoError(t, err)
	testutil.NotNil(t, body["tables"])
}

func TestRouterSetup(t *testing.T) {
	ch := newCacheHolderWithSchema(nil)
	srv := newTestServer(t, ch)

	// Health endpoint exists.
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	testutil.Equal(t, http.StatusOK, w.Code)

	// Schema endpoint exists (returns 503 since cache not loaded).
	req = httptest.NewRequest(http.MethodGet, "/api/schema", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	testutil.Equal(t, http.StatusServiceUnavailable, w.Code)

	// Unknown API route returns 404.
	req = httptest.NewRequest(http.MethodGet, "/api/nonexistent", nil)
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	testutil.Equal(t, http.StatusNotFound, w.Code)
}

func TestHealthEndpointReturnsJSON(t *testing.T) {
	ch := newCacheHolderWithSchema(nil)
	srv := newTestServer(t, ch)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	// Verify valid JSON.
	var result map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &result)
	testutil.NoError(t, err)
	testutil.Equal(t, "ok", result["status"])
}

// TestSchemaEndpointWithLoadedCache tests the 200 case using SetForTesting.
func TestSchemaEndpointWithLoadedCache(t *testing.T) {
	sc := &schema.SchemaCache{
		Tables: map[string]*schema.Table{
			"public.posts": {Schema: "public", Name: "posts", Kind: "table"},
		},
		Schemas: []string{"public"},
		BuiltAt: time.Now(),
	}
	ch := newCacheHolderWithSchema(sc)
	srv := newTestServer(t, ch)

	req := httptest.NewRequest(http.MethodGet, "/api/schema", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, http.StatusOK, w.Code)
	testutil.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &body)
	testutil.NoError(t, err)
	testutil.NotNil(t, body["tables"])
}

// TestCacheHolderGetBeforeLoad verifies that Get() returns nil before Load().
func TestCacheHolderGetBeforeLoad(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ch := schema.NewCacheHolder(nil, logger)

	got := ch.Get()
	testutil.True(t, got == nil, "expected nil before Load()")
}

func TestOpenAPISpecEndpoint(t *testing.T) {
	ch := newCacheHolderWithSchema(nil)
	srv := newTestServer(t, ch)

	req := httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil)
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, http.StatusOK, w.Code)
	testutil.Equal(t, "application/yaml", w.Header().Get("Content-Type"))
	testutil.Contains(t, w.Body.String(), "openapi: 3.0.3")
	testutil.Contains(t, w.Body.String(), "AllYourBase API")
}

// TestCacheHolderReadyChannel verifies the ready channel is open before Load().
func TestCacheHolderReadyChannel(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ch := schema.NewCacheHolder(nil, logger)

	// Ready channel should not be closed yet.
	select {
	case <-ch.Ready():
		t.Fatal("ready channel should not be closed before Load()")
	default:
		// Expected.
	}
}

