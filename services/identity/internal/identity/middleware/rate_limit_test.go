package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

const (
	loginPath    = "/api/v1/identity/login"
	registerPath = "/api/v1/identity/register"
	mePath       = "/api/v1/identity/me"
	logoutPath   = "/api/v1/identity/logout"
)

func newRateLimitTestApp() *fiber.App {
	app := fiber.New()

	successHandler := func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	}

	app.Post(
		loginPath,
		LoginRateLimiter(),
		successHandler,
	)

	app.Post(
		registerPath,
		RegisterRateLimiter(),
		successHandler,
	)

	app.Get(mePath, successHandler)
	app.Post(logoutPath, successHandler)

	return app
}

func performRateLimitRequest(
	t *testing.T,
	app *fiber.App,
	method string,
	path string,
) *http.Response {
	t.Helper()

	req := httptest.NewRequest(method, path, nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test() error: %v", err)
	}

	return resp
}

func TestLoginRateLimiter(t *testing.T) {
	app := newRateLimitTestApp()

	for i := 1; i <= 5; i++ {
		resp := performRateLimitRequest(
			t,
			app,
			http.MethodPost,
			loginPath,
		)

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf(
				"request %d: status = %d, want %d",
				i,
				resp.StatusCode,
				fiber.StatusOK,
			)
		}

		resp.Body.Close()
	}

	resp := performRateLimitRequest(
		t,
		app,
		http.MethodPost,
		loginPath,
	)
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf(
			"request 6: status = %d, want %d",
			resp.StatusCode,
			fiber.StatusTooManyRequests,
		)
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.Error.Code != "RATE_LIMITED" || body.Error.Message != "too many requests" {
		t.Errorf(
			"error = %q, want %q",
			body.Error.Message,
			"too many requests",
		)
	}
}

func TestRegisterRateLimiter(t *testing.T) {
	app := newRateLimitTestApp()

	for i := 1; i <= 3; i++ {
		resp := performRateLimitRequest(
			t,
			app,
			http.MethodPost,
			registerPath,
		)

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf(
				"request %d: status = %d, want %d",
				i,
				resp.StatusCode,
				fiber.StatusOK,
			)
		}

		resp.Body.Close()
	}

	resp := performRateLimitRequest(
		t,
		app,
		http.MethodPost,
		registerPath,
	)
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf(
			"request 4: status = %d, want %d",
			resp.StatusCode,
			fiber.StatusTooManyRequests,
		)
	}
}

func TestRateLimitersAreIndependent(t *testing.T) {
	app := newRateLimitTestApp()

	// Habiskan kuota login.
	for i := 0; i < 5; i++ {
		resp := performRateLimitRequest(
			t,
			app,
			http.MethodPost,
			loginPath,
		)
		resp.Body.Close()
	}

	// Login sudah dibatasi.
	resp := performRateLimitRequest(
		t,
		app,
		http.MethodPost,
		loginPath,
	)

	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf(
			"login status = %d, want %d",
			resp.StatusCode,
			fiber.StatusTooManyRequests,
		)
	}
	resp.Body.Close()

	// Register tetap memiliki kuota tersendiri.
	resp = performRateLimitRequest(
		t,
		app,
		http.MethodPost,
		registerPath,
	)
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf(
			"register status = %d, want %d",
			resp.StatusCode,
			fiber.StatusOK,
		)
	}
}

func TestOtherRoutesAreNotRateLimited(t *testing.T) {
	app := newRateLimitTestApp()

	// Habiskan kuota login dan register.
	for i := 0; i < 6; i++ {
		resp := performRateLimitRequest(
			t,
			app,
			http.MethodPost,
			loginPath,
		)
		resp.Body.Close()
	}

	for i := 0; i < 4; i++ {
		resp := performRateLimitRequest(
			t,
			app,
			http.MethodPost,
			registerPath,
		)
		resp.Body.Close()
	}

	// Endpoint lain tetap bisa dipanggil.
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, mePath},
		{http.MethodPost, logoutPath},
	}

	for _, route := range routes {
		for i := 0; i < 10; i++ {
			resp := performRateLimitRequest(
				t,
				app,
				route.method,
				route.path,
			)

			if resp.StatusCode != fiber.StatusOK {
				t.Errorf(
					"%s %s request %d: status = %d, want %d",
					route.method,
					route.path,
					i+1,
					resp.StatusCode,
					fiber.StatusOK,
				)
			}

			resp.Body.Close()
		}
	}
}

func TestRateLimiterCountsFailedRequests(t *testing.T) {
	app := fiber.New()

	app.Post(
		"/failed-login",
		LoginRateLimiter(),
		func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusUnauthorized)
		},
	)

	for i := 1; i <= 5; i++ {
		resp := performRateLimitRequest(
			t,
			app,
			http.MethodPost,
			"/failed-login",
		)

		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf(
				"request %d: status = %d, want %d",
				i,
				resp.StatusCode,
				fiber.StatusUnauthorized,
			)
		}

		resp.Body.Close()
	}

	resp := performRateLimitRequest(
		t,
		app,
		http.MethodPost,
		"/failed-login",
	)
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf(
			"request 6: status = %d, want %d",
			resp.StatusCode,
			fiber.StatusTooManyRequests,
		)
	}
}
