package api

import (
	"encoding/json"
	"net/http"

	"github.com/mattboston/sms-gateway/internal/models"
	"github.com/mattboston/sms-gateway/internal/modem"
)

// ModemHandler handles modem-related endpoints.
type ModemHandler struct {
	modem modem.Modem
}

// NewModemHandler creates a new ModemHandler.
func NewModemHandler(m modem.Modem) *ModemHandler {
	return &ModemHandler{modem: m}
}

// HandleModemStatus checks whether the modem is responsive.
//
// @Summary      Get modem status
// @Description  Checks whether the GSM modem is responsive and returns its current status.
// @Tags         Modem
// @Produce      json
// @Success      200  {object}  models.ModemStatusResponse
// @Failure      503  {object}  models.ModemStatusResponse
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /api/v1/modem/status [get]
func (h *ModemHandler) HandleModemStatus(w http.ResponseWriter, _ *http.Request) {
	if err := h.modem.CheckStatus(); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ModemStatusResponse{Status: "error"})
		return
	}

	writeJSON(w, http.StatusOK, models.ModemStatusResponse{Status: "ok"})
}

// HandleModemSignal returns the modem signal strength.
//
// @Summary      Get modem signal strength
// @Description  Returns the GSM modem signal strength and quality assessment.
// @Tags         Modem
// @Produce      json
// @Success      200  {object}  models.ModemSignalResponse
// @Failure      503  {object}  models.ErrorResponse
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /api/v1/modem/signal [get]
func (h *ModemHandler) HandleModemSignal(w http.ResponseWriter, _ *http.Request) {
	signal, err := h.modem.GetSignal()
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, models.ErrorResponse{Error: "failed to get signal strength"})
		return
	}

	quality := signalQuality(signal)
	writeJSON(w, http.StatusOK, models.ModemSignalResponse{Signal: signal, Quality: quality})
}

// HandleSendATCommand sends a raw AT command to the modem (admin only).
//
// @Summary      Send AT command
// @Description  Sends a raw AT command to the GSM modem and returns the response. Requires admin privileges.
// @Tags         Modem
// @Accept       json
// @Produce      json
// @Param        request  body      models.ATCommandRequest   true  "AT command to send"
// @Success      200      {object}  models.ATCommandResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      500      {object}  models.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/modem/at [post]
func (h *ModemHandler) HandleSendATCommand(w http.ResponseWriter, r *http.Request) {
	var req models.ATCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Command == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "command is required"})
		return
	}

	resp, err := h.modem.SendAT(req.Command)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, models.ATCommandResponse{Response: resp})
}

func signalQuality(signal int) string {
	switch {
	case signal == 99:
		return "unknown"
	case signal >= 20:
		return "excellent"
	case signal >= 15:
		return "good"
	case signal >= 10:
		return "fair"
	case signal >= 2:
		return "poor"
	default:
		return "none"
	}
}
