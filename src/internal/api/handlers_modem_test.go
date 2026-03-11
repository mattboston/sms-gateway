package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattboston/sms-gateway/internal/models"
	"github.com/mattboston/sms-gateway/internal/modem"
)

func TestHandleModemStatus(t *testing.T) {
	mock := modem.NewMockModem()
	handler := NewModemHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/modem/status", nil)
	w := httptest.NewRecorder()

	handler.HandleModemStatus(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp models.ModemStatusResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "ok" {
		t.Errorf("resp.Status = %q, want %q", resp.Status, "ok")
	}
}

func TestHandleModemSignal(t *testing.T) {
	mock := modem.NewMockModem()
	handler := NewModemHandler(mock)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/modem/signal", nil)
	w := httptest.NewRecorder()

	handler.HandleModemSignal(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp models.ModemSignalResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Signal != 20 {
		t.Errorf("resp.Signal = %d, want 20", resp.Signal)
	}
	if resp.Quality != "excellent" {
		t.Errorf("resp.Quality = %q, want %q", resp.Quality, "excellent")
	}
}

func TestHandleSendATCommand(t *testing.T) {
	mock := modem.NewMockModem()
	handler := NewModemHandler(mock)

	body := `{"command":"AT+CSQ"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/modem/at", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleSendATCommand(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp models.ATCommandResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Response == "" {
		t.Error("resp.Response should not be empty")
	}
}

func TestHandleSendATCommand_EmptyCommand(t *testing.T) {
	mock := modem.NewMockModem()
	handler := NewModemHandler(mock)

	body := `{"command":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/modem/at", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleSendATCommand(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSignalQuality(t *testing.T) {
	tests := []struct {
		signal int
		want   string
	}{
		{99, "unknown"},
		{25, "excellent"},
		{20, "excellent"},
		{17, "good"},
		{15, "good"},
		{12, "fair"},
		{10, "fair"},
		{5, "poor"},
		{2, "poor"},
		{1, "none"},
		{0, "none"},
	}

	for _, tt := range tests {
		got := signalQuality(tt.signal)
		if got != tt.want {
			t.Errorf("signalQuality(%d) = %q, want %q", tt.signal, got, tt.want)
		}
	}
}
