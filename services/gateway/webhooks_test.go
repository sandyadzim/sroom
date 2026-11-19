package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMidtransWebhookProxy(t *testing.T) {
	const body = `{"order_id":"synthetic-order","gross_amount":"150000.00","extra":true}`
	calls := 0
	upstreamStatus := 200
	upstream := testUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		forwarded, err := io.ReadAll(r.Body)
		if err != nil || string(forwarded) != body || r.URL.Path != "/webhooks/midtrans" || r.Header.Get("Content-Type") != "application/json" || r.Header.Get("X-Request-ID") != "webhook-test" || r.Header.Get("Cookie") != "" {
			t.Error("webhook was altered")
		}
		w.WriteHeader(upstreamStatus)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	app := newRouter(upstream, upstream, upstream, upstream, testLogger())
	for _, status := range []int{200, 503} {
		upstreamStatus = status
		req := httptest.NewRequest("POST", "/webhooks/midtrans", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Request-ID", "webhook-test")
		res := request(app, req)
		if res.Code != status || res.Body.String() != `{"status":"ok"}` {
			t.Fatalf("response=%d %s", res.Code, res.Body.String())
		}
	}
	req := httptest.NewRequest("POST", "/webhooks/midtrans", strings.NewReader(strings.Repeat("x", 16385)))
	if res := request(app, req); res.Code != 413 {
		t.Fatalf("oversized=%d", res.Code)
	}
	if calls != 2 {
		t.Fatalf("upstream calls=%d", calls)
	}
	if res := request(app, httptest.NewRequest("GET", "/webhooks/midtrans", nil)); res.Code == 200 {
		t.Fatal("GET accepted")
	}
}
