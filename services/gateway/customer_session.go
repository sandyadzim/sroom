package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

const customerPath = "/api/v1/customer"
const customerCookieName = "sroom_customer_session"

func registerCustomerSession(group *gin.RouterGroup, client *http.Client, identityURL *url.URL, logger *slog.Logger) {
	secure := envOrDefault("APP_ENV", "development") != "development"
	login := forward(client, identityURL, "identity", 16*1024, 8*time.Second, logger,
		func(c *gin.Context, payload []byte) {
			token, expires, ok := readSession(payload)
			if !ok {
				writeError(c, http.StatusBadGateway, "INVALID_UPSTREAM_RESPONSE", "could not establish customer session")
				return
			}
			ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
			defer cancel()
			requestID := c.Writer.Header().Get("X-Request-ID")
			profile, status := readSessionProfile(ctx, client, identityURL, token, requestID)
			if status != http.StatusOK || profile.Data.Role != "CUSTOMER" {
				revokeLoginSession(ctx, client, identityURL, token, requestID)
				if status != http.StatusOK {
					writeError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "could not verify customer account")
				} else {
					customerRequired(c)
				}
				return
			}
			setSessionCookie(c, customerCookieName, customerPath, token, expires, secure)
			c.JSON(http.StatusOK, profile)
		})
	group.POST("/login", func(c *gin.Context) {
		request := c.Request.Clone(c.Request.Context())
		request.Header.Del("Authorization")
		request.URL.Path = identityPath + "/login"
		request.URL.RawPath = ""
		original := c.Request
		c.Request = request
		defer func() { c.Request = original }()
		login(c)
	})

	group.GET("/me", func(c *gin.Context) {
		customerSession(c, client, identityURL, secure, false)
	})
	group.POST("/logout", func(c *gin.Context) {
		customerSession(c, client, identityURL, secure, true)
	})
}

func customerSession(c *gin.Context, client *http.Client, identityURL *url.URL, secure, logout bool) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	requestID := c.Writer.Header().Get("X-Request-ID")
	token, _ := c.Cookie(customerCookieName)
	status := http.StatusUnauthorized

	if token != "" {
		profile, profileStatus := readSessionProfile(ctx, client, identityURL, token, requestID)
		status = profileStatus
		if status == http.StatusOK {
			if profile.Data.Role != "CUSTOMER" {
				customerRequired(c)
				return
			}
			if !logout {
				c.JSON(http.StatusOK, profile)
				return
			}
			response, err := identitySessionRequest(ctx, client, identityURL, http.MethodPost, "/logout", token, requestID)
			if err != nil {
				status = http.StatusServiceUnavailable
			} else {
				defer response.Body.Close()
				status = response.StatusCode
			}
		}
	}

	if logout && (status == http.StatusNoContent || status == http.StatusUnauthorized) {
		setSessionCookie(c, customerCookieName, customerPath, "", time.Unix(0, 0), secure)
		c.Status(http.StatusNoContent)
	} else if status == http.StatusUnauthorized {
		writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
	} else {
		writeError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "could not check customer session")
	}
}

func customerRequired(c *gin.Context) {
	writeError(c, http.StatusForbidden, "CUSTOMER_REQUIRED", "Use a customer account to sign in here.")
}
