package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattboston/sms-gateway/internal/database"
	"github.com/mattboston/sms-gateway/internal/modem"
	_ "modernc.org/sqlite"
)

func setupTestRepo(t *testing.T) *database.Repository {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
	CREATE TABLE users (id TEXT PRIMARY KEY, username TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, is_admin INTEGER NOT NULL DEFAULT 0, must_change_password INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT '');
	CREATE TABLE api_keys (id TEXT PRIMARY KEY, key TEXT NOT NULL UNIQUE, label TEXT NOT NULL DEFAULT '', user_id TEXT NOT NULL, is_active INTEGER NOT NULL DEFAULT 1, created_at TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT '');
	CREATE TABLE messages (id TEXT PRIMARY KEY, direction TEXT NOT NULL, phone_number TEXT NOT NULL, body TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending', api_key_id TEXT, modem_response TEXT, error_message TEXT, created_at TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL DEFAULT '');`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("creating schema: %v", err)
	}
	return database.NewRepository(db)
}

func TestHandleHealth_OK(t *testing.T) {
	repo := setupTestRepo(t)
	mock := modem.NewMockModem()
	handler := NewHealthHandler(repo, mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler.HandleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("resp.Status = %q, want %q", resp.Status, "ok")
	}
	if resp.Database != "ok" {
		t.Errorf("resp.Database = %q, want %q", resp.Database, "ok")
	}
	if resp.Modem != "mock" {
		t.Errorf("resp.Modem = %q, want %q", resp.Modem, "mock")
	}
}
