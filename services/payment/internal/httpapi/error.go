package httpapi

import "github.com/gin-gonic/gin"

func WriteError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
			"details": gin.H{},
		},
		"request_id": c.Writer.Header().Get("X-Request-ID"),
	})
}
