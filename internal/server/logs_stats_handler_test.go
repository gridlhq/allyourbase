package server_test

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/allyourbase/ayb/internal/auth"
	"github.com/allyourbase/ayb/internal/config"
	"github.com/allyourbase/ayb/internal/schema"
	"github.com/allyourbase/ayb/internal/server"
	"github.com/allyourbase/ayb/internal/testutil"
)

// newTestServerWithAuth creates a test server with admin auth + auth service for secrets testing.
func newTestServerWithAuth(t *testing.T, password string) (*server.Server, *auth.Service) {
	t.Helper()
	cfg := config.Default()
	cfg.Admin.Password = password
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ch := schema.NewCacheHolder(nil, logger)
	authSvc := auth.NewService(nil, "test-secret-that-is-at-least-32-chars!!", time.Hour, 7*24*time.Hour, 8, logger)
	srv := server.New(cfg, logger, ch, nil, authSvc, nil)
	return srv, authSvc
}

func adminLogin(t *testing.T, srv *server.Server) string {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/auth", strings.NewReader(`{"password":"testpass"}`))
	req.Header.Set("Content-Type", "application/json")
	srv.Router().ServeHTTP(w, req)
	var body map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	return body["token"]
}

// --- Logs endpoint tests ---

func TestAdminLogsReturnsEmptyWithoutLogBuffer(t *testing.T) {
	srv := newTestServerWithPassword(t, "testpass")
	token := adminLogin(t, srv)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/logs/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusOK)
	var body map[string]any
	testutil.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	entries := body["entries"].([]any)
	testutil.Equal(t, len(entries), 0)
	testutil.Contains(t, body["message"].(string), "not enabled")
}

func TestAdminLogsReturnsBufferedEntries(t *testing.T) {
	cfg := config.Default()
	cfg.Admin.Password = "testpass"
	// Create log buffer wrapping a discard handler.
	inner := slog.NewTextHandler(io.Discard, nil)
	lb := server.NewLogBuffer(inner, 100)
	logger := slog.New(lb)

	ch := schema.NewCacheHolder(nil, logger)
	srv := server.New(cfg, logger, ch, nil, nil, nil)
	srv.SetLogBuffer(lb)

	// Log some entries.
	logger.Info("test message one", "key", "value1")
	logger.Warn("test message two", "count", 42)

	token := adminLogin(t, srv)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/logs/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusOK)
	var body map[string]any
	testutil.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	entries := body["entries"].([]any)
	testutil.True(t, len(entries) >= 2, "expected at least 2 log entries")

	// Find our test log entries (server initialization may have logged other entries)
	var testEntries []map[string]any
	for _, e := range entries {
		entry := e.(map[string]any)
		msg := entry["message"].(string)
		if msg == "test message one" || msg == "test message two" {
			testEntries = append(testEntries, entry)
		}
	}

	testutil.Equal(t, len(testEntries), 2)

	// Verify actual entry content, not just structure.
	first := testEntries[0]
	testutil.Equal(t, first["message"], "test message one")
	testutil.Equal(t, first["level"], "INFO")
	testutil.True(t, first["time"] != nil, "entry should have time")

	second := testEntries[1]
	testutil.Equal(t, second["message"], "test message two")
	testutil.Equal(t, second["level"], "WARN")
}

func TestAdminLogsRequiresAuth(t *testing.T) {
	srv := newTestServerWithPassword(t, "testpass")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/logs/", nil)
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusUnauthorized)
	testutil.Contains(t, w.Body.String(), "admin authentication required")
}

// --- Stats endpoint tests ---

func TestAdminStatsReturnsRuntimeInfo(t *testing.T) {
	srv := newTestServerWithPassword(t, "testpass")
	token := adminLogin(t, srv)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusOK)
	var stats map[string]any
	testutil.NoError(t, json.Unmarshal(w.Body.Bytes(), &stats))

	// Verify required fields exist.
	testutil.True(t, stats["uptime_seconds"] != nil, "should have uptime_seconds")
	testutil.True(t, stats["go_version"] != nil, "should have go_version")
	testutil.True(t, stats["goroutines"] != nil, "should have goroutines")
	testutil.True(t, stats["memory_alloc"] != nil, "should have memory_alloc")
	testutil.True(t, stats["memory_sys"] != nil, "should have memory_sys")
	testutil.True(t, stats["gc_cycles"] != nil, "should have gc_cycles")

	// Verify values are reasonable.
	uptime := stats["uptime_seconds"].(float64)
	testutil.True(t, uptime >= 0, "uptime should be non-negative")
	goroutines := stats["goroutines"].(float64)
	testutil.True(t, goroutines > 0, "goroutines should be positive")
	testutil.Contains(t, stats["go_version"].(string), "go1.")
}

func TestAdminStatsNoDBPoolFields(t *testing.T) {
	srv := newTestServerWithPassword(t, "testpass")
	token := adminLogin(t, srv)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusOK)
	var stats map[string]any
	testutil.NoError(t, json.Unmarshal(w.Body.Bytes(), &stats))

	// Without a pool, DB stats should not be present.
	testutil.True(t, stats["db_pool_total"] == nil, "should not have db_pool_total without pool")
}

func TestAdminStatsRequiresAuth(t *testing.T) {
	srv := newTestServerWithPassword(t, "testpass")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats/", nil)
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusUnauthorized)
	testutil.Contains(t, w.Body.String(), "admin authentication required")
}

// --- Secrets rotate endpoint tests ---

func TestAdminSecretsRotateSuccess(t *testing.T) {
	srv, _ := newTestServerWithAuth(t, "testpass")
	token := adminLogin(t, srv)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/secrets/rotate", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusOK)
	var body map[string]string
	testutil.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	testutil.Contains(t, body["message"], "rotated successfully")
}

func TestAdminSecretsRotateInvalidatesOldTokens(t *testing.T) {
	srv, authSvc := newTestServerWithAuth(t, "testpass")
	token := adminLogin(t, srv)

	// Generate a JWT before rotation.
	oldJWT, err := authSvc.IssueTestToken("user-1", "test@example.com")
	testutil.NoError(t, err)
	testutil.True(t, oldJWT != "", "should have generated a token")

	// Rotate.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/secrets/rotate", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)
	testutil.Equal(t, w.Code, http.StatusOK)

	// Old JWT should no longer validate.
	_, err = authSvc.ValidateToken(oldJWT)
	testutil.ErrorContains(t, err, "invalid token")

	// New JWT should validate after rotation.
	newJWT, err := authSvc.IssueTestToken("user-2", "new@example.com")
	testutil.NoError(t, err)
	claims, err := authSvc.ValidateToken(newJWT)
	testutil.NoError(t, err)
	testutil.Equal(t, claims.Email, "new@example.com")
}

func TestAdminSecretsRotateRequiresAuth(t *testing.T) {
	srv, _ := newTestServerWithAuth(t, "testpass")

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/secrets/rotate", nil)
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusUnauthorized)
	testutil.Contains(t, w.Body.String(), "admin authentication required")
}

func TestAdminSecretsNotRegisteredWithoutAuthSvc(t *testing.T) {
	// When authSvc is nil, the secrets route should not be registered.
	srv := newTestServerWithPassword(t, "testpass")
	token := adminLogin(t, srv)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/secrets/rotate", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	srv.Router().ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusNotFound)
}

// --- LogBuffer tests ---

func TestLogBufferCapturesEntries(t *testing.T) {
	inner := slog.NewTextHandler(io.Discard, nil)
	lb := server.NewLogBuffer(inner, 5)
	logger := slog.New(lb)

	logger.Info("one")
	logger.Warn("two")
	logger.Error("three")

	entries := lb.Entries()
	testutil.Equal(t, len(entries), 3)
	testutil.Equal(t, entries[0].Message, "one")
	testutil.Equal(t, entries[0].Level, "INFO")
	testutil.Equal(t, entries[1].Message, "two")
	testutil.Equal(t, entries[1].Level, "WARN")
	testutil.Equal(t, entries[2].Message, "three")
	testutil.Equal(t, entries[2].Level, "ERROR")
}

func TestLogBufferRingOverflow(t *testing.T) {
	inner := slog.NewTextHandler(io.Discard, nil)
	lb := server.NewLogBuffer(inner, 3)
	logger := slog.New(lb)

	logger.Info("a")
	logger.Info("b")
	logger.Info("c")
	logger.Info("d") // overflow: pushes out "a"

	entries := lb.Entries()
	testutil.Equal(t, len(entries), 3)
	testutil.Equal(t, entries[0].Message, "b")
	testutil.Equal(t, entries[1].Message, "c")
	testutil.Equal(t, entries[2].Message, "d")
}

func TestLogBufferCapturesAttrs(t *testing.T) {
	inner := slog.NewTextHandler(io.Discard, nil)
	lb := server.NewLogBuffer(inner, 10)
	logger := slog.New(lb)

	logger.Info("test", "key1", "value1", "key2", 42)

	entries := lb.Entries()
	testutil.Equal(t, len(entries), 1)
	testutil.Equal(t, entries[0].Attrs["key1"].(string), "value1")
	testutil.Equal(t, entries[0].Attrs["key2"].(int64), int64(42))
}

func TestLogBufferEmptyEntries(t *testing.T) {
	inner := slog.NewTextHandler(io.Discard, nil)
	lb := server.NewLogBuffer(inner, 10)

	entries := lb.Entries()
	testutil.Equal(t, len(entries), 0)
}

func TestLogBufferExactCapacity(t *testing.T) {
	inner := slog.NewTextHandler(io.Discard, nil)
	lb := server.NewLogBuffer(inner, 3)
	logger := slog.New(lb)

	// Fill exactly to capacity — no overflow.
	logger.Info("a")
	logger.Info("b")
	logger.Info("c")

	entries := lb.Entries()
	testutil.Equal(t, len(entries), 3)
	testutil.Equal(t, entries[0].Message, "a")
	testutil.Equal(t, entries[2].Message, "c")
}

func TestLogBufferMultipleWraps(t *testing.T) {
	inner := slog.NewTextHandler(io.Discard, nil)
	lb := server.NewLogBuffer(inner, 2)
	logger := slog.New(lb)

	// Write 6 entries into a buffer of size 2 — wraps multiple times.
	for i := 0; i < 6; i++ {
		logger.Info(fmt.Sprintf("msg-%d", i))
	}

	entries := lb.Entries()
	testutil.Equal(t, len(entries), 2)
	testutil.Equal(t, entries[0].Message, "msg-4")
	testutil.Equal(t, entries[1].Message, "msg-5")
}

func TestLogBufferConcurrentLogging(t *testing.T) {
	inner := slog.NewTextHandler(io.Discard, nil)
	lb := server.NewLogBuffer(inner, 100)
	logger := slog.New(lb)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			logger.Info(fmt.Sprintf("goroutine-%d", n))
		}(i)
	}
	wg.Wait()

	entries := lb.Entries()
	testutil.Equal(t, len(entries), 20)
}
