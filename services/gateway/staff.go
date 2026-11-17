package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const staffPath = "/api/v1/staff"
const staffCookieName = "sroom_staff_session"
const identityPath = "/api/v1/identity"

func registerStaff(staff *gin.RouterGroup, client *http.Client, identityURL *url.URL, identity, catalog, upload gin.HandlerFunc, logger *slog.Logger) {
	secureCookie := envOrDefault("APP_ENV", "development") != "development"
	login := forward(client, identityURL, "identity", 16*1024, 8*time.Second, logger,
		finishStaffLogin(client, identityURL, secureCookie))

	staff.POST("/login", staffProxy(identityPath, false, login))
	staffIdentity := staffProxy(identityPath, true, identity)
	staff.GET("/me", staffIdentity)
	staff.POST("/change-password", staffIdentity)

	logout := forward(client, identityURL, "identity", 16*1024, 8*time.Second, logger,
		func(c *gin.Context, _ []byte) {
			setStaffCookie(c, "", time.Unix(0, 0), secureCookie)
			c.Status(http.StatusNoContent)
		})
	staff.POST("/logout", staffProxy(identityPath, true, logout))

	staff.GET("/admins", staffIdentity)
	staff.POST("/admins", staffIdentity)
	staff.GET("/admins/:id", staffIdentity)
	staff.PATCH("/admins/:id", staffIdentity)

	staff.GET("/customers", staffIdentity)
	staff.GET("/customers/:id", staffIdentity)
	staff.PATCH("/customers/:id/status", staffIdentity)

	registerCatalogAdmin(staff.Group("/catalog/admin"),
		staffProxy("/api/v1", true, catalog), staffProxy("/api/v1", true, upload))
}

func staffProxy(path string, authenticated bool, handler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		request := c.Request.Clone(c.Request.Context())
		request.Header.Del("Authorization")
		if authenticated {
			cookie, err := request.Cookie(staffCookieName)
			if err != nil || cookie.Value == "" {
				writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "authentication required")
				return
			}
			request.Header.Set("Authorization", "Bearer "+cookie.Value)
		}

		request.URL.Path = path + strings.TrimPrefix(request.URL.Path, staffPath)
		request.URL.RawPath = ""
		original := c.Request
		c.Request = request
		defer func() { c.Request = original }()
		handler(c)
	}
}

func finishStaffLogin(client *http.Client, identityURL *url.URL, secureCookie bool) func(*gin.Context, []byte) {
	return func(c *gin.Context, payload []byte) {
		token, expires, ok := readSession(payload)
		if !ok {
			writeError(c, http.StatusBadGateway, "INVALID_UPSTREAM_RESPONSE", "could not establish staff session")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()
		requestID := c.Writer.Header().Get("X-Request-ID")
		profile, status := readSessionProfile(ctx, client, identityURL, token, requestID)
		if status != http.StatusOK {
			writeError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "could not verify staff account")
			return
		}
		if profile.Data.Role != "ADMIN" && profile.Data.Role != "OWNER" {
			// Identity login creates a session for every role. Revoke the unused one.
			revokeLoginSession(ctx, client, identityURL, token, requestID)
			writeError(c, http.StatusForbidden, "STAFF_REQUIRED", "Use a staff account to sign in here.")
			return
		}

		setStaffCookie(c, token, expires, secureCookie)
		c.JSON(http.StatusOK, profile)
	}
}

func readSession(payload []byte) (string, time.Time, bool) {
	var login struct {
		Data struct {
			Token     string    `json:"token"`
			ExpiresAt time.Time `json:"expires_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &login); err != nil {
		return "", time.Time{}, false
	}
	return login.Data.Token, login.Data.ExpiresAt, login.Data.Token != "" && login.Data.ExpiresAt.After(time.Now())
}

type loginProfile struct {
	Data struct {
		ID        string `json:"id"`
		FullName  string `json:"full_name"`
		Email     string `json:"email"`
		Role      string `json:"role"`
		CreatedAt string `json:"created_at"`
	} `json:"data"`
}

func readSessionProfile(ctx context.Context, client *http.Client, base *url.URL, token, requestID string) (loginProfile, int) {
	var profile loginProfile
	response, err := identitySessionRequest(ctx, client, base, http.MethodGet, "/me", token, requestID)
	if err != nil {
		return profile, http.StatusServiceUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusUnauthorized {
		return profile, http.StatusUnauthorized
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 16*1024)).Decode(&profile) != nil ||
		profile.Data.ID == "" || profile.Data.Role == "" {
		return profile, http.StatusServiceUnavailable
	}
	return profile, http.StatusOK
}

func revokeLoginSession(ctx context.Context, client *http.Client, base *url.URL, token, requestID string) {
	response, err := identitySessionRequest(ctx, client, base, http.MethodPost, "/logout", token, requestID)
	if err == nil {
		response.Body.Close()
	}
}

func identitySessionRequest(ctx context.Context, client *http.Client, base *url.URL, method, path, token, requestID string) (*http.Response, error) {
	target := *base
	target.Path = identityPath + path
	request, err := http.NewRequestWithContext(ctx, method, target.String(), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("X-Request-ID", requestID)
	return client.Do(request)
}

func setStaffCookie(c *gin.Context, token string, expires time.Time, secure bool) {
	setSessionCookie(c, staffCookieName, staffPath, token, expires, secure)
}

func setSessionCookie(c *gin.Context, name, path, token string, expires time.Time, secure bool) {
	cookie := &http.Cookie{
		Name: name, Value: token, Path: path,
		HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode, Expires: expires,
	}
	if token == "" {
		cookie.MaxAge = -1
	}
	http.SetCookie(c.Writer, cookie)
}
