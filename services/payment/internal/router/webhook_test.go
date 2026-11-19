package router

import (
	"net/http/httptest"
	"strings"
	"testing"

	"payment/internal/handler"
	"payment/internal/service"
)

func TestWebhookInput(t *testing.T) {
	const valid = `{"order_id":"demo-order","status_code":"200","gross_amount":"150000.00","signature_key":"invalid"}`
	for _, test := range []struct {
		name, body, contentType, key string
		status                       int
		code                         string
	}{
		{"signature without customer authentication", valid, "application/json", "synthetic-key", 401, "INVALID_SIGNATURE"},
		{"unknown fields allowed", strings.TrimSuffix(valid, "}") + `,"future_field":{"hello":true}}`, "application/json", "synthetic-key", 401, "INVALID_SIGNATURE"},
		{"unconfigured", valid, "application/json", "", 503, "PAYMENT_NOT_CONFIGURED"},
		{"malformed", `{`, "application/json", "", 400, "INVALID_REQUEST"},
		{"array", `[]`, "application/json", "", 400, "INVALID_REQUEST"},
		{"numeric signed field", strings.Replace(valid, `"150000.00"`, "150000", 1), "application/json", "", 400, "INVALID_REQUEST"},
		{"trailing object", valid + valid, "application/json", "", 400, "INVALID_REQUEST"},
		{"missing signature", `{}`, "application/json", "", 400, "INVALID_REQUEST"},
		{"media type", valid, "text/plain", "", 400, "INVALID_REQUEST"},
		{"oversized", strings.Repeat("a", 16*1024+1), "application/json", "", 413, "PAYLOAD_TOO_LARGE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			app, err := New(handler.NewHealthHandler(nil))
			if err != nil {
				t.Fatal(err)
			}
			RegisterWebhook(app, handler.NewWebhookHandler(service.NewWebhookService(nil, nil, test.key)))
			req := httptest.NewRequest("POST", "/webhooks/midtrans", strings.NewReader(test.body))
			req.Header.Set("Content-Type", test.contentType)
			res := httptest.NewRecorder()
			app.ServeHTTP(res, req)
			if res.Code != test.status || !strings.Contains(res.Body.String(), `"code":"`+test.code+`"`) {
				t.Fatalf("response=%d %s", res.Code, res.Body.String())
			}
		})
	}
}
