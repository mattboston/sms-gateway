package database

import (
	"database/sql"
	"testing"

	"github.com/mattboston/sms-gateway/internal/models"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *Repository {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("opening test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	schema := `
	CREATE TABLE users (
		id TEXT PRIMARY KEY,
		username TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		is_admin INTEGER NOT NULL DEFAULT 0,
		must_change_password INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	CREATE TABLE api_keys (
		id TEXT PRIMARY KEY,
		key TEXT NOT NULL UNIQUE,
		label TEXT NOT NULL DEFAULT '',
		user_id TEXT NOT NULL,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		FOREIGN KEY (user_id) REFERENCES users(id)
	);
	CREATE TABLE messages (
		id TEXT PRIMARY KEY,
		direction TEXT NOT NULL,
		phone_number TEXT NOT NULL,
		body TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending',
		api_key_id TEXT,
		modem_response TEXT,
		error_message TEXT,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL,
		FOREIGN KEY (api_key_id) REFERENCES api_keys(id)
	);`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	return NewRepository(db)
}

func TestCreateUser(t *testing.T) {
	repo := setupTestDB(t)

	user, err := repo.CreateUser("testuser", "hash123", false, false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if user.Username != "testuser" {
		t.Errorf("user.Username = %q, want %q", user.Username, "testuser")
	}
	if user.IsAdmin {
		t.Error("user.IsAdmin should be false")
	}
	if user.ID == "" {
		t.Error("user.ID should not be empty")
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	repo := setupTestDB(t)

	_, err := repo.CreateUser("dup", "hash1", false, false)
	if err != nil {
		t.Fatalf("first CreateUser() error = %v", err)
	}

	_, err = repo.CreateUser("dup", "hash2", false, false)
	if err == nil {
		t.Error("second CreateUser() should fail for duplicate username")
	}
}

func TestGetUserByUsername(t *testing.T) {
	repo := setupTestDB(t)

	_, err := repo.CreateUser("findme", "hash", true, false)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	user, err := repo.GetUserByUsername("findme")
	if err != nil {
		t.Fatalf("GetUserByUsername() error = %v", err)
	}
	if user.Username != "findme" {
		t.Errorf("user.Username = %q, want %q", user.Username, "findme")
	}
	if !user.IsAdmin {
		t.Error("user.IsAdmin should be true")
	}
}

func TestGetUserByUsername_NotFound(t *testing.T) {
	repo := setupTestDB(t)

	_, err := repo.GetUserByUsername("nonexistent")
	if err == nil {
		t.Error("GetUserByUsername() should return error for missing user")
	}
}

func TestSeedDefaultAdmin(t *testing.T) {
	repo := setupTestDB(t)

	seeded, err := repo.SeedDefaultAdmin("adminhash")
	if err != nil {
		t.Fatalf("SeedDefaultAdmin() error = %v", err)
	}
	if !seeded {
		t.Error("SeedDefaultAdmin() should return true on first call")
	}

	seeded, err = repo.SeedDefaultAdmin("adminhash")
	if err != nil {
		t.Fatalf("SeedDefaultAdmin() second call error = %v", err)
	}
	if seeded {
		t.Error("SeedDefaultAdmin() should return false when users exist")
	}
}

func TestUpdatePassword(t *testing.T) {
	repo := setupTestDB(t)

	user, _ := repo.CreateUser("pwuser", "oldhash", false, true)
	if !user.MustChangePassword {
		t.Fatal("user.MustChangePassword should be true initially")
	}

	err := repo.UpdatePassword(user.ID, "newhash")
	if err != nil {
		t.Fatalf("UpdatePassword() error = %v", err)
	}

	updated, _ := repo.GetUserByID(user.ID)
	if updated.MustChangePassword {
		t.Error("MustChangePassword should be false after UpdatePassword")
	}
}

func TestListUsers(t *testing.T) {
	repo := setupTestDB(t)

	_, _ = repo.CreateUser("user1", "hash1", false, false)
	_, _ = repo.CreateUser("user2", "hash2", true, false)

	users, err := repo.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Errorf("ListUsers() returned %d users, want 2", len(users))
	}
}

func TestCreateAPIKey(t *testing.T) {
	repo := setupTestDB(t)

	user, _ := repo.CreateUser("keyuser", "hash", false, false)
	key, err := repo.CreateAPIKey("testapikey123", "test label", user.ID)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}
	if key.Key != "testapikey123" {
		t.Errorf("key.Key = %q, want %q", key.Key, "testapikey123")
	}
	if key.Label != "test label" {
		t.Errorf("key.Label = %q, want %q", key.Label, "test label")
	}
	if !key.IsActive {
		t.Error("key.IsActive should be true")
	}
}

func TestGetAPIKeyByKey(t *testing.T) {
	repo := setupTestDB(t)

	user, _ := repo.CreateUser("keyuser2", "hash", false, false)
	_, _ = repo.CreateAPIKey("findthiskey", "label", user.ID)

	found, err := repo.GetAPIKeyByKey("findthiskey")
	if err != nil {
		t.Fatalf("GetAPIKeyByKey() error = %v", err)
	}
	if found.Key != "findthiskey" {
		t.Errorf("found.Key = %q, want %q", found.Key, "findthiskey")
	}
}

func TestDeactivateAPIKey(t *testing.T) {
	repo := setupTestDB(t)

	user, _ := repo.CreateUser("keyuser3", "hash", false, false)
	key, _ := repo.CreateAPIKey("deactivateme", "label", user.ID)

	err := repo.DeactivateAPIKey(key.ID)
	if err != nil {
		t.Fatalf("DeactivateAPIKey() error = %v", err)
	}

	// Should no longer be findable by key (only active keys returned).
	_, err = repo.GetAPIKeyByKey("deactivateme")
	if err == nil {
		t.Error("GetAPIKeyByKey() should fail for deactivated key")
	}
}

func TestListAPIKeys(t *testing.T) {
	repo := setupTestDB(t)

	user, _ := repo.CreateUser("keyuser4", "hash", false, false)
	_, _ = repo.CreateAPIKey("key1", "label1", user.ID)
	_, _ = repo.CreateAPIKey("key2", "label2", user.ID)

	keys, err := repo.ListAPIKeys()
	if err != nil {
		t.Fatalf("ListAPIKeys() error = %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("ListAPIKeys() returned %d keys, want 2", len(keys))
	}
}

func TestCreateMessage(t *testing.T) {
	repo := setupTestDB(t)

	msg, err := repo.CreateMessage(models.DirectionOutbound, "+15551234567", "Hello", models.StatusPending, nil)
	if err != nil {
		t.Fatalf("CreateMessage() error = %v", err)
	}
	if msg.PhoneNumber != "+15551234567" {
		t.Errorf("msg.PhoneNumber = %q, want %q", msg.PhoneNumber, "+15551234567")
	}
	if msg.Direction != models.DirectionOutbound {
		t.Errorf("msg.Direction = %q, want %q", msg.Direction, models.DirectionOutbound)
	}
	if msg.Status != models.StatusPending {
		t.Errorf("msg.Status = %q, want %q", msg.Status, models.StatusPending)
	}
}

func TestGetMessage(t *testing.T) {
	repo := setupTestDB(t)

	created, _ := repo.CreateMessage(models.DirectionInbound, "+15559876543", "Hi there", models.StatusReceived, nil)

	found, err := repo.GetMessage(created.ID)
	if err != nil {
		t.Fatalf("GetMessage() error = %v", err)
	}
	if found.Body != "Hi there" {
		t.Errorf("found.Body = %q, want %q", found.Body, "Hi there")
	}
}

func TestListMessages(t *testing.T) {
	repo := setupTestDB(t)

	_, _ = repo.CreateMessage(models.DirectionInbound, "+1111", "in1", models.StatusReceived, nil)
	_, _ = repo.CreateMessage(models.DirectionInbound, "+2222", "in2", models.StatusReceived, nil)
	_, _ = repo.CreateMessage(models.DirectionOutbound, "+3333", "out1", models.StatusSent, nil)

	inbound, err := repo.ListMessages(models.DirectionInbound, nil)
	if err != nil {
		t.Fatalf("ListMessages(inbound) error = %v", err)
	}
	if len(inbound) != 2 {
		t.Errorf("inbound messages = %d, want 2", len(inbound))
	}

	outbound, err := repo.ListMessages(models.DirectionOutbound, nil)
	if err != nil {
		t.Fatalf("ListMessages(outbound) error = %v", err)
	}
	if len(outbound) != 1 {
		t.Errorf("outbound messages = %d, want 1", len(outbound))
	}
}

func TestListMessages_FilterByStatus(t *testing.T) {
	repo := setupTestDB(t)

	_, _ = repo.CreateMessage(models.DirectionOutbound, "+1111", "msg1", models.StatusSent, nil)
	_, _ = repo.CreateMessage(models.DirectionOutbound, "+2222", "msg2", models.StatusFailed, nil)
	_, _ = repo.CreateMessage(models.DirectionOutbound, "+3333", "msg3", models.StatusSent, nil)

	sent := models.StatusSent
	filtered, err := repo.ListMessages(models.DirectionOutbound, &sent)
	if err != nil {
		t.Fatalf("ListMessages(outbound, sent) error = %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("filtered messages = %d, want 2", len(filtered))
	}
}

func TestUpdateMessageStatus(t *testing.T) {
	repo := setupTestDB(t)

	msg, _ := repo.CreateMessage(models.DirectionOutbound, "+1111", "test", models.StatusPending, nil)

	err := repo.UpdateMessageStatus(msg.ID, models.StatusSent, nil, nil)
	if err != nil {
		t.Fatalf("UpdateMessageStatus() error = %v", err)
	}

	updated, _ := repo.GetMessage(msg.ID)
	if updated.Status != models.StatusSent {
		t.Errorf("updated.Status = %q, want %q", updated.Status, models.StatusSent)
	}
}

func TestUpdateMessageStatus_WithError(t *testing.T) {
	repo := setupTestDB(t)

	msg, _ := repo.CreateMessage(models.DirectionOutbound, "+1111", "test", models.StatusPending, nil)

	errMsg := "modem timeout"
	err := repo.UpdateMessageStatus(msg.ID, models.StatusFailed, nil, &errMsg)
	if err != nil {
		t.Fatalf("UpdateMessageStatus() error = %v", err)
	}

	updated, _ := repo.GetMessage(msg.ID)
	if updated.Status != models.StatusFailed {
		t.Errorf("updated.Status = %q, want %q", updated.Status, models.StatusFailed)
	}
	if updated.ErrorMessage == nil || *updated.ErrorMessage != "modem timeout" {
		t.Errorf("updated.ErrorMessage = %v, want %q", updated.ErrorMessage, "modem timeout")
	}
}

func TestMarkMessageRead(t *testing.T) {
	repo := setupTestDB(t)

	msg, _ := repo.CreateMessage(models.DirectionInbound, "+1111", "hello", models.StatusReceived, nil)

	err := repo.MarkMessageRead(msg.ID)
	if err != nil {
		t.Fatalf("MarkMessageRead() error = %v", err)
	}

	updated, _ := repo.GetMessage(msg.ID)
	if updated.Status != models.StatusRead {
		t.Errorf("status = %q, want %q", updated.Status, models.StatusRead)
	}
}

func TestMarkMessageRead_AlreadyRead(t *testing.T) {
	repo := setupTestDB(t)

	msg, _ := repo.CreateMessage(models.DirectionInbound, "+1111", "hello", models.StatusReceived, nil)
	_ = repo.MarkMessageRead(msg.ID)

	err := repo.MarkMessageRead(msg.ID)
	if err == nil {
		t.Error("MarkMessageRead() should fail for already-read message")
	}
}

func TestMarkMessageRead_OutboundMessage(t *testing.T) {
	repo := setupTestDB(t)

	msg, _ := repo.CreateMessage(models.DirectionOutbound, "+1111", "hello", models.StatusSent, nil)

	err := repo.MarkMessageRead(msg.ID)
	if err == nil {
		t.Error("MarkMessageRead() should fail for outbound message")
	}
}

func TestPing(t *testing.T) {
	repo := setupTestDB(t)
	if err := repo.Ping(); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}
}
