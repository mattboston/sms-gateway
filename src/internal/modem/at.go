package modem

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"go.bug.st/serial"
)

// AT command constants.
const (
	ATCheck          = "AT"
	ATSetTextMode    = "AT+CMGF=1"
	ATSignalQuality  = "AT+CSQ"
	ATSendSMS        = "AT+CMGS="
	ATListUnreadSMS  = "AT+CMGL=\"REC UNREAD\""
	ATListAllSMS     = "AT+CMGL=\"ALL\""
	ATDeleteReadSMS  = "AT+CMGD=1,1"
	ATSetCharsetGSM  = "AT+CSCS=\"GSM\""
	ATSetCharsetUCS2 = "AT+CSCS=\"UCS2\""
	ATSetDCSDefault  = "AT+CSMP=17,167,0,0" // GSM 7-bit encoding
	ATSetDCSUCS2     = "AT+CSMP=17,167,0,8" // UCS-2 encoding
)

// ParsedSMS holds a parsed incoming SMS from an AT+CMGL response.
type ParsedSMS struct {
	Index int
	From  string
	Body  string
}

// sendCommand writes an AT command to the serial port and reads until a final
// response line (OK, ERROR, +CME ERROR, +CMS ERROR) is received.
func sendCommand(port serial.Port, cmd string, timeout time.Duration) (string, error) {
	// Drain any stale data in the read buffer before sending.
	drainPort(port)

	if err := port.SetReadTimeout(timeout); err != nil {
		return "", fmt.Errorf("setting timeout: %w", err)
	}

	_, err := port.Write([]byte(cmd + "\r\n"))
	if err != nil {
		return "", fmt.Errorf("writing command: %w", err)
	}

	resp, err := readUntilFinal(port, timeout)
	if err != nil {
		return resp, err
	}

	if containsFinalError(resp) {
		return resp, fmt.Errorf("AT command error: %s", strings.TrimSpace(resp))
	}

	return resp, nil
}

// sendPromptCommand sends an AT command that expects a ">" prompt (e.g., AT+CMGS).
// It waits for the prompt before returning.
func sendPromptCommand(port serial.Port, cmd string, timeout time.Duration) error {
	drainPort(port)

	if err := port.SetReadTimeout(timeout); err != nil {
		return fmt.Errorf("setting timeout: %w", err)
	}

	_, err := port.Write([]byte(cmd + "\r\n"))
	if err != nil {
		return fmt.Errorf("writing command: %w", err)
	}

	// Read until we see the ">" prompt or a final error.
	buf := make([]byte, 256)
	var response strings.Builder
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		n, err := port.Read(buf)
		if err != nil {
			break
		}
		if n > 0 {
			response.Write(buf[:n])
			resp := response.String()
			if strings.Contains(resp, ">") {
				return nil
			}
			if containsFinalError(resp) {
				return fmt.Errorf("AT command error (expected prompt): %s", strings.TrimSpace(resp))
			}
		}
	}

	return fmt.Errorf("timed out waiting for prompt, got: %s", strings.TrimSpace(response.String()))
}

// sendRawData writes raw bytes to the port without appending \r\n,
// then reads until a final response.
func sendRawData(port serial.Port, data []byte, timeout time.Duration) (string, error) {
	if err := port.SetReadTimeout(timeout); err != nil {
		return "", fmt.Errorf("setting timeout: %w", err)
	}

	_, err := port.Write(data)
	if err != nil {
		return "", fmt.Errorf("writing data: %w", err)
	}

	resp, err := readUntilFinal(port, timeout)
	if err != nil {
		return resp, err
	}

	if containsFinalError(resp) {
		return resp, fmt.Errorf("AT command error: %s", strings.TrimSpace(resp))
	}

	return resp, nil
}

// readUntilFinal reads from the port until a final result code is found or timeout.
// Final result codes per 3GPP TS 27.007: OK, ERROR, +CME ERROR, +CMS ERROR.
func readUntilFinal(port serial.Port, timeout time.Duration) (string, error) {
	buf := make([]byte, 4096)
	var response strings.Builder
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		n, err := port.Read(buf)
		if err != nil {
			break
		}
		if n > 0 {
			response.Write(buf[:n])
			if hasFinalResult(response.String()) {
				return response.String(), nil
			}
		}
	}

	resp := response.String()
	if resp == "" {
		return "", fmt.Errorf("timed out with no response")
	}
	return resp, fmt.Errorf("timed out waiting for final result, partial: %s", strings.TrimSpace(resp))
}

// hasFinalResult checks whether the response contains a final result code
// on its own line. This avoids false-matching "OK" inside message bodies.
func hasFinalResult(resp string) bool {
	lines := strings.Split(resp, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "OK":
			return true
		case trimmed == "ERROR":
			return true
		case strings.HasPrefix(trimmed, "+CME ERROR:"):
			return true
		case strings.HasPrefix(trimmed, "+CMS ERROR:"):
			return true
		}
	}
	return false
}

// containsFinalError checks if the response contains an error result code on its own line.
func containsFinalError(resp string) bool {
	lines := strings.Split(resp, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "ERROR":
			return true
		case strings.HasPrefix(trimmed, "+CME ERROR:"):
			return true
		case strings.HasPrefix(trimmed, "+CMS ERROR:"):
			return true
		}
	}
	return false
}

// drainPort reads and discards any pending data from the serial port.
func drainPort(port serial.Port) {
	_ = port.SetReadTimeout(50 * time.Millisecond)
	buf := make([]byte, 1024)
	for {
		n, _ := port.Read(buf)
		if n == 0 {
			break
		}
	}
}

// parseSignalStrength parses the AT+CSQ response and returns the signal value (0-31).
// Response format: +CSQ: <rssi>,<ber>
func parseSignalStrength(resp string) (int, error) {
	idx := strings.Index(resp, "+CSQ:")
	if idx < 0 {
		return 0, fmt.Errorf("unexpected CSQ response: %s", resp)
	}

	after := strings.TrimSpace(resp[idx+5:])
	parts := strings.SplitN(after, ",", 2)
	if len(parts) < 1 {
		return 0, fmt.Errorf("malformed CSQ response: %s", resp)
	}

	val, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, fmt.Errorf("parsing signal value: %w", err)
	}

	return val, nil
}

// isGSM7 checks whether all characters in the string are within the GSM 7-bit default alphabet.
func isGSM7(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
		// GSM 7-bit covers basic ASCII printable chars, CR, LF, and a few extras.
		// Characters outside this set (like emoji) need UCS-2.
	}
	return true
}

// encodeUCS2 encodes a string as a UCS-2 hex string for AT+CMGS in UCS-2 mode.
func encodeUCS2(s string) string {
	runes := []rune(s)
	u16 := utf16.Encode(runes)
	var b strings.Builder
	for _, cp := range u16 {
		b.WriteString(fmt.Sprintf("%04X", cp))
	}
	return b.String()
}

// parseSMSList parses an AT+CMGL response into individual messages.
// Response format:
// +CMGL: <index>,"<status>","<from>",,"<timestamp>"
// <body>
func parseSMSList(resp string) []ParsedSMS {
	var messages []ParsedSMS
	lines := strings.Split(resp, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(line, "+CMGL:") {
			continue
		}

		// Parse the header line to extract index and sender.
		// Format: +CMGL: <index>,"<status>","<from>",...
		parts := strings.SplitN(line, ",", 5)
		if len(parts) < 3 {
			continue
		}
		indexStr := strings.TrimSpace(strings.TrimPrefix(parts[0], "+CMGL:"))
		index, err := strconv.Atoi(indexStr)
		if err != nil {
			continue
		}
		from := strings.Trim(parts[2], "\" ")

		// Collect body lines until the next +CMGL header, OK, or end.
		var bodyLines []string
		for i+1 < len(lines) {
			nextLine := strings.TrimSpace(lines[i+1])
			if nextLine == "" || nextLine == "OK" || strings.HasPrefix(nextLine, "+CMGL:") {
				break
			}
			bodyLines = append(bodyLines, nextLine)
			i++
		}

		if len(bodyLines) > 0 {
			messages = append(messages, ParsedSMS{
				Index: index,
				From:  from,
				Body:  strings.Join(bodyLines, "\n"),
			})
		}
	}

	return messages
}
