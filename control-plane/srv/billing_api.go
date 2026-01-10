package srv

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"srv.exe.dev/billing"
)

// HandleCreateCheckout creates a Stripe checkout session
func (s *Server) HandleCreateCheckout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeamID  string `json:"team_id"`
		Plan    string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	stripe := billing.NewStripeClient()
	if !stripe.IsConfigured() {
		s.jsonError(w, "Stripe not configured", http.StatusServiceUnavailable)
		return
	}

	plan, ok := billing.Plans[req.Plan]
	if !ok || plan.PriceID == "" {
		s.jsonError(w, "invalid plan", http.StatusBadRequest)
		return
	}

	// Get team
	team, err := s.Queries.GetTeam(r.Context(), req.TeamID)
	if err != nil {
		s.jsonError(w, "team not found", http.StatusNotFound)
		return
	}

	// Create or get Stripe customer
	customerID := "" // Would be stored in DB
	if customerID == "" {
		customerID, err = stripe.CreateCustomer(r.Context(), team.Email, team.Name)
		if err != nil {
			slog.Error("create stripe customer", "error", err)
			s.jsonError(w, "failed to create customer", http.StatusInternalServerError)
			return
		}
	}

	// Create checkout session
	baseURL := "https://" + r.Host
	checkoutURL, err := stripe.CreateCheckoutSession(
		r.Context(),
		customerID,
		plan.PriceID,
		baseURL+"/user?success=true",
		baseURL+"/user?canceled=true",
	)
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

	stripe := billing.NewStripeClient()
	if !stripe.IsConfigured() {
		s.jsonError(w, "Stripe not configured", http.StatusServiceUnavailable)
		return
	}

	// Get customer ID from DB (placeholder)
	customerID := "" // Would be stored in DB
	if customerID == "" {
		s.jsonError(w, "no subscription found", http.StatusNotFound)
		return
	}

	baseURL := "https://" + r.Host
	portalURL, err := stripe.CreateBillingPortalSession(r.Context(), customerID, baseURL+"/user")
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
	stripe := billing.NewStripeClient()

	if !stripe.VerifyWebhookSignature(body, signature) {
		s.jsonError(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	event, err := billing.ParseWebhookEvent(body)
	if err != nil {
		s.jsonError(w, "invalid event", http.StatusBadRequest)
		return
	}

	handler := billing.NewWebhookHandler(s.DB)
	if err := handler.HandleEvent(r.Context(), event); err != nil {
		slog.Error("handle webhook", "error", err)
		s.jsonError(w, "webhook handling failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// HandlePlans returns available plans
func (s *Server) HandlePlans(w http.ResponseWriter, r *http.Request) {
	plans := make([]map[string]interface{}, 0)
	for _, p := range billing.Plans {
		plans = append(plans, map[string]interface{}{
			"id":            p.ID,
			"name":          p.Name,
			"monthly_price": p.MonthlyPrice,
			"max_facilities": p.MaxFacilities,
		})
	}
	s.jsonResponse(w, plans)
}
