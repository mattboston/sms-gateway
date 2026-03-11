package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/mattboston/sms-gateway/internal/auth"
	"github.com/mattboston/sms-gateway/internal/database"
	"github.com/mattboston/sms-gateway/internal/models"
)

// AuthHandler handles authentication-related endpoints.
type AuthHandler struct {
	repo      *database.Repository
	jwtSecret string
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(repo *database.Repository, jwtSecret string) *AuthHandler {
	return &AuthHandler{repo: repo, jwtSecret: jwtSecret}
}

// HandleLogin authenticates a user and returns a JWT token.
//
// @Summary      User login
// @Description  Authenticates a user with username and password, returning a JWT token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.LoginRequest  true  "Login credentials"
// @Success      200      {object}  models.LoginResponse
// @Failure      400      {object}  models.ErrorResponse
// @Failure      401      {object}  models.ErrorResponse
// @Failure      500      {object}  models.ErrorResponse
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "username and password are required"})
		return
	}

	user, err := h.repo.GetUserByUsername(req.Username)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "invalid credentials"})
		return
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "invalid credentials"})
		return
	}

	token, err := auth.GenerateJWT(h.jwtSecret, user.ID, user.IsAdmin)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to generate token"})
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(24 * time.Hour / time.Second),
	})

	writeJSON(w, http.StatusOK, models.LoginResponse{
		Token: token,
		User:  *user,
	})
}

// HandleLogout clears the authentication cookie.
//
// @Summary      User logout
// @Description  Clears the authentication cookie, logging the user out.
// @Tags         Auth
// @Produce      json
// @Success      200  {object}  map[string]string  "message: logged out"
// @Security     BearerAuth
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// HandleChangePassword allows an authenticated user to change their password.
//
// @Summary      Change password
// @Description  Allows an authenticated user to change their password by providing the current and new passwords.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.ChangePasswordRequest  true  "Current and new password"
// @Success      200      {object}  map[string]string             "message: password updated"
// @Failure      400      {object}  models.ErrorResponse
// @Failure      401      {object}  models.ErrorResponse
// @Failure      500      {object}  models.ErrorResponse
// @Security     BearerAuth
// @Router       /api/v1/auth/change-password [post]
func (h *AuthHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := GetUserFromContext(r.Context())
	if claims == nil {
		writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		return
	}

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{Error: "current_password and new_password are required"})
		return
	}

	user, err := h.repo.GetUserByID(claims.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to get user"})
		return
	}

	if !auth.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		writeJSON(w, http.StatusUnauthorized, models.ErrorResponse{Error: "current password is incorrect"})
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to hash password"})
		return
	}

	if err := h.repo.UpdatePassword(user.ID, hash); err != nil {
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{Error: "failed to update password"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}
