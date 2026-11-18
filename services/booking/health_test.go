package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	for _, test := range []struct {
		name       string
		path       string
		pingError  error
		wantPing   bool
		wantStatus int
		wantBody   string
	}{
		{"live with unavailable database", "/health/live", errors.New("postgres driver details"), false, 200, `"status":"ok"`},
		{"ready", "/health/ready", nil, true, 200, `"status":"ok"`},
		{"database unavailable", "/health/ready", errors.New("postgres driver details"), true, 503, `"code":"DATABASE_UNAVAILABLE"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			pinged := false
			router, err := newRouter(func(ctx context.Context) error {
				pinged = true
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 2*time.Second {
					t.Error("database ping must have a deadline of at most 2 seconds")
				}
				return test.pingError
			})
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			req.Header.Set("X-Request-ID", "booking-health-1")
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			if pinged != test.wantPing {
				t.Errorf("database ping called = %t, want %t", pinged, test.wantPing)
			}
			body := res.Body.String()
			if res.Code != test.wantStatus || !strings.Contains(body, test.wantBody) {
				t.Errorf("response = %d %s, want %d and %s", res.Code, body, test.wantStatus, test.wantBody)
			}
			if strings.Contains(body, "postgres driver details") {
				t.Error("response exposed database error details")
			}
			if res.Header().Get("X-Request-ID") != "booking-health-1" {
				t.Error("response did not preserve the request ID")
			}
		})
	}
}
