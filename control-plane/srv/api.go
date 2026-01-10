package srv

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"srv.exe.dev/db/dbgen"
)

func (s *Server) jsonResponse(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("json encode", "error", err)
	}
}

func (s *Server) jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// Dashboard API
func (s *Server) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := DashboardData{}

	if count, err := s.Queries.CountTeams(ctx); err == nil {
		data.TeamCount = count
	}
	if count, err := s.Queries.CountFacilities(ctx); err == nil {
		data.FacilityCount = count
	}
	if count, err := s.Queries.CountWatchConditions(ctx); err == nil {
		data.WatchConditionCount = count
	}
	if count, err := s.Queries.CountNotifications(ctx); err == nil {
		data.NotificationCount = count
	}
	if tickets, err := s.Queries.ListOpenSupportTickets(ctx); err == nil {
		data.OpenTicketCount = int64(len(tickets))
	}
	if plans, err := s.Queries.CountTeamsByPlan(ctx); err == nil {
		for _, p := range plans {
			data.TeamsByPlan = append(data.TeamsByPlan, PlanCount{Plan: p.Plan, Count: p.Count})
		}
	}
	if jobs, err := s.Queries.ListRecentScrapeJobs(ctx, 10); err == nil {
		data.RecentJobs = jobs
	}

	s.jsonResponse(w, data)
}

// Teams API
func (s *Server) HandleListTeams(w http.ResponseWriter, r *http.Request) {
	teams, err := s.Queries.ListTeams(r.Context(), dbgen.ListTeamsParams{Limit: 100, Offset: 0})
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, teams)
}

func (s *Server) HandleCreateTeam(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Plan  string `json:"plan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Plan == "" {
		req.Plan = "free"
	}
	team, err := s.Queries.CreateTeam(r.Context(), dbgen.CreateTeamParams{
		ID:    uuid.New().String(),
		Name:  req.Name,
		Email: req.Email,
		Plan:  req.Plan,
	})
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, team)
}

// Facilities API
func (s *Server) HandleListFacilities(w http.ResponseWriter, r *http.Request) {
	facilities, err := s.Queries.ListFacilities(r.Context(), dbgen.ListFacilitiesParams{Limit: 100, Offset: 0})
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, facilities)
}

func (s *Server) HandleCreateFacility(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		Municipality string `json:"municipality"`
		ScraperType  string `json:"scraper_type"`
		URL          string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	facility, err := s.Queries.CreateFacility(r.Context(), dbgen.CreateFacilityParams{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Municipality: req.Municipality,
		ScraperType:  req.ScraperType,
		Url:          req.URL,
	})
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, facility)
}

// Support Tickets API
func (s *Server) HandleListTickets(w http.ResponseWriter, r *http.Request) {
	tickets, err := s.Queries.ListSupportTickets(r.Context(), dbgen.ListSupportTicketsParams{Limit: 100, Offset: 0})
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, tickets)
}

func (s *Server) HandleCreateTicket(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email   string `json:"email"`
		Subject string `json:"subject"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	ticketID := uuid.New().String()
	ticket, err := s.Queries.CreateSupportTicket(r.Context(), dbgen.CreateSupportTicketParams{
		ID:      ticketID,
		TeamID:  nil,
		Email:   req.Email,
		Subject: req.Subject,
	})
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Add initial message
	_, _ = s.Queries.CreateSupportMessage(r.Context(), dbgen.CreateSupportMessageParams{
		ID:       uuid.New().String(),
		TicketID: ticketID,
		Role:     "user",
		Content:  req.Message,
	})
	s.jsonResponse(w, ticket)
}

// AI Chat Handler
func (s *Server) HandleAIChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// AI response (builtin FAQ for now)
	response := s.generateAIResponse(req.Message)
	s.jsonResponse(w, map[string]string{"response": response})
}

func (s *Server) generateAIResponse(message string) string {
	// Simple FAQ-based response
	faqs := map[string]string{
		"料金":     "AkiGuraには4つのプランがあります:\n- Free: 無料、1施設まで\n- Personal: ¥500/月、5施設まで\n- Pro: ¥2,000/月、20施設まで\n- Org: ¥10,000/月、無制限",
		"プラン":    "AkiGuraには4つのプランがあります:\n- Free: 無料、1施設まで\n- Personal: ¥500/月、5施設まで\n- Pro: ¥2,000/月、20施設まで\n- Org: ¥10,000/月、無制限",
		"通知":     "空き枠が見つかると、メールまたはLINEで即座に通知します。通知設定は監視条件ごとに変更できます。",
		"監視":     "監視条件では、施設・曜日・時間帯を指定できます。条件にマッチする空きが出た時点で通知されます。",
		"解約":     "解約はいつでも可能です。設定画面から「プラン変更」→「解約」を選択してください。",
		"施設追加":   "新しい施設のサポートをご希望の場合は、施設名と予約サイトのURLをお知らせください。",
	}

	for keyword, response := range faqs {
		if containsKeyword(message, keyword) {
			return response
		}
	}

	return "お問い合わせありがとうございます。具体的なご質問があればお聞かせください。\n\nよくある質問:\n- 料金プランについて\n- 通知設定について\n- 監視条件の設定方法\n- 施設の追加リクエスト"
}

func containsKeyword(s, keyword string) bool {
	return len(s) > 0 && len(keyword) > 0 && (s == keyword || len(s) >= len(keyword) && (s[:len(keyword)] == keyword || s[len(s)-len(keyword):] == keyword || findSubstring(s, keyword)))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
