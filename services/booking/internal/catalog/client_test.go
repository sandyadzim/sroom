package catalog

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const variantID = "c67ee02e-85db-451f-9550-64e7edb8be7e"
const testToken = "booking-test-token"
const validResponse = `{"data":{"variant_id":"` + variantID + `","vehicle_label":"Toyota Innova V","daily_rate_idr":500000,"status":"active","eligible_unit_ids":[]}}`

func TestBookingSnapshot(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{"valid response", 200, validResponse, nil},
		{"missing variant", 404, `{"error":{"code":"CAR_NOT_FOUND"}}`, ErrVariantNotFound},
		{"route disabled", 404, `{"error":{"code":"NOT_FOUND"}}`, ErrUnavailable},
		{"wrong token", 401, `{"error":{"code":"UNAUTHENTICATED"}}`, ErrUnavailable},
		{"upstream failure", 500, "synthetic private detail", ErrUnavailable},
		{"invalid JSON", 200, "{", ErrUnavailable},
		{"invalid snapshot", 200, `{"data":{"variant_id":"` + variantID + `"}}`, ErrUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/internal/v1/variants/"+variantID+"/booking-snapshot" ||
					r.Header.Get("Authorization") != "Bearer "+testToken || r.Header.Get("X-Request-ID") != "booking-test-1" {
					t.Error("request method, path, authorization or request ID does not match the contract")
				}
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			client := testClient(t, server.URL)
			result, err := client.BookingSnapshot(t.Context(), variantID, "booking-test-1")
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if test.want == nil && (result.VariantID != variantID || result.DailyRateIDR != 500000 ||
				result.VehicleLabel != "Toyota Innova V" || result.Status != "active" || result.EligibleUnitIDs == nil) {
				t.Fatalf("unexpected snapshot: %+v", result)
			}
		})
	}
}

func TestBookingSnapshotDoesNotFollowRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/other" {
			t.Error("Catalog client followed a redirect")
			return
		}
		http.Redirect(w, r, "/other", http.StatusFound)
	}))
	defer server.Close()
	_, err := testClient(t, server.URL).BookingSnapshot(t.Context(), variantID, "")
	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error = %v, want %v", err, ErrUnavailable)
	}
}

func TestBookingSnapshotTimeoutAndCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	client := testClient(t, server.URL)
	if client.http.Timeout != 3*time.Second {
		t.Fatal("default timeout must be three seconds")
	}
	client.http.Timeout = 30 * time.Millisecond
	if _, err := client.BookingSnapshot(t.Context(), variantID, ""); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("timeout error = %v, want %v", err, ErrUnavailable)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := client.BookingSnapshot(ctx, variantID, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v, want context.Canceled", err)
	}
}

func TestNewClient(t *testing.T) {
	client := testClient(t, "")
	if client.baseURL != "http://localhost:8082" {
		t.Fatalf("unexpected default origin: %s", client.baseURL)
	}
	for _, base := range []string{"not-an-origin", "http://localhost/path", "http://localhost?query=1"} {
		if _, err := NewClient(base, testToken); err == nil {
			t.Errorf("expected invalid base URL %q", base)
		}
	}
	if _, err := NewClient("", ""); err == nil {
		t.Error("expected missing token to be rejected")
	}
	if _, err := client.BookingSnapshot(t.Context(), "../invalid", ""); !errors.Is(err, ErrInvalidVariantID) {
		t.Errorf("invalid variant ID error = %v, want %v", err, ErrInvalidVariantID)
	}
}

func testClient(t *testing.T, baseURL string) *Client {
	t.Helper()
	client, err := NewClient(baseURL, testToken)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	return client
}
