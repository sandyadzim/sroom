package identity

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionVerification(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
		want   error
	}{
		{200, `{"data":{"id":"customer-fixture","role":"CUSTOMER"}}`, nil},
		{401, `{}`, ErrUnauthenticated},
		{503, `{"private":"must not escape"}`, ErrUnavailable},
		{200, `{"data":{}}`, ErrUnavailable},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/identity/me" || r.Header.Get("Authorization") != "Bearer test-only-session" || r.Header.Get("X-Request-ID") != "identity-test" {
				t.Error("invalid Identity verification request")
			}
			w.WriteHeader(test.status)
			_, _ = w.Write([]byte(test.body))
		}))
		client, err := NewClient(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.Profile(t.Context(), "test-only-session", "identity-test")
		server.Close()
		if !errors.Is(err, test.want) {
			t.Fatalf("error=%v want=%v", err, test.want)
		}
	}
}
