package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/example/project/internal/service"
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
func (h *WebhookHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	webhook, svcErr := h.svc.RegisterWebhook(service.RegisterWebhookRequest{
		URL:    req.URL,
		Events: req.Events,
		Secret: req.Secret,
	})
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	eventStrings := make([]string, len(webhook.Events))
	for i, e := range webhook.Events {
		eventStrings[i] = string(e)
	}

	writeJSON(w, http.StatusCreated, webhookResponse{
		ID:        webhook.ID,
		URL:       webhook.URL,
		Events:    eventStrings,
		Active:    webhook.Active,
		CreatedAt: webhook.CreatedAt.Format(time.RFC3339),
	})
}
