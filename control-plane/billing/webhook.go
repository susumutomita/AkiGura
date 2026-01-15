package billing

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"srv.exe.dev/db/dbgen"
)

// WebhookHandler handles Stripe webhook events
type WebhookHandler struct {
	DB      *sql.DB
	Queries *dbgen.Queries
	Stripe  *StripeClient
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(db *sql.DB, queries *dbgen.Queries) *WebhookHandler {
	return &WebhookHandler{
		DB:      db,
		Queries: queries,
		Stripe:  NewStripeClient(),
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

	slog.Info("checkout completed", "customer", session.Customer, "subscription", session.Subscription, "team_id", session.Metadata.TeamID)

	// Get subscription details from Stripe
	sub, err := h.Stripe.GetSubscription(ctx, session.Subscription)
	if err != nil {
		slog.Error("failed to get subscription", "error", err)
		return err
	}

	// Determine plan from price ID and interval
	plan, interval := h.determinePlanFromSubscription(ctx, session.Subscription)

	// Update team with subscription info
	return h.Queries.UpdateTeamSubscription(ctx, dbgen.UpdateTeamSubscriptionParams{
		ID:                   session.Metadata.TeamID,
		StripeSubscriptionID: sql.NullString{String: session.Subscription, Valid: true},
		Plan:                 plan,
		BillingInterval:      sql.NullString{String: interval, Valid: true},
		CurrentPeriodEnd:     sql.NullTime{Time: time.Unix(sub.CurrentPeriodEnd, 0), Valid: true},
	})
}

func (h *WebhookHandler) handleSubscriptionUpdated(ctx context.Context, data json.RawMessage) error {
	var sub struct {
		ID               string `json:"id"`
		Customer         string `json:"customer"`
		Status           string `json:"status"`
		CurrentPeriodEnd int64  `json:"current_period_end"`
		Items            struct {
			Data []struct {
				Price struct {
					ID        string `json:"id"`
					Recurring struct {
						Interval string `json:"interval"`
					} `json:"recurring"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
		Metadata struct {
			TeamID string `json:"team_id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &sub); err != nil {
		return err
	}

	slog.Info("subscription updated", "id", sub.ID, "status", sub.Status, "customer", sub.Customer)

	// Get team by stripe customer ID
	team, err := h.Queries.GetTeamByStripeCustomer(ctx, sql.NullString{String: sub.Customer, Valid: true})
	if err != nil {
		// Try metadata team_id
		if sub.Metadata.TeamID != "" {
			team, err = h.Queries.GetTeam(ctx, sub.Metadata.TeamID)
		}
		if err != nil {
			slog.Error("failed to find team for subscription", "customer", sub.Customer, "error", err)
			return nil // Don't fail webhook
		}
	}

	// Map price ID to plan
	plan := "free"
	interval := "monthly"
	if len(sub.Items.Data) > 0 {
		priceID := sub.Items.Data[0].Price.ID
		for _, p := range Plans {
			if p.MonthlyPriceID == priceID {
				plan = p.ID
				interval = "monthly"
				break
			}
			if p.YearlyPriceID == priceID {
				plan = p.ID
				interval = "yearly"
				break
			}
		}
	}

	// Update team subscription
	if sub.Status == "active" || sub.Status == "trialing" {
		return h.Queries.UpdateTeamSubscription(ctx, dbgen.UpdateTeamSubscriptionParams{
			ID:                   team.ID,
			StripeSubscriptionID: sql.NullString{String: sub.ID, Valid: true},
			Plan:                 plan,
			BillingInterval:      sql.NullString{String: interval, Valid: true},
			CurrentPeriodEnd:     sql.NullTime{Time: time.Unix(sub.CurrentPeriodEnd, 0), Valid: true},
		})
	}

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
	return h.Queries.CancelTeamSubscription(ctx, sql.NullString{String: sub.Customer, Valid: true})
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

func (h *WebhookHandler) determinePlanFromSubscription(ctx context.Context, subscriptionID string) (string, string) {
	sub, err := h.Stripe.GetSubscription(ctx, subscriptionID)
	if err != nil {
		return "personal", "monthly" // Default
	}

	// This is a simplified version - in production, get price ID from subscription items
	_ = sub
	return "personal", "monthly"
}
