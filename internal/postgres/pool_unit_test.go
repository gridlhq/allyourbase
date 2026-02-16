package postgres

import (
	"testing"
	"time"

	"github.com/allyourbase/ayb/internal/testutil"
)

func TestConfigDefaults(t *testing.T) {
	var cfg Config
	testutil.Equal(t, cfg.URL, "")
	testutil.Equal(t, cfg.MaxConns, int32(0))
	testutil.Equal(t, cfg.MinConns, int32(0))
	testutil.Equal(t, cfg.MaxConnLifetime, time.Duration(0))
	testutil.Equal(t, cfg.MaxConnIdleTime, time.Duration(0))
	testutil.Equal(t, cfg.HealthCheckSecs, 0)
}

func TestNewEmptyURL(t *testing.T) {
	// Empty URL should be rejected before attempting any connection.
	_, err := New(t.Context(), Config{URL: ""}, testutil.DiscardLogger())
	testutil.ErrorContains(t, err, "database URL is required")
}

func TestNewInvalidURLFormat(t *testing.T) {
	// A URL that pgxpool.ParseConfig cannot parse should return a parse error.
	_, err := New(t.Context(), Config{URL: "://bad"}, testutil.DiscardLogger())
	testutil.True(t, err != nil, "expected error for invalid URL format")
	testutil.ErrorContains(t, err, "parsing database URL")
}

func TestNewUnreachableHost(t *testing.T) {
	// A syntactically valid URL pointing at a host that doesn't exist.
	// pgxpool.NewWithConfig or Ping should fail.
	_, err := New(t.Context(), Config{
		URL:      "postgresql://nouser:nopass@127.0.0.1:1/nodb?connect_timeout=1",
		MaxConns: 1,
		MinConns: 0,
	}, testutil.DiscardLogger())
	testutil.True(t, err != nil, "expected error for unreachable host")
}

func TestConfigFieldAssignment(t *testing.T) {
	cfg := Config{
		URL:             "postgresql://localhost:5432/test",
		MaxConns:        10,
		MinConns:        2,
		MaxConnLifetime: 30 * time.Minute,
		MaxConnIdleTime: 5 * time.Minute,
		HealthCheckSecs: 15,
	}
	testutil.Equal(t, cfg.URL, "postgresql://localhost:5432/test")
	testutil.Equal(t, cfg.MaxConns, int32(10))
	testutil.Equal(t, cfg.MinConns, int32(2))
	testutil.Equal(t, cfg.MaxConnLifetime, 30*time.Minute)
	testutil.Equal(t, cfg.MaxConnIdleTime, 5*time.Minute)
	testutil.Equal(t, cfg.HealthCheckSecs, 15)
}
