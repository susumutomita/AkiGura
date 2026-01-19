package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/stripe/stripe-go/v79/webhook"
)

// Plan represents a subscription plan
type Plan struct {
	ID             string
	Name           string
	MonthlyPriceID string // Stripe Price ID for monthly
	YearlyPriceID  string // Stripe Price ID for yearly
	MonthlyPrice   int    // in JPY
	YearlyPrice    int    // in JPY (usually ~20% off)
	MaxFacilities  int
}

// GetPriceID returns the appropriate Stripe price ID based on interval
func (p Plan) GetPriceID(interval string) string {
	if interval == "yearly" {
		return p.YearlyPriceID
	}
	return p.MonthlyPriceID
}

// GetPrice returns the price for the given interval
func (p Plan) GetPrice(interval string) int {
	if interval == "yearly" {
		return p.YearlyPrice
	}
	return p.MonthlyPrice
}

var Plans = map[string]Plan{
	"free": {
		ID:             "free",
		Name:           "Free",
		MonthlyPriceID: "",
		YearlyPriceID:  "",
		MonthlyPrice:   0,
		YearlyPrice:    0,
		MaxFacilities:  1,
	},
	"personal": {
		ID:             "personal",
		Name:           "Personal",
		MonthlyPriceID: os.Getenv("STRIPE_PRICE_PERSONAL_MONTHLY"),
		YearlyPriceID:  os.Getenv("STRIPE_PRICE_PERSONAL_YEARLY"),
		MonthlyPrice:   500,
		YearlyPrice:    4800, // 20% off (500*12*0.8)
		MaxFacilities:  5,
	},
	"pro": {
		ID:             "pro",
		Name:           "Pro",
		MonthlyPriceID: os.Getenv("STRIPE_PRICE_PRO_MONTHLY"),
		YearlyPriceID:  os.Getenv("STRIPE_PRICE_PRO_YEARLY"),
		MonthlyPrice:   2000,
		YearlyPrice:    19200, // 20% off (2000*12*0.8)
		MaxFacilities:  20,
	},
	"org": {
		ID:             "org",
		Name:           "Organization",
		MonthlyPriceID: os.Getenv("STRIPE_PRICE_ORG_MONTHLY"),
		YearlyPriceID:  os.Getenv("STRIPE_PRICE_ORG_YEARLY"),
		MonthlyPrice:   10000,
		YearlyPrice:    96000, // 20% off (10000*12*0.8)
		MaxFacilities:  -1,
	},
}

// StripeClient handles Stripe API interactions
type StripeClient struct {
	SecretKey     string
	WebhookSecret string
	BaseURL       string
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

// CheckoutOptions contains options for creating a checkout session
type CheckoutOptions struct {
	CustomerID     string
	PriceID        string
	SuccessURL     string
	CancelURL      string
	TeamID         string // metadata
	AllowPromoCode bool   // allow user to enter promo code
	StripeCouponID string // pre-applied coupon
}

// CreateCheckoutSession creates a Stripe Checkout session
func (s *StripeClient) CreateCheckoutSession(ctx context.Context, opts CheckoutOptions) (string, error) {
	data := url.Values{}
	data.Set("customer", opts.CustomerID)
	data.Set("mode", "subscription")
	data.Set("line_items[0][price]", opts.PriceID)
	data.Set("line_items[0][quantity]", "1")
	data.Set("success_url", opts.SuccessURL)
	data.Set("cancel_url", opts.CancelURL)

	if opts.TeamID != "" {
		data.Set("metadata[team_id]", opts.TeamID)
		data.Set("subscription_data[metadata][team_id]", opts.TeamID)
	}

	if opts.AllowPromoCode {
		data.Set("allow_promotion_codes", "true")
	}

	if opts.StripeCouponID != "" {
		data.Set("discounts[0][coupon]", opts.StripeCouponID)
	}

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

// CreateCoupon creates a Stripe coupon
func (s *StripeClient) CreateCoupon(ctx context.Context, id string, percentOff int, durationMonths int) (string, error) {
	data := url.Values{}
	data.Set("id", id)
	data.Set("percent_off", fmt.Sprintf("%d", percentOff))
	data.Set("duration", "repeating")
	data.Set("duration_in_months", fmt.Sprintf("%d", durationMonths))
	data.Set("currency", "jpy")

	resp, err := s.post(ctx, "/coupons", data)
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

// CreateFixedAmountCoupon creates a Stripe coupon with fixed amount off
func (s *StripeClient) CreateFixedAmountCoupon(ctx context.Context, id string, amountOff int, durationMonths int) (string, error) {
	data := url.Values{}
	data.Set("id", id)
	data.Set("amount_off", fmt.Sprintf("%d", amountOff))
	data.Set("duration", "repeating")
	data.Set("duration_in_months", fmt.Sprintf("%d", durationMonths))
	data.Set("currency", "jpy")

	resp, err := s.post(ctx, "/coupons", data)
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

type Subscription struct {
	ID                string `json:"id"`
	Status            string `json:"status"`
	CurrentPeriodEnd  int64  `json:"current_period_end"`
	CancelAtPeriodEnd bool   `json:"cancel_at_period_end"`
}

// do executes an HTTP request to the Stripe API
func (s *StripeClient) do(ctx context.Context, method, path string, data url.Values) ([]byte, error) {
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, s.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.SecretKey, "")
	if data != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("stripe error %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

func (s *StripeClient) post(ctx context.Context, path string, data url.Values) ([]byte, error) {
	return s.do(ctx, "POST", path, data)
}

func (s *StripeClient) get(ctx context.Context, path string) ([]byte, error) {
	return s.do(ctx, "GET", path, nil)
}

func (s *StripeClient) delete(ctx context.Context, path string) ([]byte, error) {
	return s.do(ctx, "DELETE", path, nil)
}

// VerifyWebhookSignature verifies a Stripe webhook signature
func (s *StripeClient) VerifyWebhookSignature(payload []byte, signature string) bool {
	if s.WebhookSecret == "" || len(payload) == 0 || signature == "" {
		return false
	}

	if _, err := webhook.ConstructEvent(payload, signature, s.WebhookSecret); err != nil {
		slog.Warn("stripe webhook signature invalid", "error", err)
		return false
	}
	return true
}

// WebhookEvent represents a Stripe webhook event
type WebhookEvent struct {
	ID   string           `json:"id"`
	Type string           `json:"type"`
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
