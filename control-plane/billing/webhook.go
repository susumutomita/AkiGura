package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
)

// WebhookHandler handles Stripe webhook events
type WebhookHandler struct {
	DB     *sql.DB
	Stripe *StripeClient
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(db *sql.DB) *WebhookHandler {
	return &WebhookHandler{
		DB:     db,
		Stripe: NewStripeClient(),
	}
}

// HandleEvent processes a Stripe webhook event
func (h *WebhookHandler) HandleEvent(ctx context.Context, event *WebhookEvent) error {
	slog.Info("handling stripe webhook", "type", event.Type, "id", event.ID)

	switch event.Type {
	case "checkout.session.completed":
		return h.handleCheckoutCompleted(ctx, event.Data.Object)
	case "customer.subscription.updated":
		return h.handleSubscriptionUpdated(ctx, event.Data.Object)
	case "customer.subscription.deleted":
		return h.handleSubscriptionDeleted(ctx, event.Data.Object)
	case "invoice.paid":
		return h.handleInvoicePaid(ctx, event.Data.Object)
	case "invoice.payment_failed":
		return h.handleInvoicePaymentFailed(ctx, event.Data.Object)
	default:
		slog.Debug("unhandled webhook event", "type", event.Type)
	}
	return nil
}

func (h *WebhookHandler) handleCheckoutCompleted(ctx context.Context, data json.RawMessage) error {
	var session struct {
		Customer     string `json:"customer"`
		Subscription string `json:"subscription"`
		Metadata     struct {
			TeamID string `json:"team_id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &session); err != nil {
		return err
	}

	slog.Info("checkout completed", "customer", session.Customer, "subscription", session.Subscription)

	// Update team with Stripe customer ID and subscription
	_, err := h.DB.ExecContext(ctx, `
		UPDATE teams SET 
			status = 'active',
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, session.Metadata.TeamID)
	return err
}

func (h *WebhookHandler) handleSubscriptionUpdated(ctx context.Context, data json.RawMessage) error {
	var sub struct {
		ID       string `json:"id"`
		Customer string `json:"customer"`
		Status   string `json:"status"`
		Items    struct {
			Data []struct {
				Price struct {
					ID string `json:"id"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &sub); err != nil {
		return err
	}

	slog.Info("subscription updated", "id", sub.ID, "status", sub.Status)

	// Map price ID to plan
	plan := "free"
	if len(sub.Items.Data) > 0 {
		priceID := sub.Items.Data[0].Price.ID
		for _, p := range Plans {
			if p.PriceID == priceID {
				plan = p.ID
				break
			}
		}
	}

	status := "active"
	if sub.Status == "canceled" || sub.Status == "unpaid" {
		status = "paused"
	}

	// Note: In production, you'd look up team by stripe_customer_id
	// For now, this is a placeholder
	slog.Info("would update team", "plan", plan, "status", status)
	return nil
}

func (h *WebhookHandler) handleSubscriptionDeleted(ctx context.Context, data json.RawMessage) error {
	var sub struct {
		ID       string `json:"id"`
		Customer string `json:"customer"`
	}
	if err := json.Unmarshal(data, &sub); err != nil {
		return err
	}

	slog.Info("subscription deleted", "id", sub.ID, "customer", sub.Customer)
	// Downgrade team to free plan
	return nil
}

func (h *WebhookHandler) handleInvoicePaid(ctx context.Context, data json.RawMessage) error {
	var invoice struct {
		ID       string `json:"id"`
		Customer string `json:"customer"`
		Total    int    `json:"total"`
	}
	if err := json.Unmarshal(data, &invoice); err != nil {
		return err
	}

	slog.Info("invoice paid", "id", invoice.ID, "amount", invoice.Total)
	return nil
}

func (h *WebhookHandler) handleInvoicePaymentFailed(ctx context.Context, data json.RawMessage) error {
	var invoice struct {
		ID       string `json:"id"`
		Customer string `json:"customer"`
	}
	if err := json.Unmarshal(data, &invoice); err != nil {
		return err
	}

	slog.Warn("invoice payment failed", "id", invoice.ID, "customer", invoice.Customer)
	// Could send notification to team or pause account
	return nil
}
