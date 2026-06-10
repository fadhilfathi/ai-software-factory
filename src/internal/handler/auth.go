package handler

import (
	"encoding/json"
	"net/http"

	"github.com/example/project/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	result, svcErr := h.svc.Login(service.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
