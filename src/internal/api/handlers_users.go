package api

import (
	"encoding/json"
	"net/http"

	"github.com/mattboston/sms-gateway/internal/auth"
	"github.com/mattboston/sms-gateway/internal/database"
	"github.com/mattboston/sms-gateway/internal/models"
)

// UserHandler handles user management endpoints (admin only).
type UserHandler struct {
	repo *database.Repository
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(repo *database.Repository) *UserHandler {
	return &UserHandler{repo: repo}
}

// HandleListUsers returns all users.
//
// @Summary      List users
// @Description  Returns all user accounts. Requires admin privileges.
// @Tags         Users
// @Produce      json
// @Success      200  {array}   models.User
// @Failure      500  {object}  models.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users [get]
func (h *UserHandler) HandleListUsers(w http.ResponseWriter, _ *http.Request) {
	users, err := h.repo.ListUsers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to list users"})
		return
	}

	if users == nil {
		users = []models.User{}
	}
	writeJSON(w, http.StatusOK, users)
}

// HandleCreateUser creates a new user account.
//
// @Summary      Create user
// @Description  Creates a new user account with the specified username, password, and admin status. Requires admin privileges.
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        request  body      models.CreateUserRequest  true  "User creation request"
// @Success      201      {object}  models.User
// @Failure      400      {object}  models.ErrorResponse
// @Failure      500      {object}  models.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/users [post]
func (h *UserHandler) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "username and password are required"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to hash password"})
		return
	}

	user, err := h.repo.CreateUser(req.Username, hash, req.IsAdmin, false)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to create user"})
		return
	}

	writeJSON(w, http.StatusCreated, user)
}
