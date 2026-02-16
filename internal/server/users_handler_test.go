package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/allyourbase/ayb/internal/auth"
	"github.com/allyourbase/ayb/internal/testutil"
	"github.com/go-chi/chi/v5"
)

// fakeUserManager is an in-memory fake for testing user management handlers.
type fakeUserManager struct {
	users   []auth.AdminUser
	deleted []string
	listErr error
	delErr  error
}

func (f *fakeUserManager) ListUsers(_ context.Context, page, perPage int, search string) (*auth.UserListResult, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	var filtered []auth.AdminUser
	for _, u := range f.users {
		if search == "" || contains(u.Email, search) {
			filtered = append(filtered, u)
		}
	}

	total := len(filtered)
	start := (page - 1) * perPage
	if start > total {
		start = total
	}
	end := start + perPage
	if end > total {
		end = total
	}

	items := filtered[start:end]
	if items == nil {
		items = []auth.AdminUser{}
	}

	totalPages := total / perPage
	if total%perPage != 0 {
		totalPages++
	}

	return &auth.UserListResult{
		Items:      items,
		Page:       page,
		PerPage:    perPage,
		TotalItems: total,
		TotalPages: totalPages,
	}, nil
}

func (f *fakeUserManager) DeleteUser(_ context.Context, id string) error {
	if f.delErr != nil {
		return f.delErr
	}
	for i, u := range f.users {
		if u.ID == id {
			f.users = append(f.users[:i], f.users[i+1:]...)
			f.deleted = append(f.deleted, id)
			return nil
		}
	}
	return auth.ErrUserNotFound
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchContains(s, substr)))
}

func searchContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func sampleUsers() []auth.AdminUser {
	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	return []auth.AdminUser{
		{ID: "u1", Email: "alice@example.com", EmailVerified: true, CreatedAt: now, UpdatedAt: now},
		{ID: "u2", Email: "bob@example.com", EmailVerified: false, CreatedAt: now, UpdatedAt: now},
		{ID: "u3", Email: "carol@example.com", EmailVerified: true, CreatedAt: now, UpdatedAt: now},
	}
}

// --- List users tests ---

func TestListUsersSuccess(t *testing.T) {
	mgr := &fakeUserManager{users: sampleUsers()}
	handler := handleAdminListUsers(mgr)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusOK)

	var result auth.UserListResult
	err := json.NewDecoder(w.Body).Decode(&result)
	testutil.NoError(t, err)
	testutil.Equal(t, result.TotalItems, 3)
	testutil.Equal(t, len(result.Items), 3)
	testutil.Equal(t, result.Items[0].Email, "alice@example.com")
}

func TestListUsersWithSearch(t *testing.T) {
	mgr := &fakeUserManager{users: sampleUsers()}
	handler := handleAdminListUsers(mgr)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users?search=bob", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusOK)

	var result auth.UserListResult
	err := json.NewDecoder(w.Body).Decode(&result)
	testutil.NoError(t, err)
	testutil.Equal(t, result.TotalItems, 1)
	testutil.Equal(t, result.Items[0].Email, "bob@example.com")
}

func TestListUsersWithPagination(t *testing.T) {
	mgr := &fakeUserManager{users: sampleUsers()}
	handler := handleAdminListUsers(mgr)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users?page=1&perPage=2", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusOK)

	var result auth.UserListResult
	err := json.NewDecoder(w.Body).Decode(&result)
	testutil.NoError(t, err)
	testutil.Equal(t, result.TotalItems, 3)
	testutil.Equal(t, len(result.Items), 2)
	testutil.Equal(t, result.TotalPages, 2)
	testutil.Equal(t, result.Page, 1)
	testutil.Equal(t, result.PerPage, 2)
}

func TestListUsersEmptyResult(t *testing.T) {
	mgr := &fakeUserManager{users: nil}
	handler := handleAdminListUsers(mgr)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusOK)

	var result auth.UserListResult
	err := json.NewDecoder(w.Body).Decode(&result)
	testutil.NoError(t, err)
	testutil.Equal(t, result.TotalItems, 0)
	testutil.Equal(t, len(result.Items), 0)
}

func TestListUsersServiceError(t *testing.T) {
	mgr := &fakeUserManager{listErr: fmt.Errorf("db connection lost")}
	handler := handleAdminListUsers(mgr)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusInternalServerError)
	testutil.Contains(t, w.Body.String(), "failed to list users")
}

// --- Delete user tests ---

func TestDeleteUserSuccess(t *testing.T) {
	mgr := &fakeUserManager{users: sampleUsers()}
	handler := handleAdminDeleteUser(mgr)

	// Use chi to extract URL params.
	r := chi.NewRouter()
	r.Delete("/api/admin/users/{id}", handler)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/users/u2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusNoContent)
	testutil.Equal(t, len(mgr.deleted), 1)
	testutil.Equal(t, mgr.deleted[0], "u2")
	testutil.Equal(t, len(mgr.users), 2) // u2 removed
}

func TestDeleteUserNotFound(t *testing.T) {
	mgr := &fakeUserManager{users: sampleUsers()}
	handler := handleAdminDeleteUser(mgr)

	r := chi.NewRouter()
	r.Delete("/api/admin/users/{id}", handler)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/users/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusNotFound)
	testutil.Contains(t, w.Body.String(), "user not found")
}

func TestDeleteUserServiceError(t *testing.T) {
	mgr := &fakeUserManager{
		users:  sampleUsers(),
		delErr: fmt.Errorf("foreign key constraint"),
	}
	handler := handleAdminDeleteUser(mgr)

	r := chi.NewRouter()
	r.Delete("/api/admin/users/{id}", handler)

	req := httptest.NewRequest(http.MethodDelete, "/api/admin/users/u1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusInternalServerError)
	testutil.Contains(t, w.Body.String(), "failed to delete user")
}

func TestDeleteUserNoID(t *testing.T) {
	mgr := &fakeUserManager{users: sampleUsers()}
	handler := handleAdminDeleteUser(mgr)

	// Call directly without chi routing (no URL param set).
	req := httptest.NewRequest(http.MethodDelete, "/api/admin/users/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusBadRequest)
	testutil.Contains(t, w.Body.String(), "user id is required")
}

func TestListUsersResponseIncludesEmailVerified(t *testing.T) {
	mgr := &fakeUserManager{users: sampleUsers()}
	handler := handleAdminListUsers(mgr)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	testutil.Equal(t, w.Code, http.StatusOK)

	var result auth.UserListResult
	err := json.NewDecoder(w.Body).Decode(&result)
	testutil.NoError(t, err)
	testutil.True(t, result.Items[0].EmailVerified, "alice should be verified")
	testutil.True(t, !result.Items[1].EmailVerified, "bob should not be verified")
}
