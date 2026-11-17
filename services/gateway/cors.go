package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func browserCORS(origins string) gin.HandlerFunc {
	allowed := strings.Split(origins, ",")
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/v1/") {
			return
		}
		customerRoute := strings.HasPrefix(c.Request.URL.Path, customerPath+"/")
		cookieRoute := strings.HasPrefix(c.Request.URL.Path, staffPath+"/") || customerRoute
		if customerRoute {
			c.Header("Cache-Control", "no-store")
		}

		c.Writer.Header().Add("Vary", "Origin")
		origin := c.GetHeader("Origin")

		for _, candidate := range allowed {
			if origin == "" || origin == "null" || origin != strings.TrimSpace(candidate) {
				continue
			}
			c.Header("Access-Control-Allow-Origin", origin)
			if cookieRoute {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
			if c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != "" {
				c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
				c.AbortWithStatus(http.StatusNoContent)
			}
			return
		}

		// A form can send a request even when the browser blocks reading its response.
		if cookieRoute {
			switch c.Request.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
			default:
				writeError(c, http.StatusForbidden, "FORBIDDEN_ORIGIN", "request origin is not allowed")
			}
		}
	}
}
