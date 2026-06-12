package handler

import (
	"net/http"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
)

// WebhookHandler handles webhook registration.
type WebhookHandler struct {
	svc *service.WebhookService
}

func NewWebhookHandler(svc *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

type registerWebhookRequest struct {
	URL    string   `json:"url"`
	Events []string `json:"events"`
	Secret string   `json:"secret"`
}

type webhookResponse struct {
	ID        string   `json:"id"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	Active    bool     `json:"active"`
	CreatedAt string   `json:"created_at"`
}

// Register handles POST /webhooks.
func (h *WebhookHandler) Register(c *gin.Context) {
	var req registerWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	webhook, svcErr := h.svc.RegisterWebhook(service.RegisterWebhookRequest{
		URL:    req.URL,
		Events: req.Events,
		Secret: req.Secret,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	eventStrings := make([]string, len(webhook.Events))
	for i, e := range webhook.Events {
		eventStrings[i] = string(e)
	}

	writeJSON(c, http.StatusCreated, webhookResponse{
		ID:        webhook.ID,
		URL:       webhook.URL,
		Events:    eventStrings,
		Active:    webhook.Active,
		CreatedAt: webhook.CreatedAt.Format(time.RFC3339),
	})
}
