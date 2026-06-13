package service

import (
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/fadhilfathi/AI-Software-Factory/internal/validation"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles user registration and profile operations.
type UserService struct {
	store store.Store
	log   *zap.Logger
}

func NewUserService(s store.Store, log *zap.Logger) *UserService {
	return &UserService{store: s, log: log}
}

// RegisterRequest carries registration input.
type RegisterRequest struct {
	Email    string
	Password string
	Name     string
}

// Register creates a new user account.
func (s *UserService) Register(req RegisterRequest) (*model.User, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.Email, "email", "Email", &errs)
	validation.NotEmpty(req.Password, "password", "Password", &errs)
	validation.NotEmpty(req.Name, "name", "Name", &errs)
	validation.Email(req.Email, "email", &errs)
	validation.Name(req.Name, "name", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	// Check for duplicate email
	existing, _ := s.store.Users().GetByEmail(req.Email)
	if existing != nil {
		return nil, conflict("A user with this email already exists")
	}

	now := time.Now().UTC()
	// Hash password with bcrypt (cost 12)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("failed to hash password", zap.Error(err))
		return nil, internalError("Failed to create user")
	}

	user := &model.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword), // bcrypt hash stored
		Name:         req.Name,
		Role:         model.RoleMember,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.Users().Create(user); err != nil {
		s.log.Error("failed to create user", zap.Error(err))
		return nil, internalError("Failed to create user")
	}

	return user, nil
}

// GetProfile returns a user by ID.
func (s *UserService) GetProfile(userID string) (*model.User, *Error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		errs := &validation.Errors{}
		errs.Add("user_id", "Invalid User ID format")
		return nil, validationError(*errs)
	}
	user, err := s.store.Users().GetByID(uid)
	if err != nil {
		return nil, notFound("User not found")
	}
	return user, nil
}
