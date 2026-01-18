package srv

import (
	"encoding/json"
	"os"
	"testing"

	"srv.exe.dev/billing"
	"srv.exe.dev/webhooktest"
)

func TestVerifyWebhookSignature(t *testing.T) {
	secret := "whsec_test"
	os.Setenv("STRIPE_SECRET_KEY", "sk_test_dummy")
	os.Setenv("STRIPE_WEBHOOK_SECRET", secret)

	client := billing.NewStripeClient()

	event := map[string]any{
		"id":          "evt_test_webhook",
		"type":        "checkout.session.completed",
		"api_version": "2024-06-20",
		"data":        map[string]any{"object": map[string]any{"foo": "bar"}},
	}
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	sigHeader := webhooktest.NewSignedHeader(body, secret)

	if !client.VerifyWebhookSignature(body, sigHeader) {
		t.Fatalf("expected signature to be valid")
	}
}
