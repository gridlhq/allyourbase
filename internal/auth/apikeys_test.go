package auth

import (
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/allyourbase/ayb/internal/testutil"
)

func TestIsAPIKey(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{"valid api key prefix", "ayb_abc123def456", true},
		{"just prefix plus one char", "ayb_x", true},
		{"full length key", "ayb_" + hex.EncodeToString(make([]byte, 24)), true},
		{"jwt token", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.abc.def", false},
		{"empty string", "", false},
		{"prefix only no content", "ayb_", false},
		{"similar but wrong prefix", "ayb-abc123", false},
		{"random string", "someothertoken", false},
		{"case sensitive prefix", "AYB_abc123", false},
		{"partial prefix", "ayb", false},
		{"whitespace token", " ayb_abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.Equal(t, IsAPIKey(tt.token), tt.want)
		})
	}
}

func TestAPIKeyPrefix(t *testing.T) {
	testutil.Equal(t, APIKeyPrefix, "ayb_")
}

func TestAPIKeyConstants(t *testing.T) {
	// Key should be ayb_ + 48 hex chars (24 random bytes) = 52 chars total.
	testutil.Equal(t, apiKeyRawBytes, 24)
	testutil.True(t, len(APIKeyPrefix) == 4, "prefix should be 4 chars")

	// Verify the expected key length: prefix(4) + hex(48) = 52 chars.
	expectedKeyLen := len(APIKeyPrefix) + apiKeyRawBytes*2
	testutil.Equal(t, expectedKeyLen, 52)
}

func TestAPIKeyFormat(t *testing.T) {
	// Verify that a generated key has the expected format.
	raw := make([]byte, apiKeyRawBytes)
	_, err := rand.Read(raw)
	testutil.NoError(t, err)

	plaintext := APIKeyPrefix + hex.EncodeToString(raw)
	testutil.Equal(t, len(plaintext), 52)
	testutil.True(t, IsAPIKey(plaintext), "generated key should pass IsAPIKey")

	// Key prefix is first 12 chars.
	prefix := plaintext[:12]
	testutil.Equal(t, len(prefix), 12)
	testutil.True(t, prefix[:4] == "ayb_", "prefix should start with ayb_")
}

func TestAPIKeyUniqueness(t *testing.T) {
	// Two generated keys should be different.
	raw1 := make([]byte, apiKeyRawBytes)
	_, err := rand.Read(raw1)
	testutil.NoError(t, err)

	raw2 := make([]byte, apiKeyRawBytes)
	_, err = rand.Read(raw2)
	testutil.NoError(t, err)

	key1 := APIKeyPrefix + hex.EncodeToString(raw1)
	key2 := APIKeyPrefix + hex.EncodeToString(raw2)
	testutil.True(t, key1 != key2, "two generated keys should not collide")
}

func TestAPIKeyHashConsistency(t *testing.T) {
	// Same plaintext should produce the same hash.
	key := "ayb_abcdef1234567890abcdef1234567890abcdef12345678"
	hash1 := hashToken(key)
	hash2 := hashToken(key)
	testutil.Equal(t, hash1, hash2)

	// Different plaintexts should produce different hashes.
	otherKey := "ayb_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	otherHash := hashToken(otherKey)
	testutil.True(t, hash1 != otherHash, "different keys should have different hashes")
}

func TestAPIKeyErrorSentinels(t *testing.T) {
	// Verify error sentinel values are distinct and have meaningful messages.
	testutil.True(t, ErrAPIKeyNotFound != ErrAPIKeyRevoked, "not found != revoked")
	testutil.True(t, ErrAPIKeyNotFound != ErrAPIKeyExpired, "not found != expired")
	testutil.True(t, ErrAPIKeyRevoked != ErrAPIKeyExpired, "revoked != expired")
	testutil.True(t, ErrInvalidScope != ErrAPIKeyNotFound, "invalid scope != not found")

	testutil.Contains(t, ErrAPIKeyNotFound.Error(), "not found")
	testutil.Contains(t, ErrAPIKeyRevoked.Error(), "revoked")
	testutil.Contains(t, ErrAPIKeyExpired.Error(), "expired")
	testutil.Contains(t, ErrInvalidScope.Error(), "invalid scope")
}

// --- Scope tests ---

func TestValidScopes(t *testing.T) {
	testutil.True(t, ValidScopes[ScopeFullAccess], "* should be valid")
	testutil.True(t, ValidScopes[ScopeReadOnly], "readonly should be valid")
	testutil.True(t, ValidScopes[ScopeReadWrite], "readwrite should be valid")
	testutil.True(t, !ValidScopes["admin"], "admin should not be valid")
	testutil.True(t, !ValidScopes[""], "empty should not be valid")
	testutil.True(t, !ValidScopes["READONLY"], "uppercase should not be valid")
}

func TestScopeConstants(t *testing.T) {
	testutil.Equal(t, ScopeFullAccess, "*")
	testutil.Equal(t, ScopeReadOnly, "readonly")
	testutil.Equal(t, ScopeReadWrite, "readwrite")
}

func TestClaimsIsReadAllowed(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		want  bool
	}{
		{"empty scope (JWT)", "", true},
		{"full access", "*", true},
		{"readonly", "readonly", true},
		{"readwrite", "readwrite", true},
		{"invalid scope", "bogus", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Claims{APIKeyScope: tt.scope}
			testutil.Equal(t, c.IsReadAllowed(), tt.want)
		})
	}
}

func TestClaimsIsWriteAllowed(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		want  bool
	}{
		{"empty scope (JWT)", "", true},
		{"full access", "*", true},
		{"readwrite", "readwrite", true},
		{"readonly blocks writes", "readonly", false},
		{"invalid scope blocks writes", "bogus", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Claims{APIKeyScope: tt.scope}
			testutil.Equal(t, c.IsWriteAllowed(), tt.want)
		})
	}
}

func TestClaimsIsTableAllowed(t *testing.T) {
	tests := []struct {
		name   string
		tables []string
		table  string
		want   bool
	}{
		{"no restrictions", nil, "posts", true},
		{"empty restrictions", []string{}, "posts", true},
		{"allowed table", []string{"posts", "comments"}, "posts", true},
		{"allowed second table", []string{"posts", "comments"}, "comments", true},
		{"denied table", []string{"posts", "comments"}, "users", false},
		{"single table allowed", []string{"posts"}, "posts", true},
		{"single table denied", []string{"posts"}, "comments", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Claims{AllowedTables: tt.tables}
			testutil.Equal(t, c.IsTableAllowed(tt.table), tt.want)
		})
	}
}

func TestCheckWriteScope(t *testing.T) {
	// nil claims should pass (no-auth mode)
	testutil.NoError(t, CheckWriteScope(nil))

	// JWT claims (no scope) should pass
	testutil.NoError(t, CheckWriteScope(&Claims{}))

	// Full access should pass
	testutil.NoError(t, CheckWriteScope(&Claims{APIKeyScope: "*"}))

	// Readwrite should pass
	testutil.NoError(t, CheckWriteScope(&Claims{APIKeyScope: "readwrite"}))

	// Readonly should fail
	err := CheckWriteScope(&Claims{APIKeyScope: "readonly"})
	testutil.True(t, err != nil, "readonly should deny writes")
	testutil.Equal(t, err, ErrScopeReadOnly)
}

func TestCheckTableScope(t *testing.T) {
	// nil claims should pass
	testutil.NoError(t, CheckTableScope(nil, "posts"))

	// No restrictions should pass
	testutil.NoError(t, CheckTableScope(&Claims{}, "posts"))

	// Allowed table should pass
	testutil.NoError(t, CheckTableScope(&Claims{AllowedTables: []string{"posts"}}, "posts"))

	// Denied table should fail
	err := CheckTableScope(&Claims{AllowedTables: []string{"posts"}}, "users")
	testutil.True(t, err != nil, "denied table should fail")
	testutil.Equal(t, err, ErrScopeTableDenied)
}

func TestAPIKeyListResultEmptyItems(t *testing.T) {
	// Verify zero-value result has non-nil items after using the constructor pattern.
	result := &APIKeyListResult{
		Items:      []APIKey{},
		Page:       1,
		PerPage:    20,
		TotalItems: 0,
		TotalPages: 0,
	}
	testutil.Equal(t, len(result.Items), 0)
	testutil.Equal(t, result.Page, 1)
	testutil.NotNil(t, result.Items) // not nil, just empty slice
}

func TestAPIKeyListResultPaginationStruct(t *testing.T) {
	// Verify that APIKeyListResult carries pagination fields correctly.
	result := &APIKeyListResult{
		Items:      []APIKey{{ID: "a"}, {ID: "b"}},
		Page:       2,
		PerPage:    10,
		TotalItems: 25,
		TotalPages: 3,
	}
	testutil.Equal(t, len(result.Items), 2)
	testutil.Equal(t, result.Items[0].ID, "a")
	testutil.Equal(t, result.Page, 2)
	testutil.Equal(t, result.PerPage, 10)
	testutil.Equal(t, result.TotalItems, 25)
	testutil.Equal(t, result.TotalPages, 3)
}

func TestCheckWriteScopeInvalidScope(t *testing.T) {
	// An unrecognized scope should deny writes (fail closed).
	err := CheckWriteScope(&Claims{APIKeyScope: "bogus"})
	testutil.True(t, err != nil, "invalid scope should deny writes")
	testutil.Equal(t, err, ErrScopeReadOnly)
}

func TestCheckTableScopeCaseSensitive(t *testing.T) {
	// Table matching should be case-sensitive.
	c := &Claims{AllowedTables: []string{"Posts"}}
	testutil.True(t, c.IsTableAllowed("Posts"), "exact case should match")
	testutil.True(t, !c.IsTableAllowed("posts"), "lowercase should not match uppercase restriction")
}

func TestCreateAPIKeyOptionsDefaults(t *testing.T) {
	// Verify zero-value opts uses expected defaults.
	opts := CreateAPIKeyOptions{}
	testutil.Equal(t, opts.Scope, "")       // empty means caller should default to "*"
	testutil.True(t, opts.AllowedTables == nil, "nil means all tables")
}

func TestIsAPIKeyLengthBoundary(t *testing.T) {
	// Exactly the prefix length + 1 (minimum valid key).
	testutil.True(t, IsAPIKey("ayb_x"), "prefix + 1 char should be valid")
	// The prefix alone should be invalid.
	testutil.True(t, !IsAPIKey("ayb_"), "prefix alone should not be valid")
}
