package model

import (
	"time"

	"github.com/google/uuid"
)

// WebhookEvent represents an event type a webhook can subscribe to.
type WebhookEvent string

const (
	EventProjectCreated    WebhookEvent = "project.created"
	EventProjectUpdated    WebhookEvent = "project.updated"
	EventTaskCreated       WebhookEvent = "task.created"
	EventTaskUpdated       WebhookEvent = "task.updated"
	EventCodeGenerated     WebhookEvent = "code.generated"
	EventCodeCommitted     WebhookEvent = "code.committed"
	EventReviewCompleted   WebhookEvent = "review.completed"
	EventDeploymentCreated WebhookEvent = "deployment.created"
	EventDeploymentUpdated WebhookEvent = "deployment.updated"
)

// Webhook represents a registered webhook endpoint.
type Webhook struct {
	ID        uuid.UUID      `json:"id"`
	URL       string         `json:"url"`
	Events    []WebhookEvent `json:"events"`
	Secret    string         `json:"secret,omitempty"`
	Active    bool           `json:"active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// WebhookDelivery represents a single webhook delivery attempt.
type WebhookDelivery struct {
	ID         uuid.UUID `json:"id"`
	WebhookID  uuid.UUID `json:"webhook_id"`
	Event      string    `json:"event"`
	Payload    string    `json:"payload"`
	Status     string    `json:"status"`
	StatusCode int       `json:"status_code"`
	AttemptedAt time.Time `json:"attempted_at"`
}
