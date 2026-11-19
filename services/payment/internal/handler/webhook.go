package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"
	"time"

	"payment/internal/dto"
	"payment/internal/httpapi"
	"payment/internal/service"

	"github.com/gin-gonic/gin"
)

type WebhookHandler struct{ service *service.WebhookService }

func NewWebhookHandler(service *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{service: service}
}

func (h *WebhookHandler) Midtrans(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		httpapi.WriteError(c, 400, "INVALID_REQUEST", "send an application/json notification")
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		httpapi.WriteError(c, 413, "PAYLOAD_TOO_LARGE", "notification is too large")
		return
	}
	var notification dto.MidtransNotification
	trimmed := bytes.TrimSpace(body)
	if err != nil || len(trimmed) == 0 || trimmed[0] != '{' || json.Unmarshal(body, &notification) != nil ||
		len(notification.OrderID) == 0 || len(notification.OrderID) > 50 || len(notification.StatusCode) != 3 ||
		strings.Trim(notification.StatusCode, "0123456789") != "" || notification.GrossAmount == "" || notification.SignatureKey == "" {
		httpapi.WriteError(c, 400, "INVALID_REQUEST", "invalid notification fields")
		return
	}
	digest := sha256.Sum256(body)
	requestID := c.Writer.Header().Get("X-Request-ID")
	err = h.service.Process(ctx, notification, hex.EncodeToString(digest[:]), requestID)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
	if errors.Is(err, service.ErrInvalidSignature) {
		slog.Warn("notification signature rejected", "service", "payment", "operation", "webhooks.midtrans", "request_id", requestID)
		httpapi.WriteError(c, 401, "INVALID_SIGNATURE", "invalid notification signature")
		return
	}
	if errors.Is(err, service.ErrNotConfigured) {
		httpapi.WriteError(c, 503, "PAYMENT_NOT_CONFIGURED", "payment integration is not configured")
		return
	}
	slog.Warn("notification processing unavailable", "service", "payment", "operation", "webhooks.midtrans", "request_id", requestID)
	httpapi.WriteError(c, 503, "DEPENDENCY_UNAVAILABLE", "could not verify payment notification")
}
