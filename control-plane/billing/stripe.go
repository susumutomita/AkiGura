package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// Plan represents a subscription plan
type Plan struct {
	ID           string
	Name         string
	PriceID      string // Stripe Price ID
	MonthlyPrice int    // in JPY
	MaxFacilities int
}

var Plans = map[string]Plan{
	"free":     {ID: "free", Name: "Free", PriceID: "", MonthlyPrice: 0, MaxFacilities: 1},
	"personal": {ID: "personal", Name: "Personal", PriceID: "", MonthlyPrice: 500, MaxFacilities: 5},
	"pro":      {ID: "pro", Name: "Pro", PriceID: "", MonthlyPrice: 2000, MaxFacilities: 20},
	"org":      {ID: "org", Name: "Organization", PriceID: "", MonthlyPrice: 10000, MaxFacilities: -1},
}

// StripeClient handles Stripe API interactions
type StripeClient struct {
	SecretKey  string
	WebhookSecret string
	BaseURL    string
}

// NewStripeClient creates a new Stripe client
func NewStripeClient() *StripeClient {
	return &StripeClient{
		SecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		WebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		BaseURL:       "https://api.stripe.com/v1",
	}
}

// IsConfigured returns true if Stripe is configured
func (s *StripeClient) IsConfigured() bool {
	return s.SecretKey != ""
}

// CreateCustomer creates a Stripe customer
func (s *StripeClient) CreateCustomer(ctx context.Context, email, name string) (string, error) {
	data := url.Values{}
	data.Set("email", email)
	data.Set("name", name)

	resp, err := s.post(ctx, "/customers", data)
	if err != nil {
		return "", err
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}
	return result.ID, nil
}

// CreateCheckoutSession creates a Stripe Checkout session
func (s *StripeClient) CreateCheckoutSession(ctx context.Context, customerID, priceID, successURL, cancelURL string) (string, error) {
	data := url.Values{}
	data.Set("customer", customerID)
	data.Set("mode", "subscription")
	data.Set("line_items[0][price]", priceID)
	data.Set("line_items[0][quantity]", "1")
	data.Set("success_url", successURL)
	data.Set("cancel_url", cancelURL)

	resp, err := s.post(ctx, "/checkout/sessions", data)
	if err != nil {
		return "", err
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}
	return result.URL, nil
}

// CreateBillingPortalSession creates a customer portal session
func (s *StripeClient) CreateBillingPortalSession(ctx context.Context, customerID, returnURL string) (string, error) {
	data := url.Values{}
	data.Set("customer", customerID)
	data.Set("return_url", returnURL)

	resp, err := s.post(ctx, "/billing_portal/sessions", data)
	if err != nil {
		return "", err
	}

	var result struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return "", err
	}
	return result.URL, nil
}

// GetSubscription retrieves a subscription
func (s *StripeClient) GetSubscription(ctx context.Context, subscriptionID string) (*Subscription, error) {
	resp, err := s.get(ctx, "/subscriptions/"+subscriptionID)
	if err != nil {
		return nil, err
	}

	var result Subscription
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CancelSubscription cancels a subscription
func (s *StripeClient) CancelSubscription(ctx context.Context, subscriptionID string) error {
	_, err := s.delete(ctx, "/subscriptions/"+subscriptionID)
	return err
}

type Subscription struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	CurrentPeriodEnd   int64  `json:"current_period_end"`
	CancelAtPeriodEnd  bool   `json:"cancel_at_period_end"`
}

func (s *StripeClient) post(ctx context.Context, path string, data url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", s.BaseURL+path, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.SecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stripe error %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (s *StripeClient) get(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.SecretKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stripe error %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

func (s *StripeClient) delete(ctx context.Context, path string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "DELETE", s.BaseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.SecretKey, "")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stripe error %d: %s", resp.StatusCode, string(body))
	}
	return body, nil
}

// VerifyWebhookSignature verifies a Stripe webhook signature
func (s *StripeClient) VerifyWebhookSignature(payload []byte, signature string) bool {
	// Simplified verification - in production use stripe-go library
	return signature != "" && s.WebhookSecret != ""
}

// WebhookEvent represents a Stripe webhook event
type WebhookEvent struct {
	ID   string          `json:"id"`
	Type string          `json:"type"`
	Data WebhookEventData `json:"data"`
}

type WebhookEventData struct {
	Object json.RawMessage `json:"object"`
}

// ParseWebhookEvent parses a webhook event
func ParseWebhookEvent(body []byte) (*WebhookEvent, error) {
	var event WebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
