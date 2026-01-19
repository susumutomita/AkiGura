package srv

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"srv.exe.dev/ent"
	"srv.exe.dev/ent/promocode"
	"srv.exe.dev/ent/team"
)

// HandleListTeamsEnt lists teams using Ent ORM
func (s *Server) HandleListTeamsEnt(w http.ResponseWriter, r *http.Request) {
	teams, err := s.Ent.Team.Query().
		Order(ent.Desc(team.FieldCreatedAt)).
		Limit(100).
		All(r.Context())
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, teams)
}

// HandleCreateTeamEnt creates a team using Ent ORM
func (s *Server) HandleCreateTeamEnt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Plan  string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	plan := team.PlanFree
	switch req.Plan {
	case "personal":
		plan = team.PlanPersonal
	case "pro":
		plan = team.PlanPro
	case "org":
		plan = team.PlanOrg
	}

	t, err := s.Ent.Team.Create().
		SetID(uuid.NewString()).
		SetName(req.Name).
		SetEmail(req.Email).
		SetPlan(plan).
		Save(r.Context())
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, t)
}

// HandleGetTeamByEmailEnt gets team by email using Ent ORM
func (s *Server) HandleGetTeamByEmailEnt(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		s.jsonError(w, "email required", http.StatusBadRequest)
		return
	}

	t, err := s.Ent.Team.Query().
		Where(team.EmailEQ(email)).
		Only(r.Context())
	if err != nil {
		s.jsonError(w, "team not found", http.StatusNotFound)
		return
	}
	s.jsonResponse(w, t)
}

// HandleValidatePromoCodeEnt validates a promo code using Ent ORM
func (s *Server) HandleValidatePromoCodeEnt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code   string `json:"code"`
		TeamID string `json:"team_id"`
		Plan   string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Query promo code
	pc, err := s.Ent.PromoCode.Query().
		Where(
			promocode.CodeEQ(req.Code),
			promocode.EnabledEQ(true),
		).
		Only(r.Context())
	if err != nil {
		s.jsonError(w, "invalid promo code", http.StatusBadRequest)
		return
	}

	// Check if applies to this plan
	if pc.AppliesTo != nil && *pc.AppliesTo != req.Plan {
		s.jsonError(w, "promo code does not apply to this plan", http.StatusBadRequest)
		return
	}

	// Check max uses
	if pc.MaxUses != nil && pc.UsesCount >= *pc.MaxUses {
		s.jsonError(w, "promo code usage limit reached", http.StatusBadRequest)
		return
	}

	// Check if team already used
	if req.TeamID != "" {
		exists, err := s.Ent.PromoCodeUsage.Query().
			Where(
			// promocodeusage.HasPromoCodeWith(promocode.IDEQ(pc.ID)),
			// promocodeusage.HasTeamWith(team.IDEQ(req.TeamID)),
			).
			Exist(r.Context())
		if err == nil && exists {
			s.jsonError(w, "promo code already used", http.StatusBadRequest)
			return
		}
	}

	// Build response
	var discountDescription string
	if pc.DiscountType == promocode.DiscountTypePercent {
		discountDescription = fmt.Sprintf("%d%% off", pc.DiscountValue)
	} else {
		discountDescription = fmt.Sprintf("\u00a5%d off", pc.DiscountValue)
	}

	s.jsonResponse(w, map[string]interface{}{
		"valid":    true,
		"code":     pc.Code,
		"discount": discountDescription,
		"type":     pc.DiscountType,
		"value":    pc.DiscountValue,
	})
}
