package srv

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"srv.exe.dev/billing"
	"srv.exe.dev/db/dbgen"
)

// HandleCreateCheckout creates a Stripe checkout session
func (s *Server) HandleCreateCheckout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeamID    string `json:"team_id"`
		Plan      string `json:"plan"`
		Interval  string `json:"interval"` // monthly or yearly
		PromoCode string `json:"promo_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Default to monthly
	if req.Interval == "" {
		req.Interval = "monthly"
	}
	if req.Interval != "monthly" && req.Interval != "yearly" {
		s.jsonError(w, "interval must be 'monthly' or 'yearly'", http.StatusBadRequest)
		return
	}

	stripeClient := billing.NewStripeClient()
	if !stripeClient.IsConfigured() {
		s.jsonError(w, "Stripe not configured", http.StatusServiceUnavailable)
		return
	}

	plan, ok := billing.Plans[req.Plan]
	if !ok {
		s.jsonError(w, "invalid plan", http.StatusBadRequest)
		return
	}

	priceID := plan.GetPriceID(req.Interval)
	if priceID == "" {
		s.jsonError(w, "plan not available for billing", http.StatusBadRequest)
		return
	}

	// Get team
	team, err := s.Queries.GetTeam(r.Context(), req.TeamID)
	if err != nil {
		s.jsonError(w, "team not found", http.StatusNotFound)
		return
	}

	// Create or get Stripe customer
	customerID := ""
	if team.StripeCustomerID.Valid {
		customerID = team.StripeCustomerID.String
	}
	if customerID == "" {
		customerID, err = stripeClient.CreateCustomer(r.Context(), team.Email, team.Name)
		if err != nil {
			slog.Error("create stripe customer", "error", err)
			s.jsonError(w, "failed to create customer", http.StatusInternalServerError)
			return
		}
		// Save customer ID to DB
		if err := s.Queries.UpdateTeamStripeCustomer(r.Context(), dbgen.UpdateTeamStripeCustomerParams{
			ID:               team.ID,
			StripeCustomerID: sql.NullString{String: customerID, Valid: true},
		}); err != nil {
			slog.Error("save stripe customer id", "error", err)
		}
	}

	// Validate promo code if provided
	var stripeCouponID string
	if req.PromoCode != "" {
		promoCode, err := s.Queries.GetPromoCodeByCode(r.Context(), strings.ToUpper(req.PromoCode))
		if err != nil {
			s.jsonError(w, "invalid promo code", http.StatusBadRequest)
			return
		}

		// Check if promo code applies to this plan
		if promoCode.AppliesTo.Valid && promoCode.AppliesTo.String != req.Plan {
			s.jsonError(w, "promo code does not apply to this plan", http.StatusBadRequest)
			return
		}

		// Check if team already used this promo code
		_, err = s.Queries.GetPromoCodeUsageByTeam(r.Context(), dbgen.GetPromoCodeUsageByTeamParams{
			PromoCodeID: promoCode.ID,
			TeamID:      team.ID,
		})
		if err == nil {
			s.jsonError(w, "promo code already used", http.StatusBadRequest)
			return
		}

		// Create Stripe coupon based on promo code
		couponID := "akigura_" + promoCode.Code
		if promoCode.DiscountType == "percent" {
			_, err = stripeClient.CreateCoupon(r.Context(), couponID, int(promoCode.DiscountValue), 12)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				slog.Error("create coupon", "error", err)
				// Continue without coupon if creation fails
			} else {
				stripeCouponID = couponID
			}
		} else if promoCode.DiscountType == "fixed" {
			_, err = stripeClient.CreateFixedAmountCoupon(r.Context(), couponID, int(promoCode.DiscountValue), 12)
			if err != nil && !strings.Contains(err.Error(), "already exists") {
				slog.Error("create coupon", "error", err)
			} else {
				stripeCouponID = couponID
			}
		}

		// Record promo code usage
		_, err = s.Queries.CreatePromoCodeUsage(r.Context(), dbgen.CreatePromoCodeUsageParams{
			ID:          uuid.NewString(),
			PromoCodeID: promoCode.ID,
			TeamID:      team.ID,
		})
		if err != nil {
			slog.Error("create promo code usage", "error", err)
		}

		// Increment usage count
		if err := s.Queries.IncrementPromoCodeUsage(r.Context(), promoCode.ID); err != nil {
			slog.Error("increment promo code usage", "error", err)
		}
	}

	// Create checkout session
	baseURL := "https://" + r.Host
	checkoutURL, err := stripeClient.CreateCheckoutSession(r.Context(), billing.CheckoutOptions{
		CustomerID:     customerID,
		PriceID:        priceID,
		SuccessURL:     baseURL + "/user?success=true",
		CancelURL:      baseURL + "/user?canceled=true",
		TeamID:         team.ID,
		AllowPromoCode: req.PromoCode == "", // Allow entering promo code only if not pre-applied
		StripeCouponID: stripeCouponID,
	})
	if err != nil {
		slog.Error("create checkout session", "error", err)
		s.jsonError(w, "failed to create checkout", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, map[string]string{"url": checkoutURL})
}

// HandleBillingPortal creates a Stripe billing portal session
func (s *Server) HandleBillingPortal(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeamID string `json:"team_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	stripeClient := billing.NewStripeClient()
	if !stripeClient.IsConfigured() {
		s.jsonError(w, "Stripe not configured", http.StatusServiceUnavailable)
		return
	}

	// Get team and customer ID from DB
	team, err := s.Queries.GetTeam(r.Context(), req.TeamID)
	if err != nil {
		s.jsonError(w, "team not found", http.StatusNotFound)
		return
	}

	if !team.StripeCustomerID.Valid || team.StripeCustomerID.String == "" {
		s.jsonError(w, "no subscription found", http.StatusNotFound)
		return
	}

	baseURL := "https://" + r.Host
	portalURL, err := stripeClient.CreateBillingPortalSession(r.Context(), team.StripeCustomerID.String, baseURL+"/user")
	if err != nil {
		slog.Error("create billing portal", "error", err)
		s.jsonError(w, "failed to create portal", http.StatusInternalServerError)
		return
	}

	s.jsonResponse(w, map[string]string{"url": portalURL})
}

// HandleStripeWebhook handles Stripe webhook events
func (s *Server) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.jsonError(w, "failed to read body", http.StatusBadRequest)
		return
	}

	signature := r.Header.Get("Stripe-Signature")
	stripeClient := billing.NewStripeClient()

	if !stripeClient.VerifyWebhookSignature(body, signature) {
		s.jsonError(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	event, err := billing.ParseWebhookEvent(body)
	if err != nil {
		s.jsonError(w, "invalid event", http.StatusBadRequest)
		return
	}

	handler := billing.NewWebhookHandler(s.DB, s.Queries)
	if err := handler.HandleEvent(r.Context(), event); err != nil {
		slog.Error("handle webhook", "error", err)
		s.jsonError(w, "webhook handling failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandlePlans returns available plans with pricing info
func (s *Server) HandlePlans(w http.ResponseWriter, r *http.Request) {
	plans := make([]map[string]interface{}, 0)
	for _, p := range billing.Plans {
		plans = append(plans, map[string]interface{}{
			"id":              p.ID,
			"name":            p.Name,
			"monthly_price":   p.MonthlyPrice,
			"yearly_price":    p.YearlyPrice,
			"max_facilities":  p.MaxFacilities,
			"yearly_savings":  p.MonthlyPrice*12 - p.YearlyPrice,
		})
	}
	s.jsonResponse(w, plans)
}

// HandleValidatePromoCode validates a promo code without using it
func (s *Server) HandleValidatePromoCode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code   string `json:"code"`
		TeamID string `json:"team_id"`
		Plan   string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	promoCode, err := s.Queries.GetPromoCodeByCode(r.Context(), strings.ToUpper(req.Code))
	if err != nil {
		s.jsonError(w, "invalid promo code", http.StatusBadRequest)
		return
	}

	// Check if promo code applies to this plan
	if promoCode.AppliesTo.Valid && promoCode.AppliesTo.String != req.Plan {
		s.jsonError(w, "promo code does not apply to this plan", http.StatusBadRequest)
		return
	}

	// Check if team already used this promo code
	if req.TeamID != "" {
		_, err = s.Queries.GetPromoCodeUsageByTeam(r.Context(), dbgen.GetPromoCodeUsageByTeamParams{
			PromoCodeID: promoCode.ID,
			TeamID:      req.TeamID,
		})
		if err == nil {
			s.jsonError(w, "promo code already used", http.StatusBadRequest)
			return
		}
	}

	// Calculate discount
	var discountDescription string
	if promoCode.DiscountType == "percent" {
		discountDescription = fmt.Sprintf("%d%% off", promoCode.DiscountValue)
	} else {
		discountDescription = fmt.Sprintf("¥%d off", promoCode.DiscountValue)
	}

	s.jsonResponse(w, map[string]interface{}{
		"valid":       true,
		"code":        promoCode.Code,
		"discount":    discountDescription,
		"type":        promoCode.DiscountType,
		"value":       promoCode.DiscountValue,
	})
}
