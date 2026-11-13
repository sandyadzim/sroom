package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	app := gin.New()
	app.Use(RequestHeaders)
	app.GET("/", func(c *gin.Context) { c.Status(200) })

	for _, test := range []struct {
		id       string
		preserve bool
	}{{"gateway_1-ID", true}, {strings.Repeat("A", 64), true}, {"", false}, {"invalid id", false}, {strings.Repeat("A", 65), false}} {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("X-Request-ID", test.id)
		response := httptest.NewRecorder()
		app.ServeHTTP(response, req)
		id := response.Header().Get("X-Request-ID")
		if !validRequestID(id) || (id == test.id) != test.preserve {
			t.Errorf("ID %q, got %q, want preserve = %v", test.id, id, test.preserve)
		}
	}
}
