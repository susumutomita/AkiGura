package srv

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"srv.exe.dev/db"
	"srv.exe.dev/db/dbgen"
	"srv.exe.dev/ent"
)

type Server struct {
	DB           *sql.DB
	Queries      *dbgen.Queries
	Ent          *ent.Client // Ent ORM client
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

	// Initialize Ent client
	entClient, err := db.OpenEnt(context.Background(), dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to init ent: %w", err)
	}
	srv.Ent = entClient

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

	// Admin panel mux (protected by Basic Auth)
	adminMux := http.NewServeMux()
	adminMux.HandleFunc("GET /{$}", s.HandleRoot)
	adminMux.HandleFunc("GET /api/dashboard", s.HandleDashboard)
	adminMux.HandleFunc("GET /api/teams", s.HandleListTeams)
	adminMux.HandleFunc("POST /api/teams", s.HandleCreateTeam)
	adminMux.HandleFunc("DELETE /api/teams/{id}", s.HandleDeleteTeam)
	adminMux.HandleFunc("GET /api/facilities", s.HandleListFacilities)
	adminMux.HandleFunc("POST /api/facilities", s.HandleCreateFacility)
	adminMux.HandleFunc("GET /api/conditions", s.HandleListConditions)
	adminMux.HandleFunc("DELETE /api/conditions/{id}", s.HandleDeleteCondition)
	adminMux.HandleFunc("GET /api/notifications", s.HandleListNotifications)
	adminMux.HandleFunc("GET /api/slots", s.HandleListSlots)
	adminMux.HandleFunc("GET /api/jobs", s.HandleListJobs)
	adminMux.HandleFunc("POST /api/scrape", s.HandleTriggerScrape)
	adminMux.HandleFunc("GET /api/municipalities", s.HandleListMunicipalities)
	adminMux.HandleFunc("GET /api/grounds", s.HandleListGrounds)
	adminMux.HandleFunc("GET /api/tickets", s.HandleListTickets)
	adminMux.HandleFunc("POST /api/tickets", s.HandleCreateTicket)
	adminMux.HandleFunc("POST /api/chat", s.HandleAIChat)

	// Mount admin routes with Basic Auth at /admin prefix
	mux.Handle("/admin/", http.StripPrefix("/admin", basicAuthMiddleware(adminMux)))

	// Public pages (no auth)
	mux.HandleFunc("GET /user", s.HandleUserPage)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(s.StaticDir))))

	// User API endpoints (authenticated via session/JWT, not Basic Auth)
	mux.HandleFunc("GET /api/teams/by-email", s.HandleGetTeamByEmail)
	mux.HandleFunc("POST /api/conditions", s.HandleCreateCondition)
	mux.HandleFunc("GET /api/plan-limits", s.HandleGetPlanLimits)

	// Billing API (user-facing)
	mux.HandleFunc("GET /api/plans", s.HandlePlans)
	mux.HandleFunc("POST /api/billing/checkout", s.HandleCreateCheckout)
	mux.HandleFunc("POST /api/billing/portal", s.HandleBillingPortal)
	mux.HandleFunc("POST /api/billing/webhook", s.HandleStripeWebhook)
	mux.HandleFunc("POST /api/billing/validate-promo", s.HandleValidatePromoCode)

	// Auth endpoints (public)
	mux.HandleFunc("POST /api/auth/magic-link", s.HandleRequestMagicLink)
	mux.HandleFunc("GET /auth/verify", s.HandleVerifyMagicLink)
	mux.HandleFunc("GET /api/auth/config", s.HandleOAuthConfig)
	mux.HandleFunc("GET /auth/google", s.HandleGoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", s.HandleGoogleCallback)

	// Redirect root to admin for convenience
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
	})

	slog.Info("starting server", "addr", addr)
	return http.ListenAndServe(addr, mux)
}

// basicAuthMiddleware wraps a handler with HTTP Basic Authentication.
// Credentials are read from environment variables ADMIN_USER and ADMIN_PASS.
// If either is not set, authentication is disabled.
func basicAuthMiddleware(next http.Handler) http.Handler {
	user := os.Getenv("ADMIN_USER")
	pass := os.Getenv("ADMIN_PASS")

	// If credentials not set, skip authentication
	if user == "" || pass == "" {
		slog.Warn("ADMIN_USER or ADMIN_PASS not set, admin panel has no authentication")
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || subtle.ConstantTimeCompare([]byte(u), []byte(user)) != 1 || subtle.ConstantTimeCompare([]byte(p), []byte(pass)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="AkiGura Admin"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// mainDomainFromHost extracts the main domain from a host string by removing the first subdomain.
// For example: "example.exe.cloud:8080" returns "exe.cloud:8080"
func mainDomainFromHost(host string) string {
	// Split host and port
	hostPart, portPart := host, ""
	if colonIdx := strings.LastIndex(host, ":"); colonIdx > 0 {
		// Ensure colon is after any dots (not IPv6)
		if dotIdx := strings.LastIndex(host, "."); colonIdx > dotIdx {
			hostPart = host[:colonIdx]
			portPart = host[colonIdx:]
		}
	}

	// Find the first dot and return everything after it
	if dotIdx := strings.Index(hostPart, "."); dotIdx >= 0 {
		return hostPart[dotIdx+1:] + portPart
	}
	return host
}
