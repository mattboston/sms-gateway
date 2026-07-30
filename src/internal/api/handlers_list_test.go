package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	smsgateway "github.com/mattboston/sms-gateway"
	"github.com/mattboston/sms-gateway/internal/auth"
	"github.com/mattboston/sms-gateway/internal/database"
	"github.com/mattboston/sms-gateway/internal/models"
	_ "modernc.org/sqlite"
)

func newListTestRepo(t *testing.T) *database.Repository {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.RunMigrations(db, "sqlite", smsgateway.MigrationsFS); err != nil {
		t.Fatalf("running migrations: %v", err)
	}
	return database.NewRepository(db)
}

// withClaims attaches JWT claims the way the auth middleware does, so handler
// tests can exercise the admin and non-admin paths.
func withClaims(r *http.Request, userID string, isAdmin bool) *http.Request {
	claims := &auth.JWTClaims{UserID: userID, IsAdmin: isAdmin}
	return r.WithContext(context.WithValue(r.Context(), contextKeyUser, claims))
}

func totalHeader(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	return w.Header().Get("X-Total-Count")
}

// --- Users ---

func TestHandleListUsers_Pagination(t *testing.T) {
	repo := newListTestRepo(t)
	handler := NewUserHandler(repo)

	for i := 0; i < 12; i++ {
		if _, err := repo.CreateUser(fmt.Sprintf("user%02d", i), "hash", false, false); err != nil {
			t.Fatalf("CreateUser(%d): %v", i, err)
		}
	}

	tests := []struct {
		name    string
		query   string
		wantLen int
	}{
		{"no limit returns all", "", 12},
		{"first page", "?limit=5", 5},
		{"middle page", "?limit=5&offset=5", 5},
		{"partial last page", "?limit=5&offset=10", 2},
		{"offset past end", "?limit=5&offset=99", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users"+tt.query, nil)
			w := httptest.NewRecorder()
			handler.HandleListUsers(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
			}

			var users []models.User
			if err := json.NewDecoder(w.Body).Decode(&users); err != nil {
				t.Fatalf("decoding: %v", err)
			}
			if len(users) != tt.wantLen {
				t.Errorf("returned %d users, want %d", len(users), tt.wantLen)
			}
			if got := totalHeader(t, w); got != "12" {
				t.Errorf("X-Total-Count = %q, want %q", got, "12")
			}
		})
	}
}

func TestHandleListUsers_PagesDoNotOverlap(t *testing.T) {
	repo := newListTestRepo(t)
	handler := NewUserHandler(repo)

	const total, pageSize = 17, 4
	for i := 0; i < total; i++ {
		_, _ = repo.CreateUser(fmt.Sprintf("u%02d", i), "hash", false, false)
	}

	seen := map[string]int{}
	for offset := 0; offset < total; offset += pageSize {
		url := fmt.Sprintf("/api/v1/users?limit=%d&offset=%d", pageSize, offset)
		req := httptest.NewRequest(http.MethodGet, url, nil)
		w := httptest.NewRecorder()
		handler.HandleListUsers(w, req)

		var users []models.User
		_ = json.NewDecoder(w.Body).Decode(&users)
		for _, u := range users {
			seen[u.ID]++
		}
	}

	if len(seen) != total {
		t.Errorf("paging saw %d distinct users, want %d", len(seen), total)
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("user %s appeared %d times across pages, want 1", id, count)
		}
	}
}

func TestHandleListUsers_InvalidParams(t *testing.T) {
	repo := newListTestRepo(t)
	handler := NewUserHandler(repo)

	for _, query := range []string{"?limit=abc", "?limit=-1", "?offset=-5"} {
		t.Run(query, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/users"+query, nil)
			w := httptest.NewRecorder()
			handler.HandleListUsers(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
		})
	}
}

// --- API keys ---

func seedKeys(t *testing.T, repo *database.Repository, userID, prefix string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if _, err := repo.CreateAPIKey(fmt.Sprintf("%s-key-%02d", prefix, i), "label", userID); err != nil {
			t.Fatalf("CreateAPIKey(%s,%d): %v", prefix, i, err)
		}
	}
}

func TestHandleListAPIKeys_AdminSeesAllWithTotal(t *testing.T) {
	repo := newListTestRepo(t)
	handler := NewKeyHandler(repo)

	admin, _ := repo.CreateUser("admin", "hash", true, false)
	other, _ := repo.CreateUser("other", "hash", false, false)
	seedKeys(t, repo, admin.ID, "admin", 4)
	seedKeys(t, repo, other.ID, "other", 6)

	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/apikeys?limit=3", nil), admin.ID, true)
	w := httptest.NewRecorder()
	handler.HandleListAPIKeys(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var keys []models.APIKey
	_ = json.NewDecoder(w.Body).Decode(&keys)
	if len(keys) != 3 {
		t.Errorf("returned %d keys, want 3", len(keys))
	}
	if got := totalHeader(t, w); got != "10" {
		t.Errorf("X-Total-Count = %q, want %q (all keys)", got, "10")
	}
}

// TestHandleListAPIKeys_NonAdminTotalIsScoped is the one that matters for
// privilege separation: a non-admin must not learn how many keys exist globally.
// The count has to use the same scope as the listing, or X-Total-Count leaks it.
func TestHandleListAPIKeys_NonAdminTotalIsScoped(t *testing.T) {
	repo := newListTestRepo(t)
	handler := NewKeyHandler(repo)

	admin, _ := repo.CreateUser("admin", "hash", true, false)
	user, _ := repo.CreateUser("regular", "hash", false, false)
	seedKeys(t, repo, admin.ID, "admin", 7)
	seedKeys(t, repo, user.ID, "mine", 3)

	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/apikeys", nil), user.ID, false)
	w := httptest.NewRecorder()
	handler.HandleListAPIKeys(w, req)

	var keys []models.APIKey
	_ = json.NewDecoder(w.Body).Decode(&keys)
	if len(keys) != 3 {
		t.Errorf("returned %d keys, want 3 (only the caller's)", len(keys))
	}
	for _, k := range keys {
		if k.UserID != user.ID {
			t.Errorf("returned key belonging to %s, want only %s", k.UserID, user.ID)
		}
	}
	if got := totalHeader(t, w); got != "3" {
		t.Errorf("X-Total-Count = %q, want %q — the global total (10) must not leak", got, "3")
	}
}

func TestHandleListAPIKeys_NonAdminPaginationStaysScoped(t *testing.T) {
	repo := newListTestRepo(t)
	handler := NewKeyHandler(repo)

	admin, _ := repo.CreateUser("admin", "hash", true, false)
	user, _ := repo.CreateUser("regular", "hash", false, false)
	seedKeys(t, repo, admin.ID, "admin", 9)
	seedKeys(t, repo, user.ID, "mine", 5)

	seen := map[string]int{}
	for offset := 0; offset < 5; offset += 2 {
		url := fmt.Sprintf("/api/v1/apikeys?limit=2&offset=%d", offset)
		req := withClaims(httptest.NewRequest(http.MethodGet, url, nil), user.ID, false)
		w := httptest.NewRecorder()
		handler.HandleListAPIKeys(w, req)

		var keys []models.APIKey
		_ = json.NewDecoder(w.Body).Decode(&keys)
		for _, k := range keys {
			if k.UserID != user.ID {
				t.Fatalf("page at offset %d leaked a key owned by %s", offset, k.UserID)
			}
			seen[k.ID]++
		}
		if got := totalHeader(t, w); got != "5" {
			t.Errorf("X-Total-Count = %q, want %q", got, "5")
		}
	}

	if len(seen) != 5 {
		t.Errorf("paging saw %d distinct keys, want 5", len(seen))
	}
	for id, count := range seen {
		if count != 1 {
			t.Errorf("key %s appeared %d times across pages, want 1", id, count)
		}
	}
}

func TestHandleListAPIKeys_Unauthenticated(t *testing.T) {
	repo := newListTestRepo(t)
	handler := NewKeyHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/apikeys", nil)
	w := httptest.NewRecorder()
	handler.HandleListAPIKeys(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestHandleListAPIKeys_EmptyIsArrayNotNull(t *testing.T) {
	repo := newListTestRepo(t)
	handler := NewKeyHandler(repo)

	user, _ := repo.CreateUser("nokeys", "hash", false, false)
	req := withClaims(httptest.NewRequest(http.MethodGet, "/api/v1/apikeys", nil), user.ID, false)
	w := httptest.NewRecorder()
	handler.HandleListAPIKeys(w, req)

	if body := w.Body.String(); body != "[]\n" {
		t.Errorf("body = %q, want %q", body, "[]\n")
	}
	if got := totalHeader(t, w); got != "0" {
		t.Errorf("X-Total-Count = %q, want %q", got, "0")
	}
}
