package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{DisableKeepAlives: true},
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func forwardTo(client *http.Client, base *url.URL, service string, bodyLimit int64, timeout time.Duration, logger *slog.Logger) gin.HandlerFunc {
	return forward(client, base, service, bodyLimit, timeout, logger, nil)
}

func forward(client *http.Client, base *url.URL, service string, bodyLimit int64, timeout time.Duration, logger *slog.Logger, onSuccess func(*gin.Context, []byte)) gin.HandlerFunc {
	return func(c *gin.Context) {
		incoming := c.Request
		// Dynamic segments are forwarded to another router. Keep them as individual path segments.
		for _, param := range c.Params {
			if param.Value == "." || param.Value == ".." || strings.ContainsAny(param.Value, `\%`) {
				writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid route parameter")
				return
			}
		}

		// Read the bounded body before forwarding so an oversized request cannot reach a mutation.
		var body []byte
		if incoming.Body != nil && incoming.Body != http.NoBody {
			if incoming.ContentLength > bodyLimit {
				writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
				return
			}
			limited := http.MaxBytesReader(c.Writer, incoming.Body, bodyLimit)
			defer limited.Close()
			var err error
			body, err = io.ReadAll(limited)
			if err != nil {
				if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
					writeError(c, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "request body is too large")
					return
				}
				writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "could not read request body")
				return
			}
		}

		target := *base
		target.Path = incoming.URL.Path
		target.RawQuery = incoming.URL.RawQuery
		target.ForceQuery = incoming.URL.ForceQuery
		ctx, cancel := context.WithTimeout(incoming.Context(), timeout)
		defer cancel()
		request, err := http.NewRequestWithContext(ctx, incoming.Method, target.String(), bytes.NewReader(body))
		if err != nil {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request URL")
			return
		}
		for _, header := range []string{"Authorization", "Content-Type"} {
			if value := incoming.Header.Get(header); value != "" {
				request.Header.Set(header, value)
			}
		}
		request.Header.Set("X-Request-ID", c.Writer.Header().Get("X-Request-ID"))

		response, err := client.Do(request)
		if err != nil {
			upstreamError(c, logger, service, err)
			return
		}
		defer response.Body.Close()
		// Buffer the bounded JSON response so a timeout can still return a complete error envelope.
		const maxResponseBytes = 8 * 1024 * 1024
		payload, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
		if err != nil {
			upstreamError(c, logger, service, err)
			return
		}
		if len(payload) > maxResponseBytes {
			upstreamError(c, logger, service, errors.New("response body is too large"))
			return
		}
		for _, header := range []string{
			"Content-Type", "Cache-Control", "Content-Security-Policy", "X-Frame-Options", "WWW-Authenticate",
			"Retry-After", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "Allow",
		} {
			if value := response.Header.Get(header); value != "" {
				c.Header(header, value)
			}
		}
		if onSuccess != nil && response.StatusCode >= 200 && response.StatusCode < 300 {
			onSuccess(c, payload)
			return
		}
		c.Status(response.StatusCode)
		if len(payload) > 0 {
			if _, err := c.Writer.Write(payload); err != nil {
				logger.Warn("response interrupted", "operation", "http.response", "request_id", request.Header.Get("X-Request-ID"))
			}
		}
	}
}

func upstreamError(c *gin.Context, logger *slog.Logger, service string, err error) {
	status := http.StatusServiceUnavailable
	message := "service unavailable"
	if errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusGatewayTimeout
		message = "service timed out"
	}
	logger.Error("upstream request failed", "operation", "http.forward", "upstream", service,
		"status", status, "request_id", c.Writer.Header().Get("X-Request-ID"))
	writeError(c, status, "DEPENDENCY_UNAVAILABLE", message)
}
