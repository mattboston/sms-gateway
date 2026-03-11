package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/mattboston/sms-gateway/internal/auth"
	"github.com/mattboston/sms-gateway/internal/database"
	"github.com/mattboston/sms-gateway/internal/models"
)

// KeyHandler handles API key management endpoints.
type KeyHandler struct {
	repo *database.Repository
}

// NewKeyHandler creates a new KeyHandler.
func NewKeyHandler(repo *database.Repository) *KeyHandler {
	return &KeyHandler{repo: repo}
}

// HandleListAPIKeys returns all API keys for the authenticated user.
//
// @Summary      List API keys
// @Description  Returns all API keys for the authenticated user. Admins see all keys.
// @Tags         API Keys
// @Produce      json
// @Success      200  {array}   models.APIKey
// @Failure      401  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/apikeys [get]
func (h *KeyHandler) HandleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r.Context())
	if claims == nil {
		writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		return
	}

	var keys []models.APIKey
	var err error

	if claims.IsAdmin {
		keys, err = h.repo.ListAPIKeys()
	} else {
		keys, err = h.repo.ListAPIKeysByUserID(claims.UserID)
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to list API keys"})
		return
	}

	if keys == nil {
		keys = []models.APIKey{}
	}
	writeJSON(w, http.StatusOK, keys)
}

// HandleCreateAPIKey generates a new API key for the authenticated user.
//
// @Summary      Create API key
// @Description  Generates a new API key for the authenticated user with an optional label.
// @Tags         API Keys
// @Accept       json
// @Produce      json
// @Param        request  body      models.CreateAPIKeyRequest   true  "API key creation request"
// @Success      201      {object}  models.CreateAPIKeyResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      401      {object}  models.ErrorResponse
// @Failure      500      {object}  models.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/apikeys [post]
func (h *KeyHandler) HandleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r.Context())
	if claims == nil {
		writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		return
	}

	var req models.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	key, err := auth.GenerateAPIKey()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to generate API key"})
		return
	}

	apiKey, err := h.repo.CreateAPIKey(key, req.Label, claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to create API key"})
		return
	}

	writeJSON(w, http.StatusCreated, models.CreateAPIKeyResponse{APIKey: *apiKey})
}

// HandleDeactivateAPIKey deactivates an API key by ID.
//
// @Summary      Deactivate API key
// @Description  Deactivates an API key by its unique identifier.
// @Tags         API Keys
// @Produce      json
// @Param        id   path      string  true  "API Key ID"
// @Success      200  {object}  map[string]string  "message: API key deactivated"
// @Failure      400  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/apikeys/{id} [delete]
func (h *KeyHandler) HandleDeactivateAPIKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "API key id is required"})
		return
	}

	if err := h.repo.DeactivateAPIKey(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to deactivate API key"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "API key deactivated"})
}

// HandleDeleteAPIKey permanently deletes an API key by ID.
//
// @Summary      Delete API key
// @Description  Permanently deletes an API key by its unique identifier.
// @Tags         API Keys
// @Produce      json
// @Param        id   path      string  true  "API Key ID"
// @Success      200  {object}  map[string]string  "message: API key deleted"
// @Failure      400  {object}  models.ErrorResponse
// @Failure      404  {object}  models.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/apikeys/{id}/delete [delete]
func (h *KeyHandler) HandleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "API key id is required"})
		return
	}

	if err := h.repo.DeleteAPIKey(id); err != nil {
		writeJSON(w, http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "API key deleted"})
}
