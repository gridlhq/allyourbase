package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/allyourbase/ayb/internal/testutil"
)

func TestHandleAdminSQLNoPool(t *testing.T) {
	handler := handleAdminSQL(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/sql", strings.NewReader(`{"query":"SELECT 1"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.Equal(t, http.StatusServiceUnavailable, w.Code)
	testutil.Contains(t, w.Body.String(), "database not available")
}

func TestHandleAdminSQLEmptyQuery(t *testing.T) {
	handler := handleAdminSQL(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/sql", strings.NewReader(`{"query":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.Equal(t, http.StatusBadRequest, w.Code)
	testutil.Contains(t, w.Body.String(), "query is required")
}

func TestHandleAdminSQLInvalidJSON(t *testing.T) {
	handler := handleAdminSQL(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/sql", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandleAdminSQLWhitespaceOnlyQuery(t *testing.T) {
	handler := handleAdminSQL(nil)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/sql", strings.NewReader(`{"query":"   "}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.Equal(t, http.StatusBadRequest, w.Code)
	testutil.Contains(t, w.Body.String(), "query is required")
}

func TestSqlResponseJSON(t *testing.T) {
	resp := sqlResponse{
		Columns:    []string{"id", "name"},
		Rows:       [][]any{{1, "alice"}, {2, "bob"}},
		RowCount:   2,
		DurationMs: 5,
	}

	data, err := json.Marshal(resp)
	testutil.NoError(t, err)

	var decoded sqlResponse
	err = json.Unmarshal(data, &decoded)
	testutil.NoError(t, err)

	testutil.Equal(t, 2, decoded.RowCount)
	testutil.Equal(t, 2, len(decoded.Columns))
	testutil.Equal(t, "id", decoded.Columns[0])
	testutil.Equal(t, "name", decoded.Columns[1])
	testutil.Equal(t, int64(5), decoded.DurationMs)
}

func TestSqlResponseEmptyRows(t *testing.T) {
	resp := sqlResponse{
		Columns:    []string{"id"},
		Rows:       [][]any{},
		RowCount:   0,
		DurationMs: 1,
	}

	data, err := json.Marshal(resp)
	testutil.NoError(t, err)
	testutil.Contains(t, string(data), `"rows":[]`)
	testutil.Contains(t, string(data), `"rowCount":0`)
}

func TestToJSONSafe(t *testing.T) {
	// nil
	testutil.Nil(t, toJSONSafe(nil))

	// time.Time → RFC3339
	ts := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	result := toJSONSafe(ts)
	testutil.Equal(t, "2026-02-09T12:00:00Z", result)

	// []byte JSON → parsed
	jsonBytes := []byte(`{"key":"value"}`)
	parsed := toJSONSafe(jsonBytes)
	m, ok := parsed.(map[string]any)
	testutil.True(t, ok, "JSON bytes should parse to map")
	testutil.Equal(t, "value", m["key"])

	// []byte non-JSON → string
	plainBytes := []byte("hello world")
	testutil.Equal(t, "hello world", toJSONSafe(plainBytes))

	// string passthrough
	testutil.Equal(t, "test", toJSONSafe("test"))

	// int passthrough
	testutil.Equal(t, 42, toJSONSafe(42))

	// bool passthrough
	testutil.Equal(t, true, toJSONSafe(true))
}

func TestQueryTimeout(t *testing.T) {
	testutil.Equal(t, 30*time.Second, QueryTimeout)
}
