//go:build integration

package auth_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/allyourbase/ayb/internal/auth"
	"github.com/allyourbase/ayb/internal/config"
	"github.com/allyourbase/ayb/internal/migrations"
	"github.com/allyourbase/ayb/internal/schema"
	"github.com/allyourbase/ayb/internal/server"
	"github.com/allyourbase/ayb/internal/testutil"
)

var sharedPG *testutil.PGContainer

func TestMain(m *testing.M) {
	ctx := context.Background()
	pg, cleanup := testutil.StartPostgresForTestMain(ctx)
	sharedPG = pg
	code := m.Run()
	cleanup()
	os.Exit(code)
}

const testJWTSecret = "integration-test-secret-that-is-at-least-32-chars!!"

func resetAndMigrate(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := sharedPG.Pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public")
	if err != nil {
		t.Fatalf("resetting schema: %v", err)
	}

	logger := testutil.DiscardLogger()
	runner := migrations.NewRunner(sharedPG.Pool, logger)
	if err := runner.Bootstrap(ctx); err != nil {
		t.Fatalf("bootstrapping migrations: %v", err)
	}
	if _, err := runner.Run(ctx); err != nil {
		t.Fatalf("running migrations: %v", err)
	}
}

func newAuthService() *auth.Service {
	return auth.NewService(sharedPG.Pool, testJWTSecret, time.Hour, 7*24*time.Hour, 8, testutil.DiscardLogger())
}

func setupAuthServer(t *testing.T, ctx context.Context) *server.Server {
	t.Helper()
	resetAndMigrate(t, ctx)

	logger := testutil.DiscardLogger()
	ch := schema.NewCacheHolder(sharedPG.Pool, logger)
	if err := ch.Load(ctx); err != nil {
		t.Fatalf("loading schema cache: %v", err)
	}

	cfg := config.Default()
	cfg.Auth.Enabled = true
	cfg.Auth.JWTSecret = testJWTSecret

	authSvc := newAuthService()
	return server.New(cfg, logger, ch, sharedPG.Pool, authSvc, nil)
}

func doJSON(t *testing.T, srv *server.Server, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()
	var reqBody *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	return w
}

type authResp struct {
	Token        string         `json:"token"`
	RefreshToken string         `json:"refreshToken"`
	User         map[string]any `json:"user"`
}

func parseAuthResp(t *testing.T, w *httptest.ResponseRecorder) authResp {
	t.Helper()
	var resp authResp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parsing auth response: %v (body: %s)", err, w.Body.String())
	}
	return resp
}

// --- Registration tests ---

func TestRegisterSuccess(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	w := doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "alice@example.com", "password": "password123",
	}, "")

	testutil.Equal(t, http.StatusCreated, w.Code)

	resp := parseAuthResp(t, w)
	testutil.True(t, resp.Token != "", "should return a token")
	testutil.True(t, resp.RefreshToken != "", "should return a refresh token")
	testutil.Equal(t, "alice@example.com", resp.User["email"].(string))
	testutil.True(t, resp.User["id"].(string) != "", "should have user id")
}

func TestRegisterDuplicateEmail(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	body := map[string]string{"email": "dup@example.com", "password": "password123"}
	w := doJSON(t, srv, "POST", "/api/auth/register", body, "")
	testutil.Equal(t, http.StatusCreated, w.Code)

	// Same email again.
	w = doJSON(t, srv, "POST", "/api/auth/register", body, "")
	testutil.Equal(t, http.StatusConflict, w.Code)
}

func TestRegisterDuplicateEmailCaseInsensitive(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	w := doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "User@Example.com", "password": "password123",
	}, "")
	testutil.Equal(t, http.StatusCreated, w.Code)

	// Same email, different case.
	w = doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "user@example.com", "password": "password123",
	}, "")
	testutil.Equal(t, http.StatusConflict, w.Code)
}

// --- Login tests ---

func TestLoginSuccess(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	// Register first.
	doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "login@example.com", "password": "password123",
	}, "")

	// Login.
	w := doJSON(t, srv, "POST", "/api/auth/login", map[string]string{
		"email": "login@example.com", "password": "password123",
	}, "")
	testutil.Equal(t, http.StatusOK, w.Code)

	resp := parseAuthResp(t, w)
	testutil.True(t, resp.Token != "", "should return a token")
	testutil.True(t, resp.RefreshToken != "", "should return a refresh token")
	testutil.Equal(t, "login@example.com", resp.User["email"].(string))
}

func TestLoginWrongPassword(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "wrong@example.com", "password": "password123",
	}, "")

	w := doJSON(t, srv, "POST", "/api/auth/login", map[string]string{
		"email": "wrong@example.com", "password": "wrongpassword",
	}, "")
	testutil.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLoginNonexistentEmail(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	w := doJSON(t, srv, "POST", "/api/auth/login", map[string]string{
		"email": "noone@example.com", "password": "password123",
	}, "")
	// Same status as wrong password — no enumeration.
	testutil.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- /me endpoint tests ---

func TestMeWithRegisterToken(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	w := doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "me@example.com", "password": "password123",
	}, "")
	resp := parseAuthResp(t, w)

	w = doJSON(t, srv, "GET", "/api/auth/me", nil, resp.Token)
	testutil.Equal(t, http.StatusOK, w.Code)

	var user map[string]any
	json.Unmarshal(w.Body.Bytes(), &user)
	testutil.Equal(t, "me@example.com", user["email"].(string))
}

func TestMeWithLoginToken(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "melogin@example.com", "password": "password123",
	}, "")

	w := doJSON(t, srv, "POST", "/api/auth/login", map[string]string{
		"email": "melogin@example.com", "password": "password123",
	}, "")
	resp := parseAuthResp(t, w)

	w = doJSON(t, srv, "GET", "/api/auth/me", nil, resp.Token)
	testutil.Equal(t, http.StatusOK, w.Code)
}

func TestMeWithoutToken(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	w := doJSON(t, srv, "GET", "/api/auth/me", nil, "")
	testutil.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Protected collection endpoints ---

func TestCollectionEndpointRequiresAuth(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	// Create a test table.
	_, err := sharedPG.Pool.Exec(ctx, `
		CREATE TABLE posts (
			id SERIAL PRIMARY KEY,
			title TEXT NOT NULL
		)
	`)
	testutil.NoError(t, err)

	// Reload schema.
	logger := testutil.DiscardLogger()
	ch := schema.NewCacheHolder(sharedPG.Pool, logger)
	testutil.NoError(t, ch.Load(ctx))

	cfg := config.Default()
	cfg.Auth.Enabled = true
	cfg.Auth.JWTSecret = testJWTSecret
	authSvc := newAuthService()
	srv = server.New(cfg, logger, ch, sharedPG.Pool, authSvc, nil)

	// Without token → 401.
	w := doJSON(t, srv, "GET", "/api/collections/posts/", nil, "")
	testutil.Equal(t, http.StatusUnauthorized, w.Code)

	// Register and get token.
	w = doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "authed@example.com", "password": "password123",
	}, "")
	resp := parseAuthResp(t, w)

	// With token → 200.
	w = doJSON(t, srv, "GET", "/api/collections/posts/", nil, resp.Token)
	testutil.Equal(t, http.StatusOK, w.Code)
}

// --- RLS enforcement ---

func TestRLSEnforcement(t *testing.T) {
	ctx := context.Background()
	resetAndMigrate(t, ctx)

	// Create a table with RLS.
	_, err := sharedPG.Pool.Exec(ctx, `
		CREATE TABLE notes (
			id SERIAL PRIMARY KEY,
			owner_id TEXT NOT NULL,
			content TEXT NOT NULL
		);
		ALTER TABLE notes ENABLE ROW LEVEL SECURITY;
		ALTER TABLE notes FORCE ROW LEVEL SECURITY;
		CREATE POLICY notes_owner ON notes
			USING (owner_id = current_setting('ayb.user_id', true));
	`)
	testutil.NoError(t, err)

	logger := testutil.DiscardLogger()
	ch := schema.NewCacheHolder(sharedPG.Pool, logger)
	testutil.NoError(t, ch.Load(ctx))

	cfg := config.Default()
	cfg.Auth.Enabled = true
	cfg.Auth.JWTSecret = testJWTSecret
	authSvc := newAuthService()
	srv := server.New(cfg, logger, ch, sharedPG.Pool, authSvc, nil)

	// Register two users.
	w := doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "user1@example.com", "password": "password123",
	}, "")
	user1 := parseAuthResp(t, w)

	w = doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "user2@example.com", "password": "password123",
	}, "")
	user2 := parseAuthResp(t, w)

	user1ID := user1.User["id"].(string)
	user2ID := user2.User["id"].(string)

	// Insert notes owned by each user (bypass RLS with superuser pool).
	_, err = sharedPG.Pool.Exec(ctx,
		"INSERT INTO notes (owner_id, content) VALUES ($1, 'user1 note'), ($2, 'user2 note')",
		user1ID, user2ID)
	testutil.NoError(t, err)

	// User 1 should only see their note.
	w = doJSON(t, srv, "GET", "/api/collections/notes/", nil, user1.Token)
	testutil.Equal(t, http.StatusOK, w.Code)

	var list1 struct {
		Items []map[string]any `json:"items"`
	}
	json.Unmarshal(w.Body.Bytes(), &list1)
	testutil.Equal(t, 1, len(list1.Items))
	testutil.Equal(t, "user1 note", list1.Items[0]["content"])

	// User 2 should only see their note.
	w = doJSON(t, srv, "GET", "/api/collections/notes/", nil, user2.Token)
	testutil.Equal(t, http.StatusOK, w.Code)

	var list2 struct {
		Items []map[string]any `json:"items"`
	}
	json.Unmarshal(w.Body.Bytes(), &list2)
	testutil.Equal(t, 1, len(list2.Items))
	testutil.Equal(t, "user2 note", list2.Items[0]["content"])
}

// --- Refresh token tests ---

func setupAuthServerWithRefreshDur(t *testing.T, ctx context.Context, refreshDur time.Duration) *server.Server {
	t.Helper()
	resetAndMigrate(t, ctx)

	logger := testutil.DiscardLogger()
	ch := schema.NewCacheHolder(sharedPG.Pool, logger)
	if err := ch.Load(ctx); err != nil {
		t.Fatalf("loading schema cache: %v", err)
	}

	cfg := config.Default()
	cfg.Auth.Enabled = true
	cfg.Auth.JWTSecret = testJWTSecret

	authSvc := auth.NewService(sharedPG.Pool, testJWTSecret, time.Hour, refreshDur, 8, logger)
	return server.New(cfg, logger, ch, sharedPG.Pool, authSvc, nil)
}

func TestRefreshTokenFlow(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	// Register.
	w := doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "refresh@example.com", "password": "password123",
	}, "")
	testutil.Equal(t, http.StatusCreated, w.Code)
	resp := parseAuthResp(t, w)
	testutil.True(t, resp.RefreshToken != "", "should return refresh token")

	// Use refresh token to get new tokens.
	w = doJSON(t, srv, "POST", "/api/auth/refresh", map[string]string{
		"refreshToken": resp.RefreshToken,
	}, "")
	testutil.Equal(t, http.StatusOK, w.Code)
	refreshResp := parseAuthResp(t, w)
	testutil.True(t, refreshResp.Token != "", "should return new access token")
	testutil.True(t, refreshResp.RefreshToken != "", "should return new refresh token")

	// Verify the new access token works on /me.
	w = doJSON(t, srv, "GET", "/api/auth/me", nil, refreshResp.Token)
	testutil.Equal(t, http.StatusOK, w.Code)
}

func TestRefreshTokenExpired(t *testing.T) {
	ctx := context.Background()
	// Use a 1ms refresh duration so it expires immediately.
	srv := setupAuthServerWithRefreshDur(t, ctx, time.Millisecond)

	w := doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "expired@example.com", "password": "password123",
	}, "")
	resp := parseAuthResp(t, w)

	// Wait for the refresh token to expire.
	time.Sleep(50 * time.Millisecond)

	w = doJSON(t, srv, "POST", "/api/auth/refresh", map[string]string{
		"refreshToken": resp.RefreshToken,
	}, "")
	testutil.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogout(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	w := doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "logout@example.com", "password": "password123",
	}, "")
	resp := parseAuthResp(t, w)

	// Logout.
	w = doJSON(t, srv, "POST", "/api/auth/logout", map[string]string{
		"refreshToken": resp.RefreshToken,
	}, "")
	testutil.Equal(t, http.StatusNoContent, w.Code)

	// Refresh with the logged-out token should fail.
	w = doJSON(t, srv, "POST", "/api/auth/refresh", map[string]string{
		"refreshToken": resp.RefreshToken,
	}, "")
	testutil.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- OAuth integration tests ---

func TestOAuthLoginNewUser(t *testing.T) {
	ctx := context.Background()
	resetAndMigrate(t, ctx)

	svc := newAuthService()
	info := &auth.OAuthUserInfo{
		ProviderUserID: "google-123",
		Email:          "oauth@example.com",
		Name:           "OAuth User",
	}

	user, token, refreshToken, err := svc.OAuthLogin(ctx, "google", info)
	testutil.NoError(t, err)
	testutil.True(t, user.ID != "", "should create user")
	testutil.Equal(t, "oauth@example.com", user.Email)
	testutil.True(t, token != "", "should return access token")
	testutil.True(t, refreshToken != "", "should return refresh token")

	// Verify the access token works.
	claims, err := svc.ValidateToken(token)
	testutil.NoError(t, err)
	testutil.Equal(t, user.ID, claims.Subject)
}

func TestOAuthLoginExistingIdentity(t *testing.T) {
	ctx := context.Background()
	resetAndMigrate(t, ctx)

	svc := newAuthService()
	info := &auth.OAuthUserInfo{
		ProviderUserID: "google-456",
		Email:          "repeat@example.com",
		Name:           "Repeat User",
	}

	// First login creates user.
	user1, _, _, err := svc.OAuthLogin(ctx, "google", info)
	testutil.NoError(t, err)

	// Second login with same provider identity returns same user.
	user2, _, _, err := svc.OAuthLogin(ctx, "google", info)
	testutil.NoError(t, err)
	testutil.Equal(t, user1.ID, user2.ID)
}

func TestOAuthLoginLinksToExistingEmailUser(t *testing.T) {
	ctx := context.Background()
	resetAndMigrate(t, ctx)

	svc := newAuthService()

	// Register a user with email/password first.
	emailUser, _, _, err := svc.Register(ctx, "linked@example.com", "password123")
	testutil.NoError(t, err)

	// Login via OAuth with the same email.
	info := &auth.OAuthUserInfo{
		ProviderUserID: "github-789",
		Email:          "linked@example.com",
		Name:           "Linked User",
	}
	oauthUser, _, _, err := svc.OAuthLogin(ctx, "github", info)
	testutil.NoError(t, err)

	// Should be the same user (linked, not a new account).
	testutil.Equal(t, emailUser.ID, oauthUser.ID)
}

func TestOAuthLoginMultipleProvidersSameUser(t *testing.T) {
	ctx := context.Background()
	resetAndMigrate(t, ctx)

	svc := newAuthService()

	// Login via Google.
	googleInfo := &auth.OAuthUserInfo{
		ProviderUserID: "google-multi",
		Email:          "multi@example.com",
		Name:           "Multi User",
	}
	user1, _, _, err := svc.OAuthLogin(ctx, "google", googleInfo)
	testutil.NoError(t, err)

	// Login via GitHub with same email.
	githubInfo := &auth.OAuthUserInfo{
		ProviderUserID: "github-multi",
		Email:          "multi@example.com",
		Name:           "Multi User",
	}
	user2, _, _, err := svc.OAuthLogin(ctx, "github", githubInfo)
	testutil.NoError(t, err)

	// Should be the same user.
	testutil.Equal(t, user1.ID, user2.ID)
}

func TestOAuthLoginNoEmail(t *testing.T) {
	ctx := context.Background()
	resetAndMigrate(t, ctx)

	svc := newAuthService()
	info := &auth.OAuthUserInfo{
		ProviderUserID: "github-noemail",
		Email:          "",
		Name:           "No Email User",
	}

	user, _, _, err := svc.OAuthLogin(ctx, "github", info)
	testutil.NoError(t, err)
	testutil.True(t, user.ID != "", "should create user even without email")
	// Should have a placeholder email.
	testutil.True(t, user.Email != "", "should have placeholder email")
}

func TestOAuthHandlerFullFlowMocked(t *testing.T) {
	ctx := context.Background()
	resetAndMigrate(t, ctx)

	// Set up fake OAuth provider endpoints.
	fakeProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			json.NewEncoder(w).Encode(map[string]string{
				"access_token": "fake-access-token",
			})
		case "/userinfo":
			json.NewEncoder(w).Encode(map[string]any{
				"id":    "12345",
				"email": "fakeuser@example.com",
				"name":  "Fake User",
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer fakeProvider.Close()

	// Override Google's endpoints to point to our fake server.
	auth.SetProviderURLs("google", auth.OAuthProviderConfig{
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    fakeProvider.URL + "/token",
		UserInfoURL: fakeProvider.URL + "/userinfo",
		Scopes:      []string{"openid", "email", "profile"},
	})
	defer auth.ResetProviderURLs("google")

	logger := testutil.DiscardLogger()
	ch := schema.NewCacheHolder(sharedPG.Pool, logger)
	testutil.NoError(t, ch.Load(ctx))

	cfg := config.Default()
	cfg.Auth.Enabled = true
	cfg.Auth.JWTSecret = testJWTSecret
	cfg.Auth.OAuth = map[string]config.OAuthProvider{
		"google": {Enabled: true, ClientID: "test-id", ClientSecret: "test-secret"},
	}
	cfg.Auth.OAuthRedirectURL = "http://localhost:5173/callback"

	svc := newAuthService()
	srv := server.New(cfg, logger, ch, sharedPG.Pool, svc, nil)

	// Step 1: Initiate OAuth → should redirect to Google.
	req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth/google", nil)
	req.Host = "localhost:8090"
	w := httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)
	testutil.Equal(t, http.StatusTemporaryRedirect, w.Code)
	loc := w.Header().Get("Location")
	testutil.True(t, loc != "", "should redirect")

	// Extract state from the redirect URL.
	var state string
	if idx := len("state="); true {
		for _, part := range splitQuery(loc) {
			if len(part) > idx && part[:idx] == "state=" {
				state = part[idx:]
				break
			}
		}
	}
	testutil.True(t, state != "", "redirect should include state")

	// Step 2: Simulate callback from provider.
	callbackURL := fmt.Sprintf("/api/auth/oauth/google/callback?code=test-code&state=%s", state)
	req = httptest.NewRequest(http.MethodGet, callbackURL, nil)
	req.Host = "localhost:8090"
	w = httptest.NewRecorder()
	srv.Router().ServeHTTP(w, req)

	// Should redirect to the configured redirect URL with tokens.
	testutil.Equal(t, http.StatusTemporaryRedirect, w.Code)
	redirectLoc := w.Header().Get("Location")
	testutil.True(t, redirectLoc != "", "should redirect with tokens")
	testutil.True(t, len(redirectLoc) > len("http://localhost:5173/callback#"), "redirect should have fragment")

	// Verify the user was created.
	var count int
	err := sharedPG.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM _ayb_users WHERE email = 'fakeuser@example.com'",
	).Scan(&count)
	testutil.NoError(t, err)
	testutil.Equal(t, 1, count)

	// Verify the OAuth account was linked.
	err = sharedPG.Pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM _ayb_oauth_accounts WHERE provider = 'google' AND provider_user_id = '12345'",
	).Scan(&count)
	testutil.NoError(t, err)
	testutil.Equal(t, 1, count)
}

// splitQuery splits a URL's query string into key=value pairs.
func splitQuery(rawURL string) []string {
	idx := 0
	for i, c := range rawURL {
		if c == '?' {
			idx = i + 1
			break
		}
	}
	if idx == 0 {
		return nil
	}
	query := rawURL[idx:]
	var parts []string
	for _, p := range splitOn(query, '&') {
		parts = append(parts, p)
	}
	return parts
}

func splitOn(s string, sep byte) []string {
	var result []string
	start := 0
	for i := range len(s) {
		if s[i] == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// --- Refresh token rotation tests ---

func TestRefreshTokenRotation(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	// Register and get initial tokens.
	w := doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "refresh@example.com", "password": "password123",
	}, "")
	testutil.Equal(t, http.StatusCreated, w.Code)
	resp1 := parseAuthResp(t, w)
	oldRefreshToken := resp1.RefreshToken

	// Use refresh token to get new tokens.
	w = doJSON(t, srv, "POST", "/api/auth/refresh", map[string]string{
		"refreshToken": oldRefreshToken,
	}, "")
	testutil.Equal(t, http.StatusOK, w.Code)
	resp2 := parseAuthResp(t, w)

	// Verify new tokens are different.
	testutil.NotEqual(t, resp1.Token, resp2.Token)
	testutil.NotEqual(t, resp1.RefreshToken, resp2.RefreshToken)

	// Old refresh token should no longer work (rotation invalidates it).
	w = doJSON(t, srv, "POST", "/api/auth/refresh", map[string]string{
		"refreshToken": oldRefreshToken,
	}, "")
	testutil.Equal(t, http.StatusUnauthorized, w.Code)

	// New refresh token should work.
	w = doJSON(t, srv, "POST", "/api/auth/refresh", map[string]string{
		"refreshToken": resp2.RefreshToken,
	}, "")
	testutil.Equal(t, http.StatusOK, w.Code)
}

func TestRefreshTokenCanOnlyBeUsedOnce(t *testing.T) {
	ctx := context.Background()
	srv := setupAuthServer(t, ctx)

	// Register.
	w := doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "once@example.com", "password": "password123",
	}, "")
	resp := parseAuthResp(t, w)
	refreshToken := resp.RefreshToken

	// First refresh succeeds.
	w = doJSON(t, srv, "POST", "/api/auth/refresh", map[string]string{
		"refreshToken": refreshToken,
	}, "")
	testutil.Equal(t, http.StatusOK, w.Code)

	// Second use of same token fails.
	w = doJSON(t, srv, "POST", "/api/auth/refresh", map[string]string{
		"refreshToken": refreshToken,
	}, "")
	testutil.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRefreshTokenRejectedAfterExpiry(t *testing.T) {
	ctx := context.Background()
	resetAndMigrate(t, ctx)

	// Create auth service with very short refresh token expiry (1 second).
	authSvc := auth.NewService(sharedPG.Pool, testJWTSecret, time.Hour, 1*time.Second, 8, testutil.DiscardLogger())

	logger := testutil.DiscardLogger()
	ch := schema.NewCacheHolder(sharedPG.Pool, logger)
	testutil.NoError(t, ch.Load(ctx))

	cfg := config.Default()
	cfg.Auth.Enabled = true
	cfg.Auth.JWTSecret = testJWTSecret
	srv := server.New(cfg, logger, ch, sharedPG.Pool, authSvc, nil)

	// Register.
	w := doJSON(t, srv, "POST", "/api/auth/register", map[string]string{
		"email": "expiry@example.com", "password": "password123",
	}, "")
	resp := parseAuthResp(t, w)

	// Wait for refresh token to expire.
	time.Sleep(1200 * time.Millisecond)

	// Refresh should fail.
	w = doJSON(t, srv, "POST", "/api/auth/refresh", map[string]string{
		"refreshToken": resp.RefreshToken,
	}, "")
	testutil.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- Verification token tests ---

func TestVerificationTokenReuse(t *testing.T) {
	ctx := context.Background()
	resetAndMigrate(t, ctx)

	authSvc := newAuthService()

	// Create a user.
	user, err := auth.CreateUser(ctx, sharedPG.Pool, "verify@example.com", "password123", 8)
	testutil.NoError(t, err)

	// Manually insert a verification token (simulating SendVerificationEmail).
	token := "test-verification-token-12345"
	hash := hashTokenForTest(token)
	_, err = sharedPG.Pool.Exec(ctx,
		`INSERT INTO _ayb_email_verifications (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		user.ID, hash, time.Now().Add(time.Hour),
	)
	testutil.NoError(t, err)

	// Verify email.
	err = authSvc.ConfirmEmail(ctx, token)
	testutil.NoError(t, err)

	// Check user is verified.
	var verified bool
	err = sharedPG.Pool.QueryRow(ctx,
		`SELECT email_verified FROM _ayb_users WHERE id = $1`, user.ID,
	).Scan(&verified)
	testutil.NoError(t, err)
	testutil.True(t, verified, "email should be verified")

	// Try to use same token again — should fail (token deleted after use).
	err = authSvc.ConfirmEmail(ctx, token)
	testutil.ErrorContains(t, err, "invalid or expired verification token")
}

func TestVerificationTokenExpiry(t *testing.T) {
	ctx := context.Background()
	resetAndMigrate(t, ctx)

	authSvc := newAuthService()

	// Create a user.
	user, err := auth.CreateUser(ctx, sharedPG.Pool, "expired@example.com", "password123", 8)
	testutil.NoError(t, err)

	// Insert an expired verification token.
	token := "expired-token-12345"
	hash := hashTokenForTest(token)
	_, err = sharedPG.Pool.Exec(ctx,
		`INSERT INTO _ayb_email_verifications (user_id, token_hash, expires_at)
		 VALUES ($1, $2, $3)`,
		user.ID, hash, time.Now().Add(-time.Hour), // expired
	)
	testutil.NoError(t, err)

	// Try to verify with expired token.
	err = authSvc.ConfirmEmail(ctx, token)
	testutil.ErrorContains(t, err, "invalid or expired verification token")
}

func TestVerificationTokenInvalidFormat(t *testing.T) {
	ctx := context.Background()
	resetAndMigrate(t, ctx)

	authSvc := newAuthService()

	// Try to verify with invalid token.
	err := authSvc.ConfirmEmail(ctx, "not-a-real-token")
	testutil.ErrorContains(t, err, "invalid or expired verification token")
}

// hashTokenForTest computes SHA-256 hash of a token (same as internal/auth/auth.go).
func hashTokenForTest(token string) string {
	h := fmt.Sprintf("%x", sha256.Sum256([]byte(token)))
	return h
}
