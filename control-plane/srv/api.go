package srv

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

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

// Get Team by Email
func (s *Server) HandleGetTeamByEmail(w http.ResponseWriter, r *http.Request) {
	email := r.URL.Query().Get("email")
	if email == "" {
		s.jsonError(w, "email required", http.StatusBadRequest)
		return
	}
	team, err := s.Queries.GetTeamByEmail(r.Context(), email)
	if err != nil {
		s.jsonError(w, "team not found", http.StatusNotFound)
		return
	}
	s.jsonResponse(w, team)
}

// Slots API
func (s *Server) HandleListSlots(w http.ResponseWriter, r *http.Request) {
	groundID := r.URL.Query().Get("ground_id")
	municipalityID := r.URL.Query().Get("municipality_id")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 10000 {
			limit = l
		}
	}

	var query string
	var args []interface{}

	if groundID != "" {
		query = `
			SELECT s.id, s.ground_id, s.municipality_id, g.name as ground_name, m.name as municipality_name,
			       s.slot_date, s.time_from, s.time_to, s.court_name, s.scraped_at
			FROM slots s
			LEFT JOIN grounds g ON s.ground_id = g.id
			LEFT JOIN municipalities m ON s.municipality_id = m.id
			WHERE s.ground_id = ? AND s.slot_date >= date('now')
			ORDER BY s.slot_date, s.time_from
			LIMIT ?
		`
		args = []interface{}{groundID, limit}
	} else if municipalityID != "" {
		query = `
			SELECT s.id, s.ground_id, s.municipality_id, g.name as ground_name, m.name as municipality_name,
			       s.slot_date, s.time_from, s.time_to, s.court_name, s.scraped_at
			FROM slots s
			LEFT JOIN grounds g ON s.ground_id = g.id
			LEFT JOIN municipalities m ON s.municipality_id = m.id
			WHERE s.municipality_id = ? AND s.slot_date >= date('now')
			ORDER BY s.slot_date, s.time_from
			LIMIT ?
		`
		args = []interface{}{municipalityID, limit}
	} else {
		query = `
			SELECT s.id, s.ground_id, s.municipality_id, g.name as ground_name, m.name as municipality_name,
			       s.slot_date, s.time_from, s.time_to, s.court_name, s.scraped_at
			FROM slots s
			LEFT JOIN grounds g ON s.ground_id = g.id
			LEFT JOIN municipalities m ON s.municipality_id = m.id
			WHERE s.slot_date >= date('now')
			ORDER BY s.slot_date, s.time_from
			LIMIT ?
		`
		args = []interface{}{limit}
	}

	rows, err := s.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type SlotWithGround struct {
		ID               string  `json:"id"`
		GroundID         *string `json:"ground_id"`
		MunicipalityID   *string `json:"municipality_id"`
		GroundName       *string `json:"ground_name"`
		MunicipalityName *string `json:"municipality_name"`
		SlotDate         string  `json:"slot_date"`
		TimeFrom         string  `json:"time_from"`
		TimeTo           string  `json:"time_to"`
		CourtName        *string `json:"court_name"`
		ScrapedAt        string  `json:"scraped_at"`
	}
	var slots []SlotWithGround
	for rows.Next() {
		var slot SlotWithGround
		if err := rows.Scan(&slot.ID, &slot.GroundID, &slot.MunicipalityID, &slot.GroundName, &slot.MunicipalityName,
			&slot.SlotDate, &slot.TimeFrom, &slot.TimeTo, &slot.CourtName, &slot.ScrapedAt); err != nil {
			continue
		}
		slots = append(slots, slot)
	}
	s.jsonResponse(w, slots)
}

// Scrape Jobs API
func (s *Server) HandleListJobs(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT j.id, j.municipality_id, m.name as municipality_name, j.status, 
		       j.scrape_status, j.slots_found, j.error_message, j.diagnostics,
		       j.started_at, j.completed_at, j.created_at
		FROM scrape_jobs j
		LEFT JOIN municipalities m ON j.municipality_id = m.id
		ORDER BY j.created_at DESC
		LIMIT 50
	`)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Job struct {
		ID               string  `json:"id"`
		MunicipalityID   string  `json:"municipality_id"`
		MunicipalityName *string `json:"municipality_name"`
		Status           string  `json:"status"`
		ScrapeStatus     *string `json:"scrape_status"`
		SlotsFound       int     `json:"slots_found"`
		ErrorMessage     *string `json:"error_message"`
		Diagnostics      *string `json:"diagnostics"`
		StartedAt        *string `json:"started_at"`
		CompletedAt      *string `json:"completed_at"`
		CreatedAt        string  `json:"created_at"`
	}

	var jobs []Job
	for rows.Next() {
		var j Job
		if err := rows.Scan(&j.ID, &j.MunicipalityID, &j.MunicipalityName, &j.Status,
			&j.ScrapeStatus, &j.SlotsFound, &j.ErrorMessage, &j.Diagnostics,
			&j.StartedAt, &j.CompletedAt, &j.CreatedAt); err != nil {
			continue
		}
		jobs = append(jobs, j)
	}
	s.jsonResponse(w, jobs)
}

// Municipalities API
func (s *Server) HandleListMunicipalities(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT id, name, scraper_type, url, enabled, created_at
		FROM municipalities
		WHERE enabled = 1
		ORDER BY name
	`)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Municipality struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		ScraperType string `json:"scraper_type"`
		URL         string `json:"url"`
		Enabled     bool   `json:"enabled"`
		CreatedAt   string `json:"created_at"`
	}
	var municipalities []Municipality
	for rows.Next() {
		var m Municipality
		var enabled int
		if err := rows.Scan(&m.ID, &m.Name, &m.ScraperType, &m.URL, &enabled, &m.CreatedAt); err != nil {
			continue
		}
		m.Enabled = enabled == 1
		municipalities = append(municipalities, m)
	}
	s.jsonResponse(w, municipalities)
}

// Grounds API
func (s *Server) HandleListGrounds(w http.ResponseWriter, r *http.Request) {
	municipalityID := r.URL.Query().Get("municipality_id")

	var query string
	var args []interface{}

	if municipalityID != "" {
		query = `
			SELECT g.id, g.municipality_id, m.name as municipality_name, g.name, g.court_pattern, g.enabled, g.created_at
			FROM grounds g
			JOIN municipalities m ON g.municipality_id = m.id
			WHERE g.municipality_id = ? AND g.enabled = 1
			ORDER BY g.name
		`
		args = []interface{}{municipalityID}
	} else {
		query = `
			SELECT g.id, g.municipality_id, m.name as municipality_name, g.name, g.court_pattern, g.enabled, g.created_at
			FROM grounds g
			JOIN municipalities m ON g.municipality_id = m.id
			WHERE g.enabled = 1
			ORDER BY m.name, g.name
		`
	}

	rows, err := s.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Ground struct {
		ID               string  `json:"id"`
		MunicipalityID   string  `json:"municipality_id"`
		MunicipalityName string  `json:"municipality_name"`
		Name             string  `json:"name"`
		CourtPattern     *string `json:"court_pattern"`
		Enabled          bool    `json:"enabled"`
		CreatedAt        string  `json:"created_at"`
	}
	var grounds []Ground
	for rows.Next() {
		var g Ground
		var enabled int
		if err := rows.Scan(&g.ID, &g.MunicipalityID, &g.MunicipalityName, &g.Name, &g.CourtPattern, &enabled, &g.CreatedAt); err != nil {
			continue
		}
		g.Enabled = enabled == 1
		grounds = append(grounds, g)
	}
	s.jsonResponse(w, grounds)
}

// Plan Limits API
func (s *Server) HandleGetPlanLimits(w http.ResponseWriter, r *http.Request) {
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT plan, max_grounds, weekend_only, max_conditions_per_ground, notification_priority
		FROM plan_limits
	`)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type PlanLimit struct {
		Plan                   string `json:"plan"`
		MaxGrounds             int    `json:"max_grounds"`
		WeekendOnly            bool   `json:"weekend_only"`
		MaxConditionsPerGround int    `json:"max_conditions_per_ground"`
		NotificationPriority   int    `json:"notification_priority"`
	}
	var limits []PlanLimit
	for rows.Next() {
		var l PlanLimit
		var weekendOnly int
		if err := rows.Scan(&l.Plan, &l.MaxGrounds, &weekendOnly, &l.MaxConditionsPerGround, &l.NotificationPriority); err != nil {
			continue
		}
		l.WeekendOnly = weekendOnly == 1
		limits = append(limits, l)
	}
	s.jsonResponse(w, limits)
}

// Facilities API (legacy)
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

// Watch Conditions API
func (s *Server) HandleListConditions(w http.ResponseWriter, r *http.Request) {
	teamID := r.URL.Query().Get("team_id")
	if teamID == "" {
		s.jsonError(w, "team_id required", http.StatusBadRequest)
		return
	}
	conditions, err := s.Queries.ListWatchConditionsByTeam(r.Context(), teamID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, conditions)
}

func (s *Server) HandleCreateCondition(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TeamID     string `json:"team_id"`
		FacilityID string `json:"facility_id"`
		DaysOfWeek string `json:"days_of_week"`
		TimeFrom   string `json:"time_from"`
		TimeTo     string `json:"time_to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	condition, err := s.Queries.CreateWatchCondition(r.Context(), dbgen.CreateWatchConditionParams{
		ID:         uuid.New().String(),
		TeamID:     req.TeamID,
		FacilityID: req.FacilityID,
		DaysOfWeek: req.DaysOfWeek,
		TimeFrom:   req.TimeFrom,
		TimeTo:     req.TimeTo,
	})
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, condition)
}

func (s *Server) HandleDeleteCondition(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		s.jsonError(w, "id required", http.StatusBadRequest)
		return
	}
	err := s.Queries.DeleteWatchCondition(r.Context(), id)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.jsonResponse(w, map[string]bool{"success": true})
}

// Notifications API (for users)
func (s *Server) HandleListNotifications(w http.ResponseWriter, r *http.Request) {
	teamID := r.URL.Query().Get("team_id")
	if teamID == "" {
		s.jsonError(w, "team_id required", http.StatusBadRequest)
		return
	}

	// Custom query with slot and ground info
	rows, err := s.DB.QueryContext(r.Context(), `
		SELECT 
			n.id, n.team_id, n.watch_condition_id, n.slot_id, n.channel, n.status, n.sent_at, n.created_at,
			COALESCE(g.name, '') as facility_name,
			COALESCE(s.slot_date, '') as slot_date, 
			COALESCE(s.time_from, '') as slot_time_from, 
			COALESCE(s.time_to, '') as slot_time_to, 
			COALESCE(s.court_name, '') as court_name
		FROM notifications n
		LEFT JOIN slots s ON n.slot_id = s.id
		LEFT JOIN grounds g ON s.ground_id = g.id
		WHERE n.team_id = ?
		ORDER BY n.created_at DESC
		LIMIT 50
	`, teamID)
	if err != nil {
		s.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type NotificationWithSlot struct {
		ID               string  `json:"id"`
		TeamID           string  `json:"team_id"`
		WatchConditionID string  `json:"watch_condition_id"`
		SlotID           string  `json:"slot_id"`
		Channel          string  `json:"channel"`
		Status           string  `json:"status"`
		SentAt           *string `json:"sent_at"`
		CreatedAt        string  `json:"created_at"`
		FacilityName     string  `json:"facility_name"`
		SlotDate         string  `json:"slot_date"`
		SlotTime         string  `json:"slot_time"`
		CourtName        string  `json:"court_name"`
	}

	var notifications []NotificationWithSlot
	for rows.Next() {
		var n NotificationWithSlot
		var timeFrom, timeTo string
		if err := rows.Scan(&n.ID, &n.TeamID, &n.WatchConditionID, &n.SlotID, &n.Channel, &n.Status, &n.SentAt, &n.CreatedAt,
			&n.FacilityName, &n.SlotDate, &timeFrom, &timeTo, &n.CourtName); err != nil {
			continue
		}
		n.SlotTime = timeFrom + " - " + timeTo
		notifications = append(notifications, n)
	}
	if notifications == nil {
		notifications = []NotificationWithSlot{}
	}
	s.jsonResponse(w, notifications)
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
		TeamID:  sql.NullString{},
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
		Message string        `json:"message"`
		History []ChatMessage `json:"history"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Try real AI first
	aiClient := NewAIClient()
	if aiClient != nil && aiClient.IsConfigured() {
		response, err := aiClient.Chat(r.Context(), req.Message, req.History)
		if err != nil {
			slog.Warn("AI chat error, falling back to FAQ", "error", err)
			response = s.generateFallbackResponse(req.Message)
		}
		s.jsonResponse(w, map[string]interface{}{
			"response":   response,
			"ai_powered": true,
		})
		return
	}

	// Fallback to simple FAQ
	response := s.generateFallbackResponse(req.Message)
	s.jsonResponse(w, map[string]interface{}{
		"response":   response,
		"ai_powered": false,
	})
}

func (s *Server) generateFallbackResponse(message string) string {
	// Simple FAQ-based response
	faqs := map[string]string{
		"料金":   "AkiGuraには4つのプランがあります:\n- Free: 無料、1施設まで\n- Personal: ¥500/月、5施設まで\n- Pro: ¥2,000/月、20施設まで\n- Org: ¥10,000/月、無制限",
		"プラン":  "AkiGuraには4つのプランがあります:\n- Free: 無料、1施設まで\n- Personal: ¥500/月、5施設まで\n- Pro: ¥2,000/月、20施設まで\n- Org: ¥10,000/月、無制限",
		"通知":   "空き枠が見つかると、メールまたはLINEで即座に通知します。通知設定は監視条件ごとに変更できます。",
		"監視":   "監視条件では、施設・曜日・時間帯を指定できます。条件にマッチする空きが出た時点で通知されます。",
		"解約":   "解約はいつでも可能です。設定画面から「プラン変更」→「解約」を選択してください。",
		"施設追加": "新しい施設のサポートをご希望の場合は、施設名と予約サイトのURLをお知らせください。",
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

// HandleTriggerScrape triggers a scrape job for a municipality or all municipalities
func (s *Server) HandleTriggerScrape(w http.ResponseWriter, r *http.Request) {
	municipalityID := r.URL.Query().Get("municipality_id")

	// Get municipalities to scrape
	var municipalities []struct {
		ID          string
		Name        string
		ScraperType string
	}

	if municipalityID != "" {
		row := s.DB.QueryRowContext(r.Context(), `
			SELECT id, name, scraper_type FROM municipalities WHERE id = ? AND enabled = 1
		`, municipalityID)
		var m struct {
			ID          string
			Name        string
			ScraperType string
		}
		if err := row.Scan(&m.ID, &m.Name, &m.ScraperType); err != nil {
			s.jsonError(w, "Municipality not found", http.StatusNotFound)
			return
		}
		municipalities = append(municipalities, m)
	} else {
		rows, err := s.DB.QueryContext(r.Context(), `
			SELECT id, name, scraper_type FROM municipalities WHERE enabled = 1
		`)
		if err != nil {
			s.jsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var m struct {
				ID          string
				Name        string
				ScraperType string
			}
			if err := rows.Scan(&m.ID, &m.Name, &m.ScraperType); err != nil {
				continue
			}
			municipalities = append(municipalities, m)
		}
	}

	if len(municipalities) == 0 {
		s.jsonError(w, "No municipalities to scrape", http.StatusBadRequest)
		return
	}

	// Create jobs for each municipality
	jobCount := 0
	for _, m := range municipalities {
		jobID := uuid.New().String()
		_, err := s.DB.ExecContext(r.Context(), `
			INSERT INTO scrape_jobs (id, municipality_id, status, created_at)
			VALUES (?, ?, 'pending', CURRENT_TIMESTAMP)
		`, jobID, m.ID)
		if err != nil {
			slog.Warn("Failed to create scrape job", "municipality", m.Name, "error", err)
			continue
		}
		jobCount++
	}

	s.jsonResponse(w, map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Created %d scrape job(s). Jobs will be processed by the worker.", jobCount),
		"jobs":    jobCount,
	})
}
