package handler

import (
	"net/http"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/middleware"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
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
func (h *UserHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	user, svcErr := h.svc.Register(service.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusCreated, userResponse{
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
func (h *UserHandler) GetProfile(c *gin.Context) {
	uid, exists := c.Get(middleware.UserIDKey)
	if !exists {
		uid = "user_from_jwt"
	}
	userID, _ := uid.(string)

	user, svcErr := h.svc.GetProfile(userID)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	writeJSON(c, http.StatusOK, userResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      string(user.Role),
		Teams:     user.Teams,
		Projects:  user.Projects,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	})
}
