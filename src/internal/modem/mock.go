package modem

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// MockModem simulates a GSM modem for development and testing.
type MockModem struct {
	mu       sync.Mutex
	messages []MockMessage
}

// MockMessage holds a message sent through the mock modem.
type MockMessage struct {
	To   string
	Body string
}

// NewMockModem creates a new mock modem.
func NewMockModem() *MockModem {
	log.Println("[mock modem] initialized")
	return &MockModem{}
}

// SendSMS simulates sending an SMS message.
func (m *MockModem) SendSMS(to, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Printf("[mock modem] SendSMS to=%s body=%q", to, body)
	m.messages = append(m.messages, MockMessage{To: to, Body: body})
	return nil
}

// CheckStatus simulates a modem status check.
func (m *MockModem) CheckStatus() error {
	log.Println("[mock modem] CheckStatus: OK")
	return nil
}

// GetSignal returns a fake signal strength.
func (m *MockModem) GetSignal() (int, error) {
	log.Println("[mock modem] GetSignal: 20")
	return 20, nil
}

// SendAT simulates sending a raw AT command.
func (m *MockModem) SendAT(cmd string) (string, error) {
	log.Printf("[mock modem] SendAT: %s", cmd)
	return fmt.Sprintf("OK (mock response to %s)", cmd), nil
}

// StartReceiver does nothing in mock mode.
func (m *MockModem) StartReceiver(_ context.Context, _ func(from, body string) error) {
	log.Println("[mock modem] StartReceiver: no-op in mock mode")
}

// Close does nothing in mock mode.
func (m *MockModem) Close() error {
	log.Println("[mock modem] Close")
	return nil
}

// SentMessages returns all messages sent through the mock modem.
func (m *MockModem) SentMessages() []MockMessage {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]MockMessage, len(m.messages))
	copy(out, m.messages)
	return out
}
