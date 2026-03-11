package models

import "time"

// Direction represents the direction of a message.
type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

// MessageStatus represents the status of a message.
type MessageStatus string

const (
	StatusPending  MessageStatus = "pending"
	StatusSending  MessageStatus = "sending"
	StatusSent     MessageStatus = "sent"
	StatusFailed   MessageStatus = "failed"
	StatusReceived MessageStatus = "received"
	StatusRead     MessageStatus = "read"
)

// User represents a user account.
type User struct {
	ID                 string    `json:"id"`
	Username           string    `json:"username"`
	PasswordHash       string    `json:"-"`
	IsAdmin            bool      `json:"is_admin"`
	MustChangePassword bool      `json:"must_change_password"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// APIKey represents an API key for authenticating requests.
type APIKey struct {
	ID        string    `json:"id"`
	Key       string    `json:"key"`
	Label     string    `json:"label"`
	UserID    string    `json:"user_id"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Message represents an SMS message.
type Message struct {
	ID            string        `json:"id"`
	Direction     Direction     `json:"direction"`
	PhoneNumber   string        `json:"phone_number"`
	Body          string        `json:"body"`
	Status        MessageStatus `json:"status"`
	APIKeyID      *string       `json:"api_key_id,omitempty"`
	ModemResponse *string       `json:"modem_response,omitempty"`
	ErrorMessage  *string       `json:"error_message,omitempty"`
	CreatedAt     time.Time     `json:"created_at"`
	UpdatedAt     time.Time     `json:"updated_at"`
}

// SendSMSRequest is the request body for sending an SMS.
type SendSMSRequest struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

// SendSMSResponse is the response body after sending an SMS.
type SendSMSResponse struct {
	ID      string        `json:"id"`
	Status  MessageStatus `json:"status"`
	Message string        `json:"message,omitempty"`
}

// LoginRequest is the request body for logging in.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is the response body after logging in.
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// CreateAPIKeyRequest is the request body for creating an API key.
type CreateAPIKeyRequest struct {
	Label string `json:"label"`
}

// CreateAPIKeyResponse is the response body after creating an API key.
type CreateAPIKeyResponse struct {
	APIKey APIKey `json:"api_key"`
}

// CreateUserRequest is the request body for creating a user.
type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	IsAdmin  bool   `json:"is_admin"`
}

// ModemStatusResponse is the response body for modem status.
type ModemStatusResponse struct {
	Status string `json:"status"`
}

// ModemSignalResponse is the response body for modem signal strength.
type ModemSignalResponse struct {
	Signal  int    `json:"signal"`
	Quality string `json:"quality"`
}

// ATCommandRequest is the request body for sending a raw AT command.
type ATCommandRequest struct {
	Command string `json:"command"`
}

// ATCommandResponse is the response body for a raw AT command.
type ATCommandResponse struct {
	Response string `json:"response"`
}

// ChangePasswordRequest is the request body for changing a password.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error string `json:"error"`
}
