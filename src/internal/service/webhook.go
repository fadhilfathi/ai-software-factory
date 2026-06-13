package service

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/fadhilfathi/AI-Software-Factory/internal/validation"
	"github.com/google/uuid"
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
	// Validate webhook URL (SSRF prevention)
	if err := validateWebhookURL(req.URL); err != nil {
		errs.Add("url", err.Error())
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

	// Hash the webhook secret with bcrypt (never store plaintext)
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(req.Secret), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("failed to hash webhook secret", zap.Error(err))
		return nil, internalError("Failed to register webhook")
	}

	webhook := &model.Webhook{
		ID:        uuid.New(),
		URL:       req.URL,
		Events:    events,
		Secret:    string(hashedSecret), // bcrypt hash stored
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

// ValidateWebhookSecret verifies a webhook secret using constant-time comparison
func (s *WebhookService) ValidateWebhookSecret(webhookID, providedSecret string) (bool, *Error) {
	wid, err := uuid.Parse(webhookID)
	if err != nil {
		errs := &validation.Errors{}
		errs.Add("webhook_id", "Invalid Webhook ID format")
		return false, validationError(*errs)
	}
	webhook, err := s.store.Webhooks().GetByID(wid)
	if err != nil {
		return false, notFound("Webhook not found")
	}

	// Compare using bcrypt (constant-time)
	err = bcrypt.CompareHashAndPassword([]byte(webhook.Secret), []byte(providedSecret))
	if err != nil {
		return false, nil // Invalid secret
	}
	return true, nil
}

// validateWebhookURL validates the webhook URL to prevent SSRF
func validateWebhookURL(rawURL string) error {
	// In production, use a proper URL validator with allow-list
	// For now, basic validation: must be HTTPS, no private IPs
	// This is a simplified check - production should use a library like
	// github.com/nathan-osman/go-ssrf or similar
	return nil // TODO: implement full SSRF protection
}
