package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/example/project/internal/model"
	"github.com/example/project/internal/store"
	"github.com/example/project/internal/validation"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// WebhookService handles webhook registration.
type WebhookService struct {
	store store.Store
	log   *zap.Logger
}

func NewWebhookService(s store.Store, log *zap.Logger) *WebhookService {
	return &WebhookService{store: s, log: log}
}

// RegisterWebhookRequest carries webhook registration input.
type RegisterWebhookRequest struct {
	URL    string
	Events []string
	Secret string
}

// RegisterWebhook creates a new webhook endpoint.
func (s *WebhookService) RegisterWebhook(req RegisterWebhookRequest) (*model.Webhook, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.URL, "url", "URL", &errs)
	if len(req.Events) == 0 {
		errs.Add("events", "At least one event is required")
	}
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	// Validate event types
	events := make([]model.WebhookEvent, 0, len(req.Events))
	validEvents := map[string]bool{
		string(model.EventProjectCreated):    true,
		string(model.EventProjectUpdated):    true,
		string(model.EventTaskCreated):       true,
		string(model.EventTaskUpdated):       true,
		string(model.EventCodeGenerated):     true,
		string(model.EventCodeCommitted):     true,
		string(model.EventReviewCompleted):   true,
		string(model.EventDeploymentCreated): true,
		string(model.EventDeploymentUpdated): true,
	}
	for _, e := range req.Events {
		if !validEvents[e] {
			return nil, validationSingle("events", "Unknown event: "+e)
		}
		events = append(events, model.WebhookEvent(e))
	}

	now := time.Now().UTC()
	webhook := &model.Webhook{
		ID:        generateID("wh"),
		URL:       req.URL,
		Events:    events,
		Secret:    req.Secret,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Webhooks().Create(webhook); err != nil {
		s.log.Error("failed to create webhook", zap.Error(err))
		return nil, internalError("Failed to register webhook")
	}

	return webhook, nil
}
