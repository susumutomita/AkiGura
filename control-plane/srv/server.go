package srv

import (
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime"

	"srv.exe.dev/db"
	"srv.exe.dev/db/dbgen"
)

type Server struct {
	DB           *sql.DB
	Queries      *dbgen.Queries
	Hostname     string
	TemplatesDir string
	StaticDir    string
}

type DashboardData struct {
	TeamCount           int64
	FacilityCount       int64
	WatchConditionCount int64
	NotificationCount   int64
	OpenTicketCount     int64
	TeamsByPlan         []PlanCount
	RecentJobs          []dbgen.ListRecentScrapeJobsRow
}

type PlanCount struct {
	Plan  string `json:"plan"`
	Count int64  `json:"count"`
}

func New(dbPath, hostname string) (*Server, error) {
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(thisFile)
	srv := &Server{
		Hostname:     hostname,
		TemplatesDir: filepath.Join(baseDir, "templates"),
		StaticDir:    filepath.Join(baseDir, "static"),
	}
	if err := srv.setUpDatabase(dbPath); err != nil {
		return nil, err
	}
	srv.Queries = dbgen.New(srv.DB)
	return srv, nil
}

func (s *Server) HandleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderTemplate(w, "index.html", nil); err != nil {
		slog.Warn("render template", "url", r.URL.Path, "error", err)
	}
}

func (s *Server) HandleUserPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.renderTemplate(w, "user.html", nil); err != nil {
		slog.Warn("render template", "url", r.URL.Path, "error", err)
	}
}

func (s *Server) renderTemplate(w http.ResponseWriter, name string, data any) error {
	path := filepath.Join(s.TemplatesDir, name)
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		return fmt.Errorf("parse template %q: %w", name, err)
	}
	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("execute template %q: %w", name, err)
	}
	return nil
}

// SetupDatabase initializes the database connection and runs migrations
func (s *Server) setUpDatabase(dbPath string) error {
	wdb, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}
	s.DB = wdb
	if err := db.RunMigrations(wdb); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	return nil
}

// Serve starts the HTTP server with the configured routes
func (s *Server) Serve(addr string) error {
	mux := http.NewServeMux()
	// Pages
	mux.HandleFunc("GET /{$}", s.HandleRoot)
	mux.HandleFunc("GET /user", s.HandleUserPage)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(s.StaticDir))))
	// API endpoints
	mux.HandleFunc("GET /api/dashboard", s.HandleDashboard)
	mux.HandleFunc("GET /api/teams", s.HandleListTeams)
	mux.HandleFunc("POST /api/teams", s.HandleCreateTeam)
	mux.HandleFunc("GET /api/teams/by-email", s.HandleGetTeamByEmail)
	mux.HandleFunc("GET /api/facilities", s.HandleListFacilities)
	mux.HandleFunc("POST /api/facilities", s.HandleCreateFacility)
	mux.HandleFunc("GET /api/conditions", s.HandleListConditions)
	mux.HandleFunc("POST /api/conditions", s.HandleCreateCondition)
	mux.HandleFunc("DELETE /api/conditions/{id}", s.HandleDeleteCondition)
	mux.HandleFunc("GET /api/notifications", s.HandleListNotifications)
	mux.HandleFunc("GET /api/slots", s.HandleListSlots)
	mux.HandleFunc("GET /api/jobs", s.HandleListJobs)
	mux.HandleFunc("POST /api/scrape", s.HandleTriggerScrape)
	mux.HandleFunc("GET /api/municipalities", s.HandleListMunicipalities)
	mux.HandleFunc("GET /api/grounds", s.HandleListGrounds)
	mux.HandleFunc("GET /api/plan-limits", s.HandleGetPlanLimits)
	mux.HandleFunc("GET /api/tickets", s.HandleListTickets)
	mux.HandleFunc("POST /api/tickets", s.HandleCreateTicket)
	mux.HandleFunc("POST /api/chat", s.HandleAIChat)
	// Billing API
	mux.HandleFunc("GET /api/plans", s.HandlePlans)
	mux.HandleFunc("POST /api/billing/checkout", s.HandleCreateCheckout)
	mux.HandleFunc("POST /api/billing/portal", s.HandleBillingPortal)
	mux.HandleFunc("POST /api/billing/webhook", s.HandleStripeWebhook)
	// Auth endpoints
	mux.HandleFunc("POST /api/auth/magic-link", s.HandleRequestMagicLink)
	mux.HandleFunc("GET /auth/verify", s.HandleVerifyMagicLink)
	slog.Info("starting server", "addr", addr)
	return http.ListenAndServe(addr, mux)
}

// mainDomainFromHost extracts the main domain from a host string by removing the first subdomain.
// For example: "example.exe.cloud:8080" returns "exe.cloud:8080"
func mainDomainFromHost(host string) string {
	// Split host and port
	hostPart := host
	portPart := ""
	if idx := len(host) - 1; idx > 0 {
		for i := idx; i >= 0; i-- {
			if host[i] == ':' {
				hostPart = host[:i]
				portPart = host[i:]
				break
			}
			if host[i] == '.' {
				break
			}
		}
	}

	// Find the first dot and return everything after it
	for i := 0; i < len(hostPart); i++ {
		if hostPart[i] == '.' {
			return hostPart[i+1:] + portPart
		}
	}
	return host
}
