package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/example/project/internal/service"
)

// UserHandler handles user registration and profile endpoints.
type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type userResponse struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	Name      string   `json:"name"`
	Role      string   `json:"role,omitempty"`
	Teams     []string `json:"teams,omitempty"`
	Projects  []string `json:"projects,omitempty"`
	CreatedAt string   `json:"created_at"`
}

// Register handles POST /users/register.
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	user, svcErr := h.svc.Register(service.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusCreated, userResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      string(user.Role),
		Teams:     user.Teams,
		Projects:  user.Projects,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	})
}

// GetProfile handles GET /users/me.
func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id")
	uid, _ := userID.(string)
	if uid == "" {
		uid = "user_from_jwt"
	}

	user, svcErr := h.svc.GetProfile(uid)
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      string(user.Role),
		Teams:     user.Teams,
		Projects:  user.Projects,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	})
}
