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

		c.Writer.Header().Add("Vary", "Origin")
		origin := c.GetHeader("Origin")
		if origin == "" {
			return
		}

		for _, candidate := range allowed {
			if origin != strings.TrimSpace(candidate) {
				continue
			}
			c.Header("Access-Control-Allow-Origin", origin)
			if c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != "" {
				c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE")
				c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
				c.AbortWithStatus(http.StatusNoContent)
			}
			return
		}
	}
}
