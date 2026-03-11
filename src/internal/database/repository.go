package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mattboston/sms-gateway/internal/models"
)

// Repository provides data access methods for all domain entities.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new Repository wrapping the given database connection.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Ping verifies the database connection is alive.
func (r *Repository) Ping() error {
	return r.db.Ping()
}

// --- Users ---

// CreateUser inserts a new user and returns the created user.
func (r *Repository) CreateUser(username, passwordHash string, isAdmin, mustChangePassword bool) (*models.User, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := r.db.Exec(
		`INSERT INTO users (id, username, password_hash, is_admin, must_change_password, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id, username, passwordHash, boolToInt(isAdmin), boolToInt(mustChangePassword), now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return r.GetUserByID(id)
}

// SeedDefaultAdmin creates a default admin user if no users exist.
// Returns true if the seed user was created.
func (r *Repository) SeedDefaultAdmin(passwordHash string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("checking user count: %w", err)
	}
	if count > 0 {
		return false, nil
	}

	_, err = r.CreateUser("admin", passwordHash, true, true)
	if err != nil {
		return false, fmt.Errorf("seeding default admin: %w", err)
	}
	return true, nil
}

// UpdatePassword updates a user's password and clears the must_change_password flag.
func (r *Repository) UpdatePassword(userID, passwordHash string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(
		`UPDATE users SET password_hash = ?, must_change_password = 0, updated_at = ? WHERE id = ?`,
		passwordHash, now, userID,
	)
	if err != nil {
		return fmt.Errorf("updating password: %w", err)
	}
	return nil
}

// GetUserByID retrieves a user by their ID.
func (r *Repository) GetUserByID(id string) (*models.User, error) {
	row := r.db.QueryRow(
		`SELECT id, username, password_hash, is_admin, must_change_password, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	)
	return scanUser(row)
}

// GetUserByUsername retrieves a user by their username.
func (r *Repository) GetUserByUsername(username string) (*models.User, error) {
	row := r.db.QueryRow(
		`SELECT id, username, password_hash, is_admin, must_change_password, created_at, updated_at
		 FROM users WHERE username = ?`, username,
	)
	return scanUser(row)
}

// ListUsers returns all users.
func (r *Repository) ListUsers() ([]models.User, error) {
	rows, err := r.db.Query(
		`SELECT id, username, password_hash, is_admin, must_change_password, created_at, updated_at
		 FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		u, err := scanUserRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

// --- API Keys ---

// CreateAPIKey inserts a new API key and returns it.
func (r *Repository) CreateAPIKey(key, label, userID string) (*models.APIKey, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := r.db.Exec(
		`INSERT INTO api_keys (id, key, label, user_id, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 1, ?, ?)`,
		id, key, label, userID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("creating API key: %w", err)
	}

	return r.getAPIKeyByID(id)
}

// GetAPIKeyByKey retrieves an active API key by its key value.
func (r *Repository) GetAPIKeyByKey(key string) (*models.APIKey, error) {
	row := r.db.QueryRow(
		`SELECT id, key, label, user_id, is_active, created_at, updated_at
		 FROM api_keys WHERE key = ? AND is_active = 1`, key,
	)
	return scanAPIKey(row)
}

func (r *Repository) getAPIKeyByID(id string) (*models.APIKey, error) {
	row := r.db.QueryRow(
		`SELECT id, key, label, user_id, is_active, created_at, updated_at
		 FROM api_keys WHERE id = ?`, id,
	)
	return scanAPIKey(row)
}

// ListAPIKeys returns all API keys.
func (r *Repository) ListAPIKeys() ([]models.APIKey, error) {
	rows, err := r.db.Query(
		`SELECT id, key, label, user_id, is_active, created_at, updated_at
		 FROM api_keys ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("listing API keys: %w", err)
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		k, err := scanAPIKeyRows(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, *k)
	}
	return keys, rows.Err()
}

// ListAPIKeysByUserID returns all API keys for a given user.
func (r *Repository) ListAPIKeysByUserID(userID string) ([]models.APIKey, error) {
	rows, err := r.db.Query(
		`SELECT id, key, label, user_id, is_active, created_at, updated_at
		 FROM api_keys WHERE user_id = ? ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("listing API keys by user: %w", err)
	}
	defer rows.Close()

	var keys []models.APIKey
	for rows.Next() {
		k, err := scanAPIKeyRows(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, *k)
	}
	return keys, rows.Err()
}

// DeactivateAPIKey marks an API key as inactive.
func (r *Repository) DeactivateAPIKey(id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(
		`UPDATE api_keys SET is_active = 0, updated_at = ? WHERE id = ?`, now, id,
	)
	if err != nil {
		return fmt.Errorf("deactivating API key: %w", err)
	}
	return nil
}

// DeleteAPIKey permanently removes an API key by ID.
func (r *Repository) DeleteAPIKey(id string) error {
	result, err := r.db.Exec(`DELETE FROM api_keys WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting API key: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("API key not found")
	}
	return nil
}

// --- Messages ---

// CreateMessage inserts a new message and returns it.
func (r *Repository) CreateMessage(direction models.Direction, phoneNumber, body string, status models.MessageStatus, apiKeyID *string) (*models.Message, error) {
	id := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := r.db.Exec(
		`INSERT INTO messages (id, direction, phone_number, body, status, api_key_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, string(direction), phoneNumber, body, string(status), apiKeyID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("creating message: %w", err)
	}

	return r.GetMessage(id)
}

// GetMessage retrieves a message by ID.
func (r *Repository) GetMessage(id string) (*models.Message, error) {
	row := r.db.QueryRow(
		`SELECT id, direction, phone_number, body, status, api_key_id, modem_response, error_message, created_at, updated_at
		 FROM messages WHERE id = ?`, id,
	)
	return scanMessage(row)
}

// ListMessages returns messages filtered by direction and optionally status.
func (r *Repository) ListMessages(direction models.Direction, status *models.MessageStatus) ([]models.Message, error) {
	var rows *sql.Rows
	var err error

	if status != nil {
		rows, err = r.db.Query(
			`SELECT id, direction, phone_number, body, status, api_key_id, modem_response, error_message, created_at, updated_at
			 FROM messages WHERE direction = ? AND status = ? ORDER BY created_at DESC`,
			string(direction), string(*status),
		)
	} else {
		rows, err = r.db.Query(
			`SELECT id, direction, phone_number, body, status, api_key_id, modem_response, error_message, created_at, updated_at
			 FROM messages WHERE direction = ? ORDER BY created_at DESC`,
			string(direction),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("listing messages: %w", err)
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		m, err := scanMessageRows(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, *m)
	}
	return messages, rows.Err()
}

// UpdateMessageStatus updates the status and optionally the modem response or error of a message.
func (r *Repository) UpdateMessageStatus(id string, status models.MessageStatus, modemResponse, errorMessage *string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.Exec(
		`UPDATE messages SET status = ?, modem_response = ?, error_message = ?, updated_at = ? WHERE id = ?`,
		string(status), modemResponse, errorMessage, now, id,
	)
	if err != nil {
		return fmt.Errorf("updating message status: %w", err)
	}
	return nil
}

// MarkMessageRead updates an inbound message's status from "received" to "read".
func (r *Repository) MarkMessageRead(id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.Exec(
		`UPDATE messages SET status = ?, updated_at = ? WHERE id = ? AND direction = ? AND status = ?`,
		string(models.StatusRead), now, id, string(models.DirectionInbound), string(models.StatusReceived),
	)
	if err != nil {
		return fmt.Errorf("marking message as read: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("message not found or already read")
	}
	return nil
}

// MarkMessageUnread updates an inbound message's status from "read" back to "received".
func (r *Repository) MarkMessageUnread(id string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := r.db.Exec(
		`UPDATE messages SET status = ?, updated_at = ? WHERE id = ? AND direction = ? AND status = ?`,
		string(models.StatusReceived), now, id, string(models.DirectionInbound), string(models.StatusRead),
	)
	if err != nil {
		return fmt.Errorf("marking message as unread: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("message not found or already unread")
	}
	return nil
}

// DeleteMessage deletes a message by ID. Returns an error if the message does not exist.
func (r *Repository) DeleteMessage(id string) error {
	result, err := r.db.Exec(`DELETE FROM messages WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("deleting message: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("message not found")
	}
	return nil
}

// GetPendingMessages returns all outbound messages with pending status.
func (r *Repository) GetPendingMessages() ([]models.Message, error) {
	status := models.StatusPending
	return r.ListMessages(models.DirectionOutbound, &status)
}

// --- Scan helpers ---

type scannable interface {
	Scan(dest ...interface{}) error
}

func scanUser(s scannable) (*models.User, error) {
	var u models.User
	var isAdmin, mustChangePassword int
	var createdAt, updatedAt string
	err := s.Scan(&u.ID, &u.Username, &u.PasswordHash, &isAdmin, &mustChangePassword, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("scanning user: %w", err)
	}
	u.IsAdmin = isAdmin != 0
	u.MustChangePassword = mustChangePassword != 0
	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &u, nil
}

func scanUserRows(rows *sql.Rows) (*models.User, error) {
	return scanUser(rows)
}

func scanAPIKey(s scannable) (*models.APIKey, error) {
	var k models.APIKey
	var isActive int
	var createdAt, updatedAt string
	err := s.Scan(&k.ID, &k.Key, &k.Label, &k.UserID, &isActive, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("scanning API key: %w", err)
	}
	k.IsActive = isActive != 0
	k.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	k.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &k, nil
}

func scanAPIKeyRows(rows *sql.Rows) (*models.APIKey, error) {
	return scanAPIKey(rows)
}

func scanMessage(s scannable) (*models.Message, error) {
	var m models.Message
	var direction, status string
	var createdAt, updatedAt string
	err := s.Scan(&m.ID, &direction, &m.PhoneNumber, &m.Body, &status, &m.APIKeyID, &m.ModemResponse, &m.ErrorMessage, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("scanning message: %w", err)
	}
	m.Direction = models.Direction(direction)
	m.Status = models.MessageStatus(status)
	m.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	m.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &m, nil
}

func scanMessageRows(rows *sql.Rows) (*models.Message, error) {
	return scanMessage(rows)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
