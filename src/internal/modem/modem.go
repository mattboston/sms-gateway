package modem

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

// Modem defines the interface for interacting with a GSM modem.
type Modem interface {
	SendSMS(to, body string) error
	CheckStatus() error
	GetSignal() (int, error)
	SendAT(cmd string) (string, error)
	StartReceiver(ctx context.Context, callback func(from, body string) error)
	Close() error
}

// SerialModem communicates with a GSM modem over a serial port.
type SerialModem struct {
	port serial.Port
	mu   sync.Mutex
}

// NewSerialModem opens a serial connection to the modem at the given device path and baud rate.
func NewSerialModem(devicePath string, baudRate int) (*SerialModem, error) {
	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(devicePath, mode)
	if err != nil {
		return nil, fmt.Errorf("opening serial port %s: %w", devicePath, err)
	}

	if err := port.SetReadTimeout(5 * time.Second); err != nil {
		port.Close()
		return nil, fmt.Errorf("setting read timeout: %w", err)
	}

	m := &SerialModem{port: port}

	// Give the modem a moment to stabilize after port open.
	time.Sleep(500 * time.Millisecond)

	// Disable echo so responses don't include the command we sent.
	if _, err := m.SendAT("ATE0"); err != nil {
		port.Close()
		return nil, fmt.Errorf("disabling echo: %w", err)
	}

	// Set text mode for SMS (as opposed to PDU mode).
	if _, err := m.SendAT(ATSetTextMode); err != nil {
		port.Close()
		return nil, fmt.Errorf("setting text mode: %w", err)
	}

	// Enable detailed error messages.
	if _, err := m.SendAT("AT+CMEE=1"); err != nil {
		// Not fatal — some modems don't support this.
		log.Printf("Warning: could not enable detailed errors: %v", err)
	}

	return m, nil
}

// SendSMS sends an SMS message to the given phone number.
// If the message contains non-GSM characters (e.g., emoji, unicode),
// it automatically switches to UCS-2 encoding and switches back after.
func (m *SerialModem) SendSMS(to, body string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	needsUCS2 := !isGSM7(body)

	if needsUCS2 {
		// Switch to UCS-2 charset and set Data Coding Scheme for unicode/emoji support.
		if _, err := sendCommand(m.port, ATSetCharsetUCS2, 2*time.Second); err != nil {
			return fmt.Errorf("switching to UCS-2: %w", err)
		}
		if _, err := sendCommand(m.port, ATSetDCSUCS2, 2*time.Second); err != nil {
			if _, restoreErr := sendCommand(m.port, ATSetCharsetGSM, 2*time.Second); restoreErr != nil {
				log.Printf("Warning: failed to restore GSM charset after UCS-2 setup error: %v", restoreErr)
			}
			return fmt.Errorf("setting UCS-2 DCS: %w", err)
		}
	}

	// Build the send command and message body.
	var cmd string
	var msgData []byte
	if needsUCS2 {
		cmd = fmt.Sprintf("%s\"%s\"", ATSendSMS, encodeUCS2(to))
		msgData = append([]byte(encodeUCS2(body)), 0x1A)
	} else {
		cmd = fmt.Sprintf("%s\"%s\"", ATSendSMS, to)
		msgData = append([]byte(body), 0x1A)
	}

	// Step 1: Send AT+CMGS="<number>" and wait for the ">" prompt.
	if err := sendPromptCommand(m.port, cmd, 5*time.Second); err != nil {
		if needsUCS2 {
			// Best-effort restore to GSM charset and DCS.
			if _, restoreErr := sendCommand(m.port, ATSetDCSDefault, 2*time.Second); restoreErr != nil {
				log.Printf("Warning: failed to restore default DCS after prompt error: %v", restoreErr)
			}
			if _, restoreErr := sendCommand(m.port, ATSetCharsetGSM, 2*time.Second); restoreErr != nil {
				log.Printf("Warning: failed to restore GSM charset after prompt error: %v", restoreErr)
			}
		}
		return fmt.Errorf("waiting for SMS prompt: %w", err)
	}

	// Step 2: Send the message body followed by Ctrl+Z (0x1A).
	resp, err := sendRawData(m.port, msgData, 30*time.Second)

	// Always restore GSM charset and DCS if we switched.
	if needsUCS2 {
		if _, restoreErr := sendCommand(m.port, ATSetDCSDefault, 2*time.Second); restoreErr != nil {
			log.Printf("Warning: failed to restore default DCS after send: %v", restoreErr)
		}
		if _, restoreErr := sendCommand(m.port, ATSetCharsetGSM, 2*time.Second); restoreErr != nil {
			log.Printf("Warning: failed to restore GSM charset after send: %v", restoreErr)
		}
	}

	if err != nil {
		return fmt.Errorf("sending SMS body: %w", err)
	}

	// Verify we got a +CMGS response (message reference number).
	if !containsCMGS(resp) {
		return fmt.Errorf("unexpected send response: %s", resp)
	}

	return nil
}

// CheckStatus sends an AT command to check if the modem is responsive.
func (m *SerialModem) CheckStatus() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := sendCommand(m.port, ATCheck, 2*time.Second)
	if err != nil {
		return fmt.Errorf("modem not responding: %w", err)
	}
	return nil
}

// GetSignal queries the modem for signal strength and returns a value 0-31.
func (m *SerialModem) GetSignal() (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	resp, err := sendCommand(m.port, ATSignalQuality, 2*time.Second)
	if err != nil {
		return 0, fmt.Errorf("querying signal strength: %w", err)
	}

	return parseSignalStrength(resp)
}

// SendAT sends a raw AT command and returns the response.
func (m *SerialModem) SendAT(cmd string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	resp, err := sendCommand(m.port, cmd, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("sending AT command %q: %w", cmd, err)
	}
	return resp, nil
}

// StartReceiver polls for incoming SMS messages in a background goroutine.
// The callback should return nil if the message was successfully persisted.
// Only messages that are successfully processed are deleted from the SIM.
func (m *SerialModem) StartReceiver(ctx context.Context, callback func(from, body string) error) {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.mu.Lock()
				resp, err := sendCommand(m.port, ATListUnreadSMS, 5*time.Second)
				m.mu.Unlock()
				if err != nil {
					continue
				}

				messages := parseSMSList(resp)
				for _, msg := range messages {
					if err := callback(msg.From, msg.Body); err != nil {
						log.Printf("Failed to process SMS from %s (SIM index %d): %v", msg.From, msg.Index, err)
						continue
					}
					// Only delete from SIM after successful DB insert.
					m.mu.Lock()
					deleteCmd := fmt.Sprintf("AT+CMGD=%d", msg.Index)
					_, delErr := sendCommand(m.port, deleteCmd, 2*time.Second)
					m.mu.Unlock()
					if delErr != nil {
						log.Printf("Failed to delete SMS at SIM index %d: %v", msg.Index, delErr)
					}
				}
			}
		}
	}()
}

// Close closes the serial port connection.
func (m *SerialModem) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := m.port.Close(); err != nil {
		return fmt.Errorf("closing serial port: %w", err)
	}
	return nil
}

// containsCMGS checks for a +CMGS: response indicating the SMS was accepted.
func containsCMGS(resp string) bool {
	return strings.Contains(resp, "+CMGS:")
}
