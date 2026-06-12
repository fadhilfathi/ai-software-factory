package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWebhookEventConstants(t *testing.T) {
	assert.Equal(t, WebhookEvent("project.created"), EventProjectCreated)
	assert.Equal(t, WebhookEvent("project.updated"), EventProjectUpdated)
	assert.Equal(t, WebhookEvent("task.created"), EventTaskCreated)
	assert.Equal(t, WebhookEvent("task.updated"), EventTaskUpdated)
	assert.Equal(t, WebhookEvent("code.generated"), EventCodeGenerated)
	assert.Equal(t, WebhookEvent("code.committed"), EventCodeCommitted)
	assert.Equal(t, WebhookEvent("review.completed"), EventReviewCompleted)
	assert.Equal(t, WebhookEvent("deployment.created"), EventDeploymentCreated)
	assert.Equal(t, WebhookEvent("deployment.updated"), EventDeploymentUpdated)
}

func TestWebhookStruct(t *testing.T) {
	now := time.Now().UTC()
	webhook := Webhook{
		ID:        "webhook-123",
		URL:       "https://example.com/webhook",
		Events:    []WebhookEvent{EventProjectCreated, EventTaskCreated, EventCodeGenerated},
		Secret:    "secret-key-123",
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	assert.Equal(t, "webhook-123", webhook.ID)
	assert.Equal(t, "https://example.com/webhook", webhook.URL)
	assert.Len(t, webhook.Events, 3)
	assert.Equal(t, EventProjectCreated, webhook.Events[0])
	assert.Equal(t, EventTaskCreated, webhook.Events[1])
	assert.Equal(t, EventCodeGenerated, webhook.Events[2])
	assert.Equal(t, "secret-key-123", webhook.Secret)
	assert.True(t, webhook.Active)
	assert.Equal(t, now, webhook.CreatedAt)
	assert.Equal(t, now, webhook.UpdatedAt)
}

func TestWebhookWithEmptySecret(t *testing.T) {
	webhook := Webhook{
		ID:     "webhook-no-secret",
		URL:    "https://example.com/webhook",
		Events: []WebhookEvent{EventTaskUpdated},
		Active: false,
	}
	assert.Empty(t, webhook.Secret)
	assert.False(t, webhook.Active)
	assert.Len(t, webhook.Events, 1)
}

func TestWebhookDeliveryStruct(t *testing.T) {
	now := time.Now().UTC()
	delivery := WebhookDelivery{
		ID:          "delivery-123",
		WebhookID:   "webhook-123",
		Event:       "task.created",
		Payload:     `{"task_id": "task-1", "title": "New Task"}`,
		Status:      "success",
		StatusCode:  200,
		AttemptedAt: now,
	}

	assert.Equal(t, "delivery-123", delivery.ID)
	assert.Equal(t, "webhook-123", delivery.WebhookID)
	assert.Equal(t, "task.created", delivery.Event)
	assert.Equal(t, `{"task_id": "task-1", "title": "New Task"}`, delivery.Payload)
	assert.Equal(t, "success", delivery.Status)
	assert.Equal(t, 200, delivery.StatusCode)
	assert.Equal(t, now, delivery.AttemptedAt)
}