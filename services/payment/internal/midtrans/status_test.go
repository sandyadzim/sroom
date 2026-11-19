package midtrans

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNotificationSignature(t *testing.T) {
	digest := sha512.Sum512([]byte("demo-order" + "200" + "150000.00" + "synthetic-key"))
	signature := hex.EncodeToString(digest[:])
	if !VerifySignature("demo-order", "200", "150000.00", signature, "synthetic-key") {
		t.Fatal("valid signature rejected")
	}
	for _, test := range []struct{ amount, signature, key string }{
		{"150000", signature, "synthetic-key"}, {"150000.00", signature, "wrong"},
		{"150000.00", "bad-hex", "synthetic-key"}, {"150000.00", signature, ""},
	} {
		if VerifySignature("demo-order", "200", test.amount, test.signature, test.key) {
			t.Fatal("invalid signature accepted")
		}
	}
}

func TestGetStatus(t *testing.T) {
	const body = `{"order_id":"demo-order","transaction_id":"demo-tx","merchant_id":"demo-merchant","transaction_status":"settlement","status_code":"200","payment_type":"bank_transfer","currency":"IDR","gross_amount":"150000.00","fraud_status":"accept","va_numbers":[{"bank":"bni"}],"future_field":true}`
	for _, test := range []struct {
		name, body string
		status     int
		want       error
	}{
		{"settled", body, 200, nil}, {"not found", `{}`, 404, ErrUnavailable},
		{"provider down", `{}`, 503, ErrUnavailable}, {"invalid JSON", `{`, 200, ErrInvalidResponse},
		{"fractional IDR", strings.Replace(body, "150000.00", "150000.01", 1), 200, ErrInvalidResponse},
		{"oversized", strings.Repeat("x", 65537), 200, ErrUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user, password, ok := r.BasicAuth()
				if !ok || user != "synthetic-key" || password != "" || r.Method != "GET" || r.URL.Path != "/v2/demo-order/status" || r.Header.Get("X-Request-ID") != "test-status" {
					t.Error("incorrect provider request")
				}
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.body))
			}))
			defer server.Close()
			client := NewClient("synthetic-key")
			client.baseURL = server.URL
			result, err := client.GetStatus(t.Context(), "demo-order", "test-status")
			if !errors.Is(err, test.want) {
				t.Fatalf("error=%v want=%v", err, test.want)
			}
			if err == nil && (result.AmountIDR != 150000 || result.Bank != "bni" || result.Status != "settlement" || len(result.Digest) != 64) {
				t.Fatalf("invalid parsed status: %+v", result)
			}
		})
	}
}

func TestStatusCancellation(t *testing.T) {
	client := NewClient("synthetic-key")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("canceled request reached provider") }))
	defer server.Close()
	client.baseURL = server.URL
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := client.GetStatus(ctx, "demo-order", "cancel"); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("error=%v", err)
	}
}
