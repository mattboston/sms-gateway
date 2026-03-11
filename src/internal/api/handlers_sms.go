package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mattboston/sms-gateway/internal/database"
	"github.com/mattboston/sms-gateway/internal/models"
	"github.com/mattboston/sms-gateway/internal/modem"
)

// SMSHandler handles SMS-related endpoints.
type SMSHandler struct {
	repo  *database.Repository
	modem modem.Modem
}

// NewSMSHandler creates a new SMSHandler.
func NewSMSHandler(repo *database.Repository, m modem.Modem) *SMSHandler {
	return &SMSHandler{repo: repo, modem: m}
}

// HandleSendSMS sends an SMS message.
//
// @Summary      Send SMS
// @Description  Sends an SMS message to the specified phone number via the GSM modem.
// @Tags         SMS
// @Accept       json
// @Produce      json
// @Param        request  body      models.SendSMSRequest   true  "SMS message to send"
// @Success      200      {object}  models.SendSMSResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      500      {object}  models.ErrorResponse
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /api/v1/sms/send [post]
func (h *SMSHandler) HandleSendSMS(w http.ResponseWriter, r *http.Request) {
	var req models.SendSMSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.To == "" || req.Body == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "to and body are required"})
		return
	}

	// Determine API key ID if authenticated via API key.
	var apiKeyID *string
	if ak := GetAPIKeyFromContext(r.Context()); ak != nil {
		apiKeyID = &ak.ID
	}

	// Create the message in pending status.
	msg, err := h.repo.CreateMessage(models.DirectionOutbound, req.To, req.Body, models.StatusPending, apiKeyID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to create message"})
		return
	}

	// Attempt to send via modem.
	if err := h.modem.SendSMS(req.To, req.Body); err != nil {
		errStr := err.Error()
		_ = h.repo.UpdateMessageStatus(msg.ID, models.StatusFailed, nil, &errStr)
		writeJSON(w, http.StatusOK, models.SendSMSResponse{
			ID:      msg.ID,
			Status:  models.StatusFailed,
			Message: "failed to send: " + errStr,
		})
		return
	}

	_ = h.repo.UpdateMessageStatus(msg.ID, models.StatusSent, nil, nil)

	writeJSON(w, http.StatusOK, models.SendSMSResponse{
		ID:      msg.ID,
		Status:  models.StatusSent,
		Message: "message sent",
	})
}

// HandleGetInbox returns inbound messages.
//
// @Summary      Get inbox messages
// @Description  Returns inbound SMS messages. Use status=received for unread only (default), status=read for read messages, or all=true for everything.
// @Tags         SMS
// @Produce      json
// @Param        status  query     string  false  "Filter by status: received (unread), read"
// @Param        all     query     string  false  "Set to true to return all inbound messages regardless of status"
// @Success      200     {array}   models.Message
// @Failure      500     {object}  models.ErrorResponse
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /api/v1/sms/inbox [get]
func (h *SMSHandler) HandleGetInbox(w http.ResponseWriter, r *http.Request) {
	var status *models.MessageStatus

	if s := r.URL.Query().Get("status"); r.URL.Query().Get("all") != "true" && s != "" {
		ms := models.MessageStatus(s)
		status = &ms
	} else if r.URL.Query().Get("all") != "true" {
		// Default: only unread messages.
		s := models.StatusReceived
		status = &s
	}

	messages, err := h.repo.ListMessages(models.DirectionInbound, status)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to list messages"})
		return
	}

	if messages == nil {
		messages = []models.Message{}
	}
	writeJSON(w, http.StatusOK, messages)
}

// HandleMarkRead marks an inbound message as read.
//
// @Summary      Mark message as read
// @Description  Marks an inbound message as read, changing its status from "received" to "read".
// @Tags         SMS
// @Produce      json
// @Param        id   path      string  true  "Message ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /api/v1/sms/{id}/read [put]
func (h *SMSHandler) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "message id is required"})
		return
	}

	if err := h.repo.MarkMessageRead(id); err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "marked as read"})
}

// HandleMarkUnread marks an inbound message as unread.
//
// @Summary      Mark message as unread
// @Description  Marks an inbound message as unread, changing its status from "read" back to "received".
// @Tags         SMS
// @Produce      json
// @Param        id   path      string  true  "Message ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /api/v1/sms/{id}/unread [put]
func (h *SMSHandler) HandleMarkUnread(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "message id is required"})
		return
	}

	if err := h.repo.MarkMessageUnread(id); err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "marked as unread"})
}

// HandleGetOutbox returns outbound messages.
//
// @Summary      Get outbox messages
// @Description  Returns all outbound SMS messages.
// @Tags         SMS
// @Produce      json
// @Success      200  {array}   models.Message
// @Failure      500  {object}  models.ErrorResponse
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /api/v1/sms/outbox [get]
func (h *SMSHandler) HandleGetOutbox(w http.ResponseWriter, _ *http.Request) {
	messages, err := h.repo.ListMessages(models.DirectionOutbound, nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to list messages"})
		return
	}

	if messages == nil {
		messages = []models.Message{}
	}
	writeJSON(w, http.StatusOK, messages)
}

// HandleDeleteMessage deletes a message by ID.
//
// @Summary      Delete message
// @Description  Deletes a single SMS message by its unique identifier.
// @Tags         SMS
// @Produce      json
// @Param        id   path      string  true  "Message ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /api/v1/sms/{id} [delete]
func (h *SMSHandler) HandleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "message id is required"})
		return
	}

	if err := h.repo.DeleteMessage(id); err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// HandleGetMessage returns a single message by ID.
//
// @Summary      Get message by ID
// @Description  Returns a single SMS message by its unique identifier.
// @Tags         SMS
// @Produce      json
// @Param        id   path      string  true  "Message ID"
// @Success      200  {object}  models.Message
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /api/v1/sms/{id} [get]
func (h *SMSHandler) HandleGetMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "message id is required"})
		return
	}

	msg, err := h.repo.GetMessage(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: "message not found"})
		return
	}

	writeJSON(w, http.StatusOK, msg)
}
